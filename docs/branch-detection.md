# Branch Detection User Guide

## Overview

TimeMachine CLI automatically tracks Git branch switches and creates descriptive snapshot messages, providing essential context for AI-assisted development sessions. This feature helps you understand what happened to your code when working across multiple branches.

## Key Benefits for AI Development

- **Context Preservation**: Know exactly when AI code changes occurred during branch switches
- **Safe Experimentation**: Clear history of which branch contained working vs broken code
- **Workflow Clarity**: Instant visual feedback about branch transitions in your snapshot history
- **Rollback Confidence**: Easily identify the right snapshot to restore from when AI breaks code

## How Branch Detection Works

TimeMachine monitors your Git branch status and automatically detects when you switch between branches. When this happens, it creates enhanced snapshot messages that show:

1. **Where you came from** (previous branch)
2. **Where you went** (current branch) 
3. **How many files changed**
4. **If it's a large change** (20+ files get a warning)

## Message Formats You'll See

### Initial Snapshots
When TimeMachine creates the very first snapshot:
```
[main] INITIAL: Auto-snapshot from file watcher
[feature-auth] INITIAL: User authentication work
```

### Branch Switch Detection
When you switch from one branch to another:
```
[main→feature] BRANCH SWITCH: Working on new feature
[feature→main] BRANCH SWITCH: Back to main branch (8 files)
[develop→hotfix] BRANCH SWITCH: Critical bug fix (25 files - LARGE CHANGES ⚠️)
```

### Regular Snapshots
Normal file changes within the same branch:
```
[feature] Auto-snapshot from file watcher
[main] Fixed login validation
```

## Real-World Example

Here's what your snapshot history might look like during an AI coding session:

```bash
$ timemachine list

📸 Recent snapshots:

abc12345  [main→feature] BRANCH SWITCH: AI suggested improvements (12 files)    2 minutes ago
def67890  [main] Working authentication before AI changes                        15 minutes ago  
ghi09876  [develop→main] BRANCH SWITCH: Merged latest changes                    1 hour ago
jkl12345  [develop] INITIAL: Started new feature branch                          2 hours ago
```

From this history, you can immediately see:
- You switched from `main` to `feature` branch 2 minutes ago
- AI made changes to 12 files during that switch
- You had working authentication 15 minutes ago on `main`
- You can safely restore to `def67890` if the AI changes broke something

## Using Branch Information for Recovery

### Scenario: AI Broke Your Code After Branch Switch

1. **Identify the problem**: You switched branches and let AI make changes, now code is broken

2. **Check your history**: 
   ```bash
   $ timemachine list
   abc12345  [main→feature] BRANCH SWITCH: AI refactoring (18 files)    5 minutes ago
   def67890  [main] Working version before switch                        20 minutes ago
   ```

3. **Restore to safety**:
   ```bash
   $ timemachine restore def67890
   ✨ Files restored successfully!
   ```

### Scenario: Find All Changes for a Specific Branch

```bash
# See all snapshots related to the 'feature-auth' branch
$ timemachine list | grep "feature-auth"

# Or use future filtering (coming soon):
$ timemachine list --branch feature-auth
```

### Scenario: Understanding Large Changes

When you see `LARGE CHANGES ⚠️` in a snapshot message:

```bash
# Get details about what changed
$ timemachine show abc12345

📸 Snapshot abc12345
🕐 Created: 2024-03-15 14:30:25
📝 Message: [main→refactor] BRANCH SWITCH: AI code restructuring (47 files - LARGE CHANGES ⚠️)

📊 File Changes:
+ src/components/NewComponent.js
+ src/utils/helpers/validation.js
~ src/App.js
~ src/components/Header.js
~ package.json
... (42 more files)

💡 Use: timemachine restore abc12345 to restore these files
```

This tells you AI made major structural changes across 47 files - proceed carefully!

## Best Practices for AI Development

### 1. Check Branch Context Before Restoring
Always look at the branch information in snapshot messages:
- `[main]` snapshots are usually stable
- `[experimental→main]` might contain risky changes  
- `[main→feature]` switches show when you started new work

