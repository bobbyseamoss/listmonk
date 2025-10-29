#!/usr/bin/env python3
"""
Export Configuration from Dev Database
Extracts SMTP, bounce, and app settings from local dev database
Sanitizes credentials and saves as templates for Azure deployment
"""

import json
import psycopg2
import sys
from pathlib import Path

# Database connection settings
DB_CONFIG = {
    'host': 'localhost',
    'port': 5432,
    'database': 'listmonk-dev',
    'user': 'listmonk-dev',
    'password': 'listmonk-dev'
}

# Output directory
OUTPUT_DIR = Path(__file__).parent.parent / 'configs'

def sanitize_smtp_config(smtp_servers):
    """Remove credentials from SMTP configuration"""
    sanitized = []
    credential_map = {}

    for idx, server in enumerate(smtp_servers, 1):
        server_num = idx

        # Store credentials separately for Key Vault
        if server.get('username'):
            credential_map[f'smtp-{server_num}-username'] = server['username']
        if server.get('password'):
            credential_map[f'smtp-{server_num}-password'] = server['password']

        # Create sanitized server config
        sanitized_server = {
            'uuid': server['uuid'],
            'enabled': server.get('enabled', True),
            'host': server['host'],
            'port': server.get('port', 587),
            'auth_protocol': server.get('auth_protocol', ''),
            'from_email': server.get('from_email', ''),
            'name': server.get('name', ''),
            'tls_enabled': server.get('tls_enabled', True),
            'tls_skip_verify': server.get('tls_skip_verify', False),
            'max_conns': server.get('max_conns', 10),
            'idle_timeout': server.get('idle_timeout', '15s'),
            'wait_timeout': server.get('wait_timeout', '5s'),
            'max_msg_retries': server.get('max_msg_retries', 2),
            'email_headers': server.get('email_headers', []),
            'daily_limit': server.get('daily_limit', 0),
            'sliding_window': server.get('sliding_window', False),
            'sliding_window_duration': server.get('sliding_window_duration', ''),
            'sliding_window_rate': server.get('sliding_window_rate', 0),
            'bounce_mailbox_uuid': server.get('bounce_mailbox_uuid', ''),
            # Credential references for Key Vault
            'username_secret': f'smtp-{server_num}-username',
            'password_secret': f'smtp-{server_num}-password'
        }
        sanitized.append(sanitized_server)

    return sanitized, credential_map

def sanitize_bounce_config(bounce_boxes):
    """Remove credentials from bounce mailbox configuration"""
    sanitized = []
    credential_map = {}

    for idx, box in enumerate(bounce_boxes, 1):
        box_num = idx

        # Store credentials separately for Key Vault
        if box.get('username'):
            credential_map[f'bounce-{box_num}-username'] = box['username']
        if box.get('password'):
            credential_map[f'bounce-{box_num}-password'] = box['password']

        # Create sanitized mailbox config
        sanitized_box = {
            'uuid': box['uuid'],
            'enabled': box.get('enabled', True),
            'type': box.get('type', 'pop'),
            'name': box.get('name', ''),
            'host': box['host'],
            'port': box.get('port', 995),
            'return_path': box.get('return_path', ''),
            'scan_interval': box.get('scan_interval', '15m'),
            'tls_enabled': box.get('tls_enabled', True),
            'tls_skip_verify': box.get('tls_skip_verify', False),
            # Credential references for Key Vault
            'username_secret': f'bounce-{box_num}-username',
            'password_secret': f'bounce-{box_num}-password'
        }
        sanitized.append(sanitized_box)

    return sanitized, credential_map

def export_smtp_config(conn):
    """Export SMTP server configuration"""
    print("Exporting SMTP configuration...")

    cursor = conn.cursor()
    cursor.execute("SELECT value FROM settings WHERE key = 'smtp';")
    result = cursor.fetchone()

    if not result:
        print("  No SMTP configuration found")
        return None, {}

    smtp_servers = result[0]
    sanitized_config, credentials = sanitize_smtp_config(smtp_servers)

    # Save template
    output_file = OUTPUT_DIR / 'smtp_config_template.json'
    with open(output_file, 'w') as f:
        json.dump(sanitized_config, f, indent=2)

    print(f"  Exported {len(sanitized_config)} SMTP servers to {output_file}")
    print(f"  Extracted {len(credentials)} credentials for Key Vault")

    return sanitized_config, credentials

