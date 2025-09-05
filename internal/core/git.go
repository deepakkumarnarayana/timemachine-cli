package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GitManager wraps all Git operations for the shadow repository
// Uses hybrid approach: text extraction (primary) + git notes (secondary)
type GitManager struct {
	State *AppState // Only essential state needed
}

// NewGitManager creates a new GitManager with the given state
func NewGitManager(state *AppState) *GitManager {
	return &GitManager{State: state}
}


// RunCommand executes a git command with the shadow repo as the git directory
// CRITICAL: ALWAYS uses --git-dir and --work-tree to ensure operations
// happen in shadow repo, not main repo
func (g *GitManager) RunCommand(args ...string) (string, error) {
	// Build command: git --git-dir=<shadow_repo_path> --work-tree=<project_root> <args>
	fullArgs := []string{
		"--git-dir=" + g.State.ShadowRepoDir,
		"--work-tree=" + g.State.ProjectRoot,
	}
	fullArgs = append(fullArgs, args...)
	
	cmd := exec.Command("git", fullArgs...)
	
	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return "", fmt.Errorf("git command failed: %s\nOutput: %s", err.Error(), string(output))
	}
	
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the currently active branch in the main repository
func (g *GitManager) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "--git-dir="+g.State.GitDir, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "main", nil // Default to main if detached HEAD or empty
	}
	
	return branch, nil
}

// getLastCommitBranch extracts the branch name from the last commit message
func (g *GitManager) getLastCommitBranch() string {
	// Get the last commit message
	output, err := g.RunCommand("log", "-1", "--format=%s")
	if err != nil {
		return "" // No commits yet or error
	}
	
	message := strings.TrimSpace(output)
	if message == "" {
		return ""
	}
	
	// Extract branch from format [branch-name] or [prev→curr]
	if strings.HasPrefix(message, "[") {
		endBracket := strings.Index(message, "]")
		if endBracket > 1 {
			branchPart := message[1:endBracket]
			// Handle branch switch format [prev→curr]
			if strings.Contains(branchPart, "→") {
				parts := strings.Split(branchPart, "→")
				if len(parts) >= 2 {
					return strings.TrimSpace(parts[1]) // Return current branch from switch
				}
			}
			// Handle regular format [branch-name]
			return strings.TrimSpace(branchPart)
		}
	}
	
	return ""
}


// countUncommittedFiles counts files that would be included in next commit
func (g *GitManager) countUncommittedFiles() (int, error) {
	status, err := g.RunCommand("status", "--porcelain")
	if err != nil {
		return 0, err
	}
	
	if strings.TrimSpace(status) == "" {
		return 0, nil
	}
	
	lines := strings.Split(strings.TrimSpace(status), "\n")
	return len(lines), nil
}

// SnapshotMetadata represents structured metadata for each snapshot
type SnapshotMetadata struct {
	Branch         string    `json:"branch"`
	PreviousBranch string    `json:"previousBranch,omitempty"`
	ChangeCount    int       `json:"changeCount"`
	BranchSwitch   bool      `json:"branchSwitch"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"`
	FileTypes      []string  `json:"fileTypes,omitempty"`
	LargeChange    bool      `json:"largeChange"`
}

// addCommitMetadata adds structured metadata to commit using git notes
func (g *GitManager) addCommitMetadata(commitHash, currentBranch, lastCommitBranch string, changeCount int, commitType string, isBranchSwitch bool) error {
	metadata := SnapshotMetadata{
		Branch:         currentBranch,
		PreviousBranch: lastCommitBranch,
		ChangeCount:    changeCount,
		BranchSwitch:   isBranchSwitch,
		Timestamp:      time.Now(),
		Type:           commitType,
		LargeChange:    changeCount > 20,
	}
	
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		fmt.Printf("Warning: failed to marshal metadata: %v\n", err)
		return nil // Don't fail commit for metadata issues
	}
	
	_, err = g.RunCommand("notes", "add", "-m", string(jsonData), commitHash)
	if err != nil {
		// Don't fail commit if notes fail - text extraction is primary
		fmt.Printf("Warning: failed to add commit metadata: %v\n", err)
	}
	return nil
}

