const https = require('https');

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

const loginReq = https.request(loginOptions, (res) => {
  const cookies = res.headers['set-cookie'];

  if (res.statusCode === 200 && cookies) {
    console.log('✓ Login successful\n');

    // Get webhook logs with no filter to see actual structure
    const logsOptions = {
      hostname: 'list.bobbyseamoss.com',
      port: 443,
      path: '/api/webhook-logs?page=1&per_page=10',
      method: 'GET',
      headers: {
        'Cookie': cookies.join('; ')
      },
      rejectUnauthorized: false
    };

    const logsReq = https.request(logsOptions, (logsRes) => {
      let data = '';
      logsRes.on('data', (chunk) => { data += chunk; });
      logsRes.on('end', () => {
        const result = JSON.parse(data);
        console.log('=== Sample Webhook Logs Data Structure ===\n');

        if (result.data && result.data.results && result.data.results.length > 0) {
          // Parse the JSON to see structure
          const samples = {
            delivery: null,
            engagement_view: null,
            engagement_click: null
          };

          result.data.results.forEach((log) => {
            try {
              const body = JSON.parse(log.request_body);

              if (Array.isArray(body) && body.length > 0) {
                const event = body[0];

                if (event.eventType === 'Microsoft.Communication.EmailDeliveryReportReceived' && !samples.delivery) {
                  samples.delivery = {
                    eventType: event.eventType,
                    data: event.data
                  };
                } else if (event.eventType === 'Microsoft.Communication.EmailEngagementTrackingReportReceived') {
                  const engagementType = event.data.engagementType;
                  if (engagementType === 'view' && !samples.engagement_view) {
                    samples.engagement_view = {
                      eventType: event.eventType,
                      data: event.data
                    };
                  } else if (engagementType === 'click' && !samples.engagement_click) {
                    samples.engagement_click = {
                      eventType: event.eventType,
                      data: event.data
                    };
                  }
                }
              }
            } catch (e) {
              // skip
            }
          });

          console.log('DELIVERY EVENT SAMPLE:');
          console.log(JSON.stringify(samples.delivery, null, 2));
          console.log('\n---\n');

          console.log('ENGAGEMENT VIEW EVENT SAMPLE:');
          console.log(JSON.stringify(samples.engagement_view, null, 2));
          console.log('\n---\n');

          console.log('ENGAGEMENT CLICK EVENT SAMPLE:');
          console.log(JSON.stringify(samples.engagement_click, null, 2));
          console.log('\n');

          // Now show what the JSONB query needs to match
          console.log('=== JSONB Query Patterns ===\n');

          if (samples.delivery) {
            console.log('FOR DELIVERED (status = "Delivered"):');
            console.log(`The array contains: ${JSON.stringify([{eventType: samples.delivery.eventType, data: {status: samples.delivery.data.status}}])}`);
            console.log('');
          }

          if (samples.engagement_view) {
            console.log('FOR VIEWS (engagementType = "view"):');
            console.log(`The array contains: ${JSON.stringify([{eventType: samples.engagement_view.eventType, data: {engagementType: samples.engagement_view.data.engagementType}}])}`);
            console.log('');
          }

          if (samples.engagement_click) {
            console.log('FOR CLICKS (engagementType = "click"):');
            console.log(`The array contains: ${JSON.stringify([{eventType: samples.engagement_click.eventType, data: {engagementType: samples.engagement_click.data.engagementType}}])}`);
            console.log('');
          }
        }
      });
    });

    logsReq.on('error', (e) => {
      console.error('Error:', e.message);
    });

    logsReq.end();
  } else {
    console.log('❌ Login failed');
  }
});

loginReq.on('error', (e) => {
  console.error('Error:', e.message);
});

loginReq.write(loginData);
loginReq.end();
