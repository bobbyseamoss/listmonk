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

async function testWebhookFiltering() {
  console.log('\n=== Testing Webhook Logs Filtering ===\n');

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

    // Step 2: Test each filter
    const filters = ['delivered', 'failed', 'views', 'clicks'];
    const results = {};

    for (const filter of filters) {
      console.log(`2. Testing "${filter}" filter...`);

      const logsOptions = {
        hostname: 'list.bobbyseamoss.com',
        port: 443,
        path: `/api/webhook-logs?page=1&per_page=5&status_filter=${filter}`,
        method: 'GET',
        headers: {
          'Cookie': cookies
        },
        rejectUnauthorized: false
      };

      const logsResult = await makeRequest(logsOptions);
      const data = JSON.parse(logsResult.data);

      const total = data.data.total || 0;
      const resultsCount = data.data.results ? data.data.results.length : 0;

      results[filter] = {
        total: total,
        resultsCount: resultsCount
      };

      console.log(`   Total records: ${total}`);
      console.log(`   Records returned: ${resultsCount}`);

      // Show sample of first result if available
      if (data.data.results && data.data.results.length > 0) {
        const firstLog = data.data.results[0];
        try {
          const requestBody = JSON.parse(firstLog.request_body);
          if (Array.isArray(requestBody) && requestBody.length > 0) {
            const event = requestBody[0];
            console.log(`   Sample event type: ${event.eventType}`);
            if (event.data.status) {
              console.log(`   Sample status: ${event.data.status}`);
            }
            if (event.data.engagementType) {
              console.log(`   Sample engagement: ${event.data.engagementType}`);
            }
          }
        } catch (e) {
          console.log(`   (Could not parse sample)`);
        }
      }
      console.log('');
    }

    // Step 3: Verify results are different
    console.log('=== VERIFICATION RESULTS ===\n');

    const totals = Object.values(results).map(r => r.total);
    const allSame = totals.every(t => t === totals[0]);

    console.log('Filter Results:');
    for (const [filter, data] of Object.entries(results)) {
      console.log(`  ${filter.padEnd(10)}: ${data.total} total records`);
    }
    console.log('');

    if (allSame && totals[0] !== 0) {
      console.log('❌ FAILED: All filters show the same count!');
      console.log('   The filtering is NOT working.\n');
      return false;
    } else if (totals.every(t => t === 0)) {
      console.log('⚠️  WARNING: All filters returned 0 records.');
      console.log('   Either there is no webhook data, or filtering still broken.\n');
      return false;
    } else {
      console.log('✅ SUCCESS: Filters show different counts!');
      console.log('   The filtering is working correctly.\n');
      return true;
    }

  } catch (error) {
    console.error('\n❌ Error:', error.message);
    return false;
  }
}

testWebhookFiltering().then(success => {
  process.exit(success ? 0 : 1);
});
