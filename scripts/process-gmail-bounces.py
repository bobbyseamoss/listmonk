#!/usr/bin/env python3
"""
Process Gmail bounces and blocklist subscribers in listmonk Azure database.

This script:
1. Connects to Gmail via IMAP
2. Searches inbox and spam for bounce messages
3. Extracts bounced email addresses
4. Blocklists those subscribers in Azure PostgreSQL
"""

import imaplib
import email
from email.header import decode_header
import re
import ssl
import psycopg2
from datetime import datetime
import argparse

# Gmail IMAP settings
IMAP_SERVER = "imap.gmail.com"
IMAP_PORT = 993
IMAP_USERNAME = "adam@mail2.bobbyseamoss.com"
IMAP_PASSWORD = "xgsw scvz rdui mzhi"

# Azure PostgreSQL settings
PG_HOST = "listmonk420-db.postgres.database.azure.com"
PG_PORT = 5432
PG_USER = "listmonkadmin"
PG_PASSWORD = "T@intshr3dd3r"
PG_DATABASE = "listmonk"

# Bounce detection patterns
BOUNCE_SENDERS = [
    "mailer-daemon@",
    "postmaster@",
    "mail delivery subsystem",
    "mail delivery system",
    "returned mail",
    "undeliverable",
    "delivery status notification",
]

HARD_BOUNCE_PATTERNS = [
    r"user unknown",
    r"user not found",
    r"mailbox not found",
    r"no such user",
    r"does not exist",
    r"invalid recipient",
    r"recipient rejected",
    r"address rejected",
    r"unknown user",
    r"unknown recipient",
    r"undeliverable",
    r"permanent failure",
    r"550 5\.1\.1",  # User unknown
    r"550 5\.1\.2",  # Bad destination mailbox address
    r"550 5\.2\.1",  # Mailbox disabled
    r"551 5\.1\.1",  # User unknown
    r"552 5\.2\.2",  # Mailbox full (treat as soft bounce usually, but repeated = hard)
    r"553 5\.1\.3",  # Invalid address format
    r"no longer works here",
    r"no longer with",
    r"left the company",
    r"left the organization",
    r"is no longer",
    r"has left",
    r"employee.{0,20}(left|terminated|departed)",
    r"account.{0,20}(disabled|closed|deactivated|deleted)",
    r"mailbox.{0,20}(disabled|closed|deactivated|deleted|unavailable)",
]

# Email extraction pattern
EMAIL_PATTERN = re.compile(r'[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}', re.IGNORECASE)


def decode_mime_header(header_value):
    """Decode MIME encoded header."""
    if header_value is None:
        return ""
    decoded_parts = []
    for part, encoding in decode_header(header_value):
        if isinstance(part, bytes):
            decoded_parts.append(part.decode(encoding or 'utf-8', errors='replace'))
        else:
            decoded_parts.append(part)
    return ''.join(decoded_parts)


def get_email_body(msg):
    """Extract text body from email message."""
    body = ""
    if msg.is_multipart():
        for part in msg.walk():
            content_type = part.get_content_type()
            if content_type == "text/plain":
                try:
                    payload = part.get_payload(decode=True)
                    charset = part.get_content_charset() or 'utf-8'
                    body += payload.decode(charset, errors='replace')
                except:
                    pass
            elif content_type == "text/html" and not body:
                try:
                    payload = part.get_payload(decode=True)
                    charset = part.get_content_charset() or 'utf-8'
                    body += payload.decode(charset, errors='replace')
                except:
                    pass
    else:
        try:
            payload = msg.get_payload(decode=True)
            if payload:
                charset = msg.get_content_charset() or 'utf-8'
                body = payload.decode(charset, errors='replace')
        except:
            pass
    return body


def is_bounce_message(msg):
    """Check if message is a bounce notification."""
    sender = decode_mime_header(msg.get('From', '')).lower()
    subject = decode_mime_header(msg.get('Subject', '')).lower()

    # Check sender patterns
    for pattern in BOUNCE_SENDERS:
        if pattern in sender or pattern in subject:
            return True

    return False


