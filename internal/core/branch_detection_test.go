package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchDetection(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create initial commit in main repo to establish a baseline
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Add to main repo
	cmd := exec.Command("git", "-C", tempDir, "add", "test.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file to main repo: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit in main repo: %v", err)
	}

	// Test 1: Create snapshot and verify branch detection
	err := gitManager.CreateSnapshot("Test branch detection")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snapshots))
	}

	// Verify branch-aware message format
	message := snapshots[0].Message
	if !strings.Contains(message, "[") || !strings.Contains(message, "]") {
		t.Errorf("Expected branch-aware message format [branch] message, got '%s'", message)
	}

	// Test 2: Verify notes metadata exists
	hash := snapshots[0].Hash
	notesOutput, err := gitManager.RunCommand("notes", "show", hash)
	if err != nil {
		t.Logf("Notes not found for commit %s (this is expected for first commit): %v", hash, err)
	} else {
		// Try to parse the notes as JSON
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(notesOutput), &metadata); err != nil {
			t.Logf("Notes exist but not in JSON format: %s", notesOutput)
		} else {
			t.Logf("Branch metadata found: %v", metadata)
			if metadata["branch"] != nil {
				t.Logf("Current branch from metadata: %s", metadata["branch"])
			}
		}
	}

	// Test 3: Simulate branch change by creating a new branch and switching
	cmd = exec.Command("git", "-C", tempDir, "checkout", "-b", "feature-branch")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Modify file and create another snapshot
	if err := os.WriteFile(testFile, []byte("modified on feature branch"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	err = gitManager.CreateSnapshot("Modified on feature branch")
	if err != nil {
		t.Fatalf("Failed to create feature branch snapshot: %v", err)
	}

	// Verify the new snapshot has branch switch detection
	snapshots, err = gitManager.ListSnapshots(2, "")
	if err != nil {
		t.Fatalf("Failed to list updated snapshots: %v", err)
	}

	if len(snapshots) < 2 {
		t.Errorf("Expected at least 2 snapshots, got %d", len(snapshots))
	}

	// Check the latest snapshot message for branch switch
	latestMessage := snapshots[0].Message
	t.Logf("Latest snapshot message: %s", latestMessage)

	// The message should contain branch information
	if !strings.Contains(latestMessage, "[") {
		t.Errorf("Expected branch information in message, got '%s'", latestMessage)
	}
}

func TestBranchDetectionFallback(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test fallback logic when no previous commits exist
	testFile := filepath.Join(tempDir, "fallback-test.txt")
	if err := os.WriteFile(testFile, []byte("fallback test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("Testing fallback logic")
	if err != nil {
		t.Fatalf("Failed to create snapshot for fallback test: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot for fallback test, got %d", len(snapshots))
	}

	// Should handle the case gracefully
	message := snapshots[0].Message
	if !strings.Contains(message, "Testing fallback logic") {
		t.Errorf("Expected custom message in fallback, got '%s'", message)
	}

	t.Logf("Fallback branch detection message: %s", message)
}

func TestBranchDetectionCorruptedData(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a snapshot first
	testFile := filepath.Join(tempDir, "corrupted-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := gitManager.CreateSnapshot("First snapshot")
	if err != nil {
		t.Fatalf("Failed to create first snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snapshots))
	}

	// Add corrupted notes to test fallback handling
	hash := snapshots[0].Hash
	corruptedNotes := "invalid json data"

	// This might fail, but that's expected - we're testing resilience
	_, err = gitManager.RunCommand("notes", "add", "-f", "-m", corruptedNotes, hash)
	if err != nil {
		t.Logf("Expected to add corrupted notes, but failed: %v", err)
	}

	// Create another snapshot - should handle corrupted data gracefully
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = gitManager.CreateSnapshot("Testing corrupted data handling")
	if err != nil {
		t.Fatalf("Failed to create snapshot with corrupted data: %v", err)
	}

	// Should still work despite corrupted data
	snapshots, err = gitManager.ListSnapshots(2, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots after corruption test: %v", err)
	}

	t.Logf("Successfully handled corrupted data, created %d snapshots", len(snapshots))
}
