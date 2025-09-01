package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestEnhancedIgnoreManager tests the basic functionality
func TestEnhancedIgnoreManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-ignore-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test .timemachine-ignore file
	ignoreContent := `# Test ignore patterns
*.log
*.tmp
build/
dist/
node_modules/
!important.log
*.test.*
temp/
/.vscode
/src/generated/
`

	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	// Create ignore manager
	manager := NewEnhancedIgnoreManager(tempDir)

	// Test cases
	testCases := []struct {
		path     string
		ignored  bool
		reason   string
	}{
		// Basic pattern matching
		{"app.log", true, "matches *.log"},
		{"data.tmp", true, "matches *.tmp"},
		{"main.go", false, "no matching pattern"},
		{"important.log", false, "negation pattern !important.log"},

		// Directory patterns
		{"build/output.js", true, "matches build/"},
		{"dist/bundle.js", true, "matches dist/"},
		{"node_modules/react/index.js", true, "matches node_modules/"},
		{"temp/file.txt", true, "matches temp/"},

		// Test patterns
		{"app.test.js", true, "matches *.test.*"},
		{"utils.test.go", true, "matches *.test.*"},
		{"test.js", false, "doesn't match *.test.*"},

		// Absolute patterns
		{".vscode/settings.json", true, "matches /.vscode (absolute)"},
		{"project/.vscode/settings.json", false, ".vscode not at root"},
		{"src/generated/api.go", true, "matches /src/generated/ (absolute)"},
		{"lib/src/generated/api.go", false, "src/generated not at root"},

		// Edge cases
		{"", false, "empty path"},
		{".", false, "current directory"},
		{"../parent.txt", false, "parent directory reference"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("path_%s", strings.ReplaceAll(tc.path, "/", "_")), func(t *testing.T) {
			fullPath := filepath.Join(tempDir, tc.path)
			result := manager.ShouldIgnore(fullPath)
			if result != tc.ignored {
				t.Errorf("ShouldIgnore(%q) = %v, want %v (%s)", tc.path, result, tc.ignored, tc.reason)
			}
		})
	}
}

// TestPatternParsing tests the pattern parsing logic
func TestPatternParsing(t *testing.T) {
	manager := &EnhancedIgnoreManager{}

	testCases := []struct {
		input       string
		expected    IgnorePattern
		shouldError bool
	}{
		// Basic patterns
		{
			input: "*.log",
			expected: IgnorePattern{
				Original: "*.log", Pattern: "*.log",
				IsSimple: false, IsNegation: false, IsDirectory: false, IsAbsolute: false,
			},
		},
		{
			input: "simple.txt",
			expected: IgnorePattern{
				Original: "simple.txt", Pattern: "simple.txt",
				IsSimple: true, IsNegation: false, IsDirectory: false, IsAbsolute: false,
			},
		},
		// Negation patterns
		{
			input: "!important.log",
			expected: IgnorePattern{
				Original: "!important.log", Pattern: "important.log",
				IsSimple: true, IsNegation: true, IsDirectory: false, IsAbsolute: false,
			},
		},
		// Directory patterns
		{
			input: "build/",
			expected: IgnorePattern{
				Original: "build/", Pattern: "build",
				IsSimple: true, IsNegation: false, IsDirectory: true, IsAbsolute: false,
			},
		},
		// Absolute patterns
		{
			input: "/src/generated/",
			expected: IgnorePattern{
				Original: "/src/generated/", Pattern: "src/generated",
				IsSimple: true, IsNegation: false, IsDirectory: true, IsAbsolute: true,
			},
		},
		// Complex patterns
		{
			input: "!/*.test.*",
			expected: IgnorePattern{
				Original: "!/*.test.*", Pattern: "*.test.*",
				IsSimple: false, IsNegation: true, IsDirectory: false, IsAbsolute: true,
			},
		},
		// Error cases
		{"", IgnorePattern{}, true},
		{"!", IgnorePattern{}, true},
		{"/", IgnorePattern{}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := manager.parsePattern(tc.input)
			
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for pattern %q, but got none", tc.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for pattern %q: %v", tc.input, err)
				return
			}
			
			if result != tc.expected {
				t.Errorf("Pattern %q parsed incorrectly:\ngot:  %+v\nwant: %+v", 
					tc.input, result, tc.expected)
			}
		})
	}
}

