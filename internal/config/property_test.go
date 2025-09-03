package config

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// Property-based testing for configuration validation
// These tests generate random inputs to find edge cases

// TestPropertyBasedLogLevel tests log level validation with random inputs
func TestPropertyBasedLogLevel(t *testing.T) {
	validator := NewValidator()
	validLevels := []string{"debug", "info", "warn", "error"}
	
	// Test with 1000 random strings
	for i := 0; i < 1000; i++ {
		randomLevel := generateRandomString(1, 20)
		
		config := &LogConfig{
			Level:  randomLevel,
			Format: "text",
			File:   "",
		}
		
		err := validator.validateLogConfig(config)
		
		// Check if the random level is in valid levels
		isValid := false
		for _, valid := range validLevels {
			if randomLevel == valid {
				isValid = true
				break
			}
		}
		
		if isValid && err != nil {
			t.Errorf("Valid log level '%s' was rejected: %v", randomLevel, err)
		}
		
		if !isValid && err == nil {
			t.Errorf("Invalid log level '%s' was accepted", randomLevel)
		}
	}
}

// TestPropertyBasedPathValidation tests path validation with generated attack vectors
func TestPropertyBasedPathValidation(t *testing.T) {
	validator := NewValidator()
	
	// Generate paths with various attack patterns
	generators := []func() string{
		generatePathTraversalAttack,
		generateURLEncodedAttack,
		generateUnicodeAttack,
		generateLongPathAttack,
		generateControlCharacterAttack,
		generateCaseVariationAttack,
	}
	
	for _, generator := range generators {
		for i := 0; i < 200; i++ {
			attackPath := generator()
			
			// Any path containing ".." should be rejected
			shouldBeRejected := strings.Contains(attackPath, "..")
			
			result := validator.isValidFilePath(attackPath)
			
			if shouldBeRejected && result {
				t.Errorf("Security vulnerability: malicious path was accepted: '%s'", attackPath)
			}
		}
	}
}

// TestPropertyBasedTimeRanges tests time validation with random durations
func TestPropertyBasedTimeRanges(t *testing.T) {
	validator := NewValidator()
	
	for i := 0; i < 1000; i++ {
		// Generate random duration
		randomMs := rand.Intn(3600000) // 0 to 1 hour in milliseconds
		randomDuration := time.Duration(randomMs) * time.Millisecond
		
		config := &WatcherConfig{
			DebounceDelay:   randomDuration,
			MaxWatchedFiles: 10000,
			BatchSize:       100,
		}
		
		err := validator.validateWatcherConfig(config)
		
		// Check if duration is in valid range (100ms to 10s)
		isValid := randomDuration >= 100*time.Millisecond && randomDuration <= 10*time.Second
		
		if isValid && err != nil {
			t.Errorf("Valid debounce delay %v was rejected: %v", randomDuration, err)
		}
		
		if !isValid && err == nil {
			t.Errorf("Invalid debounce delay %v was accepted", randomDuration)
		}
	}
}

// TestPropertyBasedIntegerRanges tests integer validation with random values
func TestPropertyBasedIntegerRanges(t *testing.T) {
	validator := NewValidator()
	
	// Test cache max entries
	for i := 0; i < 1000; i++ {
		randomEntries := rand.Intn(200000) // 0 to 200k
		
		config := &CacheConfig{
			MaxEntries:  randomEntries,
			MaxMemoryMB: 50,
			TTL:         time.Hour,
			EnableLRU:   true,
		}
		
		err := validator.validateCacheConfig(config)
		
		// Valid range: 1000-100000
		isValid := randomEntries >= 1000 && randomEntries <= 100000
		
		if isValid && err != nil {
			t.Errorf("Valid cache entries %d was rejected: %v", randomEntries, err)
		}
		
		if !isValid && err == nil {
			t.Errorf("Invalid cache entries %d was accepted", randomEntries)
		}
	}
	
	// Test git config relationship (cleanup_threshold < max_commits)
	for i := 0; i < 1000; i++ {
		cleanup := rand.Intn(10000) + 1    // 1-10000
		maxCommits := rand.Intn(50000) + 1 // 1-50000
		
		config := &GitConfig{
			CleanupThreshold: cleanup,
			MaxCommits:       maxCommits,
			AutoGC:           true,
			UseShallowClone:  false,
		}
		
		err := validator.validateGitConfig(config)
		
		// Both must be in valid range AND cleanup < max_commits
		cleanupValid := cleanup >= 10 && cleanup <= 10000
		maxCommitsValid := maxCommits >= 50 && maxCommits <= 50000
		relationshipValid := cleanup < maxCommits
		
		isValid := cleanupValid && maxCommitsValid && relationshipValid
		
		if isValid && err != nil {
			t.Errorf("Valid git config (cleanup: %d, max: %d) was rejected: %v", 
				cleanup, maxCommits, err)
		}
		
		if !isValid && err == nil {
			t.Errorf("Invalid git config (cleanup: %d, max: %d) was accepted", 
				cleanup, maxCommits)
		}
	}
}

