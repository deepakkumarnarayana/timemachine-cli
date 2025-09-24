package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGetSnapshotStats tests the calculation of file change statistics
func TestGetSnapshotStats(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-stats-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create mock Git repos
	repoDir := filepath.Join(tempDir, "repo")
	shadowRepoDir := filepath.Join(repoDir, ".git", "timemachine_snapshots")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Create AppState
	state := &AppState{
		ProjectRoot:   repoDir,
		GitDir:        filepath.Join(repoDir, ".git"),
		ShadowRepoDir: shadowRepoDir,
		IsInitialized: false,
	}

	// Initialize main repository
	setupStatsTestRepo(t, repoDir)

	// Create GitManager and initialize shadow repo
	gitManager := NewGitManager(state)
	err = gitManager.SetupShadowRepo()
	if err != nil {
		t.Fatalf("Failed to setup shadow repo: %v", err)
	}

	// Create test files with different content
	testFile1 := filepath.Join(repoDir, "test1.txt")
	testFile2 := filepath.Join(repoDir, "test2.go")

	// Create files with specific content to test line counting
	content1 := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	content2 := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"

	if err := os.WriteFile(testFile1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Create initial snapshot
	err = gitManager.CreateSnapshot("Initial snapshot with test files")
	if err != nil {
		t.Fatalf("Failed to create initial snapshot: %v", err)
	}

	// Get the snapshot hash
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatalf("No snapshots found")
	}

	snapshot := snapshots[0]
	t.Logf("Testing snapshot: %s - %s", snapshot.Hash, snapshot.Message)

	// Test that statistics are populated
	if snapshot.FilesChanged == 0 {
		t.Errorf("Expected FilesChanged > 0, got %d", snapshot.FilesChanged)
	}

	// Expected: 2 files (test1.txt and test2.go)
	if snapshot.FilesChanged < 2 {
		t.Errorf("Expected at least 2 files changed, got %d", snapshot.FilesChanged)
	}

	// Test getSnapshotStats directly
	stats, err := gitManager.getSnapshotStats(snapshot.Hash)
	if err != nil {
		t.Fatalf("Failed to get snapshot stats: %v", err)
	}

	if stats.FilesChanged < 2 {
		t.Errorf("Expected at least 2 files in stats, got %d", stats.FilesChanged)
	}

	if stats.LinesAdded == 0 {
		t.Errorf("Expected some lines added, got %d", stats.LinesAdded)
	}

	// Lines should match content we wrote (5 lines in test1.txt + 5 lines in test2.go)
	if stats.LinesAdded < 8 {
		t.Errorf("Expected at least 8 lines added, got %d", stats.LinesAdded)
	}

	t.Logf("Snapshot statistics: %d files, +%d/-%d lines",
		stats.FilesChanged, stats.LinesAdded, stats.LinesRemoved)
}

// TestGetSnapshotStatsModifications tests statistics for file modifications
func TestGetSnapshotStatsModifications(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-stats-mod-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create mock Git repos
	repoDir := filepath.Join(tempDir, "repo")
	shadowRepoDir := filepath.Join(repoDir, ".git", "timemachine_snapshots")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Create AppState
	state := &AppState{
		ProjectRoot:   repoDir,
		GitDir:        filepath.Join(repoDir, ".git"),
		ShadowRepoDir: shadowRepoDir,
		IsInitialized: false,
	}

	// Initialize main repository
	setupStatsTestRepo(t, repoDir)

	// Create GitManager and initialize shadow repo
	gitManager := NewGitManager(state)
	err = gitManager.SetupShadowRepo()
	if err != nil {
		t.Fatalf("Failed to setup shadow repo: %v", err)
	}

	// Create initial file
	testFile := filepath.Join(repoDir, "modify_test.txt")
	originalContent := "line 1\nline 2\nline 3\n"

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create first snapshot
	err = gitManager.CreateSnapshot("Initial file")
	if err != nil {
		t.Fatalf("Failed to create initial snapshot: %v", err)
	}

	// Modify the file (add lines and remove some)
	modifiedContent := "line 1\nline 2 modified\nline 3\nline 4 new\nline 5 new\n"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Create second snapshot
	err = gitManager.CreateSnapshot("Modified file with additions")
	if err != nil {
		t.Fatalf("Failed to create modified snapshot: %v", err)
	}

	// Get the latest snapshot (modification)
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	modSnapshot := snapshots[0]
	t.Logf("Testing modification snapshot: %s", modSnapshot.Hash)

	// Verify statistics for modification
	if modSnapshot.FilesChanged != 1 {
		t.Errorf("Expected 1 file changed for modification, got %d", modSnapshot.FilesChanged)
	}

	if modSnapshot.LinesAdded == 0 {
		t.Errorf("Expected some lines added in modification, got %d", modSnapshot.LinesAdded)
	}

	t.Logf("Modification statistics: %d files, +%d/-%d lines",
		modSnapshot.FilesChanged, modSnapshot.LinesAdded, modSnapshot.LinesRemoved)
}

