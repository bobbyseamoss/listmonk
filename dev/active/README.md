# Active Development Tasks

This directory contains active development task documentation that persists across context resets.

## Structure

Each task has its own subdirectory with three core files:

```
dev/active/[task-name]/
├── [task-name]-plan.md      # Comprehensive implementation plan
├── [task-name]-context.md   # Technical context and decisions
├── [task-name]-tasks.md     # Checklist-style task tracker
└── HANDOFF-NOTES.md         # Quick reference for context switches
```

## File Purposes

### *-plan.md
- Executive summary
- Current state analysis
- Proposed future state
- Implementation phases with detailed tasks
- Risk assessment and mitigation
- Success metrics and timeline estimates
- Lessons learned and future considerations

### *-context.md
- Key files modified with line numbers
- Architecture decisions with rationale
- Dependencies and data flows
- Critical code patterns and examples
- Testing considerations
- Known issues and debugging tips

### *-tasks.md
- Checklist format with ✅/⏳/❌ status
- Grouped by implementation phase
- Includes acceptance criteria
- Deployment and testing checklists
- Pending work clearly marked

### HANDOFF-NOTES.md (Optional)
- Quick-start guide for new sessions
- Uncommitted changes summary
- Immediate next steps
- Deployment commands ready to run
- Blockers and waiting states

### SESSION-N-SUMMARY.md (Optional)
- Per-session work summary
- Changes made in specific session
- Rationale for decisions
- Next steps from that session

## Current Active Tasks

### queue-campaign-management/
**Status**: Mostly complete, pending deployment
**Features**:
- Campaign pause/resume fix ✅ (deployed)
- Remove Sent Subscribers feature ✅ (deployed as soft delete)
- Hard delete modification ⏳ (code ready, awaiting deployment)

**Next Steps**:
- Deploy hard delete changes when user pauses campaign
- Test campaign resume functionality
- Verify hard delete behavior in database

## Usage Guidelines

### Starting a New Task
1. Create directory: `dev/active/[task-name]/`
2. Run `/dev-docs` slash command to generate documentation
3. Keep files updated as work progresses

### During Active Development
- Update *-tasks.md frequently (mark completed items)
- Add session summaries for major milestones
- Update *-context.md when making architectural decisions
- Keep HANDOFF-NOTES.md current for easy context switching

### Before Context Reset
- Run `/dev-docs-update` to capture current state
- Ensure all uncommitted changes are documented
- Update pending tasks and blockers
- Add specific next steps

### After Context Reset
1. Read HANDOFF-NOTES.md first (quick orientation)
2. Check *-tasks.md for current status
3. Review *-context.md for technical details
4. Refer to *-plan.md for overall strategy

## Best Practices

### Keep Documentation Fresh
- Update "Last Updated" timestamps
- Mark completed tasks immediately
- Document blockers as they occur
- Capture decisions and rationale in real-time

### Be Specific
- Include file paths with line numbers
- Show exact code changes (diff format)
- Provide runnable commands
- Include expected outputs

### Think Forward
- What would you need to know after a context reset?
- What commands need to run next?
- What was being worked on when interrupted?
- What decisions would be hard to reconstruct?

### Minimize Discovery Time
- Link to related files and documentation
- Explain non-obvious design choices
- Document failed approaches (save others from repeating)
- Include database queries for verification

## Integration with Git

Documentation files are NOT typically committed to the main repository. They are:
- Stored locally in `/dev/active/`
- Added to `.gitignore` (if desired)
- Used for development workflow only
- Archived when task is complete

## Archiving Completed Tasks

When a task is fully complete and tested:
1. Move to `dev/archive/[task-name]/`
2. Add final summary with completion date
3. Document any follow-up work needed
4. Keep for future reference

## Tools and Commands

- `/dev-docs [description]` - Generate initial plan documentation
- `/dev-docs-update [context]` - Update docs before context reset
- Manual updates as work progresses

---

**Last Updated**: 2025-11-07
**Active Tasks**: 1 (queue-campaign-management)
**Archived Tasks**: 0
