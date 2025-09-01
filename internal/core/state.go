package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// AppState contains the application state and paths
type AppState struct {
	ProjectRoot   string // Absolute path to project root (parent of .git)
	GitDir        string // Path to .git directory
	ShadowRepoDir string // Path to .git/timemachine_snapshots
	IsInitialized bool   // Whether shadow repo exists and is valid
}

// NewAppState creates a new AppState by finding the Git repository
// and checking if the shadow repository is initialized
func NewAppState() (*AppState, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Walk up directory tree looking for .git directory
	gitDir := findGitDir(cwd)
	if gitDir == "" {
		return nil, errors.New("not in a Git repository (or any parent directory)")
	}

	// Set ProjectRoot to parent of .git
	projectRoot := filepath.Dir(gitDir)
	
	// Set ShadowRepoDir to .git/timemachine_snapshots
	shadowRepoDir := filepath.Join(gitDir, "timemachine_snapshots")
	
	// Check if shadow repo exists by looking for HEAD file
	headFile := filepath.Join(shadowRepoDir, "HEAD")
	isInitialized := false
	if _, err := os.Stat(headFile); err == nil {
		isInitialized = true
	}

	return &AppState{
		ProjectRoot:   projectRoot,
		GitDir:        gitDir,
		ShadowRepoDir: shadowRepoDir,
		IsInitialized: isInitialized,
	}, nil
}

// findGitDir searches for a .git directory starting from the given directory
// and walking up the directory tree until it finds one or reaches the filesystem root
func findGitDir(startDir string) string {
	currentDir := startDir
	
	for {
		// Check for .git directory in current directory
		gitPath := filepath.Join(currentDir, ".git")
		
		// Check if .git exists and is a directory (not a file, which could be a submodule)
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return gitPath
		}
		
		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		
		// Stop if we've reached the filesystem root
		if parentDir == currentDir {
			break
		}
		
		currentDir = parentDir
	}
	
	// Not found
	return ""
}