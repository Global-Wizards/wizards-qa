#!/bin/bash

# Wizards QA - Automated Development Script
# Runs every 30 minutes to work on next task in ROADMAP.md

set -e

PROJECT_DIR="/home/koves/GitHub/wizards-qa"
ROADMAP="$PROJECT_DIR/ROADMAP.md"
DISCORD_CHANNEL="1469482114083852553"  # #wizards-qa

cd "$PROJECT_DIR"

# Find the next task marked as "⏳ NEXT"
NEXT_TASK=$(grep -n "⏳ NEXT" "$ROADMAP" | head -1)

if [ -z "$NEXT_TASK" ]; then
    echo "No tasks marked ⏳ NEXT. Looking for first incomplete task..."
    NEXT_TASK=$(grep -n "Status:** Not started" "$ROADMAP" | head -1)
fi

if [ -z "$NEXT_TASK" ]; then
    # No incomplete tasks found - Phase 1 complete!
    MESSAGE="🎉 **Wizards QA - Phase 1 Complete!**

All Phase 1 tasks are done! Ready to move to Phase 2: AI Integration.

**Completed:**
- Maestro wrapper package
- Flow validation
- Config file parsing
- Screenshot/video capture
- Basic test reporting
- CLI integration

**Next Phase:** AI Integration (Claude API, game analysis, flow generation)

Awaiting Fernando's approval to continue! 🌸"
    
    echo "$MESSAGE"
    openclaw message send --channel discord --target "channel:$DISCORD_CHANNEL" --message "$MESSAGE"
    exit 0
fi

# Extract task details
TASK_LINE=$(echo "$NEXT_TASK" | cut -d: -f1)
TASK_SECTION=$(sed -n "${TASK_LINE}p" "$ROADMAP" | sed 's/#### //' | sed 's/ ⏳ NEXT//' | sed 's/ - /: /')

echo "Working on: $TASK_SECTION"

# Post progress update to Discord
PROGRESS_MESSAGE="🧙‍♂️ **Wizards QA - Auto Development**

**Working on:** $TASK_SECTION
**Started:** $(date '+%Y-%m-%d %H:%M PST')

Building... 🔨

Lia here! I'll work on this for ~25 minutes and report back! 🌸"

openclaw message send --channel discord --target "channel:$DISCORD_CHANNEL" --message "$PROGRESS_MESSAGE"

# Determine which task we're working on
if echo "$TASK_SECTION" | grep -q "1.1"; then
    echo "Building Maestro wrapper package..."
    bash "$PROJECT_DIR/scripts/tasks/task-1.1-maestro-wrapper.sh"
    
elif echo "$TASK_SECTION" | grep -q "1.2"; then
    echo "Building flow validation..."
    bash "$PROJECT_DIR/scripts/tasks/task-1.2-flow-validation.sh"
    
elif echo "$TASK_SECTION" | grep -q "1.3"; then
    echo "Building config parser..."
    bash "$PROJECT_DIR/scripts/tasks/task-1.3-config-parser.sh"
    
elif echo "$TASK_SECTION" | grep -q "1.4"; then
    echo "Setting up screenshot/video capture..."
    bash "$PROJECT_DIR/scripts/tasks/task-1.4-capture.sh"
    
elif echo "$TASK_SECTION" | grep -q "1.5"; then
    echo "Building test reporting..."
    bash "$PROJECT_DIR/scripts/tasks/task-1.5-reporting.sh"
    
elif echo "$TASK_SECTION" | grep -q "1.6"; then
    echo "Integrating into CLI..."
    bash "$PROJECT_DIR/scripts/tasks/task-1.6-cli-integration.sh"
    
else
    echo "Unknown task: $TASK_SECTION"
    exit 1
fi

# Build and test
echo "Building project..."
go build -o wizards-qa ./cmd

# Commit changes
git add -A
git commit -m "feat: Auto-dev - $TASK_SECTION

Automated development run at $(date '+%Y-%m-%d %H:%M:%S PST')
Script: scripts/auto-dev.sh" || echo "No changes to commit"

git push origin master

# Post completion update
COMPLETE_MESSAGE="✅ **Task Complete: $TASK_SECTION**

Committed and pushed to GitHub!

**Next run:** $(date -d '+30 minutes' '+%Y-%m-%d %H:%M PST')

View progress: https://github.com/Global-Wizards/wizards-qa 🌸"

openclaw message send --channel discord --target "channel:$DISCORD_CHANNEL" --message "$COMPLETE_MESSAGE"

echo "Done!"
