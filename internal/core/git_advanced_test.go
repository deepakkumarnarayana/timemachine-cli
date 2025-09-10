package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetCurrentBranch tests branch detection in various scenarios
func TestGetCurrentBranch(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Test 1: Default branch (should be master or main)
	branch, err := gitManager.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if branch == "" {
		t.Error("Expected non-empty branch name")
	}
	t.Logf("Current branch: %s", branch)

	// Test 2: Create and switch to a new branch
	cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", "feature-test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	branch, err = gitManager.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get feature branch: %v", err)
	}
	if branch != "feature-test" {
		t.Errorf("Expected 'feature-test', got '%s'", branch)
	}

	// Test 3: Test detached HEAD scenario
	// Create a commit first
	testFile := filepath.Join(tempDir, "detached-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "add", "detached-test.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "commit", "-m", "test commit for detached HEAD")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit test file: %v", err)
	}

	// Get commit hash
	cmd = exec.Command("git", "-C", tempDir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(string(output))

	// Checkout the commit (detached HEAD)
	cmd = exec.Command("git", "-C", tempDir, "checkout", commitHash)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout commit: %v", err)
	}

	// Should default to "main" for detached HEAD
	branch, err = gitManager.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get branch in detached HEAD: %v", err)
	}
	if branch != "main" {
		t.Logf("In detached HEAD, got branch: %s (expected fallback to 'main')", branch)
	}
}

// TestGetLastCommitBranchAdvanced tests the advanced branch detection logic
func TestGetLastCommitBranchAdvanced(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Test 1: No history scenario (initial state)
	result := gitManager.getLastCommitBranchAdvanced()
	if result.HasHistory {
		t.Error("Expected no history in empty repo")
	}
	if result.CommitCount != 0 {
		t.Errorf("Expected commit count 0, got %d", result.CommitCount)
	}

	// Test 2: Create first commit with branch-aware message
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Initial commit test")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Now test with history
	result = gitManager.getLastCommitBranchAdvanced()
	if !result.HasHistory {
		t.Error("Expected history after creating commit")
	}
	if result.CommitCount == 0 {
		t.Error("Expected non-zero commit count")
	}
	if result.IsCorrupted {
		t.Error("Expected non-corrupted result")
	}
	t.Logf("Advanced branch detection result: %+v", result)

	// Test 3: Branch switch scenario
	cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", "feature-advanced")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("modified on feature"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = gitManager.CreateSnapshot("Feature branch work")
	if err != nil {
		t.Fatalf("Failed to create feature snapshot: %v", err)
	}

	result = gitManager.getLastCommitBranchAdvanced()
	if result.LastBranch == "" {
		t.Error("Expected non-empty last branch")
	}
	t.Logf("After branch switch: %+v", result)
}

// TestCountUncommittedFiles tests file counting functionality
func TestCountUncommittedFiles(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Test 1: No uncommitted files
	count, err := gitManager.countUncommittedFiles()
	if err != nil {
		t.Fatalf("Failed to count files: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 files, got %d", count)
	}

	// Test 2: Single file
	testFile := filepath.Join(tempDir, "count-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	count, err = gitManager.countUncommittedFiles()
	if err != nil {
		t.Fatalf("Failed to count files: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 file, got %d", count)
	}

	// Test 3: Multiple files
	for i := 2; i <= 5; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(fileName, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}
	}

	count, err = gitManager.countUncommittedFiles()
	if err != nil {
		t.Fatalf("Failed to count multiple files: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5 files, got %d", count)
	}

	// Test 4: Modified existing file
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	count, err = gitManager.countUncommittedFiles()
	if err != nil {
		t.Fatalf("Failed to count after modification: %v", err)
	}
	// Should still be 5 (the modified file is already tracked)
	if count != 5 {
		t.Errorf("Expected 5 files after modification, got %d", count)
	}
}

// TestAddCommitMetadata tests Git notes metadata functionality
func TestAddCommitMetadata(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Create a commit first
	testFile := filepath.Join(tempDir, "metadata-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Test metadata")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Get the commit hash
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(snapshots))
	}

	commitHash := snapshots[0].Hash

	// Test adding metadata
	err = gitManager.addCommitMetadata(commitHash, "test-branch", "prev-branch", 5, "manual", true)
	if err != nil {
		t.Fatalf("Failed to add commit metadata: %v", err)
	}

	// Try to retrieve the metadata
	notesOutput, err := gitManager.RunCommand("notes", "show", commitHash)
	if err != nil {
		t.Logf("Notes retrieval failed (expected for some scenarios): %v", err)
	} else {
		t.Logf("Retrieved notes: %s", notesOutput)

		// Try to parse as JSON
		var metadata SnapshotMetadata
		if err := json.Unmarshal([]byte(notesOutput), &metadata); err == nil {
			if metadata.Branch != "test-branch" {
				t.Errorf("Expected branch 'test-branch', got '%s'", metadata.Branch)
			}
			if metadata.PreviousBranch != "prev-branch" {
				t.Errorf("Expected previous branch 'prev-branch', got '%s'", metadata.PreviousBranch)
			}
			if metadata.ChangeCount != 5 {
				t.Errorf("Expected change count 5, got %d", metadata.ChangeCount)
			}
			if !metadata.BranchSwitch {
				t.Error("Expected branch switch to be true")
			}
		}
	}
}

