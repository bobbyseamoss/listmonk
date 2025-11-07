#!/usr/bin/env python3
"""
Diagnostic script to check webhook configuration and bounce processing.
This script verifies that all webhook events are being processed correctly.
"""

import os
import sys
import json
import psycopg2
from datetime import datetime, timedelta

DB_HOST = "listmonk420-db.postgres.database.azure.com"
DB_PORT = 5432
DB_USER = "listmonkadmin"
DB_NAME = "listmonk"
DB_PASSWORD = "T@intshr3dd3r"

def connect_db():
    """Connect to the PostgreSQL database."""
    try:
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD,
            database=DB_NAME,
            connect_timeout=10
        )
        return conn
    except Exception as e:
        print(f"❌ Failed to connect to database: {e}")
        sys.exit(1)

def check_bounce_actions(conn):
    """Check bounce.actions configuration."""
    print("\n" + "="*60)
    print("1. BOUNCE ACTIONS CONFIGURATION")
    print("="*60)

    cursor = conn.cursor()
    cursor.execute("SELECT value FROM settings WHERE key = 'bounce.actions'")
    result = cursor.fetchone()

    if result:
        bounce_actions = json.loads(result[0])
        print("✓ bounce.actions setting found:")
        print(json.dumps(bounce_actions, indent=2))

        # Check each bounce type
        for bounce_type in ['hard', 'soft', 'complaint']:
            if bounce_type in bounce_actions:
                config = bounce_actions[bounce_type]
                action = config.get('action', 'none')
                count = config.get('count', 0)

                if action == 'blocklist':
                    print(f"  ✓ {bounce_type.upper()}: Blocklist after {count} bounces")
                elif action == 'delete':
                    print(f"  ⚠️  {bounce_type.upper()}: DELETE after {count} bounces (destructive!)")
                else:
                    print(f"  ⚠️  {bounce_type.upper()}: NO ACTION after {count} bounces")
            else:
                print(f"  ❌ {bounce_type.upper()}: Not configured!")
    else:
        print("❌ bounce.actions setting NOT FOUND!")
        print("   This is critical - bounces will not trigger blocklisting!")

    cursor.close()

def check_recent_bounces(conn):
    """Check recent bounces and their subscriber status."""
    print("\n" + "="*60)
    print("2. RECENT BOUNCES (Last 24 hours)")
    print("="*60)

    cursor = conn.cursor()

    # Get bounces from last 24 hours
    cursor.execute("""
        SELECT
            b.id,
            b.email,
            b.type,
            b.source,
            b.created_at,
            s.id as subscriber_id,
            s.status as subscriber_status,
            COUNT(*) OVER (PARTITION BY b.subscriber_id, b.type) as bounce_count
        FROM bounces b
        LEFT JOIN subscribers s ON b.subscriber_id = s.id
        WHERE b.created_at > NOW() - INTERVAL '24 hours'
        ORDER BY b.created_at DESC
        LIMIT 20
    """)

    bounces = cursor.fetchall()

    if bounces:
        print(f"Found {len(bounces)} recent bounces:")
        for bounce in bounces:
            (bid, email, btype, source, created, sub_id, sub_status, bounce_count) = bounce
            status_icon = "✓" if sub_status == "blocklisted" else "⚠️"
            print(f"  {status_icon} {email[:30]:30} | {btype:10} | {source:10} | Count: {bounce_count} | Status: {sub_status}")

            if bounce_count >= 2 and sub_status != "blocklisted" and btype in ['hard', 'complaint']:
                print(f"     🔴 ISSUE: Subscriber should be blocklisted but is {sub_status}!")
    else:
        print("No bounces in the last 24 hours")

    cursor.close()

def check_azure_events(conn):
    """Check Azure webhook events."""
    print("\n" + "="*60)
    print("3. AZURE WEBHOOK EVENTS")
    print("="*60)

    cursor = conn.cursor()

    # Check delivery events
    cursor.execute("""
        SELECT
            status,
            COUNT(*) as count,
            MAX(event_timestamp) as last_event
        FROM azure_delivery_events
        WHERE event_timestamp > NOW() - INTERVAL '24 hours'
        GROUP BY status
        ORDER BY count DESC
    """)

    delivery_events = cursor.fetchall()

    if delivery_events:
        print("\nDelivery Events (Last 24 hours):")
        total = sum(e[1] for e in delivery_events)
        for status, count, last_event in delivery_events:
            print(f"  {status:15} : {count:5} ({count*100/total:.1f}%)")
    else:
        print("⚠️  No delivery events in last 24 hours")

    # Check engagement events
    cursor.execute("""
        SELECT
            engagement_type,
            COUNT(*) as count,
            MAX(event_timestamp) as last_event
        FROM azure_engagement_events
        WHERE event_timestamp > NOW() - INTERVAL '24 hours'
        GROUP BY engagement_type
    """)

    engagement_events = cursor.fetchall()

    if engagement_events:
        print("\nEngagement Events (Last 24 hours):")
        for eng_type, count, last_event in engagement_events:
            print(f"  {eng_type:10} : {count:5}")
    else:
        print("⚠️  No engagement events in last 24 hours")

    cursor.close()