// TestCachePerformance tests the caching mechanism
func TestCachePerformance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create ignore file with some patterns
	ignoreContent := "*.log\n*.tmp\nbuild/\n"
	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	// Test same path multiple times to verify caching
	testPath := filepath.Join(tempDir, "app.log")
	
	// First call - should be cache miss
	result1 := manager.ShouldIgnore(testPath)
	hits1, misses1, total1, _ := manager.GetStats()
	t.Logf("After first call: hits=%d, misses=%d, total=%d", hits1, misses1, total1)
	
	// Second call - should be cache hit
	result2 := manager.ShouldIgnore(testPath)
	hits2, misses2, total2, hitRate := manager.GetStats()
	t.Logf("After second call: hits=%d, misses=%d, total=%d", hits2, misses2, total2)
	
	// Verify results are consistent
	if result1 != result2 {
		t.Errorf("Cache inconsistency: first=%v, second=%v", result1, result2)
	}
	
	// Verify cache stats
	if misses2 != misses1 {
		t.Errorf("Second call should not increase cache misses, got misses: %d -> %d", misses1, misses2)
	}
	
	if hits2 != hits1+1 {
		t.Errorf("Second call should increase cache hits, got hits: %d -> %d", hits1, hits2)
	}
	
	if hitRate <= 0 || hitRate > 100 {
		t.Errorf("Invalid hit rate: %f%% (should be between 0-100)", hitRate)
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create ignore file
	ignoreContent := "*.log\n*.tmp\nbuild/\n"
	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	// Test concurrent access
	const numGoroutines = 50
	const callsPerGoroutine = 100
	
	var wg sync.WaitGroup
	results := make([][]bool, numGoroutines)
	
	// Launch goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			results[goroutineID] = make([]bool, callsPerGoroutine)
			
			for j := 0; j < callsPerGoroutine; j++ {
				testPath := filepath.Join(tempDir, fmt.Sprintf("test%d_%d.log", goroutineID, j))
				results[goroutineID][j] = manager.ShouldIgnore(testPath)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify all results are consistent (all .log files should be ignored)
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < callsPerGoroutine; j++ {
			if !results[i][j] {
				t.Errorf("Expected .log files to be ignored, but got false for goroutine %d, call %d", i, j)
			}
		}
	}
	
	// Verify cache stats
	hits, misses, total, hitRate := manager.GetStats()
	expectedTotal := int64(numGoroutines * callsPerGoroutine)
	
	if total != expectedTotal {
		t.Errorf("Expected %d total calls, got %d", expectedTotal, total)
	}
	
	if hits+misses != total {
		t.Errorf("Cache stats don't add up: hits(%d) + misses(%d) != total(%d)", hits, misses, total)
	}
	
	t.Logf("Concurrent test stats: hits=%d, misses=%d, total=%d, hit rate=%.2f%%", 
		hits, misses, total, hitRate)
}

// TestSecurityLimits tests security limits and error handling
func TestSecurityLimits(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-security-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("TooManyLines", func(t *testing.T) {
		// Create ignore file with too many lines
		var content strings.Builder
		for i := 0; i < MaxIgnoreLines+100; i++ {
			content.WriteString(fmt.Sprintf("pattern%d\n", i))
		}
		
		ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
		if err := os.WriteFile(ignoreFile, []byte(content.String()), 0644); err != nil {
			t.Fatalf("Failed to write ignore file: %v", err)
		}
		
		manager := NewEnhancedIgnoreManager(tempDir)
		patternCount := manager.GetPatternsCount()
		
		if patternCount > MaxPatterns {
			t.Errorf("Too many patterns loaded: %d (max %d)", patternCount, MaxPatterns)
		}
	})
	
	t.Run("InvalidPatterns", func(t *testing.T) {
		// Create ignore file with invalid patterns
		ignoreContent := `valid.txt
!
/
*.log
invalid pattern with 	tab
!another-valid.txt
`
		
		ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile+".invalid")
		if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
			t.Fatalf("Failed to write ignore file: %v", err)
		}
		
		manager := &EnhancedIgnoreManager{
			projectRoot: tempDir,
			ignoreFile:  ignoreFile,
			pathCache:   make(map[string]bool),
		}
		
		err := manager.loadIgnoreFile()
		if err != nil {
			t.Fatalf("loadIgnoreFile failed: %v", err)
		}
		
		// Should have loaded only valid patterns
		patternCount := manager.GetPatternsCount()
		if patternCount <= 0 {
			t.Errorf("Expected some valid patterns to be loaded, got %d", patternCount)
		}
	})
}

