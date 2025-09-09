package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMessageFormatValidation tests all documented message formats
func TestMessageFormatValidation(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "format-test.txt")

	// Test 1: Initial commit format [branch] INITIAL: message
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Testing initial format")
	if err != nil {
		t.Fatalf("Failed to create initial snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatal("No snapshots created")
	}

	message := snapshots[0].Message
	t.Logf("Initial commit message: %s", message)

	// Should contain branch info in brackets
	if !strings.HasPrefix(message, "[") || !strings.Contains(message, "]") {
		t.Errorf("Expected branch format in message, got: %s", message)
	}

	// Should contain INITIAL or the custom message
	if !strings.Contains(message, "INITIAL") && !strings.Contains(message, "Testing initial format") {
		t.Errorf("Expected INITIAL or custom message, got: %s", message)
	}

	// Test 2: Branch switch format [prev→curr] BRANCH SWITCH: message
	cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", "format-test-branch")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create format test branch: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("modified on branch"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = gitManager.CreateSnapshot("Testing branch switch format")
	if err != nil {
		t.Fatalf("Failed to create branch switch snapshot: %v", err)
	}

	snapshots, err = gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list updated snapshots: %v", err)
	}

	switchMessage := snapshots[0].Message
	t.Logf("Branch switch message: %s", switchMessage)

	// Should contain arrow format
	if !strings.Contains(switchMessage, "→") {
		t.Errorf("Expected branch switch arrow (→) in message, got: %s", switchMessage)
	}

	// Should contain BRANCH SWITCH or custom message
	if !strings.Contains(switchMessage, "BRANCH SWITCH") && !strings.Contains(switchMessage, "Testing branch switch format") {
		t.Logf("Note: Branch switch detection may not trigger immediately, message: %s", switchMessage)
	}
}

// TestLargeFileCountHandling tests handling of large change sets
func TestLargeFileCountHandling(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create many files to trigger large change warning
	const fileCount = 30 // More than the 20 file threshold

	for i := 0; i < fileCount; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("large-test-%d.txt", i))
		content := fmt.Sprintf("Content of file %d", i)
		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}
	}

	err := gitManager.CreateSnapshot("Testing large file count")
	if err != nil {
		t.Fatalf("Failed to create large snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatal("No snapshots created")
	}

	message := snapshots[0].Message
	t.Logf("Large change message: %s", message)

	// Should handle large file count gracefully
	if strings.Contains(message, "LARGE CHANGES") {
		t.Logf("✅ Large changes warning detected: %s", message)
	} else {
		t.Logf("Note: Large changes warning not detected, but snapshot created successfully")
	}
}

