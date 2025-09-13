package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateSystemPath validates system directory paths (shadow repo, project root, git dirs)
// This function allows absolute paths which are required for system operations
func ValidateSystemPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path not allowed")
	}

	// Clean the path using OS-appropriate rules
	cleaned := filepath.Clean(path)

	// For absolute paths (system-internal paths like ShadowRepoDir), basic validation
	if filepath.IsAbs(cleaned) {
		// Just prevent obvious traversal attempts in absolute paths
		if strings.Contains(path, "..") {
			return "", fmt.Errorf("path traversal not allowed in absolute path")
		}
		return cleaned, nil
	}

	// For relative paths, use Go's built-in security validation
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("path must be local and relative")
	}

	return cleaned, nil
}

// ValidateUserInputPath validates user file paths (simple validation for local CLI tool)
// For a Git-based local tool, we just need basic traversal prevention
func ValidateUserInputPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path not allowed")
	}

	// Clean the path and use Go's built-in security validation
	cleaned := filepath.Clean(path)
	
	// Go 1.20+ filepath.IsLocal handles all the complex cross-platform security cases
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("path must be local and relative")
	}

	return cleaned, nil
}