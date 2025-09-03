package commands

import (
	"strings"
	"testing"
)

// TestValidateGitHash tests the git hash validation function
func TestValidateGitHash(t *testing.T) {
	testCases := []struct {
		name     string
		hash     string
		wantErr  bool
		errMsg   string
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
		name     string
		path     string
		want     string
		wantErr  bool
		errMsg   string
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
			name:    "directory traversal attack",
			path:    "../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal not allowed",
		},
		{
			name:    "directory traversal in middle",
			path:    "src/../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal not allowed",
		},
		{
			name:    "absolute path",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "absolute paths not allowed",
		},
		{
			name:    "windows absolute path",
			path:    "C:\\Windows\\System32",
			wantErr: true,
			errMsg:  "absolute paths not allowed",
		},
		{
			name:    "windows absolute path with forward slash",
			path:    "C:/Windows/System32",
			wantErr: true,
			errMsg:  "absolute paths not allowed",
		},
		{
			name:    "path becomes absolute after cleaning",
			path:    "/../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal not allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeFilePath(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("sanitizeFilePath(%q) expected error, got nil", tc.path)
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("sanitizeFilePath(%q) error = %v, want error containing %q", 
						tc.path, err, tc.errMsg)
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
		"../etc/passwd",                        // Path traversal
		"/etc/passwd",                          // Unix absolute path
		"../../.ssh/id_rsa",                    // Multiple path traversal
		"/home/user/.ssh/id_rsa",               // Unix absolute path
		"C:\\Windows\\System32\\config\\SAM",   // Windows absolute path with backslash
		"C:/Windows/System32/config/SAM",       // Windows absolute path with forward slash
		"D:\\data\\sensitive.txt",              // Different drive letter
		"src/../../../etc/passwd",              // Path traversal through valid directory
		"\\\\server\\share\\file.txt",          // UNC path
		"/usr/bin/../../../etc/passwd",         // Complex traversal
	}

	for _, path := range badPaths {
		if _, err := sanitizeFilePath(path); err == nil {
			t.Errorf("sanitizeFilePath should reject malicious input: %q", path)
		}
	}
}