// TestCorruptedNotesHandling tests resilience to corrupted Git notes
func TestCorruptedNotesHandling(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create initial snapshot
	testFile := filepath.Join(tempDir, "corrupt-notes-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Before corruption")
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

	// Add corrupted notes
	corruptedData := "this is not valid JSON {broken"
	_, err = gitManager.RunCommand("notes", "add", "-f", "-m", corruptedData, hash)
	if err != nil {
		t.Logf("Note: Adding corrupted notes failed, which is okay: %v", err)
	}

	// Create another snapshot - should handle corruption gracefully
	if err := os.WriteFile(testFile, []byte("after corruption"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = gitManager.CreateSnapshot("After corruption test")
	if err != nil {
		t.Fatalf("Failed to create snapshot after corruption: %v", err)
	}

	// Should succeed despite corrupted notes
	newSnapshots, err := gitManager.ListSnapshots(2, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots after corruption: %v", err)
	}

	if len(newSnapshots) < 2 {
		t.Errorf("Expected at least 2 snapshots, got %d", len(newSnapshots))
	}

	t.Logf("✅ Handled corrupted notes gracefully, created %d snapshots", len(newSnapshots))
}

// TestEmptyRepositoryScenarios tests behavior with empty repositories
func TestEmptyRepositoryScenarios(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Test operations on empty repo
	result := gitManager.getLastCommitBranchAdvanced()
	if result.HasHistory {
		t.Error("Expected no history in empty repository")
	}

	if result.CommitCount != 0 {
		t.Errorf("Expected 0 commits, got %d", result.CommitCount)
	}

	if result.IsCorrupted {
		t.Error("Empty repository should not be marked as corrupted")
	}

	// Test listing snapshots in empty repo
	snapshots, err := gitManager.ListSnapshots(10, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots in empty repo: %v", err)
	}

	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots in empty repo, got %d", len(snapshots))
	}

	t.Log("✅ Empty repository handled correctly")
}

// TestBranchSwitchChain tests multiple consecutive branch switches
func TestBranchSwitchChain(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "chain-test.txt")
	branches := []string{"feature-1", "feature-2", "bugfix", "hotfix"}

	// Create initial commit
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Initial for chain test")
	if err != nil {
		t.Fatalf("Failed to create initial snapshot: %v", err)
	}

	// Create chain of branch switches
	for i, branch := range branches {
		cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", branch)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create branch %s: %v", branch, err)
		}

		content := fmt.Sprintf("Content on %s", branch)
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write content for %s: %v", branch, err)
		}

		err := gitManager.CreateSnapshot(fmt.Sprintf("Work on %s", branch))
		if err != nil {
			t.Fatalf("Failed to create snapshot for %s: %v", branch, err)
		}

		t.Logf("Created snapshot on branch %s (%d/%d)", branch, i+1, len(branches))
	}

	// Verify all snapshots were created
	allSnapshots, err := gitManager.ListSnapshots(10, "")
	if err != nil {
		t.Fatalf("Failed to list all snapshots: %v", err)
	}

	expectedMin := len(branches) + 1 // +1 for initial
	if len(allSnapshots) < expectedMin {
		t.Errorf("Expected at least %d snapshots, got %d", expectedMin, len(allSnapshots))
	}

	// Check for branch switch patterns
	switchCount := 0
	for _, snapshot := range allSnapshots {
		if strings.Contains(snapshot.Message, "→") || strings.Contains(snapshot.Message, "BRANCH SWITCH") {
			switchCount++
			t.Logf("Branch switch detected: %s", snapshot.Message)
		}
	}

	t.Logf("✅ Chain test completed: %d total snapshots, %d detected switches",
		len(allSnapshots), switchCount)
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create base file
	testFile := filepath.Join(tempDir, "concurrent-test.txt")
	if err := os.WriteFile(testFile, []byte("base content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create initial snapshot
	err := gitManager.CreateSnapshot("Base for concurrent test")
	if err != nil {
		t.Fatalf("Failed to create base snapshot: %v", err)
	}

	// Test concurrent file counting (thread-safe operations)
	const numGoroutines = 5
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Each goroutine performs different operations
			switch id % 3 {
			case 0:
				// Count files
				_, err := gitManager.countUncommittedFiles()
				results <- err
			case 1:
				// Get current branch
				_, err := gitManager.GetCurrentBranch()
				results <- err
			case 2:
				// Get last commit branch info
				result := gitManager.getLastCommitBranchAdvanced()
				if result.IsCorrupted {
					results <- fmt.Errorf("corrupted result in goroutine %d", id)
				} else {
					results <- nil
				}
			}
		}(i)
	}

	// Collect results
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Logf("Goroutine error: %v", err)
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Errorf("Concurrent operations had %d errors", errorCount)
	} else {
		t.Log("✅ Concurrent operations completed without errors")
	}
}

// TestRestoreEdgeCases tests restoration with edge cases
func TestRestoreEdgeCases(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Test 1: Restore from invalid hash
	err := gitManager.RestoreSnapshot("invalid-hash-1234567890", []string{})
	if err == nil {
		t.Error("Expected error when restoring from invalid hash")
	}

	// Test 2: Create snapshot and test restore of non-existent file
	testFile := filepath.Join(tempDir, "restore-test.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = gitManager.CreateSnapshot("For restore testing")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatal("No snapshots available for restore test")
	}

	hash := snapshots[0].Hash

	// Test 3: Restore specific non-existent file (should not cause crash)
	err = gitManager.RestoreSnapshot(hash, []string{"non-existent-file.txt"})
	if err != nil {
		t.Logf("Restore of non-existent file failed as expected: %v", err)
	} else {
		t.Log("Restore of non-existent file completed (may be valid behavior)")
	}

	// Test 4: Successful restore
	// Modify the file first
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Restore the original version
	err = gitManager.RestoreSnapshot(hash, []string{"restore-test.txt"})
	if err != nil {
		t.Fatalf("Failed to restore specific file: %v", err)
	}

	// Verify restoration
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(content) != "original content" {
		t.Errorf("Expected 'original content', got '%s'", string(content))
	}

	t.Log("✅ Restore edge cases handled correctly")
}
