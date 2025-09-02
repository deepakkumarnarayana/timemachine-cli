package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	
	if manager.viper == nil {
		t.Error("Manager viper instance is nil")
	}
	
	if manager.validator == nil {
		t.Error("Manager validator instance is nil")
	}
	
	if manager.config == nil {
		t.Error("Manager config instance is nil")
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	manager := NewManager()
	
	// Load configuration without any config file (should use defaults)
	err = manager.Load(tempDir)
	if err != nil {
		t.Errorf("Load() failed: %v", err)
	}
	
	config := manager.Get()
	
	// Test default values
	if config.Log.Level != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", config.Log.Level)
	}
	
	if config.Log.Format != "text" {
		t.Errorf("Expected default log format 'text', got '%s'", config.Log.Format)
	}
	
	if config.Watcher.DebounceDelay != 2*time.Second {
		t.Errorf("Expected default debounce delay '2s', got '%v'", config.Watcher.DebounceDelay)
	}
	
	if config.Watcher.MaxWatchedFiles != 100000 {
		t.Errorf("Expected default max watched files '100000', got '%d'", config.Watcher.MaxWatchedFiles)
	}
	
	if config.Cache.MaxEntries != 10000 {
		t.Errorf("Expected default cache max entries '10000', got '%d'", config.Cache.MaxEntries)
	}
	
	if config.Git.CleanupThreshold != 100 {
		t.Errorf("Expected default cleanup threshold '100', got '%d'", config.Git.CleanupThreshold)
	}
	
	if !config.UI.ProgressIndicators {
		t.Error("Expected default progress indicators to be true")
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test config file
	configContent := `
log:
  level: debug
  format: json
  file: "/tmp/test.log"

watcher:
  debounce_delay: 5s
  max_watched_files: 50000
  batch_size: 200
  enable_recursive: false

cache:
  max_entries: 5000
  max_memory_mb: 100
  ttl: 30m
  enable_lru: false

git:
  cleanup_threshold: 50
  auto_gc: false
  max_commits: 500
  use_shallow_clone: true

ui:
  progress_indicators: false
  color_output: false
  pager: never
  table_format: json
`
	
	configPath := filepath.Join(tempDir, "timemachine.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	manager := NewManager()
	
	// Load configuration
	err = manager.Load(tempDir)
	if err != nil {
		t.Errorf("Load() failed: %v", err)
	}
	
	config := manager.Get()
	
	// Test loaded values
	if config.Log.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.Log.Level)
	}
	
	if config.Log.Format != "json" {
		t.Errorf("Expected log format 'json', got '%s'", config.Log.Format)
	}
	
	if config.Log.File != "/tmp/test.log" {
		t.Errorf("Expected log file '/tmp/test.log', got '%s'", config.Log.File)
	}
	
	if config.Watcher.DebounceDelay != 5*time.Second {
		t.Errorf("Expected debounce delay '5s', got '%v'", config.Watcher.DebounceDelay)
	}
	
	if config.Watcher.MaxWatchedFiles != 50000 {
		t.Errorf("Expected max watched files '50000', got '%d'", config.Watcher.MaxWatchedFiles)
	}
	
	if config.Watcher.EnableRecursive != false {
		t.Error("Expected enable recursive to be false")
	}
	
	if config.Cache.MaxEntries != 5000 {
		t.Errorf("Expected cache max entries '5000', got '%d'", config.Cache.MaxEntries)
	}
	
	if config.Cache.TTL != 30*time.Minute {
		t.Errorf("Expected cache TTL '30m', got '%v'", config.Cache.TTL)
	}
	
	if config.Git.CleanupThreshold != 50 {
		t.Errorf("Expected cleanup threshold '50', got '%d'", config.Git.CleanupThreshold)
	}
	
	if config.Git.AutoGC != false {
		t.Error("Expected auto GC to be false")
	}
	
	if config.UI.ProgressIndicators != false {
		t.Error("Expected progress indicators to be false")
	}
	
	if config.UI.Pager != "never" {
		t.Errorf("Expected pager 'never', got '%s'", config.UI.Pager)
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Set environment variables
	originalEnvs := make(map[string]string)
	envVars := map[string]string{
		"TIMEMACHINE_LOG_LEVEL":             "warn",
		"TIMEMACHINE_LOG_FORMAT":            "json",
		"TIMEMACHINE_WATCHER_DEBOUNCE":      "3s",
		"TIMEMACHINE_WATCHER_MAX_FILES":     "75000",
		"TIMEMACHINE_CACHE_MAX_ENTRIES":     "8000",
		"TIMEMACHINE_GIT_CLEANUP_THRESHOLD": "75",
		"TIMEMACHINE_UI_COLOR":              "false",
	}
	
	// Save original env values and set test values
	for key, value := range envVars {
		originalEnvs[key] = os.Getenv(key)
		os.Setenv(key, value)
	}
	
	// Restore original env values after test
	defer func() {
		for key, originalValue := range originalEnvs {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}()
	
	manager := NewManager()
	
	// Load configuration
	err = manager.Load(tempDir)
	if err != nil {
		t.Errorf("Load() failed: %v", err)
	}
	
	config := manager.Get()
	
	// Test environment variable overrides
	if config.Log.Level != "warn" {
		t.Errorf("Expected log level 'warn' from env, got '%s'", config.Log.Level)
	}
	
	if config.Log.Format != "json" {
		t.Errorf("Expected log format 'json' from env, got '%s'", config.Log.Format)
	}
	
	if config.Watcher.DebounceDelay != 3*time.Second {
		t.Errorf("Expected debounce delay '3s' from env, got '%v'", config.Watcher.DebounceDelay)
	}
	
	if config.Watcher.MaxWatchedFiles != 75000 {
		t.Errorf("Expected max watched files '75000' from env, got '%d'", config.Watcher.MaxWatchedFiles)
	}
	
	if config.Cache.MaxEntries != 8000 {
		t.Errorf("Expected cache max entries '8000' from env, got '%d'", config.Cache.MaxEntries)
	}
	
	if config.Git.CleanupThreshold != 75 {
		t.Errorf("Expected cleanup threshold '75' from env, got '%d'", config.Git.CleanupThreshold)
	}
	
	if config.UI.ColorOutput != false {
		t.Error("Expected color output 'false' from env")
	}
}

func TestCreateDefaultConfigFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	manager := NewManager()
	
	// Create default config file
	err = manager.CreateDefaultConfigFile(tempDir)
	if err != nil {
		t.Errorf("CreateDefaultConfigFile() failed: %v", err)
	}
	
	// Check if file was created
	configPath := filepath.Join(tempDir, "timemachine.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Default config file was not created")
	}
	
	// Try to create again (should fail)
	err = manager.CreateDefaultConfigFile(tempDir)
	if err == nil {
		t.Error("CreateDefaultConfigFile() should fail when file already exists")
	}
	
	// Read and verify content contains expected sections
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	
	contentStr := string(content)
	expectedSections := []string{"log:", "watcher:", "cache:", "git:", "ui:"}
	for _, section := range expectedSections {
		if !contains(contentStr, section) {
			t.Errorf("Config file missing section: %s", section)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   len(s) > len(substr) && s[:len(substr)] == substr ||
		   (len(s) > len(substr) && func() bool {
		   	for i := 0; i <= len(s)-len(substr); i++ {
		   		if s[i:i+len(substr)] == substr {
		   			return true
		   		}
		   	}
		   	return false
		   }())
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create invalid config file
	invalidConfigContent := `
log:
  level: invalid_level
  format: invalid_format

watcher:
  debounce_delay: invalid_duration
  max_watched_files: -1

cache:
  max_entries: 0
  ttl: invalid_duration
`
	
	configPath := filepath.Join(tempDir, "timemachine.yaml")
	err = os.WriteFile(configPath, []byte(invalidConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	manager := NewManager()
	
	// Load configuration (should fail validation)
	err = manager.Load(tempDir)
	if err == nil {
		t.Error("Load() should fail with invalid configuration")
	}
}

func TestGetViper(t *testing.T) {
	manager := NewManager()
	viper := manager.GetViper()
	
	if viper == nil {
		t.Error("GetViper() returned nil")
	}
	
	// Should be the same instance
	if viper != manager.viper {
		t.Error("GetViper() returned different instance")
	}
}