// CreateWatcherSnapshot creates a snapshot specifically from file watcher
// Uses hybrid approach: text extraction (primary) + git notes (secondary)
func (g *GitManager) CreateWatcherSnapshot() error {
	// Check if there are changes to commit
	status, err := g.RunCommand("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}
	
	if strings.TrimSpace(status) == "" {
		return nil // No changes to commit
	}
	
	// Stage all changes
	_, err = g.RunCommand("add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}
	
	// Get current branch from main repository
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	
	// Count changes
	changeCount, err := g.countUncommittedFiles()
	if err != nil {
		changeCount = 0
	}
	
	// Check if this is a branch switch by comparing with last commit's branch
	// PRIMARY: Text extraction from commit history (reliable, human-readable)
	lastCommitBranch := g.getLastCommitBranch()
	isBranchSwitch := lastCommitBranch != "" && lastCommitBranch != currentBranch
	
	// Create smart commit message (PRIMARY source of truth)
	var message string
	var commitType string
	
	if isBranchSwitch {
		// Watcher detected branch change
		if changeCount > 20 {
			message = fmt.Sprintf("[%s→%s] Auto-snapshot after branch switch (%d files - LARGE CHANGES ⚠️)", 
				lastCommitBranch, currentBranch, changeCount)
		} else {
			message = fmt.Sprintf("[%s→%s] Auto-snapshot after branch switch (%d files)", 
				lastCommitBranch, currentBranch, changeCount)
		}
		commitType = "watcher-branch-switch"
	} else {
		// Regular watcher snapshot
		message = fmt.Sprintf("[%s] Auto-snapshot from file watcher", currentBranch)
		commitType = "watcher-auto"
	}
	
	// Create commit with human-readable message
	output, err := g.RunCommand("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create watcher snapshot: %w", err)
	}
	
	// Add structured metadata using git notes (SECONDARY - fast queries)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		commitInfo := lines[0]
		if strings.Contains(commitInfo, "[") {
			parts := strings.Fields(commitInfo)
			if len(parts) >= 2 {
				commitHash := strings.Trim(parts[1], "[]")
				// Store rich structured metadata in git notes
				g.addCommitMetadata(commitHash, currentBranch, lastCommitBranch, changeCount, commitType, isBranchSwitch)
			}
		}
	}
	
	return nil
}

// InitializeShadowRepo creates and initializes the shadow repository
func (g *GitManager) InitializeShadowRepo() error {
	// Create .git/timemachine_snapshots directory
	if err := os.MkdirAll(g.State.ShadowRepoDir, 0755); err != nil {
		return fmt.Errorf("failed to create shadow repo directory: %w", err)
	}
	
	// Initialize the shadow repo
	_, err := g.RunCommand("init")
	if err != nil {
		return fmt.Errorf("failed to initialize shadow repository: %w", err)
	}
	
	// Copy user.name and user.email from main repo
	if err := g.CopyGitConfig(); err != nil {
		return fmt.Errorf("failed to copy git config: %w", err)
	}
	
	// Create initial empty commit to establish repository history
	if err := g.createInitialCommit(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}
	
	// Update state
	g.State.IsInitialized = true
	
	return nil
}

// SetupShadowRepo creates and initializes the shadow repository without any commits
// Use this + CreateSnapshot() instead of InitializeShadowRepo() to avoid redundant empty commits
func (g *GitManager) SetupShadowRepo() error {
	// Create .git/timemachine_snapshots directory
	if err := os.MkdirAll(g.State.ShadowRepoDir, 0755); err != nil {
		return fmt.Errorf("failed to create shadow repo directory: %w", err)
	}
	
	// Initialize the shadow repo
	_, err := g.RunCommand("init")
	if err != nil {
		return fmt.Errorf("failed to initialize shadow repository: %w", err)
	}
	
	// Copy user.name and user.email from main repo
	if err := g.CopyGitConfig(); err != nil {
		return fmt.Errorf("failed to copy git config: %w", err)
	}
	
	// DO NOT create any commits - let CreateSnapshot() handle that
	// Update state
	g.State.IsInitialized = true
	
	return nil
}

// CopyGitConfig copies user.name and user.email from the main repo to shadow repo
func (g *GitManager) CopyGitConfig() error {
	// Get user.name from main repo
	cmd := exec.Command("git", "--git-dir="+g.State.GitDir, "config", "user.name")
	nameOutput, err := cmd.Output()
	if err == nil && len(nameOutput) > 0 {
		name := strings.TrimSpace(string(nameOutput))
		_, err = g.RunCommand("config", "user.name", name)
		if err != nil {
			return fmt.Errorf("failed to set user.name: %w", err)
		}
	}
	
	// Get user.email from main repo
	cmd = exec.Command("git", "--git-dir="+g.State.GitDir, "config", "user.email")
	emailOutput, err := cmd.Output()
	if err == nil && len(emailOutput) > 0 {
		email := strings.TrimSpace(string(emailOutput))
		_, err = g.RunCommand("config", "user.email", email)
		if err != nil {
			return fmt.Errorf("failed to set user.email: %w", err)
		}
	}
	
	return nil
}

