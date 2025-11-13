#!/usr/bin/env python3
import json
import psycopg2

def update_helo_hostnames():
    # Connect to database
    conn = psycopg2.connect(
        host='listmonk420-db.postgres.database.azure.com',
        port=5432,
        user='listmonkadmin',
        password='T@intshr3dd3r',
        database='listmonk_comma',
        sslmode='require'
    )

    try:
        cursor = conn.cursor()
        print('✓ Connected to database')

        # Get current SMTP settings
        cursor.execute("SELECT value FROM settings WHERE key = 'smtp'")
        smtp_settings = cursor.fetchone()[0]

        print(f'✓ Got {len(smtp_settings)} SMTP servers')

        # Add helo_hostname to each server
        updated_count = 0
        for server in smtp_settings:
            # Extract the mail number from the name (e.g., "email-mail10" => "10")
            name = server.get('name', '')
            if name.startswith('email-mail'):
                mail_number = name.replace('email-mail', '')
                server['helo_hostname'] = f'mail{mail_number}.enjoycomma.com'
                updated_count += 1
                print(f'  ✓ {name} => {server["helo_hostname"]}')

        print(f'\n✓ Updated {updated_count} servers with HELO hostnames')

        # Update the settings in database
        cursor.execute(
            "UPDATE settings SET value = %s, updated_at = NOW() WHERE key = 'smtp'",
            (json.dumps(smtp_settings),)
        )

        conn.commit()
        print('✓ Settings saved to database')
        print('\n⚠️  IMPORTANT: Restart the Comma container app for changes to take effect')

    except Exception as error:
        print(f'Error: {error}')
        conn.rollback()
        raise
    finally:
        cursor.close()
        conn.close()

if __name__ == '__main__':
    update_helo_hostnames()
