const fs = require('fs');

console.log('Verifying CSS fix for campaign progress bar...\n');

// Read the Campaigns.vue file
const campaignsVue = fs.readFileSync('/home/adam/listmonk/frontend/src/views/Campaigns.vue', 'utf8');

// Check for deprecated ::v-deep syntax
const hasDeprecatedSyntax = campaignsVue.includes('::v-deep .progress-wrapper');
const hasNewSyntax = campaignsVue.includes(':deep(.progress-wrapper');

console.log('CSS Syntax Check:');
console.log('================');
console.log(`✓ Deprecated ::v-deep syntax found: ${hasDeprecatedSyntax ? '❌ YES (BAD)' : '✅ NO (GOOD)'}`);
console.log(`✓ New :deep() syntax found: ${hasNewSyntax ? '✅ YES (GOOD)' : '❌ NO (BAD)'}`);

// Check the built CSS files
const cssFiles = fs.readdirSync('/home/adam/listmonk/frontend/dist/static')
  .filter(f => f.endsWith('.css') && f.includes('Campaigns'));

console.log('\nBuilt CSS Files:');
console.log('================');
cssFiles.forEach(file => {
  const content = fs.readFileSync(`/home/adam/listmonk/frontend/dist/static/${file}`, 'utf8');
  const hasProgressStyles = content.includes('.progress-wrapper') || content.includes('campaign-progress');
  console.log(`${file}: ${hasProgressStyles ? '✅ Contains progress styles' : '⚠️  No progress styles found'}`);

  if (hasProgressStyles) {
    // Look for the deep selector in the compiled CSS
    const lines = content.split('\n').filter(line =>
      line.includes('.progress') || line.includes('campaign-progress')
    );
    console.log(`  Relevant CSS rules: ${lines.length} found`);
  }
});

console.log('\n' + '='.repeat(60));
console.log('Summary:');
console.log('='.repeat(60));

if (!hasDeprecatedSyntax && hasNewSyntax) {
  console.log('✅ SUCCESS: CSS has been updated to use Vue 2.7+ compatible syntax');
  console.log('\nThe fix:');
  console.log('  - Replaced deprecated ::v-deep with :deep()');
  console.log('  - This ensures cross-browser compatibility (Chrome & Firefox)');
  console.log('\nNext steps:');
  console.log('  1. Deploy the built frontend to your server');
  console.log('  2. Test in both Chrome and Firefox');
  console.log('  3. Verify progress bars are visible in both browsers');
} else {
  console.log('❌ ISSUE: CSS still contains deprecated syntax');
}

console.log('='.repeat(60));
