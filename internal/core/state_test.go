package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitDir(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "timemachine-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test case 1: Directory with .git
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	result := findGitDir(tempDir)
	expected := gitDir
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test case 2: Subdirectory should find parent .git
	subDir := filepath.Join(tempDir, "subdir", "deeper")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirs: %v", err)
	}

	result = findGitDir(subDir)
	if result != expected {
		t.Errorf("Expected %s from subdir, got %s", expected, result)
	}

	// Test case 3: Directory without .git
	noGitDir, err := os.MkdirTemp("", "no-git")
	if err != nil {
		t.Fatalf("Failed to create no-git temp dir: %v", err)
	}
	defer os.RemoveAll(noGitDir)

	result = findGitDir(noGitDir)
	if result != "" {
		t.Errorf("Expected empty string for no-git dir, got %s", result)
	}
}

func TestNewAppState(t *testing.T) {
	// Create a temporary directory structure with .git
	tempDir, err := os.MkdirTemp("", "timemachine-appstate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Change to the temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Test case 1: Valid Git repository
	state, err := NewAppState()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.ProjectRoot != tempDir {
		t.Errorf("Expected ProjectRoot %s, got %s", tempDir, state.ProjectRoot)
	}

	if state.GitDir != gitDir {
		t.Errorf("Expected GitDir %s, got %s", gitDir, state.GitDir)
	}

	expectedShadowDir := filepath.Join(gitDir, "timemachine_snapshots")
	if state.ShadowRepoDir != expectedShadowDir {
		t.Errorf("Expected ShadowRepoDir %s, got %s", expectedShadowDir, state.ShadowRepoDir)
	}

	if state.IsInitialized {
		t.Error("Expected IsInitialized to be false initially")
	}

	// Test case 2: With initialized shadow repo
	shadowRepoDir := filepath.Join(gitDir, "timemachine_snapshots")
	if err := os.Mkdir(shadowRepoDir, 0755); err != nil {
		t.Fatalf("Failed to create shadow repo dir: %v", err)
	}

	// Create HEAD file to simulate initialized repo
	headFile := filepath.Join(shadowRepoDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("Failed to create HEAD file: %v", err)
	}

	state, err = NewAppState()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !state.IsInitialized {
		t.Error("Expected IsInitialized to be true with HEAD file present")
	}
}

func TestNewAppStateNoGit(t *testing.T) {
	// Create a temporary directory without .git
	tempDir, err := os.MkdirTemp("", "timemachine-nogit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to the temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Should return error for non-Git directory
	_, err = NewAppState()
	if err == nil {
		t.Error("Expected error for non-Git directory, got nil")
	}

	expectedErrorMsg := "not in a Git repository"
	if err != nil && !contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(substr) <= len(s) && s[0:len(substr)] == substr) || contains(s[1:], substr))
}

func TestFindGitDirNestedStructure(t *testing.T) {
	// Create a complex nested structure to test directory traversal
	tempDir, err := os.MkdirTemp("", "timemachine-nested-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create structure: tempDir/.git and tempDir/project/src/deep/nested/
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	deepDir := filepath.Join(tempDir, "project", "src", "deep", "nested")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("Failed to create deep nested dirs: %v", err)
	}

	// Test from deeply nested directory
	result := findGitDir(deepDir)
	if result != gitDir {
		t.Errorf("Expected to find .git at %s from deeply nested dir, got %s", gitDir, result)
	}
}