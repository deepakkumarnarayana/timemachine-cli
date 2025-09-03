package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Validator provides configuration validation
type Validator struct {
	// Future: could use go-validator library for complex validation
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the entire configuration
func (v *Validator) Validate(config *Config) error {
	var errors []string
	
	// Validate log configuration
	if err := v.validateLogConfig(&config.Log); err != nil {
		errors = append(errors, fmt.Sprintf("log config: %v", err))
	}
	
	// Validate watcher configuration
	if err := v.validateWatcherConfig(&config.Watcher); err != nil {
		errors = append(errors, fmt.Sprintf("watcher config: %v", err))
	}
	
	// Validate cache configuration
	if err := v.validateCacheConfig(&config.Cache); err != nil {
		errors = append(errors, fmt.Sprintf("cache config: %v", err))
	}
	
	// Validate git configuration
	if err := v.validateGitConfig(&config.Git); err != nil {
		errors = append(errors, fmt.Sprintf("git config: %v", err))
	}
	
	// Validate UI configuration
	if err := v.validateUIConfig(&config.UI); err != nil {
		errors = append(errors, fmt.Sprintf("ui config: %v", err))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// validateLogConfig validates logging configuration
func (v *Validator) validateLogConfig(config *LogConfig) error {
	var errors []string
	
	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !v.stringInSlice(config.Level, validLevels) {
		errors = append(errors, fmt.Sprintf("invalid log level '%s', must be one of: %s", 
			config.Level, strings.Join(validLevels, ", ")))
	}
	
	// Validate log format
	validFormats := []string{"text", "json"}
	if !v.stringInSlice(config.Format, validFormats) {
		errors = append(errors, fmt.Sprintf("invalid log format '%s', must be one of: %s", 
			config.Format, strings.Join(validFormats, ", ")))
	}
	
	// Validate log file path if specified
	if config.File != "" {
		if !v.isValidFilePath(config.File) {
			errors = append(errors, fmt.Sprintf("invalid log file path '%s'", config.File))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	
	return nil
}

// validateWatcherConfig validates file watcher configuration
func (v *Validator) validateWatcherConfig(config *WatcherConfig) error {
	var errors []string
	
	// Validate debounce delay
	if config.DebounceDelay < 100*time.Millisecond {
		errors = append(errors, "debounce_delay must be at least 100ms")
	}
	if config.DebounceDelay > 10*time.Second {
		errors = append(errors, "debounce_delay must be at most 10s")
	}
	
	// Validate max watched files
	if config.MaxWatchedFiles < 1000 {
		errors = append(errors, "max_watched_files must be at least 1000")
	}
	if config.MaxWatchedFiles > 1000000 {
		errors = append(errors, "max_watched_files must be at most 1000000")
	}
	
	// Validate batch size
	if config.BatchSize < 1 {
		errors = append(errors, "batch_size must be at least 1")
	}
	if config.BatchSize > 1000 {
		errors = append(errors, "batch_size must be at most 1000")
	}
	
	// Validate ignore patterns (basic syntax check)
	for i, pattern := range config.IgnorePatterns {
		if strings.Contains(pattern, "..") {
			errors = append(errors, fmt.Sprintf("ignore pattern %d contains invalid '..' sequence", i))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	
	return nil
}

// validateCacheConfig validates cache configuration
func (v *Validator) validateCacheConfig(config *CacheConfig) error {
	var errors []string
	
	// Validate max entries
	if config.MaxEntries < 1000 {
		errors = append(errors, "max_entries must be at least 1000")
	}
	if config.MaxEntries > 100000 {
		errors = append(errors, "max_entries must be at most 100000")
	}
	
	// Validate max memory
	if config.MaxMemoryMB < 10 {
		errors = append(errors, "max_memory_mb must be at least 10")
	}
	if config.MaxMemoryMB > 1024 {
		errors = append(errors, "max_memory_mb must be at most 1024")
	}
	
	// Validate TTL
	if config.TTL < 1*time.Minute {
		errors = append(errors, "ttl must be at least 1m")
	}
	if config.TTL > 24*time.Hour {
		errors = append(errors, "ttl must be at most 24h")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	
	return nil
}

// validateGitConfig validates git configuration
func (v *Validator) validateGitConfig(config *GitConfig) error {
	var errors []string
	
	// Validate cleanup threshold
	if config.CleanupThreshold < 10 {
		errors = append(errors, "cleanup_threshold must be at least 10")
	}
	if config.CleanupThreshold > 10000 {
		errors = append(errors, "cleanup_threshold must be at most 10000")
	}
	
	// Validate max commits
	if config.MaxCommits < 50 {
		errors = append(errors, "max_commits must be at least 50")
	}
	if config.MaxCommits > 50000 {
		errors = append(errors, "max_commits must be at most 50000")
	}
	
	// Ensure cleanup threshold is less than max commits
	if config.CleanupThreshold >= config.MaxCommits {
		errors = append(errors, "cleanup_threshold must be less than max_commits")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	
	return nil
}

// validateUIConfig validates UI configuration
func (v *Validator) validateUIConfig(config *UIConfig) error {
	var errors []string
	
	// Validate pager setting
	validPagerSettings := []string{"auto", "always", "never"}
	if !v.stringInSlice(config.Pager, validPagerSettings) {
		errors = append(errors, fmt.Sprintf("invalid pager setting '%s', must be one of: %s", 
			config.Pager, strings.Join(validPagerSettings, ", ")))
	}
	
	// Validate table format
	validTableFormats := []string{"table", "json", "yaml"}
	if !v.stringInSlice(config.TableFormat, validTableFormats) {
		errors = append(errors, fmt.Sprintf("invalid table_format '%s', must be one of: %s", 
			config.TableFormat, strings.Join(validTableFormats, ", ")))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	
	return nil
}

// Helper methods

// stringInSlice checks if a string is in a slice
func (v *Validator) stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// isValidFilePath performs comprehensive file path validation to prevent security vulnerabilities
func (v *Validator) isValidFilePath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	
	originalPath := path
	
	// Normalize and clean the path
	cleanPath := filepath.Clean(path)
	
	// SECURITY: Multiple layers of path traversal detection
	
	// 1. Check for basic path traversal patterns in original path
	if strings.Contains(path, "..") {
		return false
	}
	
	// 2. Check for URL encoded path traversal (single encoding)
	decodedPath, err := url.QueryUnescape(path)
	if err == nil && strings.Contains(decodedPath, "..") {
		return false
	}
	
	// 3. Check for double URL encoding
	doubleDecoded, err := url.QueryUnescape(decodedPath)
	if err == nil && strings.Contains(doubleDecoded, "..") {
		return false
	}
	
	// 4. Check for Unicode fullwidth characters (．／)
	unicodeNormalized := strings.ReplaceAll(path, "\uff0e", ".")
	unicodeNormalized = strings.ReplaceAll(unicodeNormalized, "\uff0f", "/")
	if strings.Contains(unicodeNormalized, "..") {
		return false
	}
	
	// 5. Check for Unicode dots in various forms
	unicodeDots := []string{
		"\u002e\u002e", // Standard Unicode dots
		"\u2024\u2024", // One dot leader
		"\uff0e\uff0e", // Fullwidth period
	}
	for _, dots := range unicodeDots {
		if strings.Contains(path, dots) {
			return false
		}
	}
	
	// 6. Check for Windows path traversal attempts
	if strings.Contains(strings.ToLower(path), "..\\") {
		return false
	}
	
	// 7. Check for overlong UTF-8 sequences (common attack)
	// Convert bytes that could represent ".." in overlong form
	pathBytes := []byte(path)
	for i := 0; i < len(pathBytes)-1; i++ {
		// Check for overlong encoding patterns
		if pathBytes[i] == 0xc0 && pathBytes[i+1] == 0xae {
			return false // Overlong encoding for '.'
		}
	}
	
	// 8. Check for path traversal in cleaned path (additional safety)
	if strings.Contains(cleanPath, "..") {
		return false
	}
	
	// 9. Check for suspicious character sequences that could bypass filters
	suspiciousPatterns := []string{
		".../",   // Triple dots
		"....//", // Quad dots with double slash
		".\\./",  // Mixed separators
		".//../", // Hidden traversal
	}
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}
	
	// 10. Check for Unicode URL encoding patterns (%uXXXX)
	if strings.Contains(path, "%u00") {
		return false
	}
	
	// 11. Check for malformed URL encoding patterns that could bypass filters
	malformedPatterns := []string{
		"%2", "%G", "%X", // Incomplete or invalid hex
		"%u002e%u002e%u002f", // Unicode URL encoding for ../
	}
	for _, pattern := range malformedPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}
	
	// 12. Validate that non-encoded paths that look like base64 don't decode to traversal
	// This catches attempts to bypass by using base64-like strings
	if !strings.Contains(originalPath, "/") && !strings.Contains(originalPath, "\\") {
		// Could be an encoded attempt - reject if it's suspiciously short and uniform
		if len(originalPath) > 6 && len(originalPath) < 50 {
			// Check if it's entirely alphanumeric (potential base64)
			isAlphaNumeric := true
			for _, r := range originalPath {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
					isAlphaNumeric = false
					break
				}
			}
			if isAlphaNumeric {
				return false
			}
		}
	}
	
	// If it's an absolute path, validate it's in a safe location
	if filepath.IsAbs(cleanPath) {
		// Allow absolute paths only to safe directories
		safeDirs := []string{
			"/tmp/",
			"/var/tmp/",
			"/var/log/",
			"/home/",
			"/Users/", // macOS
		}
		
		// Check if the absolute path starts with a safe directory
		pathLower := strings.ToLower(cleanPath)
		isSafe := false
		for _, safeDir := range safeDirs {
			if strings.HasPrefix(pathLower, strings.ToLower(safeDir)) {
				isSafe = true
				break
			}
		}
		
		// Reject unsafe absolute paths
		if !isSafe {
			// Additional check: allow if it's within user's home or current working directory tree
			if homeDir, err := os.UserHomeDir(); err == nil {
				if strings.HasPrefix(cleanPath, homeDir) {
					isSafe = true
				}
			}
			
			if currentDir, err := filepath.Abs("."); err == nil {
				if strings.HasPrefix(cleanPath, currentDir) {
					isSafe = true
				}
			}
			
			if !isSafe {
				return false
			}
		}
	}
	
	return true
}

// ValidateUpdate validates a configuration update
func (v *Validator) ValidateUpdate(field string, value interface{}) error {
	switch field {
	case "log.level":
		if str, ok := value.(string); ok {
			validLevels := []string{"debug", "info", "warn", "error"}
			if !v.stringInSlice(str, validLevels) {
				return fmt.Errorf("invalid log level '%s', must be one of: %s", 
					str, strings.Join(validLevels, ", "))
			}
		} else {
			return fmt.Errorf("log.level must be a string")
		}
	
	case "watcher.debounce_delay":
		if duration, ok := value.(time.Duration); ok {
			if duration < 100*time.Millisecond || duration > 10*time.Second {
				return fmt.Errorf("debounce_delay must be between 100ms and 10s")
			}
		} else {
			return fmt.Errorf("watcher.debounce_delay must be a duration")
		}
		
	case "cache.max_entries":
		if num, ok := value.(int); ok {
			if num < 1000 || num > 100000 {
				return fmt.Errorf("cache.max_entries must be between 1000 and 100000")
			}
		} else {
			return fmt.Errorf("cache.max_entries must be an integer")
		}
		
	// Add more field-specific validations as needed
	default:
		// For unknown fields, do basic type validation
		return v.validateType(field, value)
	}
	
	return nil
}

// validateType performs basic type validation for unknown fields
func (v *Validator) validateType(field string, value interface{}) error {
	// Allow basic types
	switch value.(type) {
	case string, int, bool, time.Duration, []string:
		return nil
	default:
		return fmt.Errorf("unsupported type %T for field %s", value, field)
	}
}

// GetValidationHelp returns help text for configuration validation
func (v *Validator) GetValidationHelp() string {
	return `Configuration Validation Rules:

Log Configuration:
  - level: must be one of 'debug', 'info', 'warn', 'error'
  - format: must be 'text' or 'json'
  - file: optional file path (no path traversal allowed)

Watcher Configuration:
  - debounce_delay: between 100ms and 10s
  - max_watched_files: between 1,000 and 1,000,000
  - batch_size: between 1 and 1,000
  - ignore_patterns: no '..' sequences allowed

Cache Configuration:
  - max_entries: between 1,000 and 100,000
  - max_memory_mb: between 10 and 1,024 MB
  - ttl: between 1m and 24h

Git Configuration:
  - cleanup_threshold: between 10 and 10,000 (must be < max_commits)
  - max_commits: between 50 and 50,000

UI Configuration:
  - pager: must be 'auto', 'always', or 'never'
  - table_format: must be 'table', 'json', or 'yaml'
`
}