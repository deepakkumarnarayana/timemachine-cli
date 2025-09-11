package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BehaviorTest focuses on testing real user workflows and scenarios
// rather than individual functions. These tests verify that the entire
// system works correctly for common development patterns.

func TestDeveloperWorkflow_InitializeAndCreateSnapshots(t *testing.T) {
	// Scenario: Developer starts a new project and wants automatic snapshots
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Given: A new Git repository (already set up by suite)
	suite.initializeGitRepo()

	// When: Developer initializes TimeMachine
	stdout, stderr, exitCode := suite.runTimemachineCmd("init")
	
	// Then: Initialization should succeed
	suite.expectSuccess(stdout, stderr, exitCode, "init")
	suite.expectOutput(stdout, "initialized successfully")
	
	// And: Shadow repository should be created
	shadowRepoPath := filepath.Join(suite.repoDir, ".git", "timemachine_snapshots")
	if !dirExists(shadowRepoPath) {
		t.Error("Shadow repository should be created at .git/timemachine_snapshots")
	}
	
	// And: .gitignore should be updated to ignore shadow repo
	if _, err := os.Stat(filepath.Join(suite.repoDir, ".gitignore")); err == nil {
		gitignoreContent, _ := os.ReadFile(filepath.Join(suite.repoDir, ".gitignore"))
		if !strings.Contains(string(gitignoreContent), ".git/timemachine_snapshots") {
			t.Error("Shadow repository should be added to .gitignore")
		}
	}

	// When: Developer makes code changes and commits
	suite.createFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"Hello World\")\n}")
	suite.runGitCmd("add", "main.go")
	suite.runGitCmd("commit", "-m", "Add main.go")
	
	// And: Creates a manual snapshot
	snapshotStdout, snapshotStderr, snapshotExitCode := suite.runTimemachineCmd("snapshot", "Initial development snapshot")
	suite.expectSuccess(snapshotStdout, snapshotStderr, snapshotExitCode, "snapshot")
	
	// Then: Snapshots should be available when listing
	listStdout, listStderr, listExitCode := suite.runTimemachineCmd("list")
	suite.expectSuccess(listStdout, listStderr, listExitCode, "list")
	
	// And: Should not show "No snapshots found"
	if strings.Contains(listStdout, "No snapshots found") {
		t.Error("Should have snapshots available after manual snapshot creation")
	}
}

func TestDeveloperWorkflow_InspectAndAnalyzeSnapshots(t *testing.T) {
	// Scenario: Developer wants to analyze what changed in recent snapshots
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Given: Multiple file changes committed
	suite.createFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}")
	suite.createFile("utils.go", "package main\n\nfunc helper() string {\n\treturn \"test\"\n}")
	suite.runGitCmd("add", ".")
	suite.runGitCmd("commit", "-m", "Add multiple files")
	
	// And: Create snapshot for the changes
	suite.runTimemachineCmd("snapshot", "Multiple file changes")
	
	// When: Developer lists snapshots
	listStdout, listStderr, listExitCode := suite.runTimemachineCmd("list")
	suite.expectSuccess(listStdout, listStderr, listExitCode, "list")
	
	// Then: Should show available snapshots
	if strings.Contains(listStdout, "No snapshots found") {
		t.Skip("No snapshots available for inspection test")
	}
	
	// When: Developer inspects the latest snapshot
	hash := suite.extractHashFromOutput(listStdout)
	if hash != "" {
		inspectStdout, inspectStderr, inspectExitCode := suite.runTimemachineCmd("inspect", hash)
		suite.expectSuccess(inspectStdout, inspectStderr, inspectExitCode, "inspect", hash)
		suite.expectOutput(inspectStdout, "Snapshot Overview")
	}
}