// TestPropertyBasedConfigCombinations tests random configuration combinations
func TestPropertyBasedConfigCombinations(t *testing.T) {
	validator := NewValidator()
	
	for i := 0; i < 500; i++ {
		config := generateRandomConfig()
		
		err := validator.Validate(config)
		
		// Manually validate each section
		logValid := isLogConfigValid(&config.Log)
		watcherValid := isWatcherConfigValid(&config.Watcher)
		cacheValid := isCacheConfigValid(&config.Cache)
		gitValid := isGitConfigValid(&config.Git)
		uiValid := isUIConfigValid(&config.UI)
		
		expectedValid := logValid && watcherValid && cacheValid && gitValid && uiValid
		
		if expectedValid && err != nil {
			t.Errorf("Valid config combination was rejected: %+v, error: %v", config, err)
		}
		
		if !expectedValid && err == nil {
			t.Errorf("Invalid config combination was accepted: %+v", config)
		}
	}
}

// TestFuzzConfigFileContent simulates malformed/corrupted config files
func TestFuzzConfigFileContent(t *testing.T) {
	// Generate various types of malformed YAML content
	for i := 0; i < 100; i++ {
		fuzzContent := generateFuzzYAML()
		
		// Write to temp file and try to load
		tempDir := t.TempDir()
		configPath := tempDir + "/timemachine.yaml"
		
		if err := writeFile(configPath, fuzzContent); err != nil {
			continue // Skip if we can't write the file
		}
		
		manager := NewManager()
		err := manager.Load(tempDir)
		
		// The key requirement: either succeed with valid config or fail gracefully
		// Should never panic or produce undefined behavior
		if err == nil {
			config := manager.Get()
			// If loading succeeded, config should be valid
			validator := NewValidator()
			validateErr := validator.Validate(config)
			if validateErr != nil {
				t.Errorf("Fuzz test produced invalid config but no load error: %v", validateErr)
			}
		}
		// If err != nil, that's fine - graceful failure is expected for malformed input
	}
}

// Helper functions for property-based testing

