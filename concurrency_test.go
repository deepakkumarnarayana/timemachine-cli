package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentGitOperations tests multiple concurrent git operations on shadow repo
func TestConcurrentGitOperations(t *testing.T) {
	t.Run("multiple_git_commands_same_repo", func(t *testing.T) {
		suite := NewIntegrationTestSuite(t)
		defer suite.Cleanup()

		// Initialize timemachine in test directory
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		suite.expectSuccess(stdout, stderr, exitCode, "init")

		// Create some initial content
		suite.createFile("test1.txt", "initial content 1")
		suite.createFile("test2.txt", "initial content 2")
		suite.createFile("test3.txt", "initial content 3")

		// Create initial snapshots
		suite.runTimemachineCmd("list") // This creates snapshots if needed

		var wg sync.WaitGroup
		errors := make(chan error, 5)

		// Launch 5 concurrent git operations on shadow repo
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				// Modify different files concurrently
				testFile := fmt.Sprintf("concurrent_test_%d.txt", id)
				content := fmt.Sprintf("concurrent content from goroutine %d", id)
				
				if err := os.WriteFile(filepath.Join(suite.repoDir, testFile), []byte(content), 0644); err != nil {
					errors <- fmt.Errorf("goroutine %d failed to write file: %w", id, err)
					return
				}

				// Try to list snapshots concurrently (triggers git operations)
				cmd := exec.Command(suite.binaryPath, "list")
				cmd.Dir = suite.repoDir
				if _, err := cmd.CombinedOutput(); err != nil {
					errors <- fmt.Errorf("goroutine %d failed to list snapshots: %w", id, err)
					return
				}
			}(i)
		}

		// Wait for all operations to complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success - all operations completed
		case err := <-errors:
			t.Fatalf("Concurrent git operations failed: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("Concurrent git operations timed out")
		}

		// Verify shadow repo integrity by listing snapshots
		stdout, _, _ = suite.runTimemachineCmd("list")
		if len(stdout) == 0 {
			t.Error("Shadow repo appears corrupted - no snapshots found")
		}
	})

	t.Run("file_watcher_vs_restore", func(t *testing.T) {
		suite := NewIntegrationTestSuite(t)
		defer suite.Cleanup()

		// Initialize timemachine
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		suite.expectSuccess(stdout, stderr, exitCode, "init")

		// Create initial content and snapshot
		suite.createFile("watch_test.txt", "original content")
		time.Sleep(100 * time.Millisecond) // Let watcher create snapshot

		// Get first snapshot hash
		listOutput, _, _ := suite.runTimemachineCmd("list")
		if len(listOutput) == 0 {
			t.Skip("No snapshots available for restore test")
		}

		// Start a simulated file watcher in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		errors := make(chan error, 2)

		// Simulate file watcher creating snapshots
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					counter++
					testFile := fmt.Sprintf("watcher_file_%d.txt", counter)
					content := fmt.Sprintf("watcher content %d", counter)
					
					if err := os.WriteFile(filepath.Join(suite.repoDir, testFile), []byte(content), 0644); err != nil {
						errors <- fmt.Errorf("watcher simulation failed: %w", err)
						return
					}

					if counter >= 5 {
						return // Limit iterations
					}
				}
			}
		}()

		// Simultaneously run restore command
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Start slightly after watcher

			// Modify the file that we'll restore
			if err := os.WriteFile(filepath.Join(suite.repoDir, "watch_test.txt"), []byte("modified content"), 0644); err != nil {
				errors <- fmt.Errorf("failed to modify file before restore: %w", err)
				return
			}

			// Try to restore while watcher is active
			cmd := exec.Command(suite.binaryPath, "restore", "watch_test.txt")
			cmd.Dir = suite.repoDir
			if _, err := cmd.CombinedOutput(); err != nil {
				errors <- fmt.Errorf("restore operation failed: %w", err)
				return
			}
		}()

		// Wait for completion with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			cancel() // Stop the watcher simulation
		case err := <-errors:
			cancel()
			t.Fatalf("File watcher vs restore test failed: %v", err)
		case <-time.After(10 * time.Second):
			cancel()
			t.Fatal("File watcher vs restore test timed out")
		}

		// Verify no corruption occurred
		content, err := os.ReadFile(filepath.Join(suite.repoDir, "watch_test.txt"))
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}
		
		if string(content) != "original content" {
			t.Errorf("File was not properly restored. Expected 'original content', got '%s'", string(content))
		}
	})

	t.Run("multiple_timemachine_processes", func(t *testing.T) {
		suite := NewIntegrationTestSuite(t)
		defer suite.Cleanup()

		// Initialize timemachine
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		suite.expectSuccess(stdout, stderr, exitCode, "init")

		// Create some content for testing
		suite.createFile("process_test.txt", "content for process testing")
		time.Sleep(100 * time.Millisecond)

		var wg sync.WaitGroup
		errors := make(chan error, 3)

		// Launch 3 timemachine processes simultaneously
		commands := [][]string{
			{"list"},
			{"list", "--limit", "1"},
			{"list", "--limit", "5"},
		}

		for i, cmd := range commands {
			wg.Add(1)
			go func(id int, command []string) {
				defer wg.Done()
				
				execCmd := exec.Command(suite.binaryPath, command...)
				execCmd.Dir = suite.repoDir
				output, err := execCmd.CombinedOutput()
				
				if err != nil {
					errors <- fmt.Errorf("process %d (command %v) failed: %w\nOutput: %s", id, command, err, string(output))
					return
				}
			}(i, cmd)
		}

		// Wait for all processes with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success - all processes completed
		case err := <-errors:
			t.Fatalf("Multiple timemachine processes test failed: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("Multiple timemachine processes test timed out")
		}

		// Verify shadow repo integrity
		stdout, _, _ = suite.runTimemachineCmd("list")
		if len(stdout) == 0 {
			t.Error("Shadow repo appears corrupted after concurrent processes")
		}
	})
}