// CreateSnapshot creates a new snapshot with branch-aware intelligence
// Uses hybrid approach: text extraction (primary) + git notes (secondary)
func (g *GitManager) CreateSnapshot(message string) error {
	// Stage everything including untracked files
	_, err := g.RunCommand("add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}
	
	// Check if there are any changes to commit
	status, err := g.RunCommand("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}
	
	// If no changes, don't create empty commits
	if strings.TrimSpace(status) == "" {
		return nil
	}
	
	// Count files being committed
	changeCount, err := g.countUncommittedFiles()
	if err != nil {
		changeCount = 0 // Continue even if count fails
	}
	
	// Get current branch from main repository
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	
	// Check if this is a branch switch by comparing with last commit's branch
	// PRIMARY: Text extraction from commit history (reliable, human-readable)
	lastCommitBranch := g.getLastCommitBranch()
	isBranchSwitch := lastCommitBranch != "" && lastCommitBranch != currentBranch
	
	// Create smart commit message (PRIMARY source of truth)
	var enhancedMessage string
	var commitType string
	
	if isBranchSwitch {
		// Branch switch detected
		if changeCount > 20 {
			enhancedMessage = fmt.Sprintf("[%s→%s] BRANCH SWITCH: %s (%d files - LARGE CHANGES ⚠️)", 
				lastCommitBranch, currentBranch, message, changeCount)
		} else if changeCount > 5 {
			enhancedMessage = fmt.Sprintf("[%s→%s] BRANCH SWITCH: %s (%d files)", 
				lastCommitBranch, currentBranch, message, changeCount)
		} else {
			enhancedMessage = fmt.Sprintf("[%s→%s] BRANCH SWITCH: %s", 
				lastCommitBranch, currentBranch, message)
		}
		commitType = "branch-switch"
	} else {
		// Normal commit on same branch
		if message == "" {
			now := time.Now()
			message = fmt.Sprintf("Snapshot at %s", now.Format("15:04:05"))
		}
		enhancedMessage = fmt.Sprintf("[%s] %s", currentBranch, message)
		commitType = "manual"
	}
	
	// Create the commit with human-readable message
	output, err := g.RunCommand("commit", "-m", enhancedMessage)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	
	// Extract commit hash from git commit output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	commitInfo := lines[0] // First line contains commit info
	
	// Add structured metadata using git notes (SECONDARY - fast queries)
	if strings.Contains(commitInfo, "[") {
		parts := strings.Fields(commitInfo)
		if len(parts) >= 2 {
			commitHash := strings.Trim(parts[1], "[]")
			// Store rich structured metadata in git notes
			g.addCommitMetadata(commitHash, currentBranch, lastCommitBranch, changeCount, commitType, isBranchSwitch)
		}
	}
	
	return nil
}

// Snapshot represents a Git commit snapshot
type Snapshot struct {
	Hash    string // Full commit hash
	Message string // Commit message
	Time    string // Relative time (e.g., "2 minutes ago")
}

// ListSnapshots returns a list of snapshots, optionally filtered by file
func (g *GitManager) ListSnapshots(limit int, filePath string) ([]Snapshot, error) {
	// Build git log command
	args := []string{"log", "--oneline", "--date=relative"}
	
	// Add pretty format to get hash, message, and relative time
	args = append(args, "--pretty=format:%H|%s|%ar")
	
	// Add limit if specified
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}
	
	// Add file filter if specified
	if filePath != "" {
		args = append(args, "--", filePath)
	}
	
	output, err := g.RunCommand(args...)
	if err != nil {
		// If no commits exist yet, return empty slice (not error)
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return []Snapshot{}, nil
		}
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	
	// Parse output into Snapshot structs
	lines := strings.Split(strings.TrimSpace(output), "\n")
	snapshots := make([]Snapshot, 0, len(lines))
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		
		snapshots = append(snapshots, Snapshot{
			Hash:    parts[0],
			Message: parts[1],
			Time:    parts[2],
		})
	}
	
	return snapshots, nil
}

