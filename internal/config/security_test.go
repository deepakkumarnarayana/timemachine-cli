package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSecurityPathTraversal tests comprehensive path traversal attack vectors
func TestSecurityPathTraversal(t *testing.T) {
	validator := NewValidator()

	// Comprehensive path traversal attack vectors
	attacks := []struct {
		name        string
		path        string
		description string
	}{
		// Basic path traversal
		{"basic_dotdot", "../../../etc/passwd", "basic directory traversal"},
		{"relative_dotdot", "logs/../../../etc/passwd", "relative path with traversal"},
		{"absolute_dotdot", "/tmp/../../../etc/passwd", "absolute path with traversal"},
		
		// URL encoded attacks
		{"url_encoded_basic", "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd", "URL encoded path traversal"},
		{"url_encoded_mixed", "../%2e%2e/%2e%2e/etc/passwd", "mixed encoding path traversal"},
		{"double_url_encoded", "%252e%252e%252f", "double URL encoded"},
		
		// Unicode attacks
		{"unicode_dotdot", "\u002e\u002e\u002f\u002e\u002e\u002f", "Unicode encoded dots"},
		{"unicode_fullwidth", "\uff0e\uff0e\uff0f", "Unicode fullwidth characters"},
		
		// Windows path traversal
		{"windows_backslash", "..\\..\\..\\windows\\system32\\config", "Windows backslash traversal"},
		{"windows_mixed", "../..\\windows/system32", "mixed Windows/Unix separators"},
		{"windows_unc", "\\\\server\\share\\..\\..\\system", "Windows UNC path traversal"},
		
		// Null byte injection
		{"null_byte_1", "../../../etc/passwd\x00.log", "null byte injection"},
		{"null_byte_2", "/etc/passwd\x00", "null byte termination"},
		
		// Overlong UTF-8
		{"overlong_utf8", "\xc0\xae\xc0\xae/", "overlong UTF-8 encoding"},
		
		// Case variations
		{"case_variation_1", "../../../ETC/PASSWD", "uppercase variations"},
		{"case_variation_2", "../../../Etc/Passwd", "mixed case variations"},
		
		// Space and control character attacks
		{"leading_spaces", "   ../../../etc/passwd", "leading spaces"},
		{"trailing_spaces", "../../../etc/passwd   ", "trailing spaces"},
		{"tab_chars", "\t../../../etc/passwd", "tab characters"},
		{"newline_chars", "\n../../../etc/passwd", "newline injection"},
		{"carriage_return", "\r../../../etc/passwd", "carriage return injection"},
		
		// Multiple encoding attacks
		{"hex_encoded", "\x2e\x2e\x2f\x2e\x2e\x2f", "hex encoded traversal"},
		{"base64_like", "Li4vLi4vLi4v", "base64-like encoding (not actually decoded)"},
		
		// Symbolic link style attacks
		{"symlink_style", "logs -> ../../../etc/passwd", "symbolic link style"},
		
		// Long path attacks
		{"long_path", strings.Repeat("../", 1000) + "etc/passwd", "extremely long path traversal"},
		
		// Mixed attacks
		{"mixed_attack_1", "%2e%2e/..\\..%2f%2e%2e/etc/passwd", "mixed encoding and separators"},
		{"mixed_attack_2", "../\x2e\x2e/%2e%2e/etc/passwd", "hex, URL, and plain encoding"},
	}

	for _, attack := range attacks {
		t.Run(attack.name, func(t *testing.T) {
			if validator.isValidFilePath(attack.path) {
				t.Errorf("Security vulnerability: path traversal attack '%s' was allowed: %s (description: %s)",
					attack.name, attack.path, attack.description)
			}
		})
	}
}

