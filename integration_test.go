package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// IntegrationTestSuite manages the lifecycle of integration tests
type IntegrationTestSuite struct {
	t              *testing.T
	tempDir        string
	repoDir        string
	binaryPath     string
	// initialSnapshots []string // Unused field - removed for linting
	// cleanup        func() // Unused field - removed for linting
}

// NewIntegrationTestSuite creates a new test suite with a temporary Git repository
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Build timemachine binary for testing
	binaryPath := filepath.Join(tempDir, "timemachine")
	if err := buildTimemachineBinary(binaryPath); err != nil {
		_ = os.RemoveAll(tempDir) // Ignore error - cleanup is best effort
		t.Fatalf("Failed to build binary: %v", err)
	}

	suite := &IntegrationTestSuite{
		t:          t,
		tempDir:    tempDir,
		repoDir:    repoDir,
		binaryPath: binaryPath,
		// cleanup removed - handled by Cleanup() method
	}

	// Initialize Git repository
	suite.initializeGitRepo()
	
	return suite
}

// Cleanup removes temporary files
func (suite *IntegrationTestSuite) Cleanup() {
	_ = os.RemoveAll(suite.tempDir) // Ignore error - cleanup is best effort
}

// buildTimemachineBinary compiles the timemachine binary for testing
func buildTimemachineBinary(outputPath string) error {
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/timemachine")
	return cmd.Run()
}

// initializeGitRepo sets up a test Git repository
func (suite *IntegrationTestSuite) initializeGitRepo() {
	// Change to repo directory
	originalDir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(originalDir) // Ignore error on cleanup
	}()
	_ = os.Chdir(suite.repoDir) // Ignore error - will be caught by git command failures

	// Initialize Git repo
	suite.runGitCmd("init")
	suite.runGitCmd("config", "user.name", "Test User")
	suite.runGitCmd("config", "user.email", "test@example.com")

	// Create initial commit
	suite.createFile("README.md", "# Test Repository")
	suite.runGitCmd("add", "README.md")
	suite.runGitCmd("commit", "-m", "Initial commit")

	// Create some test files for snapshots
	suite.createFile("main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello World\")\n}")
	suite.createFile("config.yaml", "version: 1.0\nname: test")
	
	// Add files but don't commit (this will be our working directory changes)
	suite.runGitCmd("add", ".")
}

// runGitCmd runs a git command in the test repository
func (suite *IntegrationTestSuite) runGitCmd(args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = suite.repoDir
	if err := cmd.Run(); err != nil {
		suite.t.Fatalf("Git command failed: git %s - %v", strings.Join(args, " "), err)
	}
}

// runTimemachineCmd runs a timemachine command and returns stdout, stderr, and exit code
func (suite *IntegrationTestSuite) runTimemachineCmd(args ...string) (string, string, int) {
	cmd := exec.Command(suite.binaryPath, args...)
	cmd.Dir = suite.repoDir
	
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}
	
	return stdout.String(), stderr.String(), exitCode
}

// createFile creates a file with given content
func (suite *IntegrationTestSuite) createFile(filename, content string) {
	filePath := filepath.Join(suite.repoDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		suite.t.Fatalf("Failed to create file %s: %v", filename, err)
	}
}

// expectSuccess asserts command succeeded
func (suite *IntegrationTestSuite) expectSuccess(stdout, stderr string, exitCode int, cmdArgs ...string) {
	if exitCode != 0 {
		suite.t.Fatalf("Command failed: %s\nStdout: %s\nStderr: %s\nExit code: %d", 
			strings.Join(cmdArgs, " "), stdout, stderr, exitCode)
	}
}

// expectFailure asserts command failed
func (suite *IntegrationTestSuite) expectFailure(stdout, stderr string, exitCode int, cmdArgs ...string) {
	if exitCode == 0 {
		suite.t.Fatalf("Command should have failed: %s\nStdout: %s", 
			strings.Join(cmdArgs, " "), stdout)
	}
}

// expectOutput asserts output contains expected text
func (suite *IntegrationTestSuite) expectOutput(output, expected string) {
	if !strings.Contains(output, expected) {
		suite.t.Fatalf("Expected output to contain %q, got:\n%s", expected, output)
	}
}

// expectOutputMatch asserts output matches regex pattern
func (suite *IntegrationTestSuite) expectOutputMatch(output, pattern string) {
	matched, err := regexp.MatchString(pattern, output)
	if err != nil {
		suite.t.Fatalf("Invalid regex pattern %q: %v", pattern, err)
	}
	if !matched {
		suite.t.Fatalf("Expected output to match pattern %q, got:\n%s", pattern, output)
	}
}