def extract_bounced_email(msg):
    """Extract the bounced email address from a bounce message."""
    body = get_email_body(msg).lower()
    subject = decode_mime_header(msg.get('Subject', '')).lower()
    full_text = subject + " " + body

    # Check for hard bounce patterns
    is_hard_bounce = False
    matched_pattern = None
    for pattern in HARD_BOUNCE_PATTERNS:
        if re.search(pattern, full_text, re.IGNORECASE):
            is_hard_bounce = True
            matched_pattern = pattern
            break

    if not is_hard_bounce:
        return None, None, None

    # Extract email addresses from the message
    # Look for the original recipient email
    emails_found = EMAIL_PATTERN.findall(full_text)

    # Filter out common system addresses
    filtered_emails = []
    exclude_patterns = [
        'mailer-daemon', 'postmaster', 'mail-delivery', 'noreply',
        'no-reply', 'donotreply', 'bounce', 'return', 'daemon',
        IMAP_USERNAME.lower(), 'mail2.bobbyseamoss.com', 'bobbyseamoss.com',
        'google.com', 'gmail.com', 'googlemail.com'
    ]

    for em in emails_found:
        em_lower = em.lower()
        should_exclude = False
        for exclude in exclude_patterns:
            if exclude in em_lower:
                should_exclude = True
                break
        if not should_exclude:
            filtered_emails.append(em_lower)

    # Return the first valid email found (usually the bounced recipient)
    if filtered_emails:
        return filtered_emails[0], matched_pattern, body[:500]

    return None, None, None


def connect_imap():
    """Connect to Gmail IMAP server."""
    print(f"Connecting to {IMAP_SERVER}:{IMAP_PORT}...")

    context = ssl.create_default_context()
    mail = imaplib.IMAP4_SSL(IMAP_SERVER, IMAP_PORT, ssl_context=context)
    mail.login(IMAP_USERNAME, IMAP_PASSWORD)

    print("Connected successfully!")
    return mail


def search_folder(mail, folder_name, search_criteria="ALL"):
    """Search a folder for messages matching criteria."""
    print(f"\nSearching folder: {folder_name}")

    try:
        status, _ = mail.select(folder_name, readonly=True)
        if status != 'OK':
            print(f"  Could not select folder {folder_name}")
            return []
    except Exception as e:
        print(f"  Error selecting folder {folder_name}: {e}")
        return []

    # Search for messages
    status, message_ids = mail.search(None, search_criteria)
    if status != 'OK':
        print(f"  Search failed")
        return []

    ids = message_ids[0].split()
    print(f"  Found {len(ids)} messages")
    return ids


def process_messages(mail, message_ids, folder_name):
    """Process messages and extract bounce information."""
    bounced_emails = {}

    for i, msg_id in enumerate(message_ids):
        if i % 100 == 0 and i > 0:
            print(f"  Processed {i}/{len(message_ids)} messages...")

        try:
            status, msg_data = mail.fetch(msg_id, '(RFC822)')
            if status != 'OK':
                continue

            raw_email = msg_data[0][1]
            msg = email.message_from_bytes(raw_email)

            # Check if it's a bounce message
            if is_bounce_message(msg):
                bounced_email, pattern, excerpt = extract_bounced_email(msg)
                if bounced_email and bounced_email not in bounced_emails:
                    date_str = msg.get('Date', 'Unknown date')
                    subject = decode_mime_header(msg.get('Subject', 'No subject'))[:100]
                    bounced_emails[bounced_email] = {
                        'pattern': pattern,
                        'date': date_str,
                        'subject': subject,
                        'folder': folder_name,
                        'excerpt': excerpt
                    }
                    print(f"    Found bounce: {bounced_email} (pattern: {pattern})")

        except Exception as e:
            continue

    return bounced_emails