// TestSecurityEnvironmentVariableWhitelist ensures only allowed env vars are processed
func TestSecurityEnvironmentVariableWhitelist(t *testing.T) {
	// Test malicious environment variable injection attempts
	maliciousEnvVars := []struct {
		name     string
		envVar   string
		expected string
	}{
		{"injection_attempt_1", "TIMEMACHINE_EVIL_INJECT", ""},
		{"injection_attempt_2", "MALICIOUS_VAR", ""},
		{"injection_attempt_3", "TIMEMACHINE_LOG_LEVEL_INJECT", ""},
		{"case_variation_1", "timemachine_log_level", ""}, // lowercase
		{"case_variation_2", "TIMEMACHINE_log_level", ""}, // mixed case
		{"prefix_attack", "XTIMEMACHINE_LOG_LEVEL", ""},
		{"suffix_attack", "TIMEMACHINE_LOG_LEVELX", ""},
	}

	for _, test := range maliciousEnvVars {
		t.Run(test.name, func(t *testing.T) {
			// Set the malicious environment variable
			os.Setenv(test.envVar, "malicious_value")
			defer os.Unsetenv(test.envVar)

			// Create manager and load config
			manager := NewManager()
			tempDir, err := os.MkdirTemp("", "security-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			err = manager.Load(tempDir)
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			// Verify the malicious env var didn't affect config
			// The malicious value should not appear anywhere in the config
			config := manager.Get()
			if containsMaliciousValue(config, "malicious_value") {
				t.Errorf("Security vulnerability: malicious environment variable '%s' affected configuration", test.envVar)
			}
		})
	}
}

// TestSecurityFilePermissions verifies secure file permissions
func TestSecurityFilePermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "security-perm-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager()
	err = manager.CreateDefaultConfigFile(tempDir)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	configPath := filepath.Join(tempDir, "timemachine.yaml")
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	// Check permissions (should be 0600 - owner read/write only)
	perm := fileInfo.Mode().Perm()
	expectedPerm := os.FileMode(0600)
	
	if perm != expectedPerm {
		t.Errorf("Security vulnerability: config file has incorrect permissions %o, expected %o", perm, expectedPerm)
	}
}

// TestSecurityConfigFileInjection tests against malicious config file content
func TestSecurityConfigFileInjection(t *testing.T) {
	maliciousConfigs := []struct {
		name    string
		content string
		desc    string
	}{
		{
			name: "yaml_injection",
			content: `
log:
  level: !!python/object/apply:os.system ["rm -rf /"]
  format: text`,
			desc: "YAML deserialization attack",
		},
		{
			name: "billion_laughs",
			content: `
lol: &lol "lol"
lol2: &lol2 [*lol,*lol,*lol,*lol,*lol,*lol,*lol,*lol,*lol]
lol3: &lol3 [*lol2,*lol2,*lol2,*lol2,*lol2,*lol2,*lol2,*lol2,*lol2]
lol4: &lol4 [*lol3,*lol3,*lol3,*lol3,*lol3,*lol3,*lol3,*lol3,*lol3]
log:
  level: *lol4`,
			desc: "Billion laughs YAML bomb",
		},
		{
			name: "path_injection",
			content: `
log:
  file: "../../../etc/passwd"`,
			desc: "Path injection in config values",
		},
		{
			name: "command_injection_attempt",
			content: `
log:
  level: "info; rm -rf /"
  format: "text && malicious_command"`,
			desc: "Command injection attempt",
		},
	}

	for _, test := range maliciousConfigs {
		t.Run(test.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "malicious-config-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "timemachine.yaml")
			err = os.WriteFile(configPath, []byte(test.content), 0600)
			if err != nil {
				t.Fatalf("Failed to write malicious config: %v", err)
			}

			manager := NewManager()
			err = manager.Load(tempDir)

			// The load should either fail gracefully or sanitize the input
			// We specifically test that validation catches malicious content
			if err == nil {
				config := manager.Get()
				// Ensure no path traversal made it through validation
				if config.Log.File != "" && strings.Contains(config.Log.File, "..") {
					t.Errorf("Security vulnerability: path traversal in config value: %s", config.Log.File)
				}
			}
		})
	}
}

// TestSecurityLargeConfigFiles tests DoS via large config files
func TestSecurityLargeConfigFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "large-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a very large config file (but not so large it breaks the test system)
	largeContent := "log:\n  level: info\n"
	largeContent += "  # " + strings.Repeat("x", 10*1024*1024) + "\n" // 10MB comment
	largeContent += "watcher:\n  debounce_delay: 2s"

	configPath := filepath.Join(tempDir, "timemachine.yaml")
	err = os.WriteFile(configPath, []byte(largeContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write large config: %v", err)
	}

	manager := NewManager()
	
	// This should either handle gracefully or fail with proper error
	// The key is it shouldn't cause memory exhaustion or hang
	err = manager.Load(tempDir)
	// We accept either success (if system handles large files) or controlled failure
	if err != nil {
		t.Logf("Large config file handling: %v (this may be expected behavior)", err)
	}
}

// Helper function to check if malicious value appears in config
func containsMaliciousValue(config *Config, maliciousValue string) bool {
	return strings.Contains(config.Log.Level, maliciousValue) ||
		strings.Contains(config.Log.Format, maliciousValue) ||
		strings.Contains(config.Log.File, maliciousValue) ||
		strings.Contains(config.UI.Pager, maliciousValue) ||
		strings.Contains(config.UI.TableFormat, maliciousValue)
}