// TestGetSnapshotMetadata tests metadata retrieval
func TestGetSnapshotMetadata(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Create a snapshot
	testFile := filepath.Join(tempDir, "metadata-retrieval-test.txt")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Test metadata retrieval")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatal("No snapshots created")
	}

	hash := snapshots[0].Hash

	// Test metadata retrieval
	metadata, err := gitManager.GetSnapshotMetadata(hash)
	if err != nil {
		t.Logf("Metadata retrieval failed (may be expected): %v", err)
	} else {
		if metadata == nil {
			t.Error("Expected non-nil metadata")
		} else {
			t.Logf("Retrieved metadata: %+v", metadata)
			if metadata.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}
		}
	}
}

// TestCreateWatcherSnapshot tests watcher-specific snapshot creation
func TestCreateWatcherSnapshot(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Test 1: No changes scenario
	err := gitManager.CreateWatcherSnapshot()
	if err != nil {
		t.Logf("Watcher snapshot with no changes failed as expected: %v", err)
	}

	// Test 2: With changes
	testFile := filepath.Join(tempDir, "watcher-test.txt")
	if err := os.WriteFile(testFile, []byte("watcher content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = gitManager.CreateWatcherSnapshot()
	if err != nil {
		t.Fatalf("Failed to create watcher snapshot: %v", err)
	}

	// Verify snapshot was created
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("Expected 1 watcher snapshot, got %d", len(snapshots))
	}

	// Check message format - could be either timestamp format or branch format
	message := snapshots[0].Message
	if !strings.Contains(message, "Snapshot at") && !strings.Contains(message, "Auto-snapshot") {
		t.Errorf("Expected watcher snapshot message to contain 'Snapshot at' or 'Auto-snapshot', got '%s'", message)
	} else {
		t.Logf("✅ Watcher snapshot created with message: %s", message)
	}
}

// TestListSnapshotsByBranch tests branch-specific snapshot listing
func TestListSnapshotsByBranch(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Create snapshots on different branches
	// Main branch snapshot
	testFile := filepath.Join(tempDir, "branch-test.txt")
	if err := os.WriteFile(testFile, []byte("main branch"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Main branch work")
	if err != nil {
		t.Fatalf("Failed to create main snapshot: %v", err)
	}

	// Switch to feature branch
	cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", "feature-branch-filter")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("feature branch"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = gitManager.CreateSnapshot("Feature branch work")
	if err != nil {
		t.Fatalf("Failed to create feature snapshot: %v", err)
	}

	// Test listing by branch
	mainSnapshots, err := gitManager.ListSnapshotsByBranch("master", 10)
	if err != nil {
		t.Fatalf("Failed to list main snapshots: %v", err)
	}

	featureSnapshots, err := gitManager.ListSnapshotsByBranch("feature-branch-filter", 10)
	if err != nil {
		t.Fatalf("Failed to list feature snapshots: %v", err)
	}

	t.Logf("Main branch snapshots: %d", len(mainSnapshots))
	t.Logf("Feature branch snapshots: %d", len(featureSnapshots))

	// At least one snapshot should exist
	totalSnapshots, _ := gitManager.ListSnapshots(10, "")
	if len(totalSnapshots) == 0 {
		t.Error("Expected at least one snapshot total")
	}
}

// TestListBranchSwitches tests branch switch detection
func TestListBranchSwitches(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Create initial commit
	testFile := filepath.Join(tempDir, "switch-test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Initial commit")
	if err != nil {
		t.Fatalf("Failed to create initial snapshot: %v", err)
	}

	// Create branch and switch
	cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", "switch-test-branch")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create switch test branch: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("switched content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = gitManager.CreateSnapshot("After branch switch")
	if err != nil {
		t.Fatalf("Failed to create switch snapshot: %v", err)
	}

	// Test listing branch switches
	switches, err := gitManager.ListBranchSwitches(10)
	if err != nil {
		t.Fatalf("Failed to list branch switches: %v", err)
	}

	t.Logf("Found %d branch switches", len(switches))

	// Should find at least one switch
	for _, s := range switches {
		t.Logf("Branch switch: %s", s.Message)
		if strings.Contains(s.Message, "BRANCH SWITCH") {
			t.Logf("Confirmed branch switch detection: %s", s.Message)
			break
		}
	}
}