// TestListSnapshotsWithStats tests that ListSnapshots returns populated statistics
func TestListSnapshotsWithStats(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-list-stats-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create mock Git repos
	repoDir := filepath.Join(tempDir, "repo")
	shadowRepoDir := filepath.Join(repoDir, ".git", "timemachine_snapshots")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Create AppState
	state := &AppState{
		ProjectRoot:   repoDir,
		GitDir:        filepath.Join(repoDir, ".git"),
		ShadowRepoDir: shadowRepoDir,
		IsInitialized: false,
	}

	// Initialize main repository
	setupStatsTestRepo(t, repoDir)

	// Create GitManager and initialize shadow repo
	gitManager := NewGitManager(state)
	err = gitManager.SetupShadowRepo()
	if err != nil {
		t.Fatalf("Failed to setup shadow repo: %v", err)
	}

	// Create multiple files for comprehensive test
	files := map[string]string{
		"file1.txt": "content1\ncontent2\ncontent3\n",
		"file2.go":  "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n",
		"file3.md":  "# Title\n\nSome content\nMore content\n",
	}

	for filename, content := range files {
		filePath := filepath.Join(repoDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Create snapshot
	err = gitManager.CreateSnapshot("Multiple files with content")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Test ListSnapshots returns proper statistics
	snapshots, err := gitManager.ListSnapshots(5, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatalf("No snapshots returned")
	}

	snapshot := snapshots[0]

	// Verify all new fields are populated
	if snapshot.Hash == "" {
		t.Errorf("Hash should not be empty")
	}
	if snapshot.Message == "" {
		t.Errorf("Message should not be empty")
	}
	if snapshot.Time == "" {
		t.Errorf("Time should not be empty")
	}
	if snapshot.Author == "" {
		t.Errorf("Author should not be empty")
	}
	if snapshot.AbsoluteTime == "" {
		t.Errorf("AbsoluteTime should not be empty")
	}

	// Verify statistics are reasonable
	if snapshot.FilesChanged < 3 {
		t.Errorf("Expected at least 3 files changed, got %d", snapshot.FilesChanged)
	}
	if snapshot.LinesAdded < 10 {
		t.Errorf("Expected at least 10 lines added, got %d", snapshot.LinesAdded)
	}

	// Removed lines should be 0 for new files
	if snapshot.LinesRemoved != 0 {
		t.Errorf("Expected 0 lines removed for new files, got %d", snapshot.LinesRemoved)
	}

	t.Logf("Snapshot: %s", snapshot.Hash[:8])
	t.Logf("Message: %s", snapshot.Message)
	t.Logf("Author: %s", snapshot.Author)
	t.Logf("Statistics: %d files, +%d/-%d lines",
		snapshot.FilesChanged, snapshot.LinesAdded, snapshot.LinesRemoved)
}

// TestGetSnapshotStatsEmpty tests behavior with empty/non-existent snapshots
func TestGetSnapshotStatsEmpty(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-stats-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create mock Git repos
	repoDir := filepath.Join(tempDir, "repo")
	shadowRepoDir := filepath.Join(repoDir, ".git", "timemachine_snapshots")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Create AppState
	state := &AppState{
		ProjectRoot:   repoDir,
		GitDir:        filepath.Join(repoDir, ".git"),
		ShadowRepoDir: shadowRepoDir,
		IsInitialized: false,
	}

	// Initialize main repository
	setupStatsTestRepo(t, repoDir)

	// Create GitManager and initialize shadow repo
	gitManager := NewGitManager(state)
	err = gitManager.SetupShadowRepo()
	if err != nil {
		t.Fatalf("Failed to setup shadow repo: %v", err)
	}

	// Test with invalid hash - should return zero stats without error
	stats, err := gitManager.getSnapshotStats("invalid-hash-123")
	if err != nil {
		// Function should not return error, but return zero stats
		t.Logf("Expected no error for invalid hash, but got: %v", err)
	}

	// Should return zero statistics
	if stats.FilesChanged != 0 || stats.LinesAdded != 0 || stats.LinesRemoved != 0 {
		t.Errorf("Expected zero stats for invalid hash, got %+v", stats)
	}

	t.Logf("Invalid hash correctly returns zero stats: %+v", stats)
}

// Helper function to setup test repository for stats testing
func setupStatsTestRepo(t *testing.T, repoDir string) {
	// Change to repo directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo dir: %v", err)
	}

	// Initialize Git repo
	if err := runGitCommand("init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runGitCommand("config", "user.name", "Test User"); err != nil {
		t.Fatalf("Failed to set git user name: %v", err)
	}
	if err := runGitCommand("config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("Failed to set git user email: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if err := runGitCommand("add", "README.md"); err != nil {
		t.Fatalf("Failed to add README: %v", err)
	}
	if err := runGitCommand("commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
}

// Helper function to run git commands
func runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	return cmd.Run()
}

// TestGetSnapshotStatsBinaryFiles tests handling of binary files
func TestGetSnapshotStatsBinaryFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-stats-binary-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create mock Git repos
	repoDir := filepath.Join(tempDir, "repo")
	shadowRepoDir := filepath.Join(repoDir, ".git", "timemachine_snapshots")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Create AppState
	state := &AppState{
		ProjectRoot:   repoDir,
		GitDir:        filepath.Join(repoDir, ".git"),
		ShadowRepoDir: shadowRepoDir,
		IsInitialized: false,
	}

	// Initialize main repository
	setupStatsTestRepo(t, repoDir)

	// Create GitManager and initialize shadow repo
	gitManager := NewGitManager(state)
	err = gitManager.SetupShadowRepo()
	if err != nil {
		t.Fatalf("Failed to setup shadow repo: %v", err)
	}

	// Create a text file and simulate binary file content
	textFile := filepath.Join(repoDir, "text.txt")
	binaryFile := filepath.Join(repoDir, "binary.dat")

	textContent := "This is a text file\nwith multiple lines\nfor testing\n"
	binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header

	if err := os.WriteFile(textFile, []byte(textContent), 0644); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	// Create snapshot
	err = gitManager.CreateSnapshot("Mixed text and binary files")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Get snapshot statistics
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatalf("No snapshots found")
	}

	snapshot := snapshots[0]

	// Should count both files
	if snapshot.FilesChanged < 2 {
		t.Errorf("Expected at least 2 files (text + binary), got %d", snapshot.FilesChanged)
	}

	// Should have line count from text file (binary files show as "-" in git numstat)
	if snapshot.LinesAdded == 0 {
		t.Errorf("Expected some lines added from text file, got %d", snapshot.LinesAdded)
	}

	t.Logf("Mixed content statistics: %d files, +%d/-%d lines",
		snapshot.FilesChanged, snapshot.LinesAdded, snapshot.LinesRemoved)
}