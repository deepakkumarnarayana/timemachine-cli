package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateLogConfig(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name        string
		config      LogConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: LogConfig{
				Level:  "info",
				Format: "text",
				File:   "/tmp/log.txt",
			},
			expectError: false,
		},
		{
			name: "invalid log level",
			config: LogConfig{
				Level:  "invalid",
				Format: "text",
				File:   "",
			},
			expectError: true,
		},
		{
			name: "invalid log format",
			config: LogConfig{
				Level:  "info",
				Format: "invalid",
				File:   "",
			},
			expectError: true,
		},
		{
			name: "path traversal in file",
			config: LogConfig{
				Level:  "info",
				Format: "text",
				File:   "../../../etc/passwd",
			},
			expectError: true,
		},
		{
			name: "empty file path (valid)",
			config: LogConfig{
				Level:  "info",
				Format: "text",
				File:   "",
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateLogConfig(&tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestValidateWatcherConfig(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name        string
		config      WatcherConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: WatcherConfig{
				DebounceDelay:   2 * time.Second,
				MaxWatchedFiles: 100000,
				BatchSize:       100,
				IgnorePatterns:  []string{"*.log", "build/"},
				EnableRecursive: true,
			},
			expectError: false,
		},
		{
			name: "debounce delay too small",
			config: WatcherConfig{
				DebounceDelay:   50 * time.Millisecond,
				MaxWatchedFiles: 100000,
				BatchSize:       100,
			},
			expectError: true,
		},
		{
			name: "debounce delay too large",
			config: WatcherConfig{
				DebounceDelay:   15 * time.Second,
				MaxWatchedFiles: 100000,
				BatchSize:       100,
			},
			expectError: true,
		},
		{
			name: "max watched files too small",
			config: WatcherConfig{
				DebounceDelay:   2 * time.Second,
				MaxWatchedFiles: 500,
				BatchSize:       100,
			},
			expectError: true,
		},
		{
			name: "max watched files too large",
			config: WatcherConfig{
				DebounceDelay:   2 * time.Second,
				MaxWatchedFiles: 2000000,
				BatchSize:       100,
			},
			expectError: true,
		},
		{
			name: "batch size too small",
			config: WatcherConfig{
				DebounceDelay:   2 * time.Second,
				MaxWatchedFiles: 100000,
				BatchSize:       0,
			},
			expectError: true,
		},
		{
			name: "batch size too large",
			config: WatcherConfig{
				DebounceDelay:   2 * time.Second,
				MaxWatchedFiles: 100000,
				BatchSize:       1500,
			},
			expectError: true,
		},
		{
			name: "invalid ignore pattern with path traversal",
			config: WatcherConfig{
				DebounceDelay:   2 * time.Second,
				MaxWatchedFiles: 100000,
				BatchSize:       100,
				IgnorePatterns:  []string{"*.log", "../../../etc/passwd"},
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateWatcherConfig(&tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestValidateCacheConfig(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name        string
		config      CacheConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: CacheConfig{
				MaxEntries:  10000,
				MaxMemoryMB: 50,
				TTL:         1 * time.Hour,
				EnableLRU:   true,
			},
			expectError: false,
		},
		{
			name: "max entries too small",
			config: CacheConfig{
				MaxEntries:  500,
				MaxMemoryMB: 50,
				TTL:         1 * time.Hour,
			},
			expectError: true,
		},
		{
			name: "max entries too large",
			config: CacheConfig{
				MaxEntries:  150000,
				MaxMemoryMB: 50,
				TTL:         1 * time.Hour,
			},
			expectError: true,
		},
		{
			name: "max memory too small",
			config: CacheConfig{
				MaxEntries:  10000,
				MaxMemoryMB: 5,
				TTL:         1 * time.Hour,
			},
			expectError: true,
		},
		{
			name: "max memory too large",
			config: CacheConfig{
				MaxEntries:  10000,
				MaxMemoryMB: 2048,
				TTL:         1 * time.Hour,
			},
			expectError: true,
		},
		{
			name: "TTL too small",
			config: CacheConfig{
				MaxEntries:  10000,
				MaxMemoryMB: 50,
				TTL:         30 * time.Second,
			},
			expectError: true,
		},
		{
			name: "TTL too large",
			config: CacheConfig{
				MaxEntries:  10000,
				MaxMemoryMB: 50,
				TTL:         48 * time.Hour,
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateCacheConfig(&tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestValidateGitConfig(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name        string
		config      GitConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: GitConfig{
				CleanupThreshold: 100,
				MaxCommits:       1000,
				AutoGC:           true,
				UseShallowClone:  false,
			},
			expectError: false,
		},
		{
			name: "cleanup threshold too small",
			config: GitConfig{
				CleanupThreshold: 5,
				MaxCommits:       1000,
				AutoGC:           true,
			},
			expectError: true,
		},
		{
			name: "cleanup threshold too large",
			config: GitConfig{
				CleanupThreshold: 15000,
				MaxCommits:       50000,
				AutoGC:           true,
			},
			expectError: true,
		},
		{
			name: "max commits too small",
			config: GitConfig{
				CleanupThreshold: 100,
				MaxCommits:       25,
				AutoGC:           true,
			},
			expectError: true,
		},
		{
			name: "max commits too large",
			config: GitConfig{
				CleanupThreshold: 100,
				MaxCommits:       75000,
				AutoGC:           true,
			},
			expectError: true,
		},
		{
			name: "cleanup threshold >= max commits",
			config: GitConfig{
				CleanupThreshold: 1000,
				MaxCommits:       1000,
				AutoGC:           true,
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateGitConfig(&tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestValidateUIConfig(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name        string
		config      UIConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: UIConfig{
				ProgressIndicators: true,
				ColorOutput:        true,
				Pager:              "auto",
				TableFormat:        "table",
			},
			expectError: false,
		},
		{
			name: "invalid pager setting",
			config: UIConfig{
				ProgressIndicators: true,
				ColorOutput:        true,
				Pager:              "invalid",
				TableFormat:        "table",
			},
			expectError: true,
		},
		{
			name: "invalid table format",
			config: UIConfig{
				ProgressIndicators: true,
				ColorOutput:        true,
				Pager:              "auto",
				TableFormat:        "invalid",
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateUIConfig(&tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestValidate_FullConfig(t *testing.T) {
	validator := NewValidator()
	
	// Valid configuration
	validConfig := &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			File:   "",
		},
		Watcher: WatcherConfig{
			DebounceDelay:   2 * time.Second,
			MaxWatchedFiles: 100000,
			BatchSize:       100,
			EnableRecursive: true,
		},
		Cache: CacheConfig{
			MaxEntries:  10000,
			MaxMemoryMB: 50,
			TTL:         1 * time.Hour,
			EnableLRU:   true,
		},
		Git: GitConfig{
			CleanupThreshold: 100,
			MaxCommits:       1000,
			AutoGC:           true,
			UseShallowClone:  false,
		},
		UI: UIConfig{
			ProgressIndicators: true,
			ColorOutput:        true,
			Pager:              "auto",
			TableFormat:        "table",
		},
	}
	
	err := validator.Validate(validConfig)
	if err != nil {
		t.Errorf("Valid configuration failed validation: %v", err)
	}
	
	// Invalid configuration (multiple errors)
	invalidConfig := &Config{
		Log: LogConfig{
			Level:  "invalid",
			Format: "invalid",
			File:   "",
		},
		Watcher: WatcherConfig{
			DebounceDelay:   50 * time.Millisecond, // Too small
			MaxWatchedFiles: 500,                   // Too small
			BatchSize:       0,                     // Too small
			EnableRecursive: true,
		},
		Cache: CacheConfig{
			MaxEntries:  500, // Too small
			MaxMemoryMB: 5,   // Too small
			TTL:         30 * time.Second, // Too small
			EnableLRU:   true,
		},
		Git: GitConfig{
			CleanupThreshold: 5,  // Too small
			MaxCommits:       25, // Too small
			AutoGC:           true,
		},
		UI: UIConfig{
			ProgressIndicators: true,
			ColorOutput:        true,
			Pager:              "invalid", // Invalid
			TableFormat:        "invalid", // Invalid
		},
	}
	
	err = validator.Validate(invalidConfig)
	if err == nil {
		t.Error("Invalid configuration should have failed validation")
	}
	
	// Check that error message contains multiple validation errors
	errorMessage := err.Error()
	expectedErrors := []string{"log config", "watcher config", "cache config", "git config", "ui config"}
	for _, expectedError := range expectedErrors {
		if !strings.Contains(errorMessage, expectedError) {
			t.Errorf("Error message should contain '%s', got: %s", expectedError, errorMessage)
		}
	}
}

func TestValidateUpdate(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name        string
		field       string
		value       interface{}
		expectError bool
	}{
		{
			name:        "valid log level",
			field:       "log.level",
			value:       "debug",
			expectError: false,
		},
		{
			name:        "invalid log level",
			field:       "log.level",
			value:       "invalid",
			expectError: true,
		},
		{
			name:        "log level wrong type",
			field:       "log.level",
			value:       123,
			expectError: true,
		},
		{
			name:        "valid debounce delay",
			field:       "watcher.debounce_delay",
			value:       2 * time.Second,
			expectError: false,
		},
		{
			name:        "debounce delay too small",
			field:       "watcher.debounce_delay",
			value:       50 * time.Millisecond,
			expectError: true,
		},
		{
			name:        "debounce delay wrong type",
			field:       "watcher.debounce_delay",
			value:       "invalid",
			expectError: true,
		},
		{
			name:        "valid cache max entries",
			field:       "cache.max_entries",
			value:       5000,
			expectError: false,
		},
		{
			name:        "cache max entries too small",
			field:       "cache.max_entries",
			value:       500,
			expectError: true,
		},
		{
			name:        "cache max entries wrong type",
			field:       "cache.max_entries",
			value:       "invalid",
			expectError: true,
		},
		{
			name:        "unknown field with valid type",
			field:       "unknown.field",
			value:       "test",
			expectError: false,
		},
		{
			name:        "unknown field with invalid type",
			field:       "unknown.field",
			value:       map[string]interface{}{"invalid": "type"},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUpdate(tt.field, tt.value)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestGetValidationHelp(t *testing.T) {
	validator := NewValidator()
	help := validator.GetValidationHelp()
	
	if help == "" {
		t.Error("GetValidationHelp() returned empty string")
	}
	
	// Check that help contains expected sections
	expectedSections := []string{
		"Log Configuration:",
		"Watcher Configuration:",
		"Cache Configuration:",
		"Git Configuration:",
		"UI Configuration:",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(help, section) {
			t.Errorf("Help text should contain '%s'", section)
		}
	}
}