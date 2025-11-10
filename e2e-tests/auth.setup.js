// @ts-check
import { test as setup, expect } from '@playwright/test';

const authFile = 'e2e-tests/.auth/user.json';

setup('authenticate', async ({ page }) => {
  // Go to the login page
  await page.goto('https://list.bobbyseamoss.com/admin/login');

  // Fill in the username
  await page.locator('input[name="username"]').fill('adam');

  // Wait for user to enter password manually
  console.log('\n===========================================');
  console.log('Please enter your password in the browser');
  console.log('and click the login button');
  console.log('===========================================\n');

  // Wait for navigation to campaigns page (successful login)
  await page.waitForURL('**/admin/campaigns', { timeout: 60000 });

  // Verify we're logged in
  await expect(page.locator('.page-header')).toContainText('Campaigns');

  // Save authentication state
  await page.context().storageState({ path: authFile });

  console.log('\n✓ Authentication successful! State saved.\n');
});