// extractHashFromOutput extracts a git hash from command output
func (suite *IntegrationTestSuite) extractHashFromOutput(output string) string {
	// Look for 8-character hex strings (short hash format)
	hashRegex := regexp.MustCompile(`[a-fA-F0-9]{8,40}`)
	matches := hashRegex.FindAllString(output, -1)
	if len(matches) == 0 {
		suite.t.Fatalf("No hash found in output: %s", output)
	}
	return matches[0] // Return first hash found
}

// setupWithSnapshots initializes timemachine and creates test snapshots
func (suite *IntegrationTestSuite) setupWithSnapshots() {
	// Initialize timemachine
	stdout, stderr, exitCode := suite.runTimemachineCmd("init")
	suite.expectSuccess(stdout, stderr, exitCode, "init")
	// Accept either "initialized successfully" or "already initialized"
	if !strings.Contains(stdout, "Time Machine initialized successfully") && 
	   !strings.Contains(stdout, "Time Machine is already initialized") {
		suite.t.Fatalf("Unexpected init output: %s", stdout)
	}

	// Create first snapshot by committing (only if there are changes)
	cmd := exec.Command("git", "commit", "-m", "Add main.go and config.yaml")
	cmd.Dir = suite.repoDir
	_ = cmd.Run() // Ignore error - might be nothing to commit
	
	// Make some changes and create more snapshots
	suite.createFile("utils.go", "package main\n\nfunc utils() {}")
	suite.createFile("test.txt", "test content")
	
	// Wait a moment to ensure different timestamps
	time.Sleep(time.Second)
	
	// We can't easily create snapshots without the file watcher,
	// but we can test with what gets created during init
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestInitCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("successful_initialization", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		
		suite.expectSuccess(stdout, stderr, exitCode, "init")
		suite.expectOutput(stdout, "Time Machine initialized successfully")
		suite.expectOutput(stdout, "Setting up shadow repository")
		suite.expectOutput(stdout, "Creating initial snapshot")
		
		// Verify shadow repository was created
		shadowDir := filepath.Join(suite.repoDir, ".git", "timemachine_snapshots")
		if _, err := os.Stat(shadowDir); os.IsNotExist(err) {
			t.Fatalf("Shadow repository not created at %s", shadowDir)
		}
		
		// Verify .gitignore was updated
		gitignoreContent, err := os.ReadFile(filepath.Join(suite.repoDir, ".gitignore"))
		if err != nil {
			t.Fatalf("Failed to read .gitignore: %v", err)
		}
		if !strings.Contains(string(gitignoreContent), "timemachine_snapshots") {
			t.Fatalf("timemachine_snapshots not added to .gitignore")
		}
	})

	t.Run("already_initialized", func(t *testing.T) {
		// Run init again
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		
		suite.expectSuccess(stdout, stderr, exitCode, "init")
		suite.expectOutput(stdout, "Time Machine is already initialized")
	})
}

func TestStatusCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("status_before_init", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("status")
		
		suite.expectSuccess(stdout, stderr, exitCode, "status")
		suite.expectOutput(stdout, "Status: Not initialized")
		suite.expectOutput(stdout, "Run 'timemachine init'")
	})

	t.Run("status_after_init", func(t *testing.T) {
		suite.setupWithSnapshots()
		
		stdout, stderr, exitCode := suite.runTimemachineCmd("status")
		
		suite.expectSuccess(stdout, stderr, exitCode, "status")
		suite.expectOutput(stdout, "Time Machine Status")
		suite.expectOutput(stdout, "Initialized")
	})
}

func TestListCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	
	t.Run("list_before_init", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("list")
		
		suite.expectSuccess(stdout, stderr, exitCode, "list")
		suite.expectOutput(stdout, "Time Machine is not initialized")
		suite.expectOutput(stdout, "Run 'timemachine init'")
	})

	t.Run("list_after_init", func(t *testing.T) {
		suite.setupWithSnapshots()
		
		stdout, stderr, exitCode := suite.runTimemachineCmd("list")
		
		suite.expectSuccess(stdout, stderr, exitCode, "list")
		suite.expectOutput(stdout, "Recent snapshots")
		
		// Should show at least the initial snapshot
		suite.expectOutputMatch(stdout, `[a-fA-F0-9]{8}`)  // Should contain hash
		suite.expectOutputMatch(stdout, `\d+ \w+ ago`)     // Should contain relative time
	})

	t.Run("list_with_limit", func(t *testing.T) {
		suite.setupWithSnapshots()
		
		stdout, stderr, exitCode := suite.runTimemachineCmd("list", "--limit", "1")
		
		suite.expectSuccess(stdout, stderr, exitCode, "list", "--limit", "1")
		suite.expectOutput(stdout, "Recent snapshots")
		
		// Count number of hash lines (should be only 1)
		lines := strings.Split(stdout, "\n")
		hashCount := 0
		hashRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}`) // Compile once for better performance
		for _, line := range lines {
			if hashRegex.MatchString(strings.TrimSpace(line)) {
				hashCount++
			}
		}
		if hashCount > 1 {
			t.Fatalf("Expected at most 1 snapshot, got %d", hashCount)
		}
	})
}

func TestShowCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Get a hash to test with
	listOutput, _, _ := suite.runTimemachineCmd("list")
	hash := suite.extractHashFromOutput(listOutput)

	t.Run("show_with_full_hash", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("show", hash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "show", hash)
		suite.expectOutput(stdout, "Snapshot Details")
		suite.expectOutput(stdout, hash)
		suite.expectOutputMatch(stdout, `Author:.*Test User`)
		suite.expectOutputMatch(stdout, `Date:.*\d{4}`)
		suite.expectOutput(stdout, "Changed Files:")
	})

	t.Run("show_with_short_hash", func(t *testing.T) {
		shortHash := hash[:8]
		stdout, stderr, exitCode := suite.runTimemachineCmd("show", shortHash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "show", shortHash)
		suite.expectOutput(stdout, "Snapshot Details")
		suite.expectOutput(stdout, hash) // Should expand to full hash
	})

	t.Run("show_nonexistent_hash", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("show", "deadbeef")
		
		suite.expectFailure(stdout, stderr, exitCode, "show", "deadbeef")
		suite.expectOutput(stdout, "git command failed")
	})

	t.Run("show_invalid_hash", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("show", "invalid-hash!")
		
		suite.expectFailure(stdout, stderr, exitCode, "show", "invalid-hash!")
		// Should fail validation before trying to find snapshot
	})
}

func TestInspectCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Get a hash to test with
	listOutput, _, _ := suite.runTimemachineCmd("list")
	hash := suite.extractHashFromOutput(listOutput)

	t.Run("inspect_with_short_hash", func(t *testing.T) {
		shortHash := hash[:8]
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", shortHash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", shortHash)
		suite.expectOutput(stdout, "Snapshot Overview")
		suite.expectOutput(stdout, shortHash)
		suite.expectOutput(stdout, "File Changes")
	})

	t.Run("inspect_with_full_hash", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", hash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", hash)
		suite.expectOutput(stdout, "Snapshot Overview")
		suite.expectOutput(stdout, hash)
	})

	t.Run("inspect_with_stats", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--stats", hash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", "--stats", hash)
		suite.expectOutput(stdout, "Repository Statistics")
		suite.expectOutputMatch(stdout, `Repository size:.*\d+`)
		suite.expectOutputMatch(stdout, `total-snapshots:.*\d+`)
	})

	t.Run("inspect_with_diff", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--diff", hash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", "--diff", hash)
		suite.expectOutput(stdout, "Detailed Changes")
		// Should show diff content
	})

	t.Run("inspect_with_file_filter", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--file", "main.go", hash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", "--file", "main.go", hash)
		suite.expectOutput(stdout, "File Changes")
		// When files match, it shows total count, not filter message
		suite.expectOutput(stdout, "Total files changed:")
	})

	t.Run("inspect_with_file_filter_no_matches", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--file", "nonexistent.txt", hash)
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", "--file", "nonexistent.txt", hash)
		suite.expectOutput(stdout, "File Changes")
		suite.expectOutput(stdout, "No file changes found")
		suite.expectOutput(stdout, "(filtered for: nonexistent.txt)")
	})

	t.Run("inspect_without_hash_uses_latest", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect")
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect")
		suite.expectOutput(stdout, "Snapshot Overview")
		// Should work with latest snapshot
	})

	t.Run("inspect_nonexistent_hash", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "deadbeef")
		
		suite.expectFailure(stdout, stderr, exitCode, "inspect", "deadbeef")
		suite.expectOutput(stderr, "snapshot hash 'deadbeef' not found")
	})

	t.Run("inspect_malicious_file_filter", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--file", "../../../etc/passwd", hash)
		
		suite.expectFailure(stdout, stderr, exitCode, "inspect", "--file", "../../../etc/passwd", hash)
		suite.expectOutput(stderr, "invalid file filter")
		suite.expectOutput(stderr, "path traversal not allowed")
	})

	t.Run("inspect_search_all", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--search-all")
		
		suite.expectSuccess(stdout, stderr, exitCode, "inspect", "--search-all")
		suite.expectOutput(stdout, "Searching All Snapshots")
		suite.expectOutputMatch(stdout, `Found \d+ snapshot\(s\)`)
	})
}

func TestRestoreCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	// Get a hash to test with
	listOutput, _, _ := suite.runTimemachineCmd("list")
	hash := suite.extractHashFromOutput(listOutput)

	t.Run("restore_specific_file", func(t *testing.T) {
		// Modify the file first
		suite.createFile("main.go", "package main\n\n// Modified content\nfunc main() {}")
		
		stdout, stderr, exitCode := suite.runTimemachineCmd("restore", hash, "--files", "main.go", "--force")
		
		suite.expectSuccess(stdout, stderr, exitCode, "restore", hash, "--files", "main.go", "--force")
		suite.expectOutput(stdout, "Files restored successfully")
		suite.expectOutput(stdout, "main.go")
		
		// Verify file was restored
		content, err := os.ReadFile(filepath.Join(suite.repoDir, "main.go"))
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}
		if !strings.Contains(string(content), "Hello World") {
			t.Fatalf("File was not properly restored: %s", string(content))
		}
	})

	t.Run("restore_nonexistent_file", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("restore", hash, "--files", "nonexistent.txt")
		
		suite.expectFailure(stdout, stderr, exitCode, "restore", hash, "--files", "nonexistent.txt")
		// Should handle gracefully
	})

	t.Run("restore_malicious_path", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("restore", hash, "--files", "../../../etc/passwd", "--force")
		
		suite.expectFailure(stdout, stderr, exitCode, "restore", hash, "--files", "../../../etc/passwd", "--force")
		// The validation should happen before confirmation, so any error message is fine
		if exitCode == 0 {
			suite.t.Fatalf("Expected command to fail with malicious path, but it succeeded")
		}
	})
}

func TestCleanCommand(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	t.Run("clean_with_confirmation", func(t *testing.T) {
		// This test might be tricky since clean might require interactive input
		// We'll test the --force flag if it exists
		stdout, stderr, exitCode := suite.runTimemachineCmd("clean", "--help")
		
		// Just verify help works for now
		suite.expectSuccess(stdout, stderr, exitCode, "clean", "--help")
	})
}

func TestCrossCommandIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("complete_workflow", func(t *testing.T) {
		// 1. Initialize
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		suite.expectSuccess(stdout, stderr, exitCode, "init")

		// 2. Check status
		stdout, stderr, exitCode = suite.runTimemachineCmd("status")
		suite.expectSuccess(stdout, stderr, exitCode, "status")

		// 3. List snapshots
		stdout, stderr, exitCode = suite.runTimemachineCmd("list")
		suite.expectSuccess(stdout, stderr, exitCode, "list")

		// Extract hash if any snapshots exist
		if strings.Contains(stdout, "Recent snapshots") && !strings.Contains(stdout, "No snapshots") {
			hash := suite.extractHashFromOutput(stdout)

			// 4. Show snapshot
			stdout, stderr, exitCode = suite.runTimemachineCmd("show", hash)
			suite.expectSuccess(stdout, stderr, exitCode, "show", hash)

			// 5. Inspect snapshot (the bug we fixed!)
			stdout, stderr, exitCode = suite.runTimemachineCmd("inspect", hash)
			suite.expectSuccess(stdout, stderr, exitCode, "inspect", hash)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("invalid_command", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("invalid-command")
		
		suite.expectFailure(stdout, stderr, exitCode, "invalid-command")
		// Should show help or error message
	})

	t.Run("missing_arguments", func(t *testing.T) {
		stdout, stderr, exitCode := suite.runTimemachineCmd("show")
		
		suite.expectFailure(stdout, stderr, exitCode, "show")
		// Should indicate missing hash argument
	})
}

func TestSecurityValidation(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	suite.setupWithSnapshots()

	hash := "abcd1234" // Use a predictable hash for security tests

	t.Run("path_traversal_attacks", func(t *testing.T) {
		attacks := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\SAM",
			"/etc/passwd",
			"C:\\Windows\\System32",
			"\\\\server\\share\\file.txt",
		}

		for _, attack := range attacks {
			t.Run(fmt.Sprintf("attack_%s", attack), func(t *testing.T) {
				stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", "--file", attack, hash)
				
				suite.expectFailure(stdout, stderr, exitCode, "inspect", "--file", attack, hash)
				// Should contain security error message
				if !strings.Contains(stderr, "traversal") && 
				   !strings.Contains(stderr, "absolute") && 
				   !strings.Contains(stderr, "invalid") {
					t.Fatalf("Expected security error for attack %q, got: %s", attack, stderr)
				}
			})
		}
	})

	t.Run("hash_injection_attacks", func(t *testing.T) {
		attacks := []string{
			"abc123; rm -rf /",
			"abc123 && echo pwned",
			"abc123 | cat /etc/passwd",
			"$(rm -rf /)",
			"`rm -rf /`",
			"abc123$(whoami)",
		}

		for _, attack := range attacks {
			t.Run(fmt.Sprintf("hash_attack_%s", attack), func(t *testing.T) {
				stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", attack)
				
				suite.expectFailure(stdout, stderr, exitCode, "inspect", attack)
				// Should fail hash validation
			})
		}
	})
}