def blocklist_subscribers(bounced_emails, dry_run=False):
    """Blocklist subscribers in Azure PostgreSQL."""
    if not bounced_emails:
        print("\nNo bounced emails to blocklist.")
        return

    print(f"\n{'[DRY RUN] ' if dry_run else ''}Connecting to Azure PostgreSQL...")

    conn = psycopg2.connect(
        host=PG_HOST,
        port=PG_PORT,
        user=PG_USER,
        password=PG_PASSWORD,
        database=PG_DATABASE,
        sslmode='require'
    )

    cursor = conn.cursor()

    blocklisted_count = 0
    already_blocklisted = 0
    not_found = 0

    for email_addr, info in bounced_emails.items():
        # Check if subscriber exists and their current status
        cursor.execute(
            "SELECT id, status, email FROM subscribers WHERE LOWER(email) = LOWER(%s)",
            (email_addr,)
        )
        result = cursor.fetchone()

        if not result:
            print(f"  Not found in database: {email_addr}")
            not_found += 1
            continue

        sub_id, status, db_email = result

        if status == 'blocklisted':
            print(f"  Already blocklisted: {email_addr}")
            already_blocklisted += 1
            continue

        # Blocklist the subscriber
        if not dry_run:
            cursor.execute(
                "UPDATE subscribers SET status = 'blocklisted', updated_at = NOW() WHERE id = %s",
                (sub_id,)
            )

            # Add a bounce record - use json.dumps to properly escape
            import json
            meta = json.dumps({
                "pattern": info["pattern"],
                "date": info["date"],
                "folder": info["folder"],
                "email": db_email
            })
            cursor.execute("""
                INSERT INTO bounces (subscriber_id, campaign_id, type, source, meta, created_at)
                VALUES (%s, NULL, 'hard', 'gmail-imap', %s, NOW())
            """, (sub_id, meta))

        print(f"  {'[DRY RUN] Would blocklist' if dry_run else 'Blocklisted'}: {email_addr} (was: {status}, pattern: {info['pattern']})")
        blocklisted_count += 1

    if not dry_run:
        conn.commit()

    cursor.close()
    conn.close()

    print(f"\n{'[DRY RUN] ' if dry_run else ''}Summary:")
    print(f"  Blocklisted: {blocklisted_count}")
    print(f"  Already blocklisted: {already_blocklisted}")
    print(f"  Not found in database: {not_found}")
    print(f"  Total bounced emails found: {len(bounced_emails)}")


def main():
    parser = argparse.ArgumentParser(description='Process Gmail bounces and blocklist subscribers')
    parser.add_argument('--dry-run', action='store_true', help='Do not actually blocklist, just show what would be done')
    parser.add_argument('--analyze-only', action='store_true', help='Only analyze emails, do not blocklist')
    parser.add_argument('--limit', type=int, default=0, help='Limit number of messages to process per folder (0 = no limit)')
    args = parser.parse_args()

    print("=" * 60)
    print("Gmail Bounce Processor for listmonk")
    print("=" * 60)

    # Connect to IMAP
    mail = connect_imap()

    # List available folders
    print("\nListing folders...")
    status, folders = mail.list()
    if status == 'OK':
        for folder in folders[:20]:  # Show first 20 folders
            print(f"  {folder.decode()}")

    all_bounced_emails = {}

    # Search INBOX
    inbox_ids = search_folder(mail, "INBOX")
    if args.limit > 0:
        inbox_ids = inbox_ids[-args.limit:]  # Get most recent
    bounced = process_messages(mail, inbox_ids, "INBOX")
    all_bounced_emails.update(bounced)

    # Search Spam folder (Gmail uses [Gmail]/Spam)
    spam_ids = search_folder(mail, "[Gmail]/Spam")
    if args.limit > 0:
        spam_ids = spam_ids[-args.limit:]
    bounced = process_messages(mail, spam_ids, "[Gmail]/Spam")
    all_bounced_emails.update(bounced)

    # Search Sent folder for auto-replies that might indicate bounces
    sent_ids = search_folder(mail, "[Gmail]/Sent Mail")
    if args.limit > 0:
        sent_ids = sent_ids[-args.limit:]
    # For sent folder, we look for auto-replies

    # Search Trash folder
    trash_ids = search_folder(mail, "[Gmail]/Trash")
    if args.limit > 0:
        trash_ids = trash_ids[-args.limit:]
    bounced = process_messages(mail, trash_ids, "[Gmail]/Trash")
    all_bounced_emails.update(bounced)

    mail.logout()

    print(f"\n{'=' * 60}")
    print(f"Total unique bounced emails found: {len(all_bounced_emails)}")
    print("=" * 60)

    if all_bounced_emails:
        print("\nBounced emails:")
        for em, info in sorted(all_bounced_emails.items()):
            print(f"  {em}")
            print(f"    Pattern: {info['pattern']}")
            print(f"    Date: {info['date']}")
            print(f"    Folder: {info['folder']}")
            print()

    if not args.analyze_only:
        blocklist_subscribers(all_bounced_emails, dry_run=args.dry_run)
    else:
        print("\n[ANALYZE ONLY] Skipping blocklist step.")


if __name__ == "__main__":
    main()
