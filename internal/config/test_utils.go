// +build test

package config

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestUtilities provides helper functions for testing configuration system
type TestUtilities struct {
	tempDirs []string
	origEnvs map[string]string
}

// NewTestUtilities creates a new test utilities instance
func NewTestUtilities() *TestUtilities {
	return &TestUtilities{
		tempDirs: make([]string, 0),
		origEnvs: make(map[string]string),
	}
}

// CreateTempDir creates a temporary directory for testing
func (tu *TestUtilities) CreateTempDir(t *testing.T, name string) string {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("timemachine-test-%s", name))
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	tu.tempDirs = append(tu.tempDirs, tempDir)
	return tempDir
}

// SetTestEnvVar sets an environment variable and remembers the original value
func (tu *TestUtilities) SetTestEnvVar(key, value string) {
	if _, exists := tu.origEnvs[key]; !exists {
		tu.origEnvs[key] = os.Getenv(key)
	}
	os.Setenv(key, value)
}

// RestoreEnvVars restores all modified environment variables
func (tu *TestUtilities) RestoreEnvVars() {
	for key, originalValue := range tu.origEnvs {
		if originalValue == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, originalValue)
		}
	}
	tu.origEnvs = make(map[string]string)
}

// Cleanup performs cleanup of all test resources
func (tu *TestUtilities) Cleanup() {
	// Restore environment variables
	tu.RestoreEnvVars()
	
	// Remove temporary directories
	for _, dir := range tu.tempDirs {
		os.RemoveAll(dir)
	}
	tu.tempDirs = make([]string, 0)
}

// CreateConfigFile creates a test configuration file
func (tu *TestUtilities) CreateConfigFile(t *testing.T, dir, content string) string {
	configPath := filepath.Join(dir, "timemachine.yaml")
	err := os.WriteFile(configPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	return configPath
}

// StandardTestConfig returns a standard test configuration
func (tu *TestUtilities) StandardTestConfig() string {
	return `
log:
  level: info
  format: text
  file: "/tmp/test.log"

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  ignore_patterns: ["*.log", "*.tmp"]
  batch_size: 100
  enable_recursive: true

cache:
  max_entries: 10000
  max_memory_mb: 50
  ttl: 1h
  enable_lru: true

git:
  cleanup_threshold: 100
  auto_gc: true
  max_commits: 1000
  use_shallow_clone: false

ui:
  progress_indicators: true
  color_output: true
  pager: auto
  table_format: table
`
}

// InvalidTestConfig returns a configuration with validation errors
func (tu *TestUtilities) InvalidTestConfig() string {
	return `
log:
  level: invalid_level
  format: invalid_format

watcher:
  debounce_delay: 50ms  # Too small
  max_watched_files: 500  # Too small
  batch_size: 2000  # Too large

cache:
  max_entries: 500  # Too small
  max_memory_mb: 5  # Too small
  ttl: 30s  # Too small

git:
  cleanup_threshold: 1000
  max_commits: 500  # Less than cleanup_threshold

ui:
  pager: invalid_pager
  table_format: invalid_format
`
}

// SecurityAttackConfig returns a config with security attack attempts
func (tu *TestUtilities) SecurityAttackConfig() string {
	return `
log:
  level: info
  file: "../../../etc/passwd"  # Path traversal attempt

watcher:
  ignore_patterns: 
    - "*.log"
    - "../../../etc/shadow"  # Another attack

cache:
  max_entries: 10000

git:
  cleanup_threshold: 100
  max_commits: 1000

ui:
  pager: auto
`
}

// LargeTestConfig returns a configuration with many entries
func (tu *TestUtilities) LargeTestConfig(numPatterns int) string {
	var configBuilder strings.Builder
	
	configBuilder.WriteString(`
log:
  level: info
  format: text

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  ignore_patterns:
`)
	
	// Add many ignore patterns
	for i := 0; i < numPatterns; i++ {
		configBuilder.WriteString(fmt.Sprintf("    - \"pattern_%d_*.tmp\"\n", i))
	}
	
	configBuilder.WriteString(`
  batch_size: 100
  enable_recursive: true

cache:
  max_entries: 10000
  max_memory_mb: 50
  ttl: 1h

git:
  cleanup_threshold: 100
  max_commits: 1000

ui:
  pager: auto
  table_format: table
`)
	
	return configBuilder.String()
}

// GenerateRandomConfig creates a random configuration for testing
func (tu *TestUtilities) GenerateRandomConfig(seed int64) *Config {
	rand.Seed(seed)
	
	logLevels := []string{"debug", "info", "warn", "error"}
	logFormats := []string{"text", "json"}
	pagerSettings := []string{"auto", "always", "never"}
	tableFormats := []string{"table", "json", "yaml"}
	
	return &Config{
		Log: LogConfig{
			Level:  logLevels[rand.Intn(len(logLevels))],
			Format: logFormats[rand.Intn(len(logFormats))],
			File:   tu.generateRandomPath(),
		},
		Watcher: WatcherConfig{
			DebounceDelay:   time.Duration(rand.Intn(9000)+1000) * time.Millisecond, // 1-10s
			MaxWatchedFiles: rand.Intn(900000) + 100000,                            // 100k-1M
			BatchSize:       rand.Intn(900) + 100,                                  // 100-1000
			EnableRecursive: rand.Float32() < 0.5,
		},
		Cache: CacheConfig{
			MaxEntries:  rand.Intn(90000) + 10000,     // 10k-100k
			MaxMemoryMB: rand.Intn(1000) + 24,         // 24-1024MB
			TTL:         time.Duration(rand.Intn(23)+1) * time.Hour, // 1-24h
			EnableLRU:   rand.Float32() < 0.5,
		},
		Git: GitConfig{
			CleanupThreshold: rand.Intn(9900) + 100,   // 100-10000
			MaxCommits:       rand.Intn(49000) + 1000, // 1000-50000 (ensure > cleanup)
			AutoGC:           rand.Float32() < 0.5,
			UseShallowClone:  rand.Float32() < 0.5,
		},
		UI: UIConfig{
			ProgressIndicators: rand.Float32() < 0.5,
			ColorOutput:        rand.Float32() < 0.5,
			Pager:             pagerSettings[rand.Intn(len(pagerSettings))],
			TableFormat:       tableFormats[rand.Intn(len(tableFormats))],
		},
	}
}

// generateRandomPath creates a valid random file path
func (tu *TestUtilities) generateRandomPath() string {
	if rand.Float32() < 0.3 { // 30% empty
		return ""
	}
	
	safePaths := []string{
		"/tmp/test.log",
		"/var/log/app.log",
		"/var/tmp/temp.log",
		"relative/path.log",
		"./local.log",
	}
	
	return safePaths[rand.Intn(len(safePaths))]
}

// GenerateAttackPath creates a path traversal attack vector
func (tu *TestUtilities) GenerateAttackPath() string {
	attacks := []string{
		"../../../etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"..\\..\\..\\windows\\system32",
		"\uff0e\uff0e\uff0f\uff0e\uff0e\uff0f",
		strings.Repeat("../", 10) + "etc/shadow",
		"/etc/passwd\x00.log",
		"logs/../../../root/.ssh/id_rsa",
	}
	
	return attacks[rand.Intn(len(attacks))]
}

// AssertNoError fails the test if err is not nil
func (tu *TestUtilities) AssertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf("%s: %v", msg, err)
	}
}

