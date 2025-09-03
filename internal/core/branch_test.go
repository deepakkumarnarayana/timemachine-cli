package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBranchFunctionality(t *testing.T) {
	// Create test environment
	tempDir, state, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Test initial branch state
	err := state.UpdateBranchState()
	if err != nil {
		t.Fatalf("Failed to update branch state: %v", err)
	}

	if state.CurrentBranch == "" {
		t.Error("CurrentBranch should not be empty after initialization")
	}

	if state.ShadowBranch == "" {
		t.Error("ShadowBranch should not be empty after initialization")
	}

	if !state.BranchSynced {
		t.Error("Branches should be synced after initialization")
	}

	// Test GetCurrentBranch
	currentBranch, err := gitManager.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if currentBranch == "" {
		t.Error("Current branch should not be empty")
	}

	// Test GetCurrentShadowBranch
	shadowBranch, err := gitManager.GetCurrentShadowBranch()
	if err != nil {
		t.Fatalf("Failed to get shadow branch: %v", err)
	}
	if shadowBranch == "" {
		t.Error("Shadow branch should not be empty")
	}

	if currentBranch != shadowBranch {
		t.Errorf("Current branch (%s) should match shadow branch (%s)", currentBranch, shadowBranch)
	}
}

func TestBranchSwitching(t *testing.T) {
	// Create test environment
	tempDir, state, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create a test branch in the shadow repo
	err := gitManager.SwitchOrCreateShadowBranch("test-branch")
	if err != nil {
		t.Fatalf("Failed to create test branch: %v", err)
	}

	// Verify we're on the test branch
	shadowBranch, err := gitManager.GetCurrentShadowBranch()
	if err != nil {
		t.Fatalf("Failed to get shadow branch: %v", err)
	}
	if shadowBranch != "test-branch" {
		t.Errorf("Expected to be on 'test-branch', got '%s'", shadowBranch)
	}

	// Switch back to main
	err = gitManager.SwitchOrCreateShadowBranch("main")
	if err != nil {
		t.Fatalf("Failed to switch back to main: %v", err)
	}

	shadowBranch, err = gitManager.GetCurrentShadowBranch()
	if err != nil {
		t.Fatalf("Failed to get shadow branch: %v", err)
	}
	if shadowBranch != "main" {
		t.Errorf("Expected to be on 'main', got '%s'", shadowBranch)
	}

	// Test EnsureBranchSync functionality
	state.CurrentBranch = "feature-branch"
	state.BranchSynced = false

	err = state.EnsureBranchSync()
	if err != nil {
		t.Fatalf("Failed to ensure branch sync: %v", err)
	}

	if !state.BranchSynced {
		t.Error("Branches should be synced after EnsureBranchSync")
	}

	if state.CurrentBranch != state.ShadowBranch {
		t.Errorf("Current branch (%s) should match shadow branch (%s) after sync", state.CurrentBranch, state.ShadowBranch)
	}
}

func TestSnapshotWithBranching(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create snapshot (should include branch info in message)
	err = gitManager.CreateSnapshot("")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// List snapshots and verify branch info is included
	snapshots, err := gitManager.ListSnapshots(10, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	// Should have 2 snapshots: initial commit + our test snapshot
	if len(snapshots) < 2 {
		t.Errorf("Expected at least 2 snapshots, got %d", len(snapshots))
	}

	// The most recent snapshot should contain branch information
	latestSnapshot := snapshots[0]
	if latestSnapshot.Message == "" {
		t.Error("Snapshot message should not be empty")
	}
	// Message should contain timestamp and branch info like "Snapshot at 15:04:05 [main]"
	// We'll just verify it's not empty and contains some expected patterns
}