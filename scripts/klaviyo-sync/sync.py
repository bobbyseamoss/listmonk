#!/usr/bin/env python3
"""
Klaviyo → listmonk One-Way Sync Script

Syncs:
1. Klaviyo Segments → listmonk Lists
2. Klaviyo Suppressions → listmonk Blocklist
3. Engaged Subscribers → Gmail and Non-Gmail lists (based on opens/clicks)

Designed to run as an Azure Container App Job on an hourly schedule.
"""

import os
import sys
import json
import logging
import time
from datetime import datetime, timezone
from typing import Optional

import requests
import psycopg2
from psycopg2.extras import execute_values

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)

# Klaviyo API configuration
KLAVIYO_API_BASE = "https://a.klaviyo.com/api"
KLAVIYO_API_REVISION = "2023-12-15"  # Use stable revision

# Configuration from environment variables
KLAVIYO_API_KEY = os.environ.get("KLAVIYO_API_KEY")
DB_HOST = os.environ.get("DB_HOST", "listmonk420-db.postgres.database.azure.com")
DB_PORT = os.environ.get("DB_PORT", "5432")
DB_NAME = os.environ.get("DB_NAME", "listmonk")
DB_USER = os.environ.get("DB_USER", "listmonkadmin")
DB_PASSWORD = os.environ.get("DB_PASSWORD")
DB_SSLMODE = os.environ.get("DB_SSLMODE", "require")
LIST_PREFIX = os.environ.get("LIST_PREFIX", "Klaviyo: ")
DRY_RUN = os.environ.get("DRY_RUN", "false").lower() == "true"
BATCH_SIZE = int(os.environ.get("BATCH_SIZE", "1000"))

# Engaged subscriber list IDs
ENGAGED_GMAIL_LIST_ID = int(os.environ.get("ENGAGED_GMAIL_LIST_ID", "10"))
ENGAGED_NON_GMAIL_LIST_ID = int(os.environ.get("ENGAGED_NON_GMAIL_LIST_ID", "19"))


class KlaviyoClient:
    """Client for Klaviyo API v3"""

    def __init__(self, api_key: str):
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Klaviyo-API-Key {api_key}",
            "revision": KLAVIYO_API_REVISION,
            "Accept": "application/json",
        })

    def _get_with_retry(self, url: str, params: dict = None, max_retries: int = 5) -> dict:
        """Make GET request with retry logic and rate limit handling"""
        for attempt in range(max_retries):
            try:
                response = self.session.get(url, params=params)

                if response.status_code == 429:
                    # Rate limited - wait and retry
                    retry_after = int(response.headers.get("Retry-After", 60))
                    logger.warning(f"Rate limited, waiting {retry_after}s...")
                    time.sleep(retry_after)
                    continue

                response.raise_for_status()
                return response.json()

            except requests.exceptions.RequestException as e:
                # Log response body for debugging
                if hasattr(e, 'response') and e.response is not None:
                    try:
                        error_body = e.response.text
                        logger.error(f"API Error Response: {error_body}")
                    except:
                        pass

                if attempt < max_retries - 1:
                    wait_time = 2 ** (attempt + 1)  # 2, 4, 8, 16 seconds
                    logger.warning(f"Request failed, retrying in {wait_time}s: {e}")
                    time.sleep(wait_time)
                else:
                    raise

        return {}

    def get_segments(self) -> list:
        """Get all segments"""
        segments = []
        url = f"{KLAVIYO_API_BASE}/segments"

        while url:
            data = self._get_with_retry(url)
            segments.extend(data.get("data", []))
            # Check for next page (full URL returned by Klaviyo)
            url = data.get("links", {}).get("next")

        return segments

    def get_segment_profiles(self, segment_id: str) -> list:
        """Get all profiles in a segment"""
        profiles = []
        url = f"{KLAVIYO_API_BASE}/segments/{segment_id}/profiles"

        while url:
            data = self._get_with_retry(url)
            profiles.extend(data.get("data", []))
            # Check for next page (full URL returned by Klaviyo)
            url = data.get("links", {}).get("next")

        return profiles

    def get_suppressed_profiles(self) -> list:
        """Get all suppressed profiles (unsubscribed from email)"""
        profiles = []
        url = f"{KLAVIYO_API_BASE}/profiles"
        params = {
            "filter": "equals(subscriptions.email.marketing.suppression.reason,'UNSUBSCRIBE')"
        }

        # First request with params
        data = self._get_with_retry(url, params=params)
        profiles.extend(data.get("data", []))

        # Follow pagination links
        next_url = data.get("links", {}).get("next")
        while next_url:
            data = self._get_with_retry(next_url)
            profiles.extend(data.get("data", []))
            next_url = data.get("links", {}).get("next")

        return profiles