// ListSnapshotsByBranch returns snapshots filtered by branch name
func (g *GitManager) ListSnapshotsByBranch(branchName string, limit int) ([]Snapshot, error) {
	// Get all snapshots first
	snapshots, err := g.ListSnapshots(0, "") // Get all snapshots
	if err != nil {
		return nil, err
	}
	
	// Filter by branch name in commit message
	var filtered []Snapshot
	branchPattern := fmt.Sprintf("[%s]", branchName)
	switchPattern := fmt.Sprintf("→%s]", branchName) // Also catch branch switches TO this branch
	
	for _, snapshot := range snapshots {
		if strings.Contains(snapshot.Message, branchPattern) || strings.Contains(snapshot.Message, switchPattern) {
			filtered = append(filtered, snapshot)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}
	
	return filtered, nil
}

// GetSnapshotMetadata retrieves structured metadata for a snapshot using git notes
func (g *GitManager) GetSnapshotMetadata(hash string) (*SnapshotMetadata, error) {
	output, err := g.RunCommand("notes", "show", hash)
	if err != nil {
		// Notes don't exist for this commit - fallback to text extraction
		return g.extractMetadataFromCommitMessage(hash)
	}
	
	var metadata SnapshotMetadata
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &metadata); err != nil {
		// Notes exist but are corrupted - fallback to text extraction
		fmt.Printf("Warning: corrupted metadata notes for %s, using text extraction\n", hash)
		return g.extractMetadataFromCommitMessage(hash)
	}
	
	return &metadata, nil
}

// extractMetadataFromCommitMessage creates metadata by parsing commit message (fallback)
func (g *GitManager) extractMetadataFromCommitMessage(hash string) (*SnapshotMetadata, error) {
	output, err := g.RunCommand("log", "-1", "--format=%s", hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit message: %w", err)
	}
	
	message := strings.TrimSpace(output)
	metadata := &SnapshotMetadata{
		Timestamp: time.Now(), // Approximate, could get actual commit time
		Type:      "unknown",
	}
	
	// Parse branch information from commit message
	if strings.HasPrefix(message, "[") {
		endBracket := strings.Index(message, "]")
		if endBracket > 1 {
			branchPart := message[1:endBracket]
			
			// Handle branch switch format [prev→curr]
			if strings.Contains(branchPart, "→") {
				parts := strings.Split(branchPart, "→")
				if len(parts) >= 2 {
					metadata.PreviousBranch = strings.TrimSpace(parts[0])
					metadata.Branch = strings.TrimSpace(parts[1])
					metadata.BranchSwitch = true
					metadata.Type = "branch-switch"
				}
			} else {
				// Handle regular format [branch-name]
				metadata.Branch = strings.TrimSpace(branchPart)
				metadata.BranchSwitch = false
			}
		}
	}
	
	return metadata, nil
}

// ListBranchSwitches returns all commits that represent branch switches
func (g *GitManager) ListBranchSwitches(limit int) ([]Snapshot, error) {
	snapshots, err := g.ListSnapshots(0, "")
	if err != nil {
		return nil, err
	}
	
	var switches []Snapshot
	for _, snapshot := range snapshots {
		// Look for branch switch indicators in commit message
		if strings.Contains(snapshot.Message, "→") && strings.Contains(snapshot.Message, "BRANCH SWITCH") {
			switches = append(switches, snapshot)
			if limit > 0 && len(switches) >= limit {
				break
			}
		}
	}
	
	return switches, nil
}

// RestoreSnapshot restores files from a specific snapshot
// NEVER use checkout or reset - they affect staging area
// ALWAYS use git restore --source=<hash> --worktree
func (g *GitManager) RestoreSnapshot(hash string, files []string) error {
	args := []string{"restore", "--source=" + hash, "--worktree"}
	
	if len(files) == 0 {
		// Restore everything
		args = append(args, ".")
	} else {
		// Restore specific files
		args = append(args, files...)
	}
	
	_, err := g.RunCommand(args...)
	if err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}
	
	return nil
}

// createInitialCommit creates an initial commit with current project state
func (g *GitManager) createInitialCommit() error {
	// Stage all current files
	_, err := g.RunCommand("add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage files for initial commit: %w", err)
	}
	
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		currentBranch = "main"
	}
	
	message := fmt.Sprintf("[%s] Initial TimeMachine shadow repository", currentBranch)
	
	// Check if there are files to commit
	status, err := g.RunCommand("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check status for initial commit: %w", err)
	}
	
	if strings.TrimSpace(status) == "" {
		// No files to commit, create empty commit
		_, err = g.RunCommand("commit", "--allow-empty", "-m", message)
	} else {
		// Files exist, create normal commit
		_, err = g.RunCommand("commit", "-m", message)
	}
	
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}
	
	return nil
}