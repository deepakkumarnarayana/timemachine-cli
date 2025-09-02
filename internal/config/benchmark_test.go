package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// BenchmarkConfigLoading benchmarks configuration loading performance
func BenchmarkConfigLoading(b *testing.B) {
	// Setup: Create temporary config files of different sizes
	tempDir := b.TempDir()
	projectRoot := filepath.Join(tempDir, "project")
	
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		b.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a standard config file
	standardConfig := `
log:
  level: info
  format: text
  file: "/tmp/benchmark.log"

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  ignore_patterns: ["*.log", "*.tmp", "node_modules/", ".git/"]
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

	configPath := filepath.Join(projectRoot, "timemachine.yaml")
	err := os.WriteFile(configPath, []byte(standardConfig), 0600)
	if err != nil {
		b.Fatalf("Failed to write config: %v", err)
	}

	b.ResetTimer()

	b.Run("StandardConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager := NewManager()
			err := manager.Load(projectRoot)
			if err != nil {
				b.Errorf("Config loading failed: %v", err)
			}
		}
	})
}

// BenchmarkConfigLoadingWithLargeFile tests performance with larger config files
func BenchmarkConfigLoadingWithLargeFile(b *testing.B) {
	tempDir := b.TempDir()
	projectRoot := filepath.Join(tempDir, "project")
	
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		b.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a large config file with many ignore patterns
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

	// Add many ignore patterns to simulate a large config
	for i := 0; i < 1000; i++ {
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

	configPath := filepath.Join(projectRoot, "timemachine.yaml")
	err := os.WriteFile(configPath, []byte(configBuilder.String()), 0600)
	if err != nil {
		b.Fatalf("Failed to write large config: %v", err)
	}

	b.ResetTimer()

	b.Run("LargeConfigFile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager := NewManager()
			err := manager.Load(projectRoot)
			if err != nil {
				b.Errorf("Large config loading failed: %v", err)
			}
		}
	})
}

// BenchmarkConfigValidation benchmarks validation performance
func BenchmarkConfigValidation(b *testing.B) {
	validator := NewValidator()
	
	// Create a complex but valid config
	config := &Config{
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
			File:   "/tmp/benchmark.log",
		},
		Watcher: WatcherConfig{
			DebounceDelay:   5 * time.Second,
			MaxWatchedFiles: 500000,
			BatchSize:       500,
			IgnorePatterns:  make([]string, 100), // 100 patterns
			EnableRecursive: true,
		},
		Cache: CacheConfig{
			MaxEntries:  50000,
			MaxMemoryMB: 256,
			TTL:         12 * time.Hour,
			EnableLRU:   true,
		},
		Git: GitConfig{
			CleanupThreshold: 250,
			MaxCommits:       5000,
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

	// Fill ignore patterns
	for i := 0; i < 100; i++ {
		config.Watcher.IgnorePatterns[i] = fmt.Sprintf("pattern_%d_*.tmp", i)
	}

	b.ResetTimer()

	b.Run("ComplexValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := validator.Validate(config)
			if err != nil {
				b.Errorf("Validation failed: %v", err)
			}
		}
	})
}

// BenchmarkPathValidation benchmarks security path validation
func BenchmarkPathValidation(b *testing.B) {
	validator := NewValidator()
	
	testPaths := []string{
		"/tmp/safe.log",
		"/var/log/app.log",
		"/home/user/documents/config.log",
		"relative/path/file.log",
		"/Users/testuser/Documents/app.log",
		"/var/tmp/temporary.log",
	}

	b.ResetTimer()

	b.Run("SafePaths", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := testPaths[i%len(testPaths)]
			validator.isValidFilePath(path)
		}
	})

	// Test with attack paths (these should be fast to reject)
	attackPaths := []string{
		"../../../etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"..\\..\\..\\windows\\system32\\config",
		"\uff0e\uff0e\uff0f\uff0e\uff0e\uff0f",
		strings.Repeat("../", 100) + "etc/passwd",
	}

	b.Run("AttackPaths", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := attackPaths[i%len(attackPaths)]
			validator.isValidFilePath(path)
		}
	})
}

// BenchmarkConcurrentConfigLoading tests performance under concurrent load
func BenchmarkConcurrentConfigLoading(b *testing.B) {
	tempDir := b.TempDir()
	projectRoot := filepath.Join(tempDir, "project")
	
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		b.Fatalf("Failed to create project dir: %v", err)
	}

	// Create config file
	standardConfig := `
log:
  level: info
  format: text

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  batch_size: 100

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
`

	configPath := filepath.Join(projectRoot, "timemachine.yaml")
	err := os.WriteFile(configPath, []byte(standardConfig), 0600)
	if err != nil {
		b.Fatalf("Failed to write config: %v", err)
	}

	b.ResetTimer()

	b.Run("ConcurrentLoading", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				manager := NewManager()
				err := manager.Load(projectRoot)
				if err != nil {
					b.Errorf("Concurrent config loading failed: %v", err)
				}
			}
		})
	})
}

// BenchmarkEnvironmentVariableProcessing benchmarks env var handling
func BenchmarkEnvironmentVariableProcessing(b *testing.B) {
	// Set up environment variables
	testEnvVars := map[string]string{
		"TIMEMACHINE_LOG_LEVEL":             "debug",
		"TIMEMACHINE_LOG_FORMAT":            "json",
		"TIMEMACHINE_WATCHER_DEBOUNCE":      "3s",
		"TIMEMACHINE_WATCHER_MAX_FILES":     "200000",
		"TIMEMACHINE_CACHE_MAX_ENTRIES":     "20000",
		"TIMEMACHINE_CACHE_MAX_MEMORY":      "100",
		"TIMEMACHINE_GIT_CLEANUP_THRESHOLD": "200",
		"TIMEMACHINE_UI_COLOR":              "false",
		"TIMEMACHINE_UI_PAGER":              "never",
	}

	// Set environment variables
	originalEnvs := make(map[string]string)
	for key, value := range testEnvVars {
		originalEnvs[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	// Clean up after benchmark
	defer func() {
		for key, originalValue := range originalEnvs {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}()

	tempDir := b.TempDir()
	projectRoot := filepath.Join(tempDir, "project")
	
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		b.Fatalf("Failed to create project dir: %v", err)
	}

	b.ResetTimer()

	b.Run("WithEnvironmentVars", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager := NewManager()
			err := manager.Load(projectRoot)
			if err != nil {
				b.Errorf("Config loading with env vars failed: %v", err)
			}
		}
	})
}

// BenchmarkValidationUpdate benchmarks incremental validation
func BenchmarkValidationUpdate(b *testing.B) {
	validator := NewValidator()

	testUpdates := []struct {
		field string
		value interface{}
	}{
		{"log.level", "debug"},
		{"log.level", "info"},
		{"watcher.debounce_delay", 3 * time.Second},
		{"cache.max_entries", 15000},
		{"cache.max_entries", 25000},
	}

	b.ResetTimer()

	b.Run("IncrementalValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			update := testUpdates[i%len(testUpdates)]
			validator.ValidateUpdate(update.field, update.value)
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	tempDir := b.TempDir()
	projectRoot := filepath.Join(tempDir, "project")
	
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		b.Fatalf("Failed to create project dir: %v", err)
	}

	// Create config file
	standardConfig := `
log:
  level: info
  format: text

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  ignore_patterns: ["*.log", "*.tmp", "build/", "dist/", "node_modules/"]
  batch_size: 100

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
`

	configPath := filepath.Join(projectRoot, "timemachine.yaml")
	err := os.WriteFile(configPath, []byte(standardConfig), 0600)
	if err != nil {
		b.Fatalf("Failed to write config: %v", err)
	}

	b.ResetTimer()

	b.Run("MemoryAllocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			manager := NewManager()
			err := manager.Load(projectRoot)
			if err != nil {
				b.Errorf("Config loading failed: %v", err)
			}
			
			// Access config to ensure full initialization
			config := manager.Get()
			_ = config.Log.Level
			_ = config.Watcher.IgnorePatterns
		}
	})
}

// BenchmarkCreateDefaultConfig benchmarks default config file creation
func BenchmarkCreateDefaultConfig(b *testing.B) {
	b.Run("DefaultConfigCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tempDir := b.TempDir()
			manager := NewManager()
			err := manager.CreateDefaultConfigFile(tempDir)
			if err != nil {
				b.Errorf("Default config creation failed: %v", err)
			}
		}
	})
}

// Performance regression tests to ensure optimizations don't break functionality
func BenchmarkRegressionTests(b *testing.B) {
	// These benchmarks help detect performance regressions
	tempDir := b.TempDir()
	projectRoot := filepath.Join(tempDir, "project")
	
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		b.Fatalf("Failed to create project dir: %v", err)
	}

	// Standard config for regression testing
	standardConfig := `
log:
  level: info
  format: text
watcher:
  debounce_delay: 2s
  max_watched_files: 100000
cache:
  max_entries: 10000
  ttl: 1h
git:
  cleanup_threshold: 100
  max_commits: 1000
ui:
  pager: auto
`

	configPath := filepath.Join(projectRoot, "timemachine.yaml")
	err := os.WriteFile(configPath, []byte(standardConfig), 0600)
	if err != nil {
		b.Fatalf("Failed to write config: %v", err)
	}

	b.Run("BaselinePerformance", func(b *testing.B) {
		// This should complete in reasonable time for CI/CD
		// If this benchmark starts failing, investigate performance regressions
		for i := 0; i < b.N; i++ {
			manager := NewManager()
			err := manager.Load(projectRoot)
			if err != nil {
				b.Errorf("Baseline config loading failed: %v", err)
			}
			
			validator := NewValidator()
			config := manager.Get()
			err = validator.Validate(config)
			if err != nil {
				b.Errorf("Baseline validation failed: %v", err)
			}
		}
	})
}