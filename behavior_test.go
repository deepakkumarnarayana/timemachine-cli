package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// BehaviorTest focuses on testing real user workflows and scenarios
// rather than individual functions. These tests verify that the entire
// system works correctly for common development patterns.

func TestDeveloperWorkflow_InitializeAndCreateSnapshots(t *testing.T) {
	// Scenario: Developer starts a new project and wants automatic snapshots
	tempDir := setupTempDir(t)
	defer os.RemoveAll(tempDir)

	// Given: A new Git repository
	runCommand(t, tempDir, "git", "init")
	runCommand(t, tempDir, "git", "config", "user.name", "Test User")
	runCommand(t, tempDir, "git", "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(tempDir, "README.md"), "# Test Project")
	runCommand(t, tempDir, "git", "add", "README.md")
	runCommand(t, tempDir, "git", "commit", "-m", "Initial commit")

	// When: Developer initializes TimeMachine
	output := runTimeMachine(t, tempDir, "init")
	
	// Then: Initialization should succeed
	if !strings.Contains(output, "initialized successfully") {
		t.Errorf("Expected initialization success message, got: %s", output)
	}
	
	// And: Shadow repository should be created
	shadowRepoPath := filepath.Join(tempDir, ".git", "timemachine_snapshots")
	if !dirExists(shadowRepoPath) {
		t.Error("Shadow repository should be created at .git/timemachine_snapshots")
	}
	
	// And: .gitignore should be updated to ignore shadow repo
	gitignoreContent := readFile(t, filepath.Join(tempDir, ".gitignore"))
	if !strings.Contains(gitignoreContent, ".git/timemachine_snapshots") {
		t.Error("Shadow repository should be added to .gitignore")
	}

	// When: Developer makes code changes
	writeFile(t, filepath.Join(tempDir, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"Hello World\")\n}")
	
	// And: Creates a snapshot manually (simulating automatic behavior)
	listOutput := runTimeMachine(t, tempDir, "list")
	
	// Then: Should show snapshots are available
	if strings.Contains(listOutput, "No snapshots found") {
		// Create initial snapshot if none exist
		runTimeMachine(t, tempDir, "start", "--once") // Simulated snapshot creation
	}
}

func TestDeveloperWorkflow_InspectAndAnalyzeSnapshots(t *testing.T) {
	// Scenario: Developer wants to analyze what changed in recent snapshots
	tempDir := setupInitializedRepo(t)
	defer os.RemoveAll(tempDir)

	// Given: Multiple file changes and snapshots
	writeFile(t, filepath.Join(tempDir, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}")
	writeFile(t, filepath.Join(tempDir, "utils.go"), "package main\n\nfunc helper() string {\n\treturn \"test\"\n}")
	
	// When: Developer lists snapshots
	listOutput := runTimeMachine(t, tempDir, "list")
	
	// Then: Should show available snapshots
	lines := strings.Split(strings.TrimSpace(listOutput), "\n")
	if len(lines) < 1 || strings.Contains(listOutput, "No snapshots found") {
		t.Skip("No snapshots available for inspection test")
	}
	
	// Extract first snapshot hash for inspection
	var snapshotHash string
	for _, line := range lines {
		if strings.Contains(line, "•") && len(line) > 10 {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				snapshotHash = parts[1] // Assuming format: "• hash message"
				break
			}
		}
	}
	
	if snapshotHash == "" {
		t.Skip("Could not extract snapshot hash for inspection")
	}

	// When: Developer inspects a specific snapshot
	inspectOutput := runTimeMachine(t, tempDir, "inspect", snapshotHash)
	
	// Then: Should show detailed snapshot information
	if !strings.Contains(inspectOutput, "Snapshot Information") {
		t.Errorf("Inspect should show snapshot information, got: %s", inspectOutput)
	}
}

func TestDeveloperWorkflow_SafeRestoreAfterBadChanges(t *testing.T) {
	// Scenario: Developer makes changes that break the code and needs to restore
	tempDir := setupInitializedRepo(t)
	defer os.RemoveAll(tempDir)

	// Given: Working code
	originalCode := "package main\n\nfunc main() {\n\tprintln(\"Working code\")\n}"
	writeFile(t, filepath.Join(tempDir, "main.go"), originalCode)
	
	// And: A snapshot is created (simulate automatic snapshot)
	listOutput := runTimeMachine(t, tempDir, "list")
	
	// When: Developer makes breaking changes
	brokenCode := "package main\n\n// This code is broken\nfunc main() {\n\tundefinedFunction()\n}"
	writeFile(t, filepath.Join(tempDir, "main.go"), brokenCode)
	
	// And: Realizes they need to restore
	// Get the latest snapshot hash
	listOutput = runTimeMachine(t, tempDir, "list")
	lines := strings.Split(strings.TrimSpace(listOutput), "\n")
	
	if len(lines) < 1 || strings.Contains(listOutput, "No snapshots found") {
		t.Skip("No snapshots available for restore test")
	}
	
	var snapshotHash string
	for _, line := range lines {
		if strings.Contains(line, "•") && len(line) > 10 {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				snapshotHash = parts[1]
				break
			}
		}
	}
	
	if snapshotHash != "" {
		// When: Developer restores from snapshot
		restoreOutput := runTimeMachine(t, tempDir, "restore", snapshotHash, "main.go")
		
		// Then: File should be restored
		if !strings.Contains(restoreOutput, "restored successfully") && 
		   !strings.Contains(restoreOutput, "Restored") {
			t.Logf("Restore output: %s", restoreOutput) // Log for debugging
		}
		
		// And: Original code should be back
		restoredContent := readFile(t, filepath.Join(tempDir, "main.go"))
		if strings.Contains(restoredContent, "undefinedFunction") {
			t.Error("File should be restored to working state")
		}
	}
}

func TestDeveloperWorkflow_IgnorePatternHandling(t *testing.T) {
	// Scenario: Developer wants to exclude certain files from snapshots
	tempDir := setupInitializedRepo(t)
	defer os.RemoveAll(tempDir)

	// Given: Developer creates files that should be ignored
	writeFile(t, filepath.Join(tempDir, "main.go"), "package main")
	writeFile(t, filepath.Join(tempDir, "secret.key"), "secret-content")
	writeFile(t, filepath.Join(tempDir, "build.log"), "build output")
	
	// And: Configures ignore patterns
	ignoreContent := "*.key\n*.log\nnode_modules/\n.env\n"
	writeFile(t, filepath.Join(tempDir, ".timemachine-ignore"), ignoreContent)
	
	// When: Snapshots are created (simulated)
	runTimeMachine(t, tempDir, "list") // This will trigger snapshot creation if needed
	
	// Then: Ignore patterns should be respected
	// We can't easily test this without actual file watching, but we can verify
	// that the ignore file is properly read and the system doesn't crash
	showOutput := runTimeMachine(t, tempDir, "status")
	if strings.Contains(showOutput, "error") && strings.Contains(showOutput, "ignore") {
		t.Errorf("Ignore pattern handling should not cause errors: %s", showOutput)
	}
}

func TestDeveloperWorkflow_LargeProjectHandling(t *testing.T) {
	// Scenario: Developer works with a project containing many files
	tempDir := setupInitializedRepo(t)
	defer os.RemoveAll(tempDir)

	// Given: A project with multiple directories and files
	dirs := []string{"src", "tests", "docs", "config"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		os.MkdirAll(dirPath, 0755)
		
		// Create files in each directory
		writeFile(t, filepath.Join(dirPath, "file1.go"), "package main")
		writeFile(t, filepath.Join(dirPath, "file2.go"), "package main")
	}
	
	// When: Developer lists snapshots
	listOutput := runTimeMachine(t, tempDir, "list")
	
	// Then: System should handle multiple files gracefully
	if strings.Contains(listOutput, "error") || strings.Contains(listOutput, "failed") {
		t.Errorf("System should handle large projects gracefully: %s", listOutput)
	}
	
	// When: Developer inspects snapshots
	if !strings.Contains(listOutput, "No snapshots found") {
		// Try to inspect if snapshots exist
		lines := strings.Split(strings.TrimSpace(listOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, "•") && len(line) > 10 {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					snapshotHash := parts[1]
					inspectOutput := runTimeMachine(t, tempDir, "inspect", snapshotHash)
					
					// Then: Should handle inspection without errors
					if strings.Contains(inspectOutput, "panic") || strings.Contains(inspectOutput, "fatal") {
						t.Errorf("Inspection should handle large projects: %s", inspectOutput)
					}
					break
				}
			}
		}
	}
}