class ListmonkDB:
    """Direct database connection to listmonk PostgreSQL"""

    def __init__(self, host: str, port: str, dbname: str, user: str, password: str, sslmode: str):
        self.conn = psycopg2.connect(
            host=host,
            port=port,
            dbname=dbname,
            user=user,
            password=password,
            sslmode=sslmode
        )
        self.conn.autocommit = False

    def close(self):
        self.conn.close()

    def get_list_by_name(self, name: str) -> Optional[dict]:
        """Get list by name"""
        with self.conn.cursor() as cur:
            cur.execute(
                "SELECT id, uuid, name FROM lists WHERE name = %s",
                (name,)
            )
            row = cur.fetchone()
            if row:
                return {"id": row[0], "uuid": row[1], "name": row[2]}
        return None

    def create_list(self, name: str, list_type: str = "public", optin: str = "single") -> int:
        """Create a new list and return its ID"""
        with self.conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO lists (uuid, name, type, optin, tags, created_at, updated_at)
                VALUES (gen_random_uuid(), %s, %s, %s, '{}', NOW(), NOW())
                RETURNING id
                """,
                (name, list_type, optin)
            )
            list_id = cur.fetchone()[0]
            self.conn.commit()
            return list_id

    def upsert_subscribers(self, subscribers: list) -> dict:
        """
        Upsert subscribers and return mapping of email -> subscriber_id

        subscribers: list of dicts with 'email' and 'name' keys

        IMPORTANT: Blocklisted users are protected - their status will NOT be
        changed to 'enabled' by this function. This prevents Klaviyo sync from
        re-enabling users who unsubscribed via listmonk.
        """
        if not subscribers:
            return {}

        email_to_id = {}
        emails_to_check = [s["email"].lower() for s in subscribers]

        with self.conn.cursor() as cur:
            # First, find which emails are already blocklisted - we'll skip these
            cur.execute(
                "SELECT email FROM subscribers WHERE email = ANY(%s) AND status = 'blocklisted'",
                (emails_to_check,)
            )
            blocklisted_emails = {row[0] for row in cur.fetchall()}

            if blocklisted_emails:
                logger.info(f"Skipping {len(blocklisted_emails)} blocklisted subscribers (protected from re-enabling)")

            # Filter out blocklisted subscribers
            filtered_subscribers = [s for s in subscribers if s["email"].lower() not in blocklisted_emails]

            if not filtered_subscribers:
                # All subscribers were blocklisted, just return their IDs
                cur.execute(
                    "SELECT id, email FROM subscribers WHERE email = ANY(%s)",
                    (emails_to_check,)
                )
                for row in cur.fetchall():
                    email_to_id[row[1]] = row[0]
                return email_to_id

            # Prepare data for bulk upsert - just email and name (non-blocklisted only)
            values = [(s["email"].lower(), s.get("name", "")) for s in filtered_subscribers]

            # Bulk upsert using execute_values
            # The WHERE clause provides additional protection against re-enabling blocklisted users
            execute_values(
                cur,
                """
                INSERT INTO subscribers (uuid, email, name, status, created_at, updated_at)
                VALUES %s
                ON CONFLICT (email) DO UPDATE SET
                    name = CASE
                        WHEN EXCLUDED.name != '' THEN EXCLUDED.name
                        ELSE subscribers.name
                    END,
                    updated_at = NOW()
                WHERE subscribers.status != 'blocklisted'
                RETURNING id, email
                """,
                values,
                template="(gen_random_uuid(), %s, %s, 'enabled', NOW(), NOW())"
            )

            for row in cur.fetchall():
                email_to_id[row[1]] = row[0]

            self.conn.commit()

        # For emails that didn't return from upsert (already existed with no changes),
        # fetch their IDs (including blocklisted ones for list membership tracking)
        missing_emails = [s["email"].lower() for s in subscribers if s["email"].lower() not in email_to_id]
        if missing_emails:
            with self.conn.cursor() as cur:
                cur.execute(
                    "SELECT id, email FROM subscribers WHERE email = ANY(%s)",
                    (missing_emails,)
                )
                for row in cur.fetchall():
                    email_to_id[row[1]] = row[0]

        return email_to_id

    def add_subscribers_to_list(self, subscriber_ids: list, list_id: int) -> int:
        """Add subscribers to a list, returns count of new additions"""
        if not subscriber_ids:
            return 0

        with self.conn.cursor() as cur:
            # Bulk insert with ON CONFLICT DO NOTHING
            values = [(sid, list_id) for sid in subscriber_ids]
            execute_values(
                cur,
                """
                INSERT INTO subscriber_lists (subscriber_id, list_id, status, created_at, updated_at)
                VALUES %s
                ON CONFLICT (subscriber_id, list_id) DO NOTHING
                """,
                values,
                template="(%s, %s, 'confirmed', NOW(), NOW())"
            )
            added = cur.rowcount
            self.conn.commit()
            return added

    def blocklist_subscribers(self, emails: list) -> int:
        """Blocklist subscribers by email, returns count of updated"""
        if not emails:
            return 0

        with self.conn.cursor() as cur:
            # First, ensure all emails exist as subscribers
            for email in emails:
                cur.execute(
                    """
                    INSERT INTO subscribers (uuid, email, name, status, created_at, updated_at)
                    VALUES (gen_random_uuid(), %s, '', 'blocklisted', NOW(), NOW())
                    ON CONFLICT (email) DO UPDATE SET
                        status = 'blocklisted',
                        updated_at = NOW()
                    WHERE subscribers.status != 'blocklisted'
                    """,
                    (email.lower(),)
                )

            self.conn.commit()
            return len(emails)

    def get_stats(self) -> dict:
        """Get subscriber statistics"""
        with self.conn.cursor() as cur:
            cur.execute("""
                SELECT status, COUNT(*)
                FROM subscribers
                GROUP BY status
            """)
            return {row[0]: row[1] for row in cur.fetchall()}

    def sync_engaged_subscribers(self, gmail_list_id: int, non_gmail_list_id: int) -> dict:
        """
        Sync engaged subscribers (those with opens/clicks) to Gmail and Non-Gmail lists.

        Args:
            gmail_list_id: List ID for engaged Gmail users
            non_gmail_list_id: List ID for engaged Non-Gmail users

        Returns:
            Dict with gmail_count and non_gmail_count
        """
        with self.conn.cursor() as cur:
            # Clear existing memberships for both lists
            cur.execute(
                "DELETE FROM subscriber_lists WHERE list_id IN (%s, %s)",
                (gmail_list_id, non_gmail_list_id)
            )
            deleted = cur.rowcount
            logger.info(f"Cleared {deleted} existing engaged list memberships")

            # Add engaged Gmail users
            cur.execute("""
                INSERT INTO subscriber_lists (subscriber_id, list_id, status, created_at, updated_at)
                SELECT DISTINCT e.subscriber_id, %s, 'confirmed'::subscription_status, NOW(), NOW()
                FROM (
                    SELECT subscriber_id FROM campaign_views WHERE subscriber_id IS NOT NULL
                    UNION
                    SELECT subscriber_id FROM link_clicks WHERE subscriber_id IS NOT NULL
                ) e
                JOIN subscribers s ON e.subscriber_id = s.id
                WHERE s.status = 'enabled'
                  AND s.email ILIKE '%%@gmail.com'
            """, (gmail_list_id,))
            gmail_count = cur.rowcount

            # Add engaged Non-Gmail users
            cur.execute("""
                INSERT INTO subscriber_lists (subscriber_id, list_id, status, created_at, updated_at)
                SELECT DISTINCT e.subscriber_id, %s, 'confirmed'::subscription_status, NOW(), NOW()
                FROM (
                    SELECT subscriber_id FROM campaign_views WHERE subscriber_id IS NOT NULL
                    UNION
                    SELECT subscriber_id FROM link_clicks WHERE subscriber_id IS NOT NULL
                ) e
                JOIN subscribers s ON e.subscriber_id = s.id
                WHERE s.status = 'enabled'
                  AND s.email NOT ILIKE '%%@gmail.com'
            """, (non_gmail_list_id,))
            non_gmail_count = cur.rowcount

            self.conn.commit()

        return {"gmail_count": gmail_count, "non_gmail_count": non_gmail_count}


def sync_segments(klaviyo: KlaviyoClient, db: ListmonkDB):
    """Sync all Klaviyo segments to listmonk lists"""
    logger.info("Fetching Klaviyo segments...")
    segments = klaviyo.get_segments()
    logger.info(f"Found {len(segments)} segments")

    total_synced = 0
    total_blocklisted = 0

    # Segments that indicate suppressed/unsubscribed users (case-insensitive)
    suppression_keywords = ['suppressed', 'unsubscribe', 'suppression']

    for segment in segments:
        segment_id = segment["id"]
        segment_name = segment["attributes"]["name"]
        list_name = f"{LIST_PREFIX}{segment_name}"

        # Check if this is a suppression segment
        is_suppression_segment = any(kw in segment_name.lower() for kw in suppression_keywords)

        logger.info(f"Processing segment: {segment_name} ({segment_id}){' [SUPPRESSION]' if is_suppression_segment else ''}")

        # Get or create corresponding list in listmonk
        listmonk_list = db.get_list_by_name(list_name)
        if not listmonk_list:
            if DRY_RUN:
                logger.info(f"[DRY RUN] Would create list: {list_name}")
                continue
            list_id = db.create_list(list_name)
            logger.info(f"Created new list: {list_name} (ID: {list_id})")
        else:
            list_id = listmonk_list["id"]
            logger.info(f"Using existing list: {list_name} (ID: {list_id})")

        # Fetch profiles in this segment
        logger.info(f"Fetching profiles for segment {segment_name}...")
        profiles = klaviyo.get_segment_profiles(segment_id)
        logger.info(f"Found {len(profiles)} profiles in segment")

        if not profiles:
            continue

        # Process in batches
        for i in range(0, len(profiles), BATCH_SIZE):
            batch = profiles[i:i + BATCH_SIZE]

            # Extract subscriber data
            subscribers = []
            emails_to_blocklist = []
            for profile in batch:
                attrs = profile.get("attributes", {})
                email = attrs.get("email")
                if email:
                    if is_suppression_segment:
                        emails_to_blocklist.append(email)
                    else:
                        subscribers.append({
                            "email": email,
                            "name": f"{attrs.get('first_name', '')} {attrs.get('last_name', '')}".strip()
                        })

            # Handle suppression segment - blocklist these users
            if is_suppression_segment and emails_to_blocklist:
                if DRY_RUN:
                    logger.info(f"[DRY RUN] Would blocklist {len(emails_to_blocklist)} subscribers from {segment_name}")
                else:
                    blocklisted = db.blocklist_subscribers(emails_to_blocklist)
                    total_blocklisted += len(emails_to_blocklist)
                    logger.info(f"Batch {i // BATCH_SIZE + 1}: Blocklisted {len(emails_to_blocklist)} subscribers from suppression segment")
                continue

            if not subscribers:
                continue

            if DRY_RUN:
                logger.info(f"[DRY RUN] Would sync {len(subscribers)} subscribers to list {list_name}")
                continue

            # Upsert subscribers
            email_to_id = db.upsert_subscribers(subscribers)

            # Add to list
            subscriber_ids = list(email_to_id.values())
            added = db.add_subscribers_to_list(subscriber_ids, list_id)

            total_synced += len(subscribers)
            logger.info(f"Batch {i // BATCH_SIZE + 1}: Synced {len(subscribers)} subscribers, {added} new list memberships")

    logger.info(f"Segment sync complete: {total_synced} synced, {total_blocklisted} blocklisted")
    return total_synced


def sync_engaged_lists(db: ListmonkDB):
    """Sync engaged subscribers to Gmail and Non-Gmail lists based on opens/clicks"""
    logger.info("Syncing engaged subscriber lists...")

    if DRY_RUN:
        logger.info("[DRY RUN] Would sync engaged subscriber lists")
        return {"gmail_count": 0, "non_gmail_count": 0}

    result = db.sync_engaged_subscribers(ENGAGED_GMAIL_LIST_ID, ENGAGED_NON_GMAIL_LIST_ID)

    logger.info(f"Engaged Gmail Users (list {ENGAGED_GMAIL_LIST_ID}): {result['gmail_count']} subscribers")
    logger.info(f"Engaged Non-Gmail (list {ENGAGED_NON_GMAIL_LIST_ID}): {result['non_gmail_count']} subscribers")

    return result


def sync_suppressions(klaviyo: KlaviyoClient, db: ListmonkDB):
    """Sync Klaviyo suppressions to listmonk blocklist"""
    logger.info("Fetching Klaviyo suppressed profiles...")

    try:
        profiles = klaviyo.get_suppressed_profiles()
    except Exception as e:
        logger.warning(f"Could not fetch suppressions (may require higher API access): {e}")
        return 0

    logger.info(f"Found {len(profiles)} suppressed profiles")

    if not profiles:
        return 0

    # Extract emails
    emails = []
    for profile in profiles:
        email = profile.get("attributes", {}).get("email")
        if email:
            emails.append(email)

    if DRY_RUN:
        logger.info(f"[DRY RUN] Would blocklist {len(emails)} subscribers")
        return len(emails)

    # Process in batches
    total_blocklisted = 0
    for i in range(0, len(emails), BATCH_SIZE):
        batch = emails[i:i + BATCH_SIZE]
        blocklisted = db.blocklist_subscribers(batch)
        total_blocklisted += blocklisted
        logger.info(f"Blocklisted batch {i // BATCH_SIZE + 1}: {len(batch)} emails")

    return total_blocklisted


def main():
    """Main sync function"""
    start_time = datetime.now(timezone.utc)
    logger.info("=" * 60)
    logger.info("Klaviyo → listmonk Sync Started")
    logger.info(f"Time: {start_time.isoformat()}")
    logger.info(f"Dry Run: {DRY_RUN}")
    logger.info("=" * 60)

    # Validate configuration
    if not KLAVIYO_API_KEY:
        logger.error("KLAVIYO_API_KEY environment variable is required")
        sys.exit(1)

    if not DB_PASSWORD:
        logger.error("DB_PASSWORD environment variable is required")
        sys.exit(1)

    # Initialize clients
    klaviyo = KlaviyoClient(KLAVIYO_API_KEY)
    db = ListmonkDB(DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSLMODE)

    try:
        # Get initial stats
        initial_stats = db.get_stats()
        logger.info(f"Initial subscriber stats: {initial_stats}")

        # Sync segments
        segments_synced = sync_segments(klaviyo, db)

        # Sync suppressions
        suppressions_synced = sync_suppressions(klaviyo, db)

        # Sync engaged subscriber lists
        engaged_result = sync_engaged_lists(db)

        # Get final stats
        final_stats = db.get_stats()
        logger.info(f"Final subscriber stats: {final_stats}")

        # Summary
        duration = (datetime.now(timezone.utc) - start_time).total_seconds()
        logger.info("=" * 60)
        logger.info("Sync Complete")
        logger.info(f"Duration: {duration:.1f} seconds")
        logger.info(f"Segments synced: {segments_synced} subscribers")
        logger.info(f"Suppressions synced: {suppressions_synced} emails")
        logger.info(f"Engaged Gmail: {engaged_result['gmail_count']} subscribers")
        logger.info(f"Engaged Non-Gmail: {engaged_result['non_gmail_count']} subscribers")
        logger.info("=" * 60)

    except Exception as e:
        logger.error(f"Sync failed: {e}", exc_info=True)
        sys.exit(1)
    finally:
        db.close()


if __name__ == "__main__":
    main()