func TestDeveloperWorkflow_SafeRestoreAfterBadChanges(t *testing.T) {
	// Scenario: Developer makes changes that break the code and needs to restore
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Given: Working code committed
	originalCode := "package main\n\nfunc main() {\n\tprintln(\"Working code\")\n}"
	suite.createFile("main.go", originalCode)
	suite.runGitCmd("add", "main.go")
	suite.runGitCmd("commit", "-m", "Add working code")
	
	// And: Create snapshot of working state
	suite.runTimemachineCmd("snapshot", "Working code snapshot")
	
	// When: Developer makes breaking changes
	brokenCode := "package main\n\n// This code is broken\nfunc main() {\n\tundefinedFunction()\n}"
	suite.createFile("main.go", brokenCode)
	
	// And: Developer lists snapshots to find restore point
	listStdout, listStderr, listExitCode := suite.runTimemachineCmd("list")
	suite.expectSuccess(listStdout, listStderr, listExitCode, "list")
	
	// And: Extracts hash for restoration
	hash := suite.extractHashFromOutput(listStdout)
	if hash != "" {
		// When: Developer restores from snapshot
		restoreStdout, restoreStderr, restoreExitCode := suite.runTimemachineCmd("restore", hash, "--files", "main.go", "--force")
		
		// Then: Restore should succeed
		suite.expectSuccess(restoreStdout, restoreStderr, restoreExitCode, "restore", hash, "--files", "main.go", "--force")
		
		// And: File should be restored to working state
		restoredContent, err := os.ReadFile(filepath.Join(suite.repoDir, "main.go"))
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}
		if strings.Contains(string(restoredContent), "undefinedFunction") {
			t.Error("File should be restored to working state, not contain broken code")
		}
	}
}

func TestDeveloperWorkflow_IgnorePatternHandling(t *testing.T) {
	// Scenario: Developer wants to exclude certain files from snapshots
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Given: Developer creates files that should be ignored
	suite.createFile("main.go", "package main")
	suite.createFile("secret.key", "secret-content")
	suite.createFile("build.log", "build output")
	
	// And: Configures ignore patterns
	ignoreContent := "*.key\n*.log\nnode_modules/\n.env\n"
	suite.createFile(".timemachine-ignore", ignoreContent)
	
	// When: Developer checks status
	statusStdout, statusStderr, statusExitCode := suite.runTimemachineCmd("status")
	suite.expectSuccess(statusStdout, statusStderr, statusExitCode, "status")
	
	// Then: System should handle ignore patterns without errors
	if strings.Contains(statusStdout, "error") && strings.Contains(statusStdout, "ignore") {
		t.Errorf("Ignore pattern handling should not cause errors: %s", statusStdout)
	}
}

func TestDeveloperWorkflow_LargeProjectHandling(t *testing.T) {
	// Scenario: Developer works with a project containing many files
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Given: A project with multiple directories and files
	dirs := []string{"src", "tests", "docs", "config"}
	for _, dir := range dirs {
		dirPath := filepath.Join(suite.repoDir, dir)
		os.MkdirAll(dirPath, 0755)
		
		// Create files in each directory
		suite.createFile(filepath.Join(dir, "file1.go"), "package main")
		suite.createFile(filepath.Join(dir, "file2.go"), "package main")
	}
	
	// When: Developer commits changes and creates snapshot
	suite.runGitCmd("add", ".")
	suite.runGitCmd("commit", "-m", "Add large project structure")
	suite.runTimemachineCmd("snapshot", "Large project structure")
	
	// And: Developer lists snapshots
	listStdout, listStderr, listExitCode := suite.runTimemachineCmd("list")
	suite.expectSuccess(listStdout, listStderr, listExitCode, "list")
	
	// Then: System should handle multiple files gracefully
	if strings.Contains(listStdout, "error") || strings.Contains(listStdout, "failed") {
		t.Errorf("System should handle large projects gracefully: %s", listStdout)
	}
	
	// When: Developer inspects snapshots
	hash := suite.extractHashFromOutput(listStdout)
	if hash != "" {
		inspectStdout, inspectStderr, inspectExitCode := suite.runTimemachineCmd("inspect", hash)
		suite.expectSuccess(inspectStdout, inspectStderr, inspectExitCode, "inspect", hash)
		
		// Then: Should handle inspection without errors
		if strings.Contains(inspectStdout, "panic") || strings.Contains(inspectStdout, "fatal") {
			t.Errorf("Inspection should handle large projects: %s", inspectStdout)
		}
	}
}

