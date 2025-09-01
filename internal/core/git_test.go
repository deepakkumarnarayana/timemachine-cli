package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitManager_RunCommand(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "timemachine-git-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create main .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Initialize main repo to copy config from
	cmd := exec.Command("git", "init", tempDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init main repo: %v", err)
	}

	// Set test user config in main repo
	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set user.name: %v", err)
	}
	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set user.email: %v", err)
	}

	// Create AppState
	state := &AppState{
		ProjectRoot:   tempDir,
		GitDir:        gitDir,
		ShadowRepoDir: filepath.Join(gitDir, "timemachine_snapshots"),
		IsInitialized: false,
	}

	gitManager := NewGitManager(state)

	// Test RunCommand before initialization (should fail)
	_, err = gitManager.RunCommand("status")
	if err == nil {
		t.Error("Expected error for git command before initialization")
	}

	// Initialize shadow repo
	err = gitManager.InitializeShadowRepo()
	if err != nil {
		t.Fatalf("Failed to initialize shadow repo: %v", err)
	}

	// Test RunCommand after initialization
	output, err := gitManager.RunCommand("status", "--porcelain")
	if err != nil {
		t.Errorf("Git status failed: %v", err)
	}

	// Status should be empty initially
	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected empty status, got: %s", output)
	}
}

func TestGitManager_InitializeShadowRepo(t *testing.T) {
	// Create test environment
	tempDir, err := os.MkdirTemp("", "timemachine-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Initialize main repo
	cmd := exec.Command("git", "init", tempDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init main repo: %v", err)
	}

	// Set user config in main repo
	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set user.name: %v", err)
	}
	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set user.email: %v", err)
	}

	state := &AppState{
		ProjectRoot:   tempDir,
		GitDir:        gitDir,
		ShadowRepoDir: filepath.Join(gitDir, "timemachine_snapshots"),
		IsInitialized: false,
	}

	gitManager := NewGitManager(state)

	// Test initialization
	err = gitManager.InitializeShadowRepo()
	if err != nil {
		t.Fatalf("Failed to initialize shadow repo: %v", err)
	}

	// Verify shadow repo directory exists
	if _, err := os.Stat(state.ShadowRepoDir); os.IsNotExist(err) {
		t.Error("Shadow repo directory was not created")
	}

	// Verify HEAD file exists
	headFile := filepath.Join(state.ShadowRepoDir, "HEAD")
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		t.Error("HEAD file was not created in shadow repo")
	}

	// Verify user config was copied
	name, err := gitManager.RunCommand("config", "user.name")
	if err != nil {
		t.Errorf("Failed to get user.name from shadow repo: %v", err)
	}
	if name != "Test User" {
		t.Errorf("Expected user.name 'Test User', got '%s'", name)
	}

	email, err := gitManager.RunCommand("config", "user.email")
	if err != nil {
		t.Errorf("Failed to get user.email from shadow repo: %v", err)
	}
	if email != "test@example.com" {
		t.Errorf("Expected user.email 'test@example.com', got '%s'", email)
	}
}

func TestGitManager_CreateSnapshot(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, Time Machine!"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test creating snapshot with custom message
	err := gitManager.CreateSnapshot("Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Verify commit was created
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snapshots))
	}

	if snapshots[0].Message != "Test snapshot" {
		t.Errorf("Expected message 'Test snapshot', got '%s'", snapshots[0].Message)
	}

	// Test creating snapshot with auto-generated message
	if err := os.WriteFile(testFile, []byte("Updated content"), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	err = gitManager.CreateSnapshot("")
	if err != nil {
		t.Fatalf("Failed to create auto-named snapshot: %v", err)
	}

	snapshots, err = gitManager.ListSnapshots(2, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(snapshots))
	}

	// Check auto-generated message format
	if !strings.Contains(snapshots[0].Message, "Snapshot at") {
		t.Errorf("Expected auto-generated message to contain 'Snapshot at', got '%s'", snapshots[0].Message)
	}
}