// TestMemoryManagement tests cache memory management
func TestMemoryManagement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-memory-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create ignore file
	ignoreContent := "*.log\n"
	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	// Fill cache to trigger eviction
	for i := 0; i < MaxPathCacheEntries+500; i++ {
		testPath := filepath.Join(tempDir, fmt.Sprintf("file%d.log", i))
		manager.ShouldIgnore(testPath)
	}

	// Verify cache size is controlled
	hits, misses, total, hitRate := manager.GetStats()
	memoryUsage := manager.EstimateMemoryUsage()
	
	t.Logf("Memory test stats: hits=%d, misses=%d, total=%d, hit rate=%.2f%%, memory=%d bytes", 
		hits, misses, total, hitRate, memoryUsage)
	
	// Memory usage should be reasonable (less than limit)
	maxMemoryBytes := int64(MaxCacheMemoryMB * 1024 * 1024)
	if memoryUsage > maxMemoryBytes*2 { // Allow some overhead
		t.Errorf("Memory usage too high: %d bytes (max ~%d)", memoryUsage, maxMemoryBytes)
	}
}

// TestReloadIgnoreFile tests dynamic reloading of ignore file
func TestReloadIgnoreFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-reload-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	
	// Create initial ignore file
	ignoreContent1 := "*.log\n"
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent1), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)
	
	// Test initial pattern
	testPath := filepath.Join(tempDir, "app.log")
	if !manager.ShouldIgnore(testPath) {
		t.Errorf("Expected app.log to be ignored initially")
	}

	// Update ignore file
	ignoreContent2 := "*.tmp\n" // Different pattern
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent2), 0644); err != nil {
		t.Fatalf("Failed to update ignore file: %v", err)
	}

	// Reload
	if err := manager.ReloadIgnoreFile(); err != nil {
		t.Fatalf("Failed to reload ignore file: %v", err)
	}

	// Test new pattern
	if manager.ShouldIgnore(testPath) {
		t.Errorf("app.log should not be ignored after reload")
	}
	
	tmpPath := filepath.Join(tempDir, "app.tmp")
	if !manager.ShouldIgnore(tmpPath) {
		t.Errorf("app.tmp should be ignored after reload")
	}
}

// BenchmarkIgnoreCheck benchmarks the ignore checking performance
func BenchmarkIgnoreCheck(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "timemachine-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create realistic ignore file
	ignoreContent := `# Node.js
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*
.npm
.yarn/cache
.yarn/unplugged
.yarn/build-state.yml
.yarn/install-state.gz

# Build outputs
dist/
build/
out/
*.tsbuildinfo

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/

# Temp
*.tmp
*.temp
temp/

# Test
coverage/
.nyc_output
*.test.*
*.spec.*
`

	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		b.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	// Benchmark different types of paths
	testPaths := []string{
		"src/main.go",           // Not ignored
		"app.log",              // Ignored (*.log)
		"build/output.js",      // Ignored (build/)
		"node_modules/react/index.js", // Ignored (node_modules/)
		"test.spec.js",         // Ignored (*.spec.*)
		"important.txt",        // Not ignored
		".DS_Store",           // Ignored
		"coverage/report.html", // Ignored (coverage/)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		fullPath := filepath.Join(tempDir, path)
		_ = manager.ShouldIgnore(fullPath)
	}
}

// BenchmarkCachePerformance benchmarks cache hit vs miss performance
func BenchmarkCachePerformance(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "timemachine-cache-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ignoreContent := "*.log\n*.tmp\nbuild/\n"
	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		b.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	// Pre-populate cache with some entries
	for i := 0; i < 100; i++ {
		testPath := filepath.Join(tempDir, fmt.Sprintf("file%d.log", i))
		manager.ShouldIgnore(testPath)
	}

	b.Run("CacheHit", func(b *testing.B) {
		testPath := filepath.Join(tempDir, "file50.log") // Should be in cache
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.ShouldIgnore(testPath)
		}
	})

	b.Run("CacheMiss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testPath := filepath.Join(tempDir, fmt.Sprintf("new%d.log", i))
			_ = manager.ShouldIgnore(testPath)
		}
	})
}

