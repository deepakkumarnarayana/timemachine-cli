package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

func TestUpdateGitignore(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "timemachine-gitignore-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("CreateNewGitignore", func(t *testing.T) {
		err := updateGitignore(tempDir)
		if err != nil {
			t.Fatalf("updateGitignore failed: %v", err)
		}

		// Check .gitignore was created with correct content
		gitignorePath := filepath.Join(tempDir, ".gitignore")
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			t.Fatalf("Failed to read .gitignore: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, ".git/timemachine_snapshots/") {
			t.Errorf(".gitignore does not contain timemachine exclusion")
		}
		if !strings.Contains(contentStr, "# Time Machine shadow repository") {
			t.Errorf(".gitignore does not contain timemachine comment")
		}
	})

	t.Run("PreserveExistingContent", func(t *testing.T) {
		// Create .gitignore with existing content
		gitignorePath := filepath.Join(tempDir, ".gitignore")
		existingContent := "node_modules/\n*.log\nbuild/\n"
		err := os.WriteFile(gitignorePath, []byte(existingContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing .gitignore: %v", err)
		}

		err = updateGitignore(tempDir)
		if err != nil {
			t.Fatalf("updateGitignore failed: %v", err)
		}

		// Check both existing and new content are present
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			t.Fatalf("Failed to read .gitignore: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "node_modules/") {
			t.Errorf(".gitignore does not preserve existing content")
		}
		if !strings.Contains(contentStr, ".git/timemachine_snapshots/") {
			t.Errorf(".gitignore does not contain new timemachine exclusion")
		}
	})

	t.Run("SkipIfAlreadyExists", func(t *testing.T) {
		// Create .gitignore that already contains timemachine_snapshots
		gitignorePath := filepath.Join(tempDir, ".gitignore")
		existingContent := "node_modules/\n.git/timemachine_snapshots/\n*.log\n"
		err := os.WriteFile(gitignorePath, []byte(existingContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing .gitignore: %v", err)
		}

		// Get original modification time
		stat1, err := os.Stat(gitignorePath)
		if err != nil {
			t.Fatalf("Failed to stat .gitignore: %v", err)
		}

		time.Sleep(10 * time.Millisecond) // Ensure different timestamp

		err = updateGitignore(tempDir)
		if err != nil {
			t.Fatalf("updateGitignore failed: %v", err)
		}

		// Check file was not modified (same timestamp)
		stat2, err := os.Stat(gitignorePath)
		if err != nil {
			t.Fatalf("Failed to stat .gitignore after update: %v", err)
		}

		if !stat1.ModTime().Equal(stat2.ModTime()) {
			t.Errorf(".gitignore was modified when it already contained timemachine exclusion")
		}
	})
}

func TestInstallPostPushHook(t *testing.T) {
	// Create temporary git directory structure
	tempDir, err := os.MkdirTemp("", "timemachine-hook-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitDir := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	t.Run("CreateNewHook", func(t *testing.T) {
		err := installPostPushHook(gitDir)
		if err != nil {
			t.Fatalf("installPostPushHook failed: %v", err)
		}

		// Check hook was created
		hookPath := filepath.Join(gitDir, "hooks", "post-push")
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("Failed to read post-push hook: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "#!/bin/sh") {
			t.Errorf("Hook does not contain shebang")
		}
		if !strings.Contains(contentStr, "timemachine clean --auto --quiet") {
			t.Errorf("Hook does not contain timemachine cleanup command")
		}
		if !strings.Contains(contentStr, "# Time Machine auto-cleanup") {
			t.Errorf("Hook does not contain timemachine comment")
		}

		// Check hook is executable
		stat, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("Failed to stat hook: %v", err)
		}
		if stat.Mode()&0111 == 0 {
			t.Errorf("Hook is not executable")
		}
	})

	t.Run("PreserveExistingHook", func(t *testing.T) {
		hookPath := filepath.Join(gitDir, "hooks", "post-push")
		
		// Create existing hook with custom content
		existingContent := "#!/bin/sh\necho 'Custom hook content'\n# Some existing functionality\n"
		err := os.WriteFile(hookPath, []byte(existingContent), 0755)
		if err != nil {
			t.Fatalf("Failed to create existing hook: %v", err)
		}

		err = installPostPushHook(gitDir)
		if err != nil {
			t.Fatalf("installPostPushHook failed: %v", err)
		}

		// Check both existing and new content are present
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("Failed to read hook after update: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Custom hook content") {
			t.Errorf("Hook does not preserve existing content")
		}
		if !strings.Contains(contentStr, "timemachine clean --auto --quiet") {
			t.Errorf("Hook does not contain new timemachine content")
		}
	})

	t.Run("SkipIfAlreadyExists", func(t *testing.T) {
		hookPath := filepath.Join(gitDir, "hooks", "post-push")
		
		// Create hook that already contains timemachine cleanup
		existingContent := "#!/bin/sh\necho 'Pre-existing hook'\ntimemachine clean --auto --quiet\n"
		err := os.WriteFile(hookPath, []byte(existingContent), 0755)
		if err != nil {
			t.Fatalf("Failed to create existing hook: %v", err)
		}

		// Get original modification time
		stat1, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("Failed to stat hook: %v", err)
		}

		time.Sleep(10 * time.Millisecond) // Ensure different timestamp

		err = installPostPushHook(gitDir)
		if err != nil {
			t.Fatalf("installPostPushHook failed: %v", err)
		}

		// Check file was not modified
		stat2, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("Failed to stat hook after update: %v", err)
		}

		if !stat1.ModTime().Equal(stat2.ModTime()) {
			t.Errorf("Hook was modified when it already contained timemachine cleanup")
		}
	})
}

