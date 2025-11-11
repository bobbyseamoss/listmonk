# Browser Testing Skill

## Overview

This skill provides comprehensive guidance for testing frontend fixes using Playwright across multiple environments.

## Skill Details

- **Name**: browser-testing
- **Type**: Domain skill (advisory, not blocking)
- **Enforcement**: Suggest (will appear as recommendation)
- **Priority**: Medium

## What It Provides

1. **Environment Configurations**
   - Bobby Seamoss (primary test environment)
   - Comma (secondary environment)
   - Local development

2. **Login Credentials** for all three environments

3. **Testing Best Practices**
   - Default: Test in both Chrome AND Firefox
   - Use selectors for verification
   - Capture screenshots for visual confirmation
   - Proper error handling and logging

4. **Code Templates**
   - Standard test template
   - Login flow
   - Selector-based verification
   - Screenshot capture
   - Cross-browser comparison

5. **Troubleshooting Guide**
   - Element not found issues
   - Login problems
   - Screenshot issues
   - Cross-browser differences

## Trigger Keywords

The skill automatically activates when you mention:

- **Direct keywords**: browser, test in browser, frontend fix, Chrome, Firefox, playwright
- **Related terms**: verify fix, screenshot test, DOM inspection, cross-browser
- **Intent patterns**:
  - "test in Chrome and Firefox"
  - "verify the frontend fix"
  - "take a screenshot"
  - "playwright script"

## How to Use

### Automatic Activation

The skill will automatically be suggested when you mention any trigger keywords:

```
You: "I need to test this frontend fix in Chrome"
Claude: [Skill suggestion appears]
```

### Manual Activation

You can also explicitly invoke the skill:

```
You: "Use the browser-testing skill"
```

### Expected Workflow

1. Mention need for browser testing
2. Skill is suggested automatically
3. Use the Skill tool to load guidance
4. Follow the templates and best practices
5. Test in both Chrome and Firefox
6. Verify with selectors + screenshots

## Examples

### Example 1: Testing a CSS Fix
```
User: "I fixed the progress bar CSS. Can you test it in Chrome and Firefox?"

→ browser-testing skill activates
→ Provides credentials for Bobby Seamoss
→ Shows test template
→ Reminds to test both browsers
→ Includes selector verification + screenshots
```

### Example 2: Playwright Script
```
User: "Write a playwright script to verify the campaign list loads"

→ browser-testing skill activates
→ Provides environment configs
→ Shows standard test template
→ Includes login flow and selectors
```

### Example 3: Visual Verification
```
User: "Take screenshots of the new dashboard in different browsers"

→ browser-testing skill activates
→ Shows screenshot capture techniques
→ Reminds about cross-browser testing
→ Provides full test examples
```

## Test Results

The skill has been tested with these prompts:

✅ "I need to test in browser to verify the frontend fix"
✅ "Can you verify this works in Chrome and Firefox?"
✅ "Write a playwright script to test this"

All triggers work correctly!

## Files Created

1. **SKILL.md** - Main skill content (comprehensive guide)
2. **README.md** - This file (documentation)

## Skill Content Highlights

- ✅ Under 500 lines (following best practices)
- ✅ Comprehensive environment configs
- ✅ Real code templates ready to use
- ✅ Cross-browser testing emphasis
- ✅ Bobby Seamoss as default environment
- ✅ Both selector and screenshot verification
- ✅ Troubleshooting section included

## Maintenance

To update the skill:

1. Edit `.claude/skills/browser-testing/SKILL.md` for content changes
2. Edit `.claude/skills/skill-rules.json` to modify triggers
3. Test triggers with: `echo '{"prompt":"test"}' | npx tsx .claude/hooks/skill-activation-prompt.ts`
4. Validate JSON: `jq . .claude/skills/skill-rules.json`

## Next Steps

The skill is ready to use! Try it by saying:
- "Test this fix in browser"
- "Verify this works in Chrome and Firefox"
- "Write a playwright test"

The skill will automatically suggest itself and provide all the guidance you need.