// generateRandomString creates a random string of given length range
func generateRandomString(minLen, maxLen int) string {
	length := rand.Intn(maxLen-minLen) + minLen
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:'\",.<>?/`~"
	
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// generatePathTraversalAttack creates path traversal attacks
func generatePathTraversalAttack() string {
	patterns := []string{"../", "..\\", ".../", "....//"}
	depth := rand.Intn(10) + 1
	
	var attack strings.Builder
	for i := 0; i < depth; i++ {
		pattern := patterns[rand.Intn(len(patterns))]
		attack.WriteString(pattern)
	}
	
	targets := []string{"etc/passwd", "windows/system32", "root/.ssh/id_rsa", "etc/shadow"}
	attack.WriteString(targets[rand.Intn(len(targets))])
	
	return attack.String()
}

// generateURLEncodedAttack creates URL-encoded attacks
func generateURLEncodedAttack() string {
	base := generatePathTraversalAttack()
	
	// Randomly URL encode characters
	var encoded strings.Builder
	for _, char := range base {
		if rand.Float32() < 0.3 { // 30% chance to encode
			encoded.WriteString(fmt.Sprintf("%%%02X", char))
		} else {
			encoded.WriteRune(char)
		}
	}
	
	return encoded.String()
}

// generateUnicodeAttack creates Unicode-based attacks
func generateUnicodeAttack() string {
	// Unicode variations of dots and slashes
	unicodeDots := []string{"\u002e", "\u2024", "\uff0e"}
	unicodeSlashes := []string{"\u002f", "\u2044", "\uff0f"}
	
	var attack strings.Builder
	depth := rand.Intn(5) + 1
	
	for i := 0; i < depth; i++ {
		dot := unicodeDots[rand.Intn(len(unicodeDots))]
		slash := unicodeSlashes[rand.Intn(len(unicodeSlashes))]
		attack.WriteString(dot + dot + slash)
	}
	
	attack.WriteString("etc/passwd")
	return attack.String()
}

// generateLongPathAttack creates very long path attacks
func generateLongPathAttack() string {
	segments := rand.Intn(500) + 100 // 100-600 segments
	var path strings.Builder
	
	for i := 0; i < segments; i++ {
		if i > 0 {
			path.WriteString("/")
		}
		if rand.Float32() < 0.8 { // 80% normal segments
			path.WriteString(fmt.Sprintf("segment%d", i))
		} else { // 20% traversal attempts
			path.WriteString("..")
		}
	}
	
	return path.String()
}

// generateControlCharacterAttack injects control characters
func generateControlCharacterAttack() string {
	base := "../../../etc/passwd"
	controlChars := []byte{0, 1, 7, 8, 9, 10, 11, 12, 13, 27}
	
	// Insert random control characters
	result := make([]byte, 0, len(base)*2)
	for i, char := range []byte(base) {
		if rand.Float32() < 0.2 && i > 0 { // 20% chance to insert control char
			ctrl := controlChars[rand.Intn(len(controlChars))]
			result = append(result, ctrl)
		}
		result = append(result, char)
	}
	
	return string(result)
}

// generateCaseVariationAttack creates case variation attacks
func generateCaseVariationAttack() string {
	base := "../../../etc/passwd"
	
	var result strings.Builder
	for _, char := range base {
		if rand.Float32() < 0.3 { // 30% chance to change case
			if char >= 'a' && char <= 'z' {
				result.WriteRune(char - 'a' + 'A')
			} else if char >= 'A' && char <= 'Z' {
				result.WriteRune(char - 'A' + 'a')
			} else {
				result.WriteRune(char)
			}
		} else {
			result.WriteRune(char)
		}
	}
	
	return result.String()
}

// generateRandomConfig creates random configuration for testing
func generateRandomConfig() *Config {
	logLevels := []string{"debug", "info", "warn", "error", "invalid"}
	logFormats := []string{"text", "json", "invalid"}
	pagerSettings := []string{"auto", "always", "never", "invalid"}
	tableFormats := []string{"table", "json", "yaml", "invalid"}
	
	return &Config{
		Log: LogConfig{
			Level:  logLevels[rand.Intn(len(logLevels))],
			Format: logFormats[rand.Intn(len(logFormats))],
			File:   generateRandomPath(),
		},
		Watcher: WatcherConfig{
			DebounceDelay:   time.Duration(rand.Intn(20000)) * time.Millisecond,
			MaxWatchedFiles: rand.Intn(2000000),
			BatchSize:       rand.Intn(2000),
			EnableRecursive: rand.Float32() < 0.5,
		},
		Cache: CacheConfig{
			MaxEntries:  rand.Intn(200000),
			MaxMemoryMB: rand.Intn(2000),
			TTL:         time.Duration(rand.Intn(86400)) * time.Second,
			EnableLRU:   rand.Float32() < 0.5,
		},
		Git: GitConfig{
			CleanupThreshold: rand.Intn(20000),
			MaxCommits:       rand.Intn(100000),
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

// generateRandomPath creates potentially malicious paths for testing
func generateRandomPath() string {
	if rand.Float32() < 0.2 { // 20% chance of empty path
		return ""
	}
	
	if rand.Float32() < 0.3 { // 30% chance of attack path
		generators := []func() string{
			generatePathTraversalAttack,
			generateURLEncodedAttack,
			generateUnicodeAttack,
		}
		return generators[rand.Intn(len(generators))]()
	}
	
	// Otherwise generate "normal" path
	return fmt.Sprintf("/tmp/random_%d.log", rand.Intn(10000))
}

// generateFuzzYAML creates malformed YAML content
func generateFuzzYAML() string {
	fuzzTypes := []func() string{
		generateMalformedYAML,
		generateOversizedYAML,
		generateNestedYAML,
		generateSpecialCharYAML,
		generateIncompleteYAML,
	}
	
	return fuzzTypes[rand.Intn(len(fuzzTypes))]()
}

// generateMalformedYAML creates syntactically invalid YAML
func generateMalformedYAML() string {
	templates := []string{
		"log:\n  level: [unclosed\nwatcher:",
		"invalid: yaml: syntax:\n  - missing\n    - brackets",
		"log:\nlevel: invalid_indent",
		"{{invalid_template}}",
		"log:\n  level:\n    - nested\n      - too\n        - deep",
		"tabs:\tmixed\n  spaces: bad",
	}
	
	return templates[rand.Intn(len(templates))]
}

// generateOversizedYAML creates extremely large YAML content
func generateOversizedYAML() string {
	var content strings.Builder
	content.WriteString("log:\n  level: info\n")
	content.WriteString("large_section:\n")
	
	// Add a large section to test memory handling
	for i := 0; i < 1000; i++ {
		content.WriteString(fmt.Sprintf("  key_%d: %s\n", i, strings.Repeat("x", 100)))
	}
	
	return content.String()
}

// generateNestedYAML creates deeply nested YAML
func generateNestedYAML() string {
	var content strings.Builder
	content.WriteString("log:\n  level: info\n")
	
	// Create deeply nested structure
	depth := 50
	for i := 0; i < depth; i++ {
		content.WriteString(strings.Repeat("  ", i+1))
		content.WriteString(fmt.Sprintf("nested_%d:\n", i))
	}
	
	return content.String()
}

// generateSpecialCharYAML injects special characters into YAML
func generateSpecialCharYAML() string {
	specialChars := []string{"\x00", "\x01", "\x1f", "\x7f", "\xff"}
	
	content := "log:\n  level: info"
	for _, char := range specialChars {
		if rand.Float32() < 0.3 {
			content += char
		}
	}
	
	return content + "\nwatcher:\n  debounce_delay: 2s"
}

// generateIncompleteYAML creates incomplete YAML structures
func generateIncompleteYAML() string {
	incompletes := []string{
		"log:\n  level:",
		"log:\nwatcher",
		"log:\n  level: info\nwatcher:\n  debounce_delay",
		"incomplete",
		"log:\n  level: info\n  format:",
	}
	
	return incompletes[rand.Intn(len(incompletes))]
}

// Validation helper functions

func isLogConfigValid(config *LogConfig) bool {
	validLevels := []string{"debug", "info", "warn", "error"}
	validFormats := []string{"text", "json"}
	
	levelValid := false
	for _, level := range validLevels {
		if config.Level == level {
			levelValid = true
			break
		}
	}
	
	formatValid := false
	for _, format := range validFormats {
		if config.Format == format {
			formatValid = true
			break
		}
	}
	
	// Use the actual validator for consistency
	pathValid := true
	if config.File != "" {
		validator := NewValidator()
		pathValid = validator.isValidFilePath(config.File)
	}
	
	return levelValid && formatValid && pathValid
}

func isWatcherConfigValid(config *WatcherConfig) bool {
	return config.DebounceDelay >= 100*time.Millisecond &&
		config.DebounceDelay <= 10*time.Second &&
		config.MaxWatchedFiles >= 1000 &&
		config.MaxWatchedFiles <= 1000000 &&
		config.BatchSize >= 1 &&
		config.BatchSize <= 1000
}

func isCacheConfigValid(config *CacheConfig) bool {
	return config.MaxEntries >= 1000 &&
		config.MaxEntries <= 100000 &&
		config.MaxMemoryMB >= 10 &&
		config.MaxMemoryMB <= 1024 &&
		config.TTL >= time.Minute &&
		config.TTL <= 24*time.Hour
}

func isGitConfigValid(config *GitConfig) bool {
	return config.CleanupThreshold >= 10 &&
		config.CleanupThreshold <= 10000 &&
		config.MaxCommits >= 50 &&
		config.MaxCommits <= 50000 &&
		config.CleanupThreshold < config.MaxCommits
}

func isUIConfigValid(config *UIConfig) bool {
	validPagers := []string{"auto", "always", "never"}
	validFormats := []string{"table", "json", "yaml"}
	
	pagerValid := false
	for _, pager := range validPagers {
		if config.Pager == pager {
			pagerValid = true
			break
		}
	}
	
	formatValid := false
	for _, format := range validFormats {
		if config.TableFormat == format {
			formatValid = true
			break
		}
	}
	
	return pagerValid && formatValid
}

// writeFile helper that handles errors gracefully
func writeFile(path, content string) error {
	// Validate content is valid UTF-8 to prevent issues
	if !utf8.ValidString(content) {
		return fmt.Errorf("invalid UTF-8 content")
	}
	
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	return err
}