func TestGitHookExecution(t *testing.T) {
	// This test verifies that the hook can be executed correctly
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping hook execution test")
	}

	// Create temporary repository
	tempDir, err := os.MkdirTemp("", "timemachine-hook-exec-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitDir := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	t.Run("HookIsExecutableAndValid", func(t *testing.T) {
		// Install the hook
		err := installPostPushHook(gitDir)
		if err != nil {
			t.Fatalf("Failed to install hook: %v", err)
		}

		hookPath := filepath.Join(gitDir, "hooks", "post-push")

		// Check hook exists and is executable
		stat, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("Hook file does not exist: %v", err)
		}
		if stat.Mode()&0111 == 0 {
			t.Errorf("Hook is not executable")
		}

		// Read hook content
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("Failed to read hook: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "#!/bin/sh") {
			t.Errorf("Hook does not contain shebang")
		}
		if !strings.Contains(contentStr, "timemachine clean --auto --quiet") {
			t.Errorf("Hook does not contain timemachine cleanup command")
		}

		// Test hook syntax by running it with sh -n (syntax check only)
		cmd := exec.Command("sh", "-n", hookPath)
		if err := cmd.Run(); err != nil {
			t.Errorf("Hook has syntax errors: %v", err)
		}
	})

	t.Run("HookExecutesDirectly", func(t *testing.T) {
		// Install the hook
		err := installPostPushHook(gitDir)
		if err != nil {
			t.Fatalf("Failed to install hook: %v", err)
		}

		// Create a test script that mimics timemachine
		testScriptPath := filepath.Join(tempDir, "fake-timemachine")
		testScript := `#!/bin/sh
echo "Cleanup executed at $(date)" > ` + filepath.Join(tempDir, "hook-test.log") + `
exit 0
`
		err = os.WriteFile(testScriptPath, []byte(testScript), 0755)
		if err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		// Create a modified hook that uses our test script
		hookPath := filepath.Join(gitDir, "hooks", "post-push")
		testHookContent := `#!/bin/sh

# Time Machine auto-cleanup
if command -v ` + testScriptPath + ` >/dev/null 2>&1; then
    ` + testScriptPath + ` --auto --quiet
fi
`
		err = os.WriteFile(hookPath, []byte(testHookContent), 0755)
		if err != nil {
			t.Fatalf("Failed to write test hook: %v", err)
		}

		// Execute the hook directly
		cmd := exec.Command(hookPath)
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to execute hook: %v", err)
		}

		// Check if our test script was executed
		testLogPath := filepath.Join(tempDir, "hook-test.log")
		if _, err := os.Stat(testLogPath); os.IsNotExist(err) {
			t.Errorf("Hook did not execute test command - log file not found")
		} else {
			content, err := os.ReadFile(testLogPath)
			if err != nil {
				t.Fatalf("Failed to read test log: %v", err)
			}
			if !strings.Contains(string(content), "Cleanup executed") {
				t.Errorf("Hook executed but did not produce expected output: %s", content)
			}
		}
	})

	t.Run("HookHandlesMissingTimemachine", func(t *testing.T) {
		// Install the hook
		err := installPostPushHook(gitDir)
		if err != nil {
			t.Fatalf("Failed to install hook: %v", err)
		}

		hookPath := filepath.Join(gitDir, "hooks", "post-push")

		// Execute the hook directly (timemachine command won't be found)
		// This should not fail - the hook should gracefully handle missing command
		cmd := exec.Command(hookPath)
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Errorf("Hook should not fail when timemachine command is not found: %v", err)
		}
	})
}