func TestGitManager_ListSnapshots(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Test empty repository
	snapshots, err := gitManager.ListSnapshots(10, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots from empty repo: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots from empty repo, got %d", len(snapshots))
	}

	// Create test files and snapshots
	testFiles := []string{"file1.txt", "file2.txt", "dir/file3.txt"}
	for i, fileName := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		
		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		
		content := []byte("Content " + string(rune('A'+i)))
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}

		if err := gitManager.CreateSnapshot("Snapshot " + string(rune('1'+i))); err != nil {
			t.Fatalf("Failed to create snapshot %d: %v", i+1, err)
		}
	}

	// Test listing all snapshots
	snapshots, err = gitManager.ListSnapshots(0, "")
	if err != nil {
		t.Fatalf("Failed to list all snapshots: %v", err)
	}
	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Test limit
	snapshots, err = gitManager.ListSnapshots(2, "")
	if err != nil {
		t.Fatalf("Failed to list limited snapshots: %v", err)
	}
	if len(snapshots) != 2 {
		t.Errorf("Expected 2 snapshots with limit, got %d", len(snapshots))
	}

	// Test file filter
	snapshots, err = gitManager.ListSnapshots(0, "file1.txt")
	if err != nil {
		t.Fatalf("Failed to list snapshots for specific file: %v", err)
	}
	// Should find at least the snapshot where file1.txt was added
	if len(snapshots) == 0 {
		t.Error("Expected at least 1 snapshot for file1.txt, got 0")
	}
}

func TestGitManager_RestoreSnapshot(t *testing.T) {
	// Create test environment
	tempDir, _, gitManager := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	// Create initial file and snapshot
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("Original content")
	if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := gitManager.CreateSnapshot("Original snapshot"); err != nil {
		t.Fatalf("Failed to create original snapshot: %v", err)
	}

	// Get the snapshot hash
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to get snapshots: %v", err)
	}
	originalHash := snapshots[0].Hash

	// Modify file
	modifiedContent := []byte("Modified content")
	if err := os.WriteFile(testFile, modifiedContent, 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Verify file was modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != "Modified content" {
		t.Error("File was not modified as expected")
	}

	// Restore from snapshot
	err = gitManager.RestoreSnapshot(originalHash, []string{})
	if err != nil {
		t.Fatalf("Failed to restore snapshot: %v", err)
	}

	// Verify file was restored
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}
	if string(content) != "Original content" {
		t.Errorf("Expected 'Original content', got '%s'", string(content))
	}

	// Test restoring specific files
	file2 := filepath.Join(tempDir, "test2.txt")
	if err := os.WriteFile(file2, []byte("File 2 content"), 0644); err != nil {
		t.Fatalf("Failed to create test2 file: %v", err)
	}

	if err := gitManager.CreateSnapshot("Added file2"); err != nil {
		t.Fatalf("Failed to create second snapshot: %v", err)
	}

	// Modify both files
	if err := os.WriteFile(testFile, []byte("Modified again"), 0644); err != nil {
		t.Fatalf("Failed to modify test file again: %v", err)
	}
	if err := os.WriteFile(file2, []byte("File 2 modified"), 0644); err != nil {
		t.Fatalf("Failed to modify test2 file: %v", err)
	}

	// Restore only test.txt from original snapshot
	err = gitManager.RestoreSnapshot(originalHash, []string{"test.txt"})
	if err != nil {
		t.Fatalf("Failed to restore specific file: %v", err)
	}

	// Verify test.txt was restored but test2.txt wasn't
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != "Original content" {
		t.Errorf("test.txt: Expected 'Original content', got '%s'", string(content))
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("Failed to read test2 file: %v", err)
	}
	if string(content2) != "File 2 modified" {
		t.Errorf("test2.txt: Expected 'File 2 modified', got '%s'", string(content2))
	}
}

// Helper function to set up a test repository
func setupTestRepo(t *testing.T) (string, *AppState, *GitManager) {
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

	state := &AppState{
		ProjectRoot:   tempDir,
		GitDir:        gitDir,
		ShadowRepoDir: filepath.Join(gitDir, "timemachine_snapshots"),
		IsInitialized: false,
	}

	gitManager := NewGitManager(state)

	// Initialize shadow repo
	if err := gitManager.InitializeShadowRepo(); err != nil {
		t.Fatalf("Failed to initialize shadow repo: %v", err)
	}

	return tempDir, state, gitManager
}