// TestLegacyCompatibility tests backward compatibility methods
func TestLegacyCompatibility(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-legacy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ignoreContent := "*.log\nbuild/\n"
	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	// Test legacy methods work the same as new methods
	testCases := []struct {
		path      string
		isDir     bool
		ignored   bool
	}{
		{"app.log", false, true},
		{"main.go", false, false},
		{"build", true, true},
		{"src", true, false},
	}

	for _, tc := range testCases {
		fullPath := filepath.Join(tempDir, tc.path)
		
		if tc.isDir {
			result1 := manager.ShouldIgnoreDirectory(fullPath)
			_ = manager.ShouldIgnore(fullPath) // Just test it works
			
			if result1 != tc.ignored {
				t.Errorf("ShouldIgnoreDirectory(%q) = %v, want %v", tc.path, result1, tc.ignored)
			}
			// Note: ShouldIgnore and ShouldIgnoreDirectory might differ for directories
		} else {
			result1 := manager.ShouldIgnoreFile(fullPath)
			result2 := manager.ShouldIgnore(fullPath)
			
			if result1 != tc.ignored {
				t.Errorf("ShouldIgnoreFile(%q) = %v, want %v", tc.path, result1, tc.ignored)
			}
			
			if result1 != result2 {
				t.Errorf("ShouldIgnoreFile and ShouldIgnore gave different results for %q: %v vs %v", 
					tc.path, result1, result2)
			}
		}
	}
}

// TestDirectoryPatternMatching tests patterns like "dir/subdir" that should match files within those paths
func TestDirectoryPatternMatching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-dirpattern-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create ignore file with directory path patterns (without trailing slash)
	ignoreContent := `# Test directory path patterns
dk/test.txt
dk/testdir
src/generated
logs/app
build/dist`

	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	testCases := []struct {
		path    string
		ignored bool
		desc    string
	}{
		// Exact file matches
		{"dk/test.txt", true, "exact file match"},
		
		// Directory matches - should ignore the directory itself
		{"dk/testdir", true, "directory itself"},
		{"src/generated", true, "directory itself"},
		{"logs/app", true, "directory itself"},
		{"build/dist", true, "directory itself"},
		
		// Files within directories - should be ignored
		{"dk/testdir/file1.txt", true, "file within dk/testdir"},
		{"dk/testdir/subdir/file2.txt", true, "file within dk/testdir subdirectory"},
		{"src/generated/api.go", true, "file within src/generated"},
		{"src/generated/models/user.go", true, "file within src/generated subdirectory"},
		{"logs/app/server.log", true, "file within logs/app"},
		{"logs/app/debug/trace.log", true, "file within logs/app subdirectory"},
		{"build/dist/main.js", true, "file within build/dist"},
		{"build/dist/assets/style.css", true, "file within build/dist subdirectory"},
		
		// Files that should NOT be ignored
		{"dk/other.txt", false, "different file in dk/"},
		{"dk/testdir.backup", false, "similar filename but not exact match"},
		{"src/main.go", false, "file in src/ but not src/generated"},
		{"src/generated.bak", false, "similar filename but not directory"},
		{"logs/error.log", false, "file in logs/ but not logs/app"},
		{"logs/application.log", false, "similar but not exact directory match"},
		{"build/main.js", false, "file in build/ but not build/dist"},
		{"other/testdir/file.txt", false, "testdir in different parent directory"},
		{"testdir/file.txt", false, "testdir without dk/ prefix"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, tc.path)
			result := manager.ShouldIgnore(fullPath)
			
			if result != tc.ignored {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tc.path, result, tc.ignored)
			}
		})
	}
}