func TestInitCommand(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timemachine-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available, skipping init command test")
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	cmd.Run()

	// Change to temp directory for testing
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	t.Run("InitializesCorrectly", func(t *testing.T) {
		// Create init command
		initCmd := InitCmd()
		
		// Execute init command
		err := initCmd.RunE(initCmd, []string{})
		if err != nil {
			t.Fatalf("Init command failed: %v", err)
		}

		// Verify shadow repository was created
		shadowRepoDir := filepath.Join(tempDir, ".git", "timemachine_snapshots")
		if _, err := os.Stat(shadowRepoDir); os.IsNotExist(err) {
			t.Errorf("Shadow repository not created")
		}

		// Verify .gitignore was updated
		gitignorePath := filepath.Join(tempDir, ".gitignore")
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			t.Fatalf("Failed to read .gitignore: %v", err)
		}
		if !strings.Contains(string(content), ".git/timemachine_snapshots/") {
			t.Errorf(".gitignore was not updated with timemachine exclusion")
		}

		// Verify post-push hook was installed
		hookPath := filepath.Join(tempDir, ".git", "hooks", "post-push")
		hookContent, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("Failed to read post-push hook: %v", err)
		}
		if !strings.Contains(string(hookContent), "timemachine clean") {
			t.Errorf("Post-push hook was not installed correctly")
		}

		// Verify hook is executable
		stat, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("Failed to stat hook: %v", err)
		}
		if stat.Mode()&0111 == 0 {
			t.Errorf("Post-push hook is not executable")
		}

		// Verify initial snapshot was created
		state, err := core.NewAppState()
		if err != nil {
			t.Fatalf("Failed to create app state: %v", err)
		}

		gitManager := core.NewGitManager(state)
		snapshots, err := gitManager.ListSnapshots(1, "")
		if err != nil {
			t.Fatalf("Failed to list snapshots: %v", err)
		}

		if len(snapshots) == 0 {
			t.Errorf("Initial snapshot was not created")
		} else if !strings.Contains(snapshots[0].Message, "Initial Time Machine snapshot") {
			t.Errorf("Initial snapshot has wrong message: %s", snapshots[0].Message)
		}
	})

	t.Run("HandlesAlreadyInitialized", func(t *testing.T) {
		// Run init again - should detect already initialized
		initCmd := InitCmd()
		err := initCmd.RunE(initCmd, []string{})
		if err != nil {
			t.Fatalf("Init command failed on second run: %v", err)
		}
		// Should complete without error and show "already initialized" message
	})
}