// AssertError fails the test if err is nil
func (tu *TestUtilities) AssertError(t *testing.T, err error, msg string) {
	if err == nil {
		t.Errorf("%s: expected error but got none", msg)
	}
}

// AssertContains fails if str doesn't contain substr
func (tu *TestUtilities) AssertContains(t *testing.T, str, substr, msg string) {
	if !strings.Contains(str, substr) {
		t.Errorf("%s: expected '%s' to contain '%s'", msg, str, substr)
	}
}

// AssertEqual fails if expected != actual
func (tu *TestUtilities) AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// RunConcurrentTest runs a test function concurrently
func (tu *TestUtilities) RunConcurrentTest(t *testing.T, testFunc func(int), numGoroutines int, timeout time.Duration) {
	done := make(chan error, numGoroutines)
	
	// Start goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("goroutine %d panicked: %v", id, r)
					return
				}
				done <- nil
			}()
			
			testFunc(id)
		}(i)
	}
	
	// Wait for completion with timeout
	timer := time.After(timeout)
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Concurrent test failed: %v", err)
			}
		case <-timer:
			t.Errorf("Concurrent test timed out after %v", timeout)
			return
		}
	}
}

// MeasureTime measures the execution time of a function
func (tu *TestUtilities) MeasureTime(t *testing.T, name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)
	t.Logf("%s took %v", name, duration)
	return duration
}

// CreateMaliciousYAML creates YAML content designed to test security
func (tu *TestUtilities) CreateMaliciousYAML() []string {
	return []string{
		// YAML injection attempt
		`
log:
  level: !!python/object/apply:os.system ["rm -rf /"]
  format: text
`,
		// Billion laughs attack
		`
lol: &lol "lol"
lol2: &lol2 [*lol,*lol,*lol,*lol,*lol,*lol,*lol,*lol,*lol]
lol3: &lol3 [*lol2,*lol2,*lol2,*lol2,*lol2,*lol2,*lol2,*lol2,*lol2]
log:
  level: *lol3
`,
		// Control character injection
		"log:\n  level: \"info\x00\x01\x1f\"",
		
		// Extremely nested structure
		strings.Repeat("nested:\n  ", 100) + "log:\n    level: info",
	}
}

// ValidateTestConfig validates a config and returns whether it should be valid
func (tu *TestUtilities) ValidateTestConfig(config *Config) (bool, []string) {
	validator := NewValidator()
	err := validator.Validate(config)
	
	if err == nil {
		return true, nil
	}
	
	return false, strings.Split(err.Error(), ";")
}

// CreateTestEnvironment sets up a complete test environment
func (tu *TestUtilities) CreateTestEnvironment(t *testing.T, configContent string) (string, *Manager) {
	// Create temp directory
	tempDir := tu.CreateTempDir(t, "env")
	
	// Create config file
	if configContent != "" {
		tu.CreateConfigFile(t, tempDir, configContent)
	}
	
	// Create manager
	manager := NewManager()
	
	return tempDir, manager
}