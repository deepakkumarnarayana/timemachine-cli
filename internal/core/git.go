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
	State *AppState
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
	if err := g.copyGitConfig(); err != nil {
		return fmt.Errorf("failed to copy git config: %w", err)
	}
	
	// Update state
	g.State.IsInitialized = true
	
	return nil
}

// copyGitConfig copies user.name and user.email from the main repo to shadow repo
func (g *GitManager) copyGitConfig() error {
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

// CreateSnapshot creates a new snapshot in the shadow repository
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
	
	// Use timestamp if no message provided
	if message == "" {
		now := time.Now()
		message = fmt.Sprintf("Snapshot at %s", now.Format("15:04:05"))
	}
	
	// Create the commit
	_, err = g.RunCommand("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
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