### 2. Create Manual Snapshots Before Major AI Changes
```bash
# Before asking AI to refactor large sections
$ timemachine snapshot "Working version before AI refactoring"

# Then let AI make changes, knowing you can easily rollback
```

### 3. Use Branch Switches as Checkpoints
The branch switch detection creates natural restore points:
```bash
# This creates a perfect checkpoint
$ git checkout -b ai-experiment
# AI makes changes automatically captured with branch switch context
$ timemachine list  # Shows: [main→ai-experiment] BRANCH SWITCH: ...
```

### 4. Monitor File Count Warnings
- **No file count**: Small, safe changes (1-5 files)
- **File count shown**: Medium changes (6-20 files) - review before continuing
- **LARGE CHANGES ⚠️**: Major changes (20+ files) - consider creating manual snapshot first

## Troubleshooting

### Branch Detection Not Working

**Problem**: All snapshots show `[unknown→current]` format

**Solution**: 
1. Ensure you're in a valid Git repository: `git status`
2. Check TimeMachine initialization: `timemachine status`
3. Verify Git user configuration: `git config user.name` and `git config user.email`

### Missing Branch Information

**Problem**: Commit messages don't contain branch names

**Diagnosis**:
```bash
# Check current branch detection
$ git branch --show-current

# Verify main repo status  
$ git status

# Check TimeMachine shadow repo
$ git --git-dir=.git/timemachine_snapshots log --oneline -5
```

**Solution**: Re-initialize TimeMachine if issues persist:
```bash
$ rm -rf .git/timemachine_snapshots
$ timemachine init
```

### "Corrupted Data" Warnings

**Problem**: See warnings about corrupted metadata

**Impact**: System continues working normally, just shows `unknown` for some previous branches

**Resolution**: Warnings are informational - create a few new snapshots to establish clean history

## Integration with Other Commands

### List Command
Branch information appears in all snapshot listings:
```bash
$ timemachine list
$ timemachine list --limit 10
$ timemachine list --file src/app.js  # Shows branch context for file changes
```

### Show Command  
Detailed snapshots include branch context:
```bash
$ timemachine show abc12345
# Shows branch information, file changes, and restoration commands
```

### Restore Command
Use branch context to choose the right snapshot:
```bash
$ timemachine restore abc12345  # Restore from specific branch context
```

### Status Command
Shows current branch alongside TimeMachine status:
```bash
$ timemachine status
🎯 Current branch: feature-auth
📊 Shadow repository: Healthy
...
```

## Advanced Usage

### Understanding Branch Switch Patterns

Learn to read the patterns in your history:

```bash
# Typical AI development session
[main] INITIAL: Starting project setup
[main→feature] BRANCH SWITCH: AI added login system (8 files)
[feature] Fixed AI validation bug  
[feature] AI improved error handling
[feature→main] BRANCH SWITCH: Merging stable changes (3 files)
[main] Final cleanup after merge
```

This pattern shows:
1. Stable starting point on `main`
2. AI work on `feature` branch (8 files changed)
3. Manual fixes and AI improvements 
4. Safe merge back to `main`
5. Post-merge cleanup

### Branch-Aware Restoration Strategies

1. **Always restore from same branch**: If you're on `feature`, look for `[feature]` snapshots
2. **Use branch switches as boundaries**: `BRANCH SWITCH` messages mark major transitions
3. **Prefer non-AI snapshots for stability**: Messages without "AI" or large file counts
4. **Check file counts**: Smaller changes are usually safer to restore

## Future Enhancements

Coming soon:
- **Branch filtering**: `timemachine list --branch feature-auth`
- **Branch merge detection**: Special messages for merge commits
- **Remote branch tracking**: Detection of pushes/pulls
- **File type analysis**: Track which types of files changed during switches

---

**Next Steps:**
- [Technical Implementation Details](docs/features/BRANCH_DETECTION.md) - For developers
- [Configuration Guide](docs/configuration/README.md) - Customize behavior
- [Troubleshooting Guide](docs/configuration/TROUBLESHOOTING.md) - Solve issues