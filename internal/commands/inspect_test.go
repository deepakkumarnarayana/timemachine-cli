package commands

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestValidateGitHash tests the git hash validation function
func TestValidateGitHash(t *testing.T) {
	testCases := []struct {
		name    string
		hash    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid short hash",
			hash:    "abc1",
			wantErr: false,
		},
		{
			name:    "valid long hash",
			hash:    "abc123def456789012345678901234567890abcd",
			wantErr: false,
		},
		{
			name:    "valid mixed case",
			hash:    "ABCdef123",
			wantErr: false,
		},
		{
			name:    "empty hash",
			hash:    "",
			wantErr: true,
			errMsg:  "empty hash not allowed",
		},
		{
			name:    "too short hash",
			hash:    "abc",
			wantErr: true,
			errMsg:  "invalid git hash format",
		},
		{
			name:    "too long hash",
			hash:    "abc123def456789012345678901234567890abcdef0",
			wantErr: true,
			errMsg:  "invalid git hash format",
		},
		{
			name:    "invalid characters",
			hash:    "abc123g",
			wantErr: true,
			errMsg:  "invalid git hash format",
		},
		{
			name:    "special characters",
			hash:    "abc-123",
			wantErr: true,
			errMsg:  "invalid git hash format",
		},
		{
			name:    "command injection attempt",
			hash:    "abc123; rm -rf /",
			wantErr: true,
			errMsg:  "invalid git hash format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGitHash(tc.hash)
			if tc.wantErr {
				if err == nil {
					t.Errorf("validateGitHash(%q) expected error, got nil", tc.hash)
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("validateGitHash(%q) error = %v, want error containing %q",
						tc.hash, err, tc.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateGitHash(%q) unexpected error: %v", tc.hash, err)
				}
			}
		})
	}
}

// TestSanitizeFilePath tests the file path sanitization function
func TestSanitizeFilePath(t *testing.T) {
	testCases := []struct {
		name      string
		path      string
		want      string
		wantErr   bool
		errMsgAny []string // Accept any of these error messages for cross-platform compatibility
	}{
		{
			name: "empty path allowed",
			path: "",
			want: "",
		},
		{
			name: "valid relative path",
			path: "src/main.go",
			want: "src/main.go",
		},
		{
			name: "path with dots but not traversal",
			path: "src/main.go.bak",
			want: "src/main.go.bak",
		},
		{
			name: "path cleaned by filepath.Clean",
			path: "src//main.go",
			want: "src/main.go",
		},
		{
			name:      "directory traversal attack",
			path:      "../etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"path traversal not allowed", "path must be local and relative"},
		},
		{
			name:      "directory traversal in middle",
			path:      "src/../etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"path traversal not allowed", "path must be local and relative"},
		},
		{
			name:      "absolute path",
			path:      "/etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"absolute paths not allowed", "path must be local and relative"},
		},
		{
			name:      "windows absolute path",
			path:      "C:\\Windows\\System32",
			wantErr:   true,
			errMsgAny: []string{"absolute paths not allowed", "path must be local and relative"},
		},
		{
			name:      "windows absolute path with forward slash",
			path:      "C:/Windows/System32",
			wantErr:   true,
			errMsgAny: []string{"absolute paths not allowed", "path must be local and relative"},
		},
		{
			name:      "path becomes absolute after cleaning",
			path:      "/../etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"path traversal not allowed", "path must be local and relative", "path must be relative after normalization"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeFilePath(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("sanitizeFilePath(%q) expected error, got nil", tc.path)
				} else if len(tc.errMsgAny) > 0 {
					// Check if error message contains any of the expected messages
					errStr := err.Error()
					foundMatch := false
					for _, expectedMsg := range tc.errMsgAny {
						if strings.Contains(errStr, expectedMsg) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						t.Errorf("sanitizeFilePath(%q) error = %v, want error containing one of %v",
							tc.path, err, tc.errMsgAny)
					}
				}
			} else {
				if err != nil {
					t.Errorf("sanitizeFilePath(%q) unexpected error: %v", tc.path, err)
				}
				if got != tc.want {
					t.Errorf("sanitizeFilePath(%q) = %q, want %q", tc.path, got, tc.want)
				}
			}
		})
	}
}

