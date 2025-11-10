#!/usr/bin/env node

/**
 * Direct API Verification Script
 *
 * This script directly calls the performance summary API endpoint
 * to verify the bug fix is working correctly.
 *
 * Usage:
 *   node verify-api.js <username> <password>
 *
 * Or with environment variables:
 *   LISTMONK_USERNAME=adam LISTMONK_PASSWORD=your-password node verify-api.js
 */

const https = require('https');

const BASE_URL = 'https://list.bobbyseamoss.com';
const USERNAME = process.argv[2] || process.env.LISTMONK_USERNAME || 'adam';
const PASSWORD = process.argv[3] || process.env.LISTMONK_PASSWORD;

if (!PASSWORD) {
  console.error('\n❌ Error: Password required\n');
  console.error('Usage:');
  console.error('  node verify-api.js <username> <password>');
  console.error('  LISTMONK_PASSWORD=your-password node verify-api.js');
  console.error('');
  process.exit(1);
}

// Color codes for terminal output
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
  bold: '\x1b[1m',
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

// Step 1: Login
function login() {
  return new Promise((resolve, reject) => {
    const loginData = JSON.stringify({ username: USERNAME, password: PASSWORD });

    const options = {
      hostname: 'list.bobbyseamoss.com',
      port: 443,
      path: '/api/login',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(loginData),
      },
    };

    log('\n📡 Logging in...', 'cyan');

    const req = https.request(options, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        if (res.statusCode === 200) {
          // Extract session cookie
          const cookies = res.headers['set-cookie'];
          const sessionCookie = cookies ? cookies.find(c => c.startsWith('session=')) : null;

          if (sessionCookie) {
            log('✅ Login successful', 'green');
            resolve(sessionCookie.split(';')[0]);
          } else {
            reject(new Error('No session cookie received'));
          }
        } else {
          reject(new Error(`Login failed: ${res.statusCode} ${data}`));
        }
      });
    });

    req.on('error', (error) => {
      reject(error);
    });

    req.write(loginData);
    req.end();
  });
}

// Step 2: Fetch performance summary
function fetchPerformanceSummary(sessionCookie) {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'list.bobbyseamoss.com',
      port: 443,
      path: '/api/campaigns/performance-summary',
      method: 'GET',
      headers: {
        'Cookie': sessionCookie,
      },
    };

    log('\n📊 Fetching performance summary...', 'cyan');

    const req = https.request(options, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        if (res.statusCode === 200) {
          try {
            const json = JSON.parse(data);
            resolve(json);
          } catch (error) {
            reject(new Error(`Failed to parse response: ${error.message}`));
          }
        } else {
          reject(new Error(`API call failed: ${res.statusCode} ${data}`));
        }
      });
    });

    req.on('error', (error) => {
      reject(error);
    });

    req.end();
  });
}

// Step 3: Verify the data
function verifyData(data) {
  log('\n' + '='.repeat(60), 'blue');
  log('         EMAIL PERFORMANCE SUMMARY VERIFICATION', 'bold');
  log('='.repeat(60), 'blue');

  const expectedFields = [
    { key: 'avgOpenRate', name: 'Average Open Rate', format: (v) => `${v}%` },
    { key: 'avgClickRate', name: 'Average Click Rate', format: (v) => `${v}%` },
    { key: 'orderRate', name: 'Placed Order Rate', format: (v) => `${v}%` },
    { key: 'revenuePerRecipient', name: 'Revenue Per Recipient', format: (v) => `$${v}` },
  ];

  let allPassed = true;
  const results = [];

  log('\n📋 API Response:', 'cyan');
  log(JSON.stringify(data, null, 2), 'reset');

  log('\n🔍 Verification Results:', 'cyan');
  log('-'.repeat(60));

  for (const field of expectedFields) {
    const value = data[field.key];
    const isValid = value !== undefined && value !== null;
    const isNonZero = typeof value === 'number' && value > 0;
    const passed = isValid && isNonZero;

    if (!passed) allPassed = false;

    const status = passed ? '✅ PASS' : '❌ FAIL';
    const color = passed ? 'green' : 'red';

    log(`${status} ${field.name}: ${field.format(value || 0)}`, color);

    if (!isValid) {
      log(`   ⚠️  Field "${field.key}" is missing or null`, 'yellow');
    } else if (!isNonZero) {
      log(`   ⚠️  Value is zero (expected non-zero)`, 'yellow');
    }

    results.push({ field: field.name, value, passed });
  }

  log('-'.repeat(60));

  // Additional checks
  log('\n🔍 Additional Checks:', 'cyan');
  log('-'.repeat(60));

  // Check for snake_case properties (old bug)
  const hasSnakeCase = Object.keys(data).some(key => key.includes('_'));
  if (hasSnakeCase) {
    log('❌ FAIL: Response contains snake_case properties (should be camelCase)', 'red');
    log(`   Found: ${Object.keys(data).filter(k => k.includes('_')).join(', ')}`, 'yellow');
    allPassed = false;
  } else {
    log('✅ PASS: All properties are in camelCase', 'green');
  }

  // Check data types
  let allNumbers = true;
  for (const field of expectedFields) {
    if (typeof data[field.key] !== 'number') {
      log(`❌ FAIL: ${field.name} is not a number (type: ${typeof data[field.key]})`, 'red');
      allNumbers = false;
      allPassed = false;
    }
  }
  if (allNumbers) {
    log('✅ PASS: All values are numbers', 'green');
  }

  // Check reasonable ranges
  log('\n📊 Range Validation:', 'cyan');
  log('-'.repeat(60));

  const ranges = [
    { key: 'avgOpenRate', min: 0, max: 100, name: 'Average Open Rate' },
    { key: 'avgClickRate', min: 0, max: 100, name: 'Average Click Rate' },
    { key: 'orderRate', min: 0, max: 100, name: 'Placed Order Rate' },
    { key: 'revenuePerRecipient', min: 0, max: Infinity, name: 'Revenue Per Recipient' },
  ];

  for (const range of ranges) {
    const value = data[range.key];
    if (typeof value === 'number') {
      const inRange = value >= range.min && value <= range.max;
      if (inRange) {
        log(`✅ ${range.name}: ${value} is within expected range`, 'green');
      } else {
        log(`⚠️  ${range.name}: ${value} is outside typical range [${range.min}, ${range.max}]`, 'yellow');
      }
    }
  }

  log('-'.repeat(60));

  // Final result
  log('\n' + '='.repeat(60), 'blue');
  if (allPassed) {
    log('           ✅ ALL TESTS PASSED - BUG FIX VERIFIED!', 'green');
  } else {
    log('           ❌ SOME TESTS FAILED - BUG NOT FULLY FIXED', 'red');
  }
  log('='.repeat(60) + '\n', 'blue');

  return allPassed;
}

// Main execution
async function main() {
  try {
    const sessionCookie = await login();
    const data = await fetchPerformanceSummary(sessionCookie);
    const passed = verifyData(data);

    process.exit(passed ? 0 : 1);
  } catch (error) {
    log(`\n❌ Error: ${error.message}`, 'red');
    process.exit(1);
  }
}

main();
