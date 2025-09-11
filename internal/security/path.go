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

	// For absolute paths (system-internal paths like ShadowRepoDir), check basic security constraints
	if filepath.IsAbs(cleaned) {
		// Prevent path traversal in absolute paths by checking the original path before cleaning
		// This catches cases where "../.." resolves to an unexpected absolute path
		if strings.Contains(path, "..") {
			return "", fmt.Errorf("path traversal not allowed in absolute path")
		}
		return cleaned, nil
	}

	// For relative paths (user inputs), use Go's built-in security validation (Go 1.20+)
	// This handles cross-platform path traversal prevention automatically
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("path must be local and relative")
	}

	return cleaned, nil
}

// ValidateUserInputPath validates paths from user inputs with strict relative-only validation
// This function only allows relative paths for maximum security when handling user inputs
func ValidateUserInputPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path not allowed")
	}

	// Clean the path using OS-appropriate rules
	cleaned := filepath.Clean(path)

	// For user inputs, we only allow relative paths for security
	// Use Go's built-in security validation (Go 1.20+)
	// This handles cross-platform path traversal prevention automatically
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("path must be local and relative")
	}

	return cleaned, nil
}