// TestSanitizeGitPath tests the git directory path sanitization function  
func TestSanitizeGitPath(t *testing.T) {
	// Base test cases that work on all platforms
	testCases := []struct {
		name      string
		path      string
		want      string
		wantErr   bool
		errMsgAny []string // Accept any of these error messages for cross-platform compatibility
	}{
		{
			name:      "empty path not allowed",
			path:      "",
			wantErr:   true,
			errMsgAny: []string{"empty path not allowed"},
		},
		{
			name: "valid current directory",
			path: ".",
			want: ".",
		},
		{
			name: "valid relative path",
			path: ".git/timemachine_snapshots",
			want: filepath.FromSlash(".git/timemachine_snapshots"),
		},
		{
			name: "valid nested path",
			path: "tmp/project/.git/timemachine_snapshots",
			want: filepath.FromSlash("tmp/project/.git/timemachine_snapshots"),
		},
		{
			name: "path with redundant separators",
			path: "tmp//project//.git//timemachine_snapshots",
			want: filepath.FromSlash("tmp/project/.git/timemachine_snapshots"),
		},
		{
			name: "path with dot components that resolve locally",
			path: "tmp/./project/./.git/timemachine_snapshots",
			want: filepath.FromSlash("tmp/project/.git/timemachine_snapshots"),
		},
		{
			name:      "parent directory traversal",
			path:      "..",
			wantErr:   true,
			errMsgAny: []string{"path must be local and relative"},
		},
		{
			name:      "path traversal attack",
			path:      "../etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"path must be local and relative"},
		},
		{
			name:      "nested path traversal",
			path:      "tmp/../../../etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"path must be local and relative"},
		},
		{
			name:      "unix absolute path",
			path:      "/tmp/project/.git",
			wantErr:   true,
			errMsgAny: []string{"path must be local and relative"},
		},
		{
			name:      "complex attack path",
			path:      "tmp/project/../../../.ssh/id_rsa",
			wantErr:   true,
			errMsgAny: []string{"path must be local and relative"},
		},
	}

	// Add Windows-specific test cases when running on Windows  
	// Note: filepath.IsLocal is platform-aware and only validates for the current OS
	// On Linux, Windows paths like "C:\temp" are treated as valid relative filenames
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			name      string
			path      string
			want      string
			wantErr   bool
			errMsgAny []string
		}{
			{
				name:      "windows absolute path with backslash",
				path:      "C:\\temp\\project\\.git",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "windows absolute path with forward slash",
				path:      "C:/temp/project/.git",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "windows UNC path",
				path:      "\\\\server\\share\\.git",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "windows reserved device name",
				path:      "NUL",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "windows reserved device name lowercase",
				path:      "nul",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "windows COM port",
				path:      "com1",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "windows LPT port",
				path:      "lpt1",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
		}
		testCases = append(testCases, windowsTests...)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeGitPath(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("sanitizeGitPath(%q) expected error, got nil", tc.path)
				} else if len(tc.errMsgAny) > 0 {
					// Check if error message contains any of the expected messages
					errStr := err.Error()
					foundMatch := false
					for _, expectedMsg := range tc.errMsgAny {
						if strings.Contains(errStr, expectedMsg) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						t.Errorf("sanitizeGitPath(%q) error = %v, want error containing one of %v",
							tc.path, err, tc.errMsgAny)
					}
				}
			} else {
				if err != nil {
					t.Errorf("sanitizeGitPath(%q) unexpected error: %v", tc.path, err)
				}
				if got != tc.want {
					t.Errorf("sanitizeGitPath(%q) = %q, want %q", tc.path, got, tc.want)
				}
			}
		})
	}
}

// TestSecurityValidation tests that security validation is properly called
func TestSecurityValidation(t *testing.T) {
	// Test that the security functions are working correctly
	// This ensures defense-in-depth approach is functioning

	// Test hash validation with known bad inputs
	badHashes := []string{
		"",
		"abc",
		"abc123; rm -rf /",
		"../../../etc/passwd",
		"$(rm -rf /)",
		"`rm -rf /`",
		"abc123def456789012345678901234567890abcdef0", // too long
	}

	for _, hash := range badHashes {
		if err := validateGitHash(hash); err == nil {
			t.Errorf("validateGitHash should reject malicious input: %q", hash)
		}
	}

	// Test path validation with known bad inputs
	badPaths := []string{
		"../etc/passwd",                      // Path traversal
		"/etc/passwd",                        // Unix absolute path
		"../../.ssh/id_rsa",                  // Multiple path traversal
		"/home/user/.ssh/id_rsa",             // Unix absolute path
		"C:\\Windows\\System32\\config\\SAM", // Windows absolute path with backslash
		"C:/Windows/System32/config/SAM",     // Windows absolute path with forward slash
		"D:\\data\\sensitive.txt",            // Different drive letter
		"src/../../../etc/passwd",            // Path traversal through valid directory
		"\\\\server\\share\\file.txt",        // UNC path
		"/usr/bin/../../../etc/passwd",       // Complex traversal
	}

	for _, path := range badPaths {
		if _, err := sanitizeFilePath(path); err == nil {
			t.Errorf("sanitizeFilePath should reject malicious input: %q", path)
		}
	}

	// Test git path validation with known bad inputs
	// Platform-agnostic attacks that should be blocked on all systems
	badGitPaths := []string{
		"",                                 // Empty path
		"..",                               // Parent directory
		"../etc/passwd",                    // Path traversal
		"/tmp/.git/timemachine_snapshots",  // Unix absolute path
		"../../.git/timemachine_snapshots", // Multiple path traversal
		"tmp/../../../.ssh/id_rsa",         // Complex traversal through valid directory
	}

	// Add platform-specific attacks
	if runtime.GOOS == "windows" {
		windowsBadPaths := []string{
			"C:\\temp\\.git\\timemachine_snapshots", // Windows absolute path with backslash
			"C:/temp/.git/timemachine_snapshots",    // Windows absolute path with forward slash
			"\\\\server\\share\\.git",               // UNC path
			"NUL",                                   // Windows reserved device name
			"nul",                                   // Windows reserved device name (lowercase)
			"com1",                                  // Windows COM port
			"lpt1",                                  // Windows LPT port
		}
		badGitPaths = append(badGitPaths, windowsBadPaths...)
	}

	for _, path := range badGitPaths {
		if _, err := sanitizeGitPath(path); err == nil {
			t.Errorf("sanitizeGitPath should reject malicious input: %q", path)
		}
	}
}
