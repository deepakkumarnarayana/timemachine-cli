package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GitManager wraps all Git operations for the shadow repository
type GitManager struct {
	State          *AppState
	currentBranch  string // Current main repo branch
	previousBranch string // Previous branch (for branch change detection)
	branchChanged  bool   // Flag indicating first commit after branch change
}

// NewGitManager creates a new GitManager with the given state
func NewGitManager(state *AppState) *GitManager {
	manager := &GitManager{State: state}
	// Initialize branch state on creation
	manager.initializeBranchState()
	return manager
}

// initializeBranchState sets up initial branch tracking
func (g *GitManager) initializeBranchState() {
	if branch, err := g.GetCurrentBranch(); err == nil {
		g.currentBranch = branch
		
		// Try to load previous branch from shadow repo metadata
		if prevBranch, err := g.loadPreviousBranch(); err == nil && prevBranch != "" {
			g.previousBranch = prevBranch
			// If current branch differs from stored previous branch, mark as changed
			if g.currentBranch != g.previousBranch {
				g.branchChanged = true
			} else {
				g.branchChanged = false
			}
		} else {
			// First time initialization - no previous branch known
			g.previousBranch = branch
			g.branchChanged = false
		}
	}
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

// loadPreviousBranch loads the last known branch from shadow repo metadata
func (g *GitManager) loadPreviousBranch() (string, error) {
	// Try to read the previous branch from git config in shadow repo
	output, err := g.RunCommand("config", "timemachine.previousBranch")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// savePreviousBranch saves the previous branch to shadow repo metadata
func (g *GitManager) savePreviousBranch(branch string) error {
	_, err := g.RunCommand("config", "timemachine.previousBranch", branch)
	return err
}

// updateBranchState detects and tracks branch changes in main repository
func (g *GitManager) updateBranchState() error {
	newBranch, err := g.GetCurrentBranch()
	if err != nil {
		return err
	}
	
	// Detect branch change
	if g.currentBranch != "" && g.currentBranch != newBranch {
		g.previousBranch = g.currentBranch
		g.branchChanged = true
		// Save the previous branch for next time
		g.savePreviousBranch(g.previousBranch)
	}
	
	g.currentBranch = newBranch
	return nil
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

// addCommitMetadata adds structured metadata to commit using git notes
func (g *GitManager) addCommitMetadata(commitHash string, changeCount int, commitType string) error {
	metadata := fmt.Sprintf(`{
  "branch": "%s",
  "previousBranch": "%s",
  "changeCount": %d,
  "branchSwitch": %t,
  "timestamp": "%s",
  "type": "%s"
}`, g.currentBranch, g.previousBranch, changeCount, g.branchChanged, time.Now().Format(time.RFC3339), commitType)
	
	_, err := g.RunCommand("notes", "add", "-m", metadata, commitHash)
	if err != nil {
		// Don't fail commit if notes fail - metadata is optional
		fmt.Printf("Warning: failed to add commit metadata: %v\n", err)
	}
	return nil
}

// CreateWatcherSnapshot creates a snapshot specifically from file watcher
// Used when watcher detects branch changes and needs to auto-commit
func (g *GitManager) CreateWatcherSnapshot() error {
	// Update branch state first
	if err := g.updateBranchState(); err != nil {
		return fmt.Errorf("failed to update branch state: %w", err)
	}
	
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
	
	// Count changes
	changeCount, err := g.countUncommittedFiles()
	if err != nil {
		changeCount = 0
	}
	
	var message string
	var commitType string
	
	if g.branchChanged {
		// Watcher detected branch change
		if changeCount > 20 {
			message = fmt.Sprintf("[%s→%s] Auto-snapshot after branch switch (%d files - LARGE CHANGES ⚠️)", 
				g.previousBranch, g.currentBranch, changeCount)
		} else {
			message = fmt.Sprintf("[%s→%s] Auto-snapshot after branch switch (%d files)", 
				g.previousBranch, g.currentBranch, changeCount)
		}
		commitType = "watcher-branch-switch"
		g.branchChanged = false
	} else {
		// Regular watcher snapshot
		message = fmt.Sprintf("[%s] Auto-snapshot from file watcher", g.currentBranch)
		commitType = "watcher-auto"
	}
	
	// Create commit
	output, err := g.RunCommand("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create watcher snapshot: %w", err)
	}
	
	// Add metadata
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		commitInfo := lines[0]
		if strings.Contains(commitInfo, "[") {
			parts := strings.Fields(commitInfo)
			if len(parts) >= 2 {
				commitHash := strings.Trim(parts[1], "[]")
				g.addCommitMetadata(commitHash, changeCount, commitType)
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
// Detects branch changes and creates contextual commit messages with warnings
func (g *GitManager) CreateSnapshot(message string) error {
	// Update branch state and detect changes
	if err := g.updateBranchState(); err != nil {
		return fmt.Errorf("failed to update branch state: %w", err)
	}
	
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
	
	// Create smart commit message based on branch state
	var enhancedMessage string
	var commitType string
	
	if g.branchChanged {
		// First commit after branch change - add warnings for large changes
		if changeCount > 20 {
			enhancedMessage = fmt.Sprintf("[%s→%s] BRANCH SWITCH: %s (%d files - LARGE CHANGES ⚠️)", 
				g.previousBranch, g.currentBranch, message, changeCount)
		} else if changeCount > 5 {
			enhancedMessage = fmt.Sprintf("[%s→%s] BRANCH SWITCH: %s (%d files)", 
				g.previousBranch, g.currentBranch, message, changeCount)
		} else {
			enhancedMessage = fmt.Sprintf("[%s→%s] BRANCH SWITCH: %s", 
				g.previousBranch, g.currentBranch, message)
		}
		commitType = "branch-switch"
		g.branchChanged = false // Reset flag after first commit
	} else {
		// Normal commit on same branch
		if message == "" {
			now := time.Now()
			message = fmt.Sprintf("Snapshot at %s", now.Format("15:04:05"))
		}
		enhancedMessage = fmt.Sprintf("[%s] %s", g.currentBranch, message)
		commitType = "manual"
	}
	
	// Create the commit
	output, err := g.RunCommand("commit", "-m", enhancedMessage)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	
	// Extract commit hash from output (format: "[branch hash] message")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	commitInfo := lines[0] // First line contains commit info
	
	// Add metadata for filtering and analysis
	if strings.Contains(commitInfo, "[") {
		// Try to extract hash from git commit output
		parts := strings.Fields(commitInfo)
		if len(parts) >= 2 {
			commitHash := strings.Trim(parts[1], "[]")
			g.addCommitMetadata(commitHash, changeCount, commitType)
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

// GetSnapshotMetadata retrieves metadata for a snapshot using git notes
func (g *GitManager) GetSnapshotMetadata(hash string) (string, error) {
	output, err := g.RunCommand("notes", "show", hash)
	if err != nil {
		// Notes don't exist for this commit
		return "", nil
	}
	return strings.TrimSpace(output), nil
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