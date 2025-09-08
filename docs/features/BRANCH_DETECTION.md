# Branch Detection System - Technical Overview

> **For Users**: See the [Branch Detection User Guide](../branch-detection.md) for practical usage examples and AI development workflows.

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Commit Message Formats](#commit-message-formats)
- [Implementation Details](#implementation-details)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)

## Overview

The TimeMachine CLI branch detection system provides intelligent branch-aware snapshot management by automatically detecting when you switch between Git branches and creating appropriately formatted commits. This feature is essential for AI-assisted development workflows where understanding code context across different branches is critical.

### Key Benefits

- **Automatic Branch Switch Detection**: Detects when you switch between branches without manual intervention
- **Enhanced Context**: Commit messages include branch transition information (e.g., `[main→feature] BRANCH SWITCH`)
- **Fallback Resilience**: Gracefully handles corrupted data and edge cases
- **Performance Optimized**: Hybrid approach using both text extraction and structured metadata

## Architecture

The branch detection system uses a **hybrid dual-storage approach** for maximum reliability and performance:

### Primary: Text Extraction (Human-Readable)
- **Source of Truth**: Commit messages contain branch information in human-readable format
- **Reliability**: Works even if metadata storage fails
- **Format**: `[branch]` or `[prev→curr]` prefixes in commit messages
- **Parsing**: Direct text parsing from commit history using `git log --format=%s`

### Secondary: Git Notes (Structured Metadata)  
- **Performance**: Fast querying of structured data
- **Rich Context**: JSON metadata including timestamps, file counts, change types
- **Storage**: Uses `git notes` to attach JSON metadata to commits
- **Fallback**: System continues working if notes are corrupted or missing

### Key Components

```
GitManager (internal/core/git.go)
├── GetCurrentBranch()           # Detects current branch from main repo
├── getLastCommitBranchAdvanced() # Enhanced branch detection with fallback
├── CreateSnapshot()             # Main snapshot creation with branch awareness
├── CreateWatcherSnapshot()      # File watcher integration
└── addCommitMetadata()          # Adds structured metadata via git notes
```

## Commit Message Formats

### Initial Commits
When creating the first snapshot in a new branch or repository:
```
[branch-name] INITIAL: message
```
**Example**: `[main] INITIAL: Auto-snapshot from file watcher`

### Branch Switch Detection
When switching from one branch to another:
```
[previous→current] BRANCH SWITCH: message (optional-file-count)
```
**Examples**:
- `[main→feature] BRANCH SWITCH: Working on new feature`
- `[feature→main] BRANCH SWITCH: Back to main (15 files)`
- `[develop→hotfix] BRANCH SWITCH: Critical bug fix (3 files - LARGE CHANGES ⚠️)`

### Regular Snapshots
For normal snapshots within the same branch:
```
[branch-name] message
```
**Example**: `[feature] Auto-snapshot from file watcher`

### File Count Indicators
- **No indicator**: 1-5 files changed
- **File count**: `(X files)` for 6-20 files  
- **Large change warning**: `(X files - LARGE CHANGES ⚠️)` for 20+ files

## Implementation Details

### Branch Detection Algorithm

1. **Current Branch Detection**
   ```go
   currentBranch, err := g.GetCurrentBranch()
   // Uses: git --git-dir=<main_repo> branch --show-current
   ```

2. **Historical Branch Analysis**
   ```go
   branchDetection := g.getLastCommitBranchAdvanced()
   // Returns: BranchDetectionResult with LastBranch, HasHistory, IsCorrupted, CommitCount
   ```

3. **Switch Detection Logic**
   ```go
   if !branchDetection.HasHistory {
       isInitialCommit = true
   } else if branchDetection.IsCorrupted {
       isBranchSwitch = true  // Conservative approach
   } else {
       isBranchSwitch = (branchDetection.LastBranch != currentBranch)
   }
   ```

### Enhanced Fallback Logic

The system includes comprehensive fallback handling for various edge cases:

#### Corrupted Data Recovery
- **Detection**: Invalid JSON in git notes, malformed commit messages
- **Response**: Falls back to text extraction, treats as potential branch switch
- **User Feedback**: Logs warning messages but continues operation

#### Missing History Handling
- **Scenario**: First commit in shadow repository
- **Response**: Creates `[branch] INITIAL: message` format
- **Metadata**: Marks as `type: "initial"` in notes

#### Unknown Branch Scenarios  
- **Scenario**: Previous branch cannot be determined
- **Response**: Uses `"unknown"` as previous branch name
- **Format**: `[unknown→current] BRANCH SWITCH: message`

### Thread Safety
- All Git operations use proper command isolation with `--git-dir` and `--work-tree` flags
- No shared mutable state between concurrent operations
- Error handling prevents data corruption

## API Reference

### Core Methods

#### `GetCurrentBranch() (string, error)`
Retrieves the current active branch from the main repository (not shadow repo).

**Returns**:
- Current branch name or "main" as fallback
- Error if Git operation fails

**Example**:
```go
branch, err := gitManager.GetCurrentBranch()
if err != nil {
    return fmt.Errorf("failed to get branch: %w", err)
}
```

#### `getLastCommitBranchAdvanced() BranchDetectionResult`
Performs enhanced branch detection with comprehensive fallback logic.

**Returns**:
```go
type BranchDetectionResult struct {
    LastBranch    string  // Previous branch name or empty
    HasHistory    bool    // Whether repository has commit history  
    IsCorrupted   bool    // Whether previous commit data is corrupted
    CommitCount   int     // Total commits in repository
}
```

#### `CreateSnapshot(message string) error`
Creates a branch-aware snapshot with intelligent commit message formatting.

**Parameters**:
- `message`: User-provided description for the snapshot

**Process**:
1. Stages all changes with `git add -A`
2. Detects current and previous branches
3. Formats message based on branch context
4. Creates commit with enhanced message
5. Adds structured metadata via git notes

#### `CreateWatcherSnapshot() error`  
Specialized snapshot creation for file watcher events with auto-generated messages.

**Features**:
- Automatic branch switch detection
- File count analysis 
- Large change warnings
- Debounced operations

### Metadata Structure

```go
type SnapshotMetadata struct {
    Branch         string    `json:"branch"`           // Current branch
    PreviousBranch string    `json:"previousBranch"`   // Previous branch (if switch)
    ChangeCount    int       `json:"changeCount"`      // Number of files changed
    BranchSwitch   bool      `json:"branchSwitch"`     // Whether this is a branch switch
    Timestamp      time.Time `json:"timestamp"`        // When snapshot was created
    Type           string    `json:"type"`             // Snapshot type (initial, branch-switch, manual, watcher-*)
    FileTypes      []string  `json:"fileTypes"`        // File extensions (future enhancement)
    LargeChange    bool      `json:"largeChange"`      // Whether >20 files changed
}
```

### Query Methods

#### `ListSnapshotsByBranch(branchName string, limit int) ([]Snapshot, error)`
Filters snapshots by branch name, including both regular commits and branch switches.

**Parameters**:
- `branchName`: Target branch to filter by
- `limit`: Maximum number of results (0 for unlimited)

**Matching Logic**:
- Matches `[branchName]` for regular commits
- Matches `→branchName]` for switches TO the branch

#### `ListBranchSwitches(limit int) ([]Snapshot, error)` 
Returns only commits that represent branch switches.

**Filter Criteria**:
- Contains `→` (branch transition arrow)
- Contains `BRANCH SWITCH` text

#### `GetSnapshotMetadata(hash string) (*SnapshotMetadata, error)`
Retrieves structured metadata for a specific snapshot.

**Fallback Behavior**:
1. Try to read git notes (structured JSON)
2. If notes missing/corrupted, extract from commit message
3. Return best-effort metadata structure

## Troubleshooting

### Common Issues

#### Branch Detection Not Working
**Symptoms**: All commits show as `[unknown→current]` format

**Diagnosis**:
```bash
# Check if shadow repo has history
git --git-dir=.git/timemachine_snapshots log --oneline -5

# Verify commit message format
git --git-dir=.git/timemachine_snapshots log -1 --format="%s"
```

**Solutions**:
- Ensure proper initialization with `timemachine init`
- Check that main repository is a valid Git repo
- Verify git user configuration is set

#### Corrupted Metadata Warnings
**Symptoms**: Warnings about corrupted metadata in console output

**Impact**: System continues working but may show `unknown` for previous branches

**Resolution**:
- Warnings are informational - core functionality unaffected
- Create new snapshots to establish clean metadata
- Consider reinitializing if warnings persist: `rm -rf .git/timemachine_snapshots && timemachine init`

#### Missing Branch Information
**Symptoms**: Commit messages don't contain branch information

**Diagnosis**:
```bash
# Check current branch detection
git branch --show-current

# Verify main repo is valid
git status
```

**Solutions**:
- Ensure you're in a Git repository
- Verify Git is properly configured
- Check if you're in detached HEAD state

### Debug Information

Enable detailed logging by examining shadow repository directly:
```bash
# View shadow repo commits with metadata
git --git-dir=.git/timemachine_snapshots log --format="%H %s" -10

# Check git notes for a specific commit
git --git-dir=.git/timemachine_snapshots notes show <commit-hash>

# Verify repository integrity
git --git-dir=.git/timemachine_snapshots fsck
```

### Performance Considerations

- **Text extraction** (primary) is fast - simple string parsing
- **Git notes** queries are optimized for structured data
- **Branch detection** runs only during snapshot creation, not file watching
- **Metadata fallback** prevents any single point of failure

### Integration Testing

Verify branch detection with this test sequence:
```bash
# Initialize TimeMachine
timemachine init

# Create initial snapshot
echo "test" > test.txt
timemachine snapshot "Initial test"

# Switch branches
git checkout -b feature-branch
echo "modified" > test.txt

# Create branch switch snapshot  
timemachine snapshot "Feature work"

# Verify branch detection worked
timemachine list
```

Expected output should show branch information in commit messages like:
```
[main] INITIAL: Initial test
[main→feature-branch] BRANCH SWITCH: Feature work
```

## Security Considerations

- **Input Validation**: All Git hashes validated with regex `^[a-fA-F0-9]{4,40}$`
- **Command Injection Prevention**: No user input directly passed to shell
- **Path Sanitization**: File paths checked for directory traversal attacks
- **Repository Isolation**: Shadow repo operations completely separated from main repo

## Future Enhancements

- **File Type Analysis**: Track which types of files changed during branch switches
- **Branch Merge Detection**: Identify when branches are merged
- **Remote Branch Tracking**: Detect pushes/pulls from remote branches
- **Performance Metrics**: Track branch detection accuracy and performance