// TestGitLockFileBehavior tests Git's built-in locking mechanisms
func TestGitLockFileBehavior(t *testing.T) {
	t.Run("observe_git_locking", func(t *testing.T) {
		suite := NewIntegrationTestSuite(t)
		defer suite.Cleanup()

		// Initialize timemachine
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		suite.expectSuccess(stdout, stderr, exitCode, "init")

		shadowRepoPath := filepath.Join(suite.repoDir, ".git", "timemachine_snapshots")
		
		// Check that shadow repo was created
		if _, err := os.Stat(shadowRepoPath); os.IsNotExist(err) {
			t.Fatal("Shadow repository was not created")
		}

		// Monitor for lock files during concurrent operations
		lockFiles := []string{
			filepath.Join(shadowRepoPath, "index.lock"),
			filepath.Join(shadowRepoPath, "HEAD.lock"),
			filepath.Join(shadowRepoPath, "refs", "heads", "main.lock"),
		}

		// Create concurrent git operations and monitor for lock files
		var wg sync.WaitGroup
		lockDetected := make(chan string, 1)

		// Background lock file monitor
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for i := 0; i < 100; i++ { // Monitor for 1 second
				for _, lockFile := range lockFiles {
					if _, err := os.Stat(lockFile); err == nil {
						select {
						case lockDetected <- lockFile:
						default:
						}
						return
					}
				}
				<-ticker.C
			}
		}()

		// Create concurrent git operations
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				// Create different files to trigger git operations
				testFile := fmt.Sprintf("lock_test_%d.txt", id)
				content := fmt.Sprintf("content for lock test %d", id)
				
				if err := os.WriteFile(filepath.Join(suite.repoDir, testFile), []byte(content), 0644); err != nil {
					t.Logf("Failed to create test file for lock test: %v", err)
					return
				}

				// This should trigger git operations in shadow repo
				cmd := exec.Command(suite.binaryPath, "list")
				cmd.Dir = suite.repoDir
				if _, err := cmd.CombinedOutput(); err != nil {
					t.Logf("Lock test git operation failed: %v", err)
				}
			}(i)
		}

		wg.Wait()

		// Check if Git's locking was observed
		select {
		case lockFile := <-lockDetected:
			t.Logf("✅ Git lock file detected: %s - Git's built-in locking is working", lockFile)
		default:
			t.Log("ℹ️  No lock files detected - operations may have been too fast or no conflicts occurred")
		}
	})

	t.Run("lock_file_cleanup", func(t *testing.T) {
		suite := NewIntegrationTestSuite(t)
		defer suite.Cleanup()

		// Initialize timemachine
		stdout, stderr, exitCode := suite.runTimemachineCmd("init")
		suite.expectSuccess(stdout, stderr, exitCode, "init")

		shadowRepoPath := filepath.Join(suite.repoDir, ".git", "timemachine_snapshots")
		
		// Manually create a mock lock file to test cleanup
		lockFile := filepath.Join(shadowRepoPath, "test.lock")
		if err := os.WriteFile(lockFile, []byte("test lock"), 0644); err != nil {
			t.Fatalf("Failed to create test lock file: %v", err)
		}

		// Verify lock file exists
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			t.Fatal("Test lock file was not created")
		}

		// Run a git operation (this should not be affected by our test lock file)
		suite.runTimemachineCmd("list")

		// Note: Real git lock files are automatically cleaned up by git
		// Our test lock file should still exist since it's not a real git lock
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			t.Log("ℹ️  Test lock file was cleaned up (unexpected but not necessarily bad)")
		} else {
			t.Log("✅ Test lock file still exists - git operations not affected by foreign lock files")
		}

		// Clean up our test lock file
		os.Remove(lockFile)
	})
}

// TestConcurrencyPerformance tests performance under concurrent load
func TestConcurrencyPerformance(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Initialize timemachine
	stdout, stderr, exitCode := suite.runTimemachineCmd("init")
	suite.expectSuccess(stdout, stderr, exitCode, "init")

	// Create baseline content
	for i := 0; i < 5; i++ {
		suite.createFile(fmt.Sprintf("perf_test_%d.txt", i), fmt.Sprintf("performance test content %d", i))
	}

	startTime := time.Now()
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Launch multiple concurrent operations
	operations := []string{"list", "show", "list"}
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			operation := operations[id%len(operations)]
			cmd := exec.Command(suite.binaryPath, operation)
			cmd.Dir = suite.repoDir
			
			_, err := cmd.CombinedOutput()
			if err != nil {
				errors <- fmt.Errorf("performance test operation %d (%s) failed: %w", id, operation, err)
				return
			}
		}(i)
	}

	// Wait for completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		duration := time.Since(startTime)
		t.Logf("✅ Concurrent performance test completed in %v", duration)
		
		if duration > 30*time.Second {
			t.Errorf("Performance degradation detected: took %v (expected < 30s)", duration)
		}
	case err := <-errors:
		t.Fatalf("Performance test failed: %v", err)
	case <-time.After(60 * time.Second):
		t.Fatal("Performance test timed out after 60 seconds")
	}

	// Verify final state
	stdout, _, _ = suite.runTimemachineCmd("list")
	if len(stdout) == 0 {
		t.Error("Performance test may have caused repository corruption")
	}
}