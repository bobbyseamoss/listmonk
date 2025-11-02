#!/usr/bin/env python3
"""
Enable Azure Event Grid in Listmonk

This script connects to the listmonk PostgreSQL database and enables
Azure Event Grid webhook support in the settings.

Usage:
    python3 enable_azure_event_grid.py [options]

Options:
    --host HOST          Database host (default: localhost)
    --port PORT          Database port (default: 5432)
    --user USER          Database user (default: listmonk)
    --password PASSWORD  Database password (default: listmonk)
    --database DATABASE  Database name (default: listmonk)
    --dry-run           Show what would be done without making changes

Environment Variables:
    LISTMONK_DB_HOST
    LISTMONK_DB_PORT
    LISTMONK_DB_USER
    LISTMONK_DB_PASSWORD
    LISTMONK_DB_DATABASE

Example:
    python3 enable_azure_event_grid.py --host listmonk420-db.postgres.database.azure.com --user listmonkadmin --password 'yourpassword'
"""

import sys
import os
import argparse
import json

try:
    import psycopg2
    from psycopg2 import sql
    from psycopg2.extras import RealDictCursor
except ImportError:
    print("ERROR: psycopg2 not installed")
    print("Install it with: pip3 install psycopg2-binary")
    sys.exit(1)


def get_db_config():
    """Get database configuration from environment or defaults."""
    return {
        'host': os.getenv('LISTMONK_DB_HOST', 'localhost'),
        'port': int(os.getenv('LISTMONK_DB_PORT', '5432')),
        'user': os.getenv('LISTMONK_DB_USER', 'listmonk'),
        'password': os.getenv('LISTMONK_DB_PASSWORD', 'listmonk'),
        'database': os.getenv('LISTMONK_DB_DATABASE', 'listmonk')
    }


def enable_azure_event_grid(conn, dry_run=False):
    """Enable Azure Event Grid in settings."""
    cursor = conn.cursor(cursor_factory=RealDictCursor)

    # Check if bounce settings exist
    cursor.execute("SELECT key, value FROM settings WHERE key = 'bounce'")
    result = cursor.fetchone()

    if result:
        print("✓ Found existing bounce settings")
        current_value = result['value'] if result['value'] else {}
        print(f"  Current value: {json.dumps(current_value, indent=2)}")

        # Update to enable Azure
        new_value = current_value.copy() if isinstance(current_value, dict) else {}
        if 'azure' not in new_value:
            new_value['azure'] = {}
        new_value['azure']['enabled'] = True

        if dry_run:
            print("\n[DRY RUN] Would update bounce settings to:")
            print(f"  {json.dumps(new_value, indent=2)}")
        else:
            cursor.execute(
                """
                UPDATE settings
                SET value = %s::jsonb, updated_at = NOW()
                WHERE key = 'bounce'
                """,
                (json.dumps(new_value),)
            )
            conn.commit()
            print("\n✓ Azure Event Grid enabled!")
    else:
        print("⚠ No existing bounce settings found")

        # Create new bounce settings
        new_value = {'azure': {'enabled': True}}

        if dry_run:
            print("\n[DRY RUN] Would create bounce settings:")
            print(f"  {json.dumps(new_value, indent=2)}")
        else:
            cursor.execute(
                """
                INSERT INTO settings (key, value, created_at, updated_at)
                VALUES ('bounce', %s::jsonb, NOW(), NOW())
                """,
                (json.dumps(new_value),)
            )
            conn.commit()
            print("\n✓ Bounce settings created with Azure Event Grid enabled!")

    # Verify the setting
    cursor.execute("SELECT value FROM settings WHERE key = 'bounce'")
    result = cursor.fetchone()
    if result:
        final_value = result['value']
        azure_enabled = final_value.get('azure', {}).get('enabled', False) if isinstance(final_value, dict) else False

        print("\n" + "="*50)
        print("VERIFICATION")
        print("="*50)
        print(f"Azure Event Grid Enabled: {azure_enabled}")
        print(f"Full settings: {json.dumps(final_value, indent=2)}")

        if azure_enabled:
            print("\n" + "="*50)
            print("NEXT STEPS")
            print("="*50)
            print("1. Restart listmonk:")
            print("   sudo systemctl restart listmonk")
            print("")
            print("2. Create Event Grid subscriptions:")
            print("   cd /home/adam/listmonk/deployment/scripts")
            print("   ./azure-event-grid-setup.sh https://listmonk.yourdomain.com/webhooks/service/azure")
            print("")
            print("3. Test webhook endpoint:")
            print("   ./test-azure-webhooks.sh https://listmonk.yourdomain.com/webhooks/service/azure")
            print("="*50)
        else:
            print("\n⚠ WARNING: Azure Event Grid is still disabled")

    cursor.close()


def main():
    parser = argparse.ArgumentParser(
        description='Enable Azure Event Grid webhooks in Listmonk',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )

    parser.add_argument('--host', help='Database host')
    parser.add_argument('--port', type=int, help='Database port')
    parser.add_argument('--user', help='Database user')
    parser.add_argument('--password', help='Database password')
    parser.add_argument('--database', help='Database name')
    parser.add_argument('--dry-run', action='store_true', help='Show what would be done without making changes')

    args = parser.parse_args()

    # Get default config
    config = get_db_config()

    # Override with command line arguments
    if args.host:
        config['host'] = args.host
    if args.port:
        config['port'] = args.port
    if args.user:
        config['user'] = args.user
    if args.password:
        config['password'] = args.password
    if args.database:
        config['database'] = args.database

    print("="*50)
    print("LISTMONK AZURE EVENT GRID ENABLER")
    print("="*50)
    print(f"Database Host: {config['host']}")
    print(f"Database Port: {config['port']}")
    print(f"Database User: {config['user']}")
    print(f"Database Name: {config['database']}")
    print(f"Dry Run: {args.dry_run}")
    print("="*50)
    print()

    # Connect to database
    try:
        print("Connecting to database...")
        conn = psycopg2.connect(**config)
        print("✓ Connected successfully")
        print()

        # Enable Azure Event Grid
        enable_azure_event_grid(conn, dry_run=args.dry_run)

        conn.close()
        print("\n✓ Done!")
        return 0

    except psycopg2.OperationalError as e:
        print(f"\n✗ ERROR: Could not connect to database")
        print(f"  {e}")
        print("\nPlease check:")
        print("  1. Database host and port are correct")
        print("  2. Database user and password are correct")
        print("  3. Database is accessible from this machine")
        print("  4. PostgreSQL is running")
        return 1

    except Exception as e:
        print(f"\n✗ ERROR: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == '__main__':
    sys.exit(main())
