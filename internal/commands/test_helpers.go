package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// setupTestRepo creates a test repository for testing - copied from core/git_test.go
func setupTestRepo(t *testing.T) (string, *core.AppState, *core.GitManager) {
	tempDir, err := os.MkdirTemp("", "timemachine-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}
	
	// Initialize main repo
	cmd := exec.Command("git", "init", tempDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init main repo: %v", err)
	}
	
	// Set user config
	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set user.name: %v", err)
	}
	
	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set user.email: %v", err)
	}
	
	// Create AppState
	state := &core.AppState{
		ProjectRoot:    tempDir,
		GitDir:         gitDir,
		ShadowRepoDir:  filepath.Join(gitDir, "timemachine_snapshots"),
		IsInitialized:  false,
		CurrentBranch:  "main",
		ShadowBranch:   "main",
		BranchSynced:   true,
	}
	
	gitManager := core.NewGitManager(state)
	
	// Initialize shadow repo
	if err := gitManager.InitializeShadowRepo(); err != nil {
		t.Fatalf("Failed to initialize shadow repo: %v", err)
	}
	
	state.IsInitialized = true
	
	return tempDir, state, gitManager
}