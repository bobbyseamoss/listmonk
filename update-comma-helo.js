const { Client } = require('pg');

async function updateHeloHostnames() {
  const client = new Client({
    host: 'listmonk420-db.postgres.database.azure.com',
    port: 5432,
    user: 'listmonkadmin',
    password: 'T@intshr3dd3r',
    database: 'listmonk_comma',
    ssl: { rejectUnauthorized: false }
  });

  try {
    await client.connect();
    console.log('✓ Connected to database');

    // Get current SMTP settings
    const result = await client.query("SELECT value FROM settings WHERE key = 'smtp'");
    const smtpSettings = result.rows[0].value;

    console.log(`✓ Got ${smtpSettings.length} SMTP servers`);

    // Add helo_hostname to each server
    let updatedCount = 0;
    smtpSettings.forEach(server => {
      // Extract the mail number from the name (e.g., "email-mail10" => "10")
      const match = server.name.match(/email-mail(\d+)/);
      if (match) {
        const mailNumber = match[1];
        server.helo_hostname = `mail${mailNumber}.enjoycomma.com`;
        updatedCount++;
        console.log(`  ✓ ${server.name} => ${server.helo_hostname}`);
      }
    });

    console.log(`\n✓ Updated ${updatedCount} servers with HELO hostnames`);

    // Update the settings in database
    await client.query(
      "UPDATE settings SET value = $1, updated_at = NOW() WHERE key = 'smtp'",
      [JSON.stringify(smtpSettings)]
    );

    console.log('✓ Settings saved to database');
    console.log('\n⚠️  IMPORTANT: Restart the Comma container app for changes to take effect');

  } catch (error) {
    console.error('Error:', error.message);
    throw error;
  } finally {
    await client.end();
  }
}

updateHeloHostnames().catch(console.error);