// TestPatternWithSlashesVsSimplePatterns tests the difference between patterns with and without slashes
func TestPatternWithSlashesVsSimplePatterns(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-slash-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test patterns with and without slashes
	ignoreContent := `# Patterns with slashes (path-specific)
config/database.yml
logs/app.log

# Patterns without slashes (filename-only)
*.tmp
secret.key`

	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	testCases := []struct {
		path    string
		ignored bool
		desc    string
	}{
		// Slash patterns - should match specific paths only
		{"config/database.yml", true, "exact path match"},
		{"logs/app.log", true, "exact path match"},
		{"other/config/database.yml", false, "same filename, different path"},
		{"database.yml", false, "filename only, not path"},
		{"app.log", false, "filename only, not path"},
		
		// Non-slash patterns - should match filename anywhere
		{"file.tmp", true, "*.tmp pattern matches anywhere"},
		{"dir/file.tmp", true, "*.tmp pattern matches in subdirectory"},
		{"deep/nested/dir/file.tmp", true, "*.tmp pattern matches deep in hierarchy"},
		{"secret.key", true, "exact filename matches anywhere"},
		{"dir/secret.key", true, "exact filename matches in subdirectory"},
		{"config/secret.key", true, "exact filename matches in different path"},
		
		// Should not match
		{"secret.key.backup", false, "partial filename match"},
		{"file.tmp.old", false, "partial extension match"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, tc.path)
			result := manager.ShouldIgnore(fullPath)
			
			if result != tc.ignored {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tc.path, result, tc.ignored)
			}
		})
	}
}

// TestRealWorldPatterns tests common real-world patterns like those found in .gitignore files
func TestRealWorldPatterns(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timemachine-realworld-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Real-world patterns commonly found in .gitignore files
	ignoreContent := `# Dependencies
node_modules/
vendor/

# Build outputs  
dist/
build/
target/
*.o
*.exe

# IDE files
.vscode/settings.json
.idea/workspace.xml

# Logs and temp files
logs/
*.log
*.tmp

# OS files
.DS_Store
Thumbs.db

# Project specific
src/generated/
config/local.yml`

	ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write ignore file: %v", err)
	}

	manager := NewEnhancedIgnoreManager(tempDir)

	testCases := []struct {
		path    string
		ignored bool
		desc    string
	}{
		// Directory patterns (with trailing slash)
		{"node_modules/react/index.js", true, "file in node_modules"},
		{"vendor/github.com/lib/pq/driver.go", true, "file in vendor"},
		{"dist/main.js", true, "file in dist"},
		{"build/app.exe", true, "file in build"},
		{"target/classes/Main.class", true, "file in target"},
		{"logs/app.log", true, "file in logs directory"},
		
		// File extension patterns
		{"main.o", true, "*.o anywhere"},
		{"src/main.o", true, "*.o in subdirectory"},
		{"app.exe", true, "*.exe anywhere"},
		{"build/app.exe", true, "*.exe in subdirectory"},
		{"debug.log", true, "*.log anywhere"},
		{"temp.tmp", true, "*.tmp anywhere"},
		
		// Specific path patterns
		{".vscode/settings.json", true, "specific IDE file"},
		{".idea/workspace.xml", true, "specific IDE file"},
		{"src/generated/api.go", true, "file in generated directory"},
		{"config/local.yml", true, "specific config file"},
		
		// OS-specific files
		{".DS_Store", true, "macOS file"},
		{"Thumbs.db", true, "Windows file"},
		
		// Should NOT be ignored
		{"src/main.js", false, "regular source file"},
		{"config/production.yml", false, "different config file"},
		{".vscode/extensions.json", false, "different IDE file"},
		{"generated/manual.go", false, "generated in different path"},
		{"node_modules.backup", false, "similar but different name"},
		{"logs.txt", false, "similar but different name"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, tc.path)
			result := manager.ShouldIgnore(fullPath)
			
			if result != tc.ignored {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tc.path, result, tc.ignored)
			}
		})
	}
}

// Test for edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("NonExistentIgnoreFile", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "timemachine-edge-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// No ignore file exists
		manager := NewEnhancedIgnoreManager(tempDir)
		
		// Should work without errors
		testPath := filepath.Join(tempDir, "any-file.txt")
		result := manager.ShouldIgnore(testPath)
		
		// Should not ignore anything (no patterns loaded)
		if result {
			t.Errorf("Expected no files to be ignored when no ignore file exists")
		}
	})
	
	t.Run("EmptyIgnoreFile", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "timemachine-empty-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Create empty ignore file
		ignoreFile := filepath.Join(tempDir, DefaultIgnoreFile)
		if err := os.WriteFile(ignoreFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write ignore file: %v", err)
		}
		
		manager := NewEnhancedIgnoreManager(tempDir)
		
		if manager.GetPatternsCount() != 0 {
			t.Errorf("Expected 0 patterns from empty file, got %d", manager.GetPatternsCount())
		}
	})
}