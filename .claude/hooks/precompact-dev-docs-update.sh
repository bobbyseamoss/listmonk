#!/bin/bash
# PreCompact hook that triggers /dev-docs-update
# This script outputs the expanded command prompt to Claude's context

# Read the dev-docs-update command file and output its contents
# Claude Code will process this as a prompt when PreCompact fires

cat "$CLAUDE_PROJECT_DIR/.claude/commands/dev-docs-update.md"
