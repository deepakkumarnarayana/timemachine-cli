package utils

import (
	"os"
	"runtime"
)

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// MakeExecutable sets executable permissions on Unix systems
// On Windows, it does nothing as executable permissions don't exist
func MakeExecutable(path string) error {
	if IsWindows() {
		// Windows doesn't use executable permissions
		return nil
	}
	return os.Chmod(path, 0700) // #nosec G302 - 0700 is appropriate for executable scripts (owner read/write/execute)
}

// GetHookExtension returns the appropriate file extension for git hooks
func GetHookExtension() string {
	if IsWindows() {
		return ".bat"
	}
	return ""
}

// GetShellCommand returns the appropriate shell command for the platform
func GetShellCommand() string {
	if IsWindows() {
		return "cmd"
	}
	return "sh"
}
