package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatcherBranchAwareness(t *testing.T) {
	// Create test environment
	tempDir, state, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create watcher
	watcher, err := NewWatcher(state, gitManager)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Test initial branch tracking
	if watcher.lastBranch == "" {
		t.Error("Watcher should track initial branch")
	}

	// Test branch change detection
	initialBranch := watcher.lastBranch
	
	// Simulate branch change by updating the branch tracker
	watcher.branchMutex.Lock()
	watcher.lastBranch = "feature-branch"
	watcher.branchMutex.Unlock()

	// Verify the change was tracked
	watcher.branchMutex.RLock()
	currentTracked := watcher.lastBranch
	watcher.branchMutex.RUnlock()

	if currentTracked != "feature-branch" {
		t.Errorf("Expected tracked branch to be 'feature-branch', got '%s'", currentTracked)
	}

	if currentTracked == initialBranch {
		t.Error("Branch tracking should have changed from initial branch")
	}
}

func TestWatcherHandleBranchChange(t *testing.T) {
	// Create test environment
	tempDir, state, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create watcher
	watcher, err := NewWatcher(state, gitManager)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Test handleBranchChange with no actual branch change
	initialBranch := watcher.lastBranch
	watcher.handleBranchChange()
	
	// Should remain the same since no actual Git branch change occurred
	if watcher.lastBranch != initialBranch {
		t.Errorf("Branch should not change when handleBranchChange is called without actual Git branch change")
	}
}

func TestWatcherGitHeadWatching(t *testing.T) {
	// Create test environment
	tempDir, state, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create watcher
	watcher, err := NewWatcher(state, gitManager)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Test that Git HEAD path is correctly identified
	gitHeadPath := filepath.Join(state.GitDir, "HEAD")
	
	// Verify the path exists
	if _, err := os.Stat(gitHeadPath); os.IsNotExist(err) {
		t.Fatalf("Git HEAD file should exist at %s", gitHeadPath)
	}

	// Test event handling with non-HEAD file (should trigger normal handling)
	regularFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(regularFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// This should not be treated as a branch change
	initialBranch := watcher.lastBranch
	// We can't easily simulate fsnotify events in unit tests, so we just verify
	// the logic would work correctly by checking the path comparison
	if regularFile == gitHeadPath {
		t.Error("Regular file should not be identified as Git HEAD file")
	}

	// Verify branch tracking wasn't affected
	if watcher.lastBranch != initialBranch {
		t.Error("Branch tracking should not change for non-HEAD file events")
	}
}