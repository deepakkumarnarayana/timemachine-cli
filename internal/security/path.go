package security

import (
	"fmt"
	"net/url"
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

	// For system paths, only check for basic Windows reserved device names
	// Allow Windows drive letters for legitimate system operations
	if err := validateWindowsReservedDeviceNames(cleaned); err != nil {
		return "", fmt.Errorf("windows reserved device name violation: %w", err)
	}

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

// ValidateUserInputPath validates paths from user inputs with comprehensive defense-in-depth security
// This function implements OWASP 2025 best practices with multiple security layers
func ValidateUserInputPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path not allowed")
	}

	// Layer 1: Block dangerous characters that could enable attacks
	if err := validatePathCharacters(path); err != nil {
		return "", fmt.Errorf("invalid characters in path: %w", err)
	}

	// Layer 2: Detect common attack patterns
	if err := detectAttackPatterns(path); err != nil {
		return "", fmt.Errorf("path traversal attack detected: %w", err)
	}

	// Layer 3a: Clean the path using OS-appropriate rules (handles normalization)
	cleaned := filepath.Clean(path)

	// Layer 3b: Use Go's security validation (Go 1.20+) - handles traversal prevention
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("path must be local and relative")
	}

	// Layer 4: Always validate Windows-specific patterns (like Git's core.protectNTFS=true)
	// This prevents creating files that would be invalid on Windows systems
	if err := validateWindowsSystemPath(cleaned); err != nil {
		return "", fmt.Errorf("windows security violation: %w", err)
	}

	// Layer 5: Additional boundary validation for defense-in-depth
	if err := validatePathBoundaries(cleaned); err != nil {
		return "", fmt.Errorf("path boundary violation: %w", err)
	}

	// Final verification: ensure the cleaned path is still safe
	if err := detectAttackPatterns(cleaned); err != nil {
		return "", fmt.Errorf("path became unsafe after cleaning: %w", err)
	}

	return cleaned, nil
}

// validatePathCharacters blocks dangerous characters that could enable security attacks
// Uses focused blacklist approach for characters that are definitively dangerous
func validatePathCharacters(path string) error {
	// Check for null bytes (classic path traversal bypass technique)
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("null byte injection detected")
	}

	// Check for control characters that could enable attacks
	for i, r := range path {
		// Block ASCII control characters (except tab which might be legitimate)
		if r < 32 && r != '\t' {
			return fmt.Errorf("control character at position %d", i)
		}
		
		// Block characters commonly used in command injection
		switch r {
		case '|', '&', ';', '`', '$':
			return fmt.Errorf("command injection character '%c' at position %d", r, i)
		case '<', '>':
			return fmt.Errorf("redirection character '%c' at position %d", r, i)
		}
		
		// Block Unicode control characters that could be dangerous
		if r == '\u0000' || (r >= '\u007F' && r <= '\u009F') {
			return fmt.Errorf("dangerous Unicode control character at position %d", i)
		}
	}

	return nil
}

// detectAttackPatterns identifies common path traversal attack patterns
func detectAttackPatterns(path string) error {
	// Normalize case for consistent detection
	pathLower := strings.ToLower(path)
	
	// Also check URL-decoded version to catch encoded attacks
	urlDecodedPath, err := urlDecodeForSecurityCheck(path)
	if err == nil {
		urlDecodedLower := strings.ToLower(urlDecodedPath)
		// Recursively check the decoded path for attacks
		if err := detectAttackPatternsInString(urlDecodedLower); err != nil {
			return fmt.Errorf("URL-encoded attack detected: %w", err)
		}
	}
	
	return detectAttackPatternsInString(pathLower)
}

