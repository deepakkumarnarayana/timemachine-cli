package security

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestValidateSystemPath tests the system path validation function that handles both absolute and relative paths
func TestValidateSystemPath(t *testing.T) {
	// Platform-appropriate base test cases
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
			name: "path with redundant separators",
			path: "tmp//project//.git//timemachine_snapshots",
			want: filepath.FromSlash("tmp/project/.git/timemachine_snapshots"),
		},
		{
			name:      "relative path traversal attack",
			path:      "../etc/passwd",
			wantErr:   true,
			errMsgAny: []string{"path must be local and relative"},
		},
	}

	// Platform-conditional absolute path tests
	if runtime.GOOS == "windows" {
		// On Windows, Unix absolute paths are rejected by filepath.IsLocal()
		windowsAbsTests := []struct {
			name      string
			path      string
			want      string
			wantErr   bool
			errMsgAny []string
		}{
			{
				name:      "unix absolute path (rejected on Windows)",
				path:      "/tmp/project/.git/timemachine_snapshots",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
			{
				name:      "unix absolute path traversal (rejected on Windows)",
				path:      "/tmp/project/../../../etc/passwd",
				wantErr:   true,
				errMsgAny: []string{"path must be local and relative"},
			},
		}
		testCases = append(testCases, windowsAbsTests...)
	} else {
		// On Unix, absolute paths are allowed for system operations
		unixAbsTests := []struct {
			name      string
			path      string
			want      string
			wantErr   bool
			errMsgAny []string
		}{
			{
				name: "valid absolute path",
				path: "/tmp/project/.git/timemachine_snapshots",
				want: "/tmp/project/.git/timemachine_snapshots",
			},
			{
				name:      "absolute path traversal attack",
				path:      "/tmp/project/../../../etc/passwd",
				wantErr:   true,
				errMsgAny: []string{"path traversal not allowed in absolute path"},
			},
		}
		testCases = append(testCases, unixAbsTests...)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateSystemPath(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateSystemPath(%q) expected error, got nil", tc.path)
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
						t.Errorf("ValidateSystemPath(%q) error = %v, want error containing one of %v",
							tc.path, err, tc.errMsgAny)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSystemPath(%q) unexpected error: %v", tc.path, err)
				}
				if got != tc.want {
					t.Errorf("ValidateSystemPath(%q) = %q, want %q", tc.path, got, tc.want)
				}
			}
		})
	}
}

// TestValidateUserInputPath tests the strict user input validation function  
func TestValidateUserInputPath(t *testing.T) {
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
			name: "current directory allowed",
			path: ".",
			want: ".",
		},
		{
			name: "valid relative path",
			path: ".git/timemachine_snapshots",
			want: filepath.FromSlash(".git/timemachine_snapshots"),
		},
		{
			name: "path with redundant separators",
			path: "tmp//project//.git//timemachine_snapshots",
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
		}
		testCases = append(testCases, windowsTests...)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateUserInputPath(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateUserInputPath(%q) expected error, got nil", tc.path)
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
						t.Errorf("ValidateUserInputPath(%q) error = %v, want error containing one of %v",
							tc.path, err, tc.errMsgAny)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUserInputPath(%q) unexpected error: %v", tc.path, err)
				}
				if got != tc.want {
					t.Errorf("ValidateUserInputPath(%q) = %q, want %q", tc.path, got, tc.want)
				}
			}
		})
	}
}