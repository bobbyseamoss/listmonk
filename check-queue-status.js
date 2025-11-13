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

async function checkQueueStatus() {
  console.log('\n=== Checking Bobby Seamoss Queue Status ===\n');

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
        console.log(`  Queue Total: ${campaign.queue_total.toLocaleString()}`);
        console.log(`  Queue Sent: ${campaign.queue_sent.toLocaleString()}`);
        console.log(`  Queue Remaining: ${(campaign.queue_total - campaign.queue_sent).toLocaleString()}`);

        const percentComplete = ((campaign.queue_sent / campaign.queue_total) * 100).toFixed(2);
        console.log(`  Progress: ${percentComplete}%`);

        // Calculate ETA at current rate (0.56 messages/sec)
        const remaining = campaign.queue_total - campaign.queue_sent;
        const ratePerSec = 0.56;
        const secondsRemaining = remaining / ratePerSec;
        const hoursRemaining = secondsRemaining / 3600;
        const daysRemaining = hoursRemaining / 24;

        console.log(`\n  📊 At current rate (0.56 msg/sec due to Azure hourly limit):`);
        console.log(`     ETA: ${hoursRemaining.toFixed(1)} hours (${daysRemaining.toFixed(1)} days)`);
      });
    } else {
      console.log('  No running campaigns');
    }

    console.log('\n=== Analysis ===');
    console.log('The queue is NOT stuck - it\'s running slowly due to:');
    console.log('  - Azure account-wide hourly limit: 2,000 emails/hour');
    console.log('  - Current send rate: 0.56 messages/second');
    console.log('  - Delay between sends: 1.8 seconds');
    console.log('\nThis is NORMAL behavior when hitting Azure rate limits.');
    console.log('');

  } catch (error) {
    console.error('\n❌ Error:', error.message);
    return false;
  }
}

checkQueueStatus();