def check_campaign_stats(conn):
    """Check if campaign stats match Azure events."""
    print("\n" + "="*60)
    print("4. CAMPAIGN STATS vs AZURE EVENTS")
    print("="*60)

    cursor = conn.cursor()

    # Get recent campaigns with stats
    cursor.execute("""
        SELECT
            c.id,
            c.name,
            c.views,
            c.clicks,
            (SELECT COUNT(*) FROM campaign_views cv WHERE cv.campaign_id = c.id) as actual_views,
            (SELECT COUNT(*) FROM link_clicks lc WHERE lc.campaign_id = c.id) as actual_clicks,
            (SELECT COUNT(*) FROM azure_engagement_events ae WHERE ae.campaign_id = c.id AND ae.engagement_type = 'view') as azure_views,
            (SELECT COUNT(*) FROM azure_engagement_events ae WHERE ae.campaign_id = c.id AND ae.engagement_type = 'click') as azure_clicks
        FROM campaigns c
        WHERE c.status IN ('running', 'finished')
        AND c.updated_at > NOW() - INTERVAL '7 days'
        ORDER BY c.updated_at DESC
        LIMIT 10
    """)

    campaigns = cursor.fetchall()

    if campaigns:
        print("\nRecent Campaigns:")
        print(f"{'Campaign':<20} | Views (camp/actual/azure) | Clicks (camp/actual/azure)")
        print("-" * 80)
        for row in campaigns:
            (cid, name, views, clicks, actual_views, actual_clicks, azure_views, azure_clicks) = row
            name_short = (name[:17] + '...') if len(name) > 20 else name

            views_match = "✓" if views == actual_views else "⚠️"
            clicks_match = "✓" if clicks == actual_clicks else "⚠️"

            print(f"{name_short:<20} | {views_match} {views}/{actual_views}/{azure_views:<10} | {clicks_match} {clicks}/{actual_clicks}/{azure_clicks}")

            if actual_views != azure_views:
                print(f"  🔴 View mismatch: campaign_views={actual_views}, azure_events={azure_views}")
            if actual_clicks != azure_clicks:
                print(f"  🔴 Click mismatch: link_clicks={actual_clicks}, azure_events={azure_clicks}")
    else:
        print("No recent campaigns found")

    cursor.close()

def check_webhook_logs(conn):
    """Check recent webhook logs for errors."""
    print("\n" + "="*60)
    print("5. RECENT WEBHOOK LOGS")
    print("="*60)

    cursor = conn.cursor()

    cursor.execute("""
        SELECT
            webhook_type,
            event_type,
            processed,
            error_message,
            created_at
        FROM webhook_logs
        WHERE created_at > NOW() - INTERVAL '24 hours'
        ORDER BY created_at DESC
        LIMIT 20
    """)

    logs = cursor.fetchall()

    if logs:
        print(f"\nFound {len(logs)} recent webhook logs:")
        for log in logs:
            (webhook_type, event_type, processed, error_msg, created) = log
            status_icon = "✓" if processed else "❌"
            print(f"  {status_icon} {webhook_type:10} | {event_type:50} | {created}")
            if error_msg:
                print(f"     Error: {error_msg}")
    else:
        print("No webhook logs in last 24 hours")

    cursor.close()

def main():
    """Main diagnostic function."""
    print("\n" + "="*60)
    print("LISTMONK WEBHOOK DIAGNOSTIC TOOL")
    print("="*60)

    conn = connect_db()
    print("✓ Connected to database")

    try:
        check_bounce_actions(conn)
        check_recent_bounces(conn)
        check_azure_events(conn)
        check_campaign_stats(conn)
        check_webhook_logs(conn)

        print("\n" + "="*60)
        print("DIAGNOSTIC COMPLETE")
        print("="*60)
        print("\nRecommendations:")
        print("1. If bounce.actions is missing/misconfigured, update via Settings UI")
        print("2. If bounces are not blocklisting, check bounce count threshold")
        print("3. If stats don't match, webhooks may not be processing correctly")
        print("4. Check webhook_logs for specific error messages")

    finally:
        conn.close()

if __name__ == "__main__":
    main()