// TestSecurityAbsolutePathValidation tests all edge cases for absolute path validation
func TestSecurityAbsolutePathValidation(t *testing.T) {
	validator := NewValidator()

	// Save original environment
	originalHome := os.Getenv("HOME")
	
	defer func() {
		os.Setenv("HOME", originalHome)
	}()

	tests := []struct {
		name      string
		path      string
		setupFunc func()
		expected  bool
		desc      string
	}{
		{
			name:     "safe_tmp_path",
			path:     "/tmp/safe.log",
			expected: true,
			desc:     "safe temporary directory",
		},
		{
			name:     "safe_var_log_path", 
			path:     "/var/log/app.log",
			expected: true,
			desc:     "safe system log directory",
		},
		{
			name:     "unsafe_etc_path",
			path:     "/etc/passwd",
			expected: false,
			desc:     "unsafe system configuration",
		},
		{
			name:     "unsafe_root_path",
			path:     "/root/.ssh/id_rsa",
			expected: false,
			desc:     "unsafe root directory",
		},
		{
			name:     "unsafe_bin_path",
			path:     "/bin/sh",
			expected: false,
			desc:     "unsafe binary directory",
		},
		{
			name: "safe_home_path",
			path: "/home/user/.config/app.log",
			setupFunc: func() {
				os.Setenv("HOME", "/home/user")
			},
			expected: true,
			desc:     "safe user home directory",
		},
		{
			name:     "macos_user_path",
			path:     "/Users/testuser/Documents/app.log",
			expected: true,
			desc:     "macOS user directory",
		},
		{
			name:     "case_insensitive_attack",
			path:     "/ETC/passwd",
			expected: false,
			desc:     "case variation attack on system files",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setupFunc != nil {
				test.setupFunc()
			}
			
			result := validator.isValidFilePath(test.path)
			if result != test.expected {
				t.Errorf("Security test '%s' failed: path '%s' returned %v, expected %v (%s)",
					test.name, test.path, result, test.expected, test.desc)
			}
		})
	}
}

// TestSecurityURLDecodingEdgeCases tests URL decoding attack vectors
func TestSecurityURLDecodingEdgeCases(t *testing.T) {
	validator := NewValidator()

	attacks := []struct {
		name string
		path string
		desc string
	}{
		{"single_encoded", "%2e%2e%2f", "single URL encoding"},
		{"double_encoded", "%252e%252e%252f", "double URL encoding"},
		{"mixed_encoded", "%2e%2e/../", "mixed encoding"},
		{"plus_encoded", "%2b%2e%2e%2f", "plus character encoded"},
		{"space_encoded", "%20%2e%2e%2f", "space character encoded"},
		{"null_encoded", "%00%2e%2e%2f", "null character encoded"},
		{"unicode_encoded", "%u002e%u002e%u002f", "Unicode URL encoding"},
		{"malformed_encoding", "%2", "malformed URL encoding"},
		{"invalid_hex", "%GG", "invalid hex in URL encoding"},
	}

	for _, attack := range attacks {
		t.Run(attack.name, func(t *testing.T) {
			// First verify our URL decoding catches the attack
			decoded, err := url.QueryUnescape(attack.path)
			if err == nil && strings.Contains(decoded, "..") {
				t.Logf("URL decoding detected traversal in %s -> %s", attack.path, decoded)
			}
			
			// Then verify our validator blocks it
			if validator.isValidFilePath(attack.path) {
				t.Errorf("Security vulnerability: URL encoding attack '%s' was allowed: %s (%s)",
					attack.name, attack.path, attack.desc)
			}
		})
	}
}

// TestSecurityConcurrentAccess tests thread safety of validation
func TestSecurityConcurrentAccess(t *testing.T) {
	validator := NewValidator()
	
	// Test concurrent access to validator methods
	numGoroutines := 100
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			// Test various validation methods concurrently
			testPaths := []string{
				"/tmp/safe.log",
				"../../../etc/passwd",
				"/home/user/app.log",
				fmt.Sprintf("/tmp/test_%d.log", id),
			}
			
			for _, path := range testPaths {
				validator.isValidFilePath(path)
			}
			
			// Test config validation
			config := &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					File:   fmt.Sprintf("/tmp/concurrent_%d.log", id),
				},
			}
			validator.validateLogConfig(&config.Log)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}