// detectAttackPatternsInString performs the actual pattern detection on a normalized string
func detectAttackPatternsInString(pathLower string) error {
	
	// Check for directory traversal patterns
	traversalPatterns := []string{
		"../", "..\\", 
		"..%2f", "..%5c",           // URL encoded
		"..%252f", "..%255c",       // Double URL encoded
		"....//", "....\\\\",       // Double dot bypass
		"%2e%2e%2f", "%2e%2e%5c",   // Fully URL encoded dots
		"\u002e\u002e\u002f",       // Unicode encoded
		"\uff0e\uff0e\uff0f",       // Full-width Unicode
		"..%c0%af", "..%e0%80%af",  // Unicode overlong encodings
		"..%c1%9c",                 // Additional overlong encoding
	}

	for _, pattern := range traversalPatterns {
		if strings.Contains(pathLower, pattern) {
			return fmt.Errorf("directory traversal pattern detected: %s", pattern)
		}
	}

	// Check for absolute path indicators
	absolutePatterns := []string{
		"/etc/", "/root/", "/home/", "/usr/", "/var/",     // Unix system paths
		"c:\\", "d:\\", "\\\\",                           // Windows paths
		"\\windows\\", "\\system32\\",                    // Windows system paths
		"\\program files\\", "\\programdata\\",           // Windows program paths
	}

	for _, pattern := range absolutePatterns {
		if strings.Contains(pathLower, pattern) {
			return fmt.Errorf("absolute path pattern detected: %s", pattern)
		}
	}

	// Check for excessive traversal attempts (likely attack)
	if strings.Count(pathLower, "..") > 5 {
		return fmt.Errorf("excessive directory traversal attempts: %d", strings.Count(pathLower, ".."))
	}

	// Check for UNC path patterns (Windows network paths)
	if strings.HasPrefix(pathLower, "\\\\") || strings.Contains(pathLower, "\\\\server\\") {
		return fmt.Errorf("UNC network path not allowed")
	}

	return nil
}

// urlDecodeForSecurityCheck safely attempts to URL decode a string for security validation
// Returns decoded string or error if decoding fails
func urlDecodeForSecurityCheck(path string) (string, error) {
	// Attempt standard URL decoding
	decoded, err := url.QueryUnescape(path)
	if err != nil {
		return "", err
	}
	
	// Check for multiple levels of encoding (double/triple encoding attacks)
	if strings.Contains(decoded, "%") && decoded != path {
		// Try decoding again in case of double encoding
		doubleDecoded, err := url.QueryUnescape(decoded)
		if err == nil && doubleDecoded != decoded {
			return doubleDecoded, nil
		}
	}
	
	return decoded, nil
}

// validatePathBoundaries ensures the path doesn't violate expected boundaries
func validatePathBoundaries(cleanedPath string) error {
	// Additional validation for cleaned paths
	
	// Must not start with path separators (absolute path check)
	if strings.HasPrefix(cleanedPath, "/") || strings.HasPrefix(cleanedPath, "\\") {
		return fmt.Errorf("absolute path not allowed: %s", cleanedPath)
	}

	// Must not be just dots or contain only traversal elements
	if cleanedPath == "." || cleanedPath == ".." || strings.HasPrefix(cleanedPath, "../") {
		return fmt.Errorf("invalid relative path: %s", cleanedPath)
	}

	// Check for Windows drive letter patterns
	if len(cleanedPath) >= 2 && cleanedPath[1] == ':' {
		return fmt.Errorf("windows drive letter not allowed: %s", cleanedPath)
	}

	// Validate reasonable path depth (prevent deeply nested attacks)
	pathParts := strings.Split(cleanedPath, string(filepath.Separator))
	if len(pathParts) > 20 {
		return fmt.Errorf("path depth exceeds maximum allowed: %d levels", len(pathParts))
	}

	// Validate reasonable path length (prevent buffer overflow attempts)
	if len(cleanedPath) > 4096 {
		return fmt.Errorf("path length exceeds maximum allowed: %d characters", len(cleanedPath))
	}

	return nil
}

// validateWindowsSystemPath performs Windows-specific security validation for user input paths
// This function blocks Windows drive letters, UNC paths, and reserved device names
func validateWindowsSystemPath(cleanedPath string) error {
	// Check for Windows drive letter patterns (C:, D:, etc.)
	if len(cleanedPath) >= 2 && cleanedPath[1] == ':' {
		return fmt.Errorf("windows drive letter paths not allowed for security")
	}
	
	// Check for UNC paths (\\server\share)
	if strings.HasPrefix(cleanedPath, `\\`) {
		return fmt.Errorf("windows UNC paths not allowed for security")
	}
	
	return validateWindowsReservedDeviceNames(cleanedPath)
}

// validateWindowsReservedDeviceNames checks for Windows reserved device names only
// This is used for both system and user paths since device names are never allowed
func validateWindowsReservedDeviceNames(cleanedPath string) error {
	// Check for Windows reserved device names
	baseName := filepath.Base(cleanedPath)
	// Remove extension for checking device names
	if dotIndex := strings.LastIndex(baseName, "."); dotIndex > 0 {
		baseName = baseName[:dotIndex]
	}
	
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	
	upperBaseName := strings.ToUpper(baseName)
	for _, reserved := range reservedNames {
		if upperBaseName == reserved {
			return fmt.Errorf("windows reserved device name not allowed: %s", baseName)
		}
	}
	
	return nil
}