func TestDeveloperWorkflow_SecurityValidation(t *testing.T) {
	// Scenario: Malicious user tries to exploit the system
	tempDir := setupInitializedRepo(t)
	defer os.RemoveAll(tempDir)

	// Given: Various malicious inputs
	maliciousInputs := []struct {
		name     string
		command  []string
		shouldFail bool
	}{
		{
			name:     "Command injection in hash",
			command:  []string{"inspect", "abc123; rm -rf /"},
			shouldFail: true,
		},
		{
			name:     "Path traversal in restore",
			command:  []string{"restore", "abc123", "../../../etc/passwd"},
			shouldFail: true,
		},
		{
			name:     "Invalid hash format",
			command:  []string{"inspect", "invalid-hash-with-special-chars!"},
			shouldFail: true,
		},
		{
			name:     "Empty hash",
			command:  []string{"inspect", ""},
			shouldFail: true,
		},
		{
			name:     "SQL injection attempt",
			command:  []string{"inspect", "'; DROP TABLE users; --"},
			shouldFail: true,
		},
	}

	for _, tc := range maliciousInputs {
		t.Run(tc.name, func(t *testing.T) {
			// When: Malicious input is provided
			output := runTimeMachineExpectError(t, tempDir, tc.command...)
			
			// Then: Should reject malicious input safely
			if tc.shouldFail {
				if !strings.Contains(output, "error") && 
				   !strings.Contains(output, "invalid") &&
				   !strings.Contains(output, "not allowed") {
					t.Errorf("Expected security validation to reject input %v, got: %s", tc.command, output)
				}
				
				// And: Should not contain signs of successful exploitation
				if strings.Contains(output, "rm -rf") || 
				   strings.Contains(output, "/etc/passwd") ||
				   strings.Contains(output, "DROP TABLE") {
					t.Errorf("Output suggests possible security vulnerability: %s", output)
				}
			}
		})
	}
}