def export_bounce_config(conn):
    """Export bounce mailbox configuration"""
    print("Exporting bounce configuration...")

    cursor = conn.cursor()
    cursor.execute("SELECT value FROM settings WHERE key = 'bounce';")
    result = cursor.fetchone()

    if not result:
        print("  No bounce configuration found")
        return None, {}

    bounce_settings = result[0]
    bounce_boxes = bounce_settings.get('mailboxes', [])

    sanitized_config, credentials = sanitize_bounce_config(bounce_boxes)

    # Save template
    output_file = OUTPUT_DIR / 'bounce_config_template.json'
    with open(output_file, 'w') as f:
        json.dump(sanitized_config, f, indent=2)

    print(f"  Exported {len(sanitized_config)} bounce mailboxes to {output_file}")
    print(f"  Extracted {len(credentials)} credentials for Key Vault")

    return sanitized_config, credentials

def export_app_settings(conn):
    """Export app performance settings"""
    print("Exporting app settings...")

    cursor = conn.cursor()
    cursor.execute("SELECT value FROM settings WHERE key = 'app';")
    result = cursor.fetchone()

    if not result:
        print("  No app settings found")
        return None

    app_settings = result[0]

    # Extract relevant settings
    relevant_settings = {
        'send_time_start': app_settings.get('send_time_start', ''),
        'send_time_end': app_settings.get('send_time_end', ''),
        'batch_size': app_settings.get('batch_size', 1000),
        'concurrency': app_settings.get('concurrency', 5),
        'message_rate': app_settings.get('message_rate', 10),
        'message_sliding_window_rate': app_settings.get('message_sliding_window_rate', 0),
        'message_sliding_window_duration': app_settings.get('message_sliding_window_duration', ''),
        'max_send_errors': app_settings.get('max_send_errors', 1000)
    }

    # Save template
    output_file = OUTPUT_DIR / 'app_settings.json'
    with open(output_file, 'w') as f:
        json.dump(relevant_settings, f, indent=2)

    print(f"  Exported app settings to {output_file}")

    return relevant_settings

def save_credential_map(smtp_creds, bounce_creds):
    """Save credential mapping for Key Vault upload"""
    print("Generating credential map...")

    all_credentials = {**smtp_creds, **bounce_creds}

    output_file = OUTPUT_DIR / 'credential_map.json'
    with open(output_file, 'w') as f:
        json.dump(all_credentials, f, indent=2)

    print(f"  Saved {len(all_credentials)} credentials to {output_file}")
    print(f"  ⚠️  WARNING: This file contains plaintext credentials!")
    print(f"  ⚠️  DO NOT commit to git! Use only for Key Vault upload, then delete.")

    return output_file

def main():
    print("=" * 70)
    print("Configuration Export Script")
    print("=" * 70)
    print()

    # Ensure output directory exists
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    try:
        # Connect to database
        print("Connecting to database...")
        conn = psycopg2.connect(**DB_CONFIG)
        print("  ✓ Connected")
        print()

        # Export configurations
        smtp_config, smtp_creds = export_smtp_config(conn)
        print()

        bounce_config, bounce_creds = export_bounce_config(conn)
        print()

        app_settings = export_app_settings(conn)
        print()

        # Save credential map
        if smtp_creds or bounce_creds:
            cred_file = save_credential_map(smtp_creds, bounce_creds)
            print()

        # Summary
        print("=" * 70)
        print("Export Summary")
        print("=" * 70)
        print(f"✓ SMTP servers: {len(smtp_config) if smtp_config else 0}")
        print(f"✓ Bounce mailboxes: {len(bounce_config) if bounce_config else 0}")
        print(f"✓ App settings: exported")
        print(f"✓ Credentials: {len(smtp_creds) + len(bounce_creds)}")
        print()
        print("Next steps:")
        print("1. Review the generated templates in deployment/configs/")
        print("2. Run upload_credentials.sh to store credentials in Azure Key Vault")
        print("3. Delete credential_map.json after upload")
        print()

        conn.close()

    except psycopg2.Error as e:
        print(f"❌ Database error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"❌ Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
