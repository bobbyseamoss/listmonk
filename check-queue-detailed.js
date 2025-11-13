const https = require('https');

async function makeRequest(options, postData = null) {
  return new Promise((resolve, reject) => {
    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        if (res.statusCode === 200) {
          resolve({ statusCode: res.statusCode, data: data, cookies: res.headers['set-cookie'] });
        } else {
          reject(new Error(`HTTP ${res.statusCode}: ${data}`));
        }
      });
    });

    req.on('error', (e) => {
      reject(e);
    });

    if (postData) {
      req.write(postData);
    }
    req.end();
  });
}

async function checkQueueDetailed() {
  console.log('\n=== Detailed Bobby Seamoss Queue Check ===\n');

  try {
    // Step 1: Login
    console.log('1. Logging in...');
    const loginData = JSON.stringify({
      username: 'adam',
      password: 'T@inshr3dd3r'
    });

    const loginOptions = {
      hostname: 'list.bobbyseamoss.com',
      port: 443,
      path: '/api/admin/login',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': loginData.length
      },
      rejectUnauthorized: false
    };

    const loginResult = await makeRequest(loginOptions, loginData);
    const cookies = loginResult.cookies.join('; ');
    console.log('   ✓ Login successful\n');

    // Step 2: Get queue stats
    console.log('2. Fetching queue stats...');

    const statsOptions = {
      hostname: 'list.bobbyseamoss.com',
      port: 443,
      path: '/api/queue/stats',
      method: 'GET',
      headers: {
        'Cookie': cookies
      },
      rejectUnauthorized: false
    };

    const statsResult = await makeRequest(statsOptions);
    const stats = JSON.parse(statsResult.data);

    console.log('\n=== Queue Statistics ===');
    console.log(`Total Queued:  ${stats.data.total_queued.toLocaleString()}`);
    console.log(`Total Sending: ${stats.data.total_sending.toLocaleString()}`);
    console.log(`Total Sent:    ${stats.data.total_sent.toLocaleString()}`);
    console.log(`Total Failed:  ${stats.data.total_failed.toLocaleString()}`);
    console.log('');

    // Step 3: Get running campaign stats
    console.log('3. Fetching running campaign stats...');

    const campaignsOptions = {
      hostname: 'list.bobbyseamoss.com',
      port: 443,
      path: '/api/campaigns/running/stats',
      method: 'GET',
      headers: {
        'Cookie': cookies
      },
      rejectUnauthorized: false
    };

    const campaignsResult = await makeRequest(campaignsOptions);
    const campaigns = JSON.parse(campaignsResult.data);

    console.log('\n=== Running Campaigns ===');
    if (campaigns.data && campaigns.data.length > 0) {
      campaigns.data.forEach((campaign) => {
        console.log(`\nCampaign ID ${campaign.id}: ${campaign.name}`);
        console.log(`  Status: ${campaign.status}`);
        console.log(`  Use Queue: ${campaign.use_queue}`);
        console.log(`  Queue Total: ${campaign.queue_total.toLocaleString()}`);
        console.log(`  Queue Sent: ${campaign.queue_sent.toLocaleString()}`);
        console.log(`  Azure Sent: ${campaign.azure_sent || 0}`);
        console.log(`  Queue Remaining: ${(campaign.queue_total - campaign.queue_sent).toLocaleString()}`);

        const percentComplete = ((campaign.queue_sent / campaign.queue_total) * 100).toFixed(2);
        console.log(`  Progress: ${percentComplete}%`);
      });
    } else {
      console.log('  No running campaigns');
    }

    // Step 4: Analysis
    console.log('\n=== Analysis ===');

    if (stats.data.total_sending > 10) {
      console.log(`⚠️  WARNING: ${stats.data.total_sending} emails are in "sending" status!`);
      console.log('   This is unusually high and may indicate stuck emails.');
      console.log('   Normal: 0-5 emails sending at any time');
      console.log('   Current: ' + stats.data.total_sending + ' emails sending');
      console.log('');
      console.log('Possible causes:');
      console.log('  - Emails marked as "sending" but processor crashed before marking them "sent"');
      console.log('  - Network timeouts leaving emails in limbo state');
      console.log('  - Need to reset stuck emails back to "queued" status');
    } else {
      console.log(`✓ ${stats.data.total_sending} emails currently sending (normal)`);
    }

    console.log('');

  } catch (error) {
    console.error('\n❌ Error:', error.message);
    return false;
  }
}

checkQueueDetailed();