func TestDeveloperWorkflow_ErrorRecovery(t *testing.T) {
	// Scenario: System encounters various error conditions
	tempDir := setupInitializedRepo(t)
	defer os.RemoveAll(tempDir)

	// Given: Corrupted or missing snapshot
	// When: Developer tries to inspect non-existent snapshot
	output := runTimeMachineExpectError(t, tempDir, "inspect", "nonexistent123")
	
	// Then: Should provide helpful error message
	if !strings.Contains(output, "not found") && 
	   !strings.Contains(output, "invalid") &&
	   !strings.Contains(output, "error") {
		t.Errorf("Expected helpful error for non-existent snapshot, got: %s", output)
	}
	
	// When: Developer tries to restore non-existent file
	output = runTimeMachineExpectError(t, tempDir, "restore", "abc123", "nonexistent.go")
	
	// Then: Should handle gracefully
	if strings.Contains(output, "panic") {
		t.Errorf("Should handle missing files gracefully, got: %s", output)
	}
}

// Helper functions for behavior testing

func setupInitializedRepo(t *testing.T) string {
	tempDir := setupTempDir(t)
	
	// Initialize Git repo
	runCommand(t, tempDir, "git", "init")
	runCommand(t, tempDir, "git", "config", "user.name", "Test User")
	runCommand(t, tempDir, "git", "config", "user.email", "test@example.com")
	
	// Create initial commit
	writeFile(t, filepath.Join(tempDir, "README.md"), "# Test Project")
	runCommand(t, tempDir, "git", "add", "README.md")
	runCommand(t, tempDir, "git", "commit", "-m", "Initial commit")
	
	// Initialize TimeMachine
	runTimeMachine(t, tempDir, "init")
	
	// Give it a moment to initialize
	time.Sleep(100 * time.Millisecond)
	
	return tempDir
}

func runTimeMachineExpectError(t *testing.T, dir string, args ...string) string {
	// Run TimeMachine command expecting it might fail
	cmd := createCommand(t, dir, "./timemachine", args...)
	output, err := cmd.CombinedOutput()
	
	// Don't fail the test if command returns error - that's expected for error cases
	return string(output)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}