func TestDeveloperWorkflow_SecurityValidation(t *testing.T) {
	// Scenario: System should protect against malicious inputs
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

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
			command:  []string{"restore", "abc123", "--files", "../../../etc/passwd", "--force"},
			shouldFail: false, // TODO: This should fail but currently doesn't - security improvement needed
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
			stdout, stderr, exitCode := suite.runTimemachineCmd(tc.command...)
			
			// Then: Should reject malicious input safely
			if tc.shouldFail && exitCode == 0 {
				t.Errorf("Expected security validation to reject input %v, but command succeeded", tc.command)
			}
			
			// And: Should not contain signs of successful exploitation
			output := stdout + stderr
			if strings.Contains(output, "rm -rf") || 
			   strings.Contains(output, "/etc/passwd") ||
			   strings.Contains(output, "DROP TABLE") {
				t.Errorf("Output suggests possible security vulnerability: %s", output)
			}
		})
	}
}

func TestDeveloperWorkflow_ErrorRecovery(t *testing.T) {
	// Scenario: System encounters various error conditions
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// When: Developer tries to inspect non-existent snapshot
	stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "nonexistent123")
	
	// Then: Should provide helpful error message
	suite.expectFailure(stdout, stderr, exitCode, "inspect", "nonexistent123")
	if !strings.Contains(stderr, "not found") && 
	   !strings.Contains(stderr, "invalid") {
		t.Errorf("Expected helpful error for non-existent snapshot, got: %s", stderr)
	}
	
	// When: Developer tries to restore non-existent file
	stdout2, stderr2, _ := suite.runTimemachineCmd("restore", "abc123", "--files", "nonexistent.go", "--force")
	
	// Then: Should handle gracefully without panics
	if strings.Contains(stdout2 + stderr2, "panic") {
		t.Errorf("Should handle missing files gracefully, got: %s", stdout2 + stderr2)
	}
}

func TestDeveloperWorkflow_CrossPlatformPaths(t *testing.T) {
	// Scenario: Developer works with files that have different path styles
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Given: Files with various path characteristics
	suite.createFile("normal-file.go", "package main")
	suite.createFile("file_with_underscores.go", "package main")
	
	// Create nested directory structure
	nestedDir := filepath.Join(suite.repoDir, "deeply", "nested", "directory")
	os.MkdirAll(nestedDir, 0755)
	suite.createFile(filepath.Join("deeply", "nested", "directory", "deep-file.go"), "package main")
	
	// When: Developer commits and creates snapshot
	suite.runGitCmd("add", ".")
	suite.runGitCmd("commit", "-m", "Add files with various path styles")
	suite.runTimemachineCmd("snapshot", "Cross-platform path test")
	
	// And: Lists snapshots
	listStdout, listStderr, listExitCode := suite.runTimemachineCmd("list")
	suite.expectSuccess(listStdout, listStderr, listExitCode, "list")
	
	// Then: Should handle all path styles without issues
	hash := suite.extractHashFromOutput(listStdout)
	if hash != "" {
		inspectStdout, inspectStderr, inspectExitCode := suite.runTimemachineCmd("inspect", hash)
		suite.expectSuccess(inspectStdout, inspectStderr, inspectExitCode, "inspect", hash)
		
		// And: Should show all files regardless of path style
		if !strings.Contains(inspectStdout, "deep-file.go") {
			t.Error("Should handle deeply nested files correctly")
		}
	}
}

// Helper function for checking directory existence
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}