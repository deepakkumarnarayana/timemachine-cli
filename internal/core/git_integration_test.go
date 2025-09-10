package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCompleteAIWorkflow tests the complete workflow as used in AI development
func TestCompleteAIWorkflow(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Simulate AI development workflow:
	// 1. Start with working code
	// 2. AI makes changes
	// 3. Code breaks
	// 4. Restore from snapshot

	// Step 1: Create working application files
	appFile := filepath.Join(tempDir, "app.js")
	configFile := filepath.Join(tempDir, "config.json")
	readmeFile := filepath.Join(tempDir, "README.md")

	workingApp := `
function calculateTotal(items) {
    return items.reduce((sum, item) => sum + item.price, 0);
}

module.exports = { calculateTotal };
`

	workingConfig := `{
    "port": 3000,
    "database": "sqlite://app.db"
}`

	workingReadme := `# My App
Working application with tests.`

	if err := os.WriteFile(appFile, []byte(workingApp), 0644); err != nil {
		t.Fatalf("Failed to create app file: %v", err)
	}
	if err := os.WriteFile(configFile, []byte(workingConfig), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	if err := os.WriteFile(readmeFile, []byte(workingReadme), 0644); err != nil {
		t.Fatalf("Failed to create readme file: %v", err)
	}

	// Create snapshot of working code
	err := gitManager.CreateSnapshot("Working application before AI changes")
	if err != nil {
		t.Fatalf("Failed to create working snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatal("No working snapshot created")
	}

	workingSnapshotHash := snapshots[0].Hash
	t.Logf("Working code snapshot: %s - %s", workingSnapshotHash, snapshots[0].Message)

	// Step 2: AI makes "improvements" that break things
	brokenApp := `
// AI "improved" version with bugs
function calculateTotal(items) {
    // AI tried to be clever but introduced bugs
    let total = 0;
    for (let i = 0; i <= items.length; i++) { // Bug: off-by-one error
        total += items[i].price; // Bug: will cause undefined access
    }
    return total.toString(); // Bug: changed return type
}

// AI added "helpful" but broken function  
function validateInput(items) {
    if (!items || items.length = 0) { // Bug: assignment instead of comparison
        throw new Error("Invalid input);  // Bug: unterminated string
    }
}

module.exports = { calculateTotal, validateInput };
`

	brokenConfig := `{
    "port": "3000", // Bug: should be number
    "database": "sqlite://app.db",
    "newFeature": {
        "enabled": true,
        "timeout": undefined // Bug: invalid JSON
    }
}` // Bug: invalid JSON syntax

	if err := os.WriteFile(appFile, []byte(brokenApp), 0644); err != nil {
		t.Fatalf("Failed to write broken app: %v", err)
	}
	if err := os.WriteFile(configFile, []byte(brokenConfig), 0644); err != nil {
		t.Fatalf("Failed to write broken config: %v", err)
	}

	// Create snapshot of broken code
	err = gitManager.CreateSnapshot("AI attempted improvements - code now broken!")
	if err != nil {
		t.Fatalf("Failed to create broken snapshot: %v", err)
	}

	// Step 3: Realize code is broken, need to restore
	t.Log("🚨 Code is now broken after AI changes - need to restore!")

	// List recent snapshots to see what's available
	recentSnapshots, err := gitManager.ListSnapshots(5, "")
	if err != nil {
		t.Fatalf("Failed to list recent snapshots: %v", err)
	}

	t.Logf("Available snapshots for recovery:")
	for i, snapshot := range recentSnapshots {
		t.Logf("  %d. %s - %s", i+1, snapshot.Hash[:8], snapshot.Message)
	}

	// Step 4: Restore to working version
	err = gitManager.RestoreSnapshot(workingSnapshotHash, []string{})
	if err != nil {
		t.Fatalf("Failed to restore working version: %v", err)
	}

	// Step 5: Verify restoration worked
	restoredApp, err := os.ReadFile(appFile)
	if err != nil {
		t.Fatalf("Failed to read restored app: %v", err)
	}

	if !strings.Contains(string(restoredApp), "calculateTotal") || strings.Contains(string(restoredApp), "off-by-one error") {
		t.Errorf("App not properly restored - still contains broken code. Content: %s", string(restoredApp))
	}

	restoredConfig, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read restored config: %v", err)
	}

	if !strings.Contains(string(restoredConfig), `"port": 3000`) {
		t.Errorf("Config not properly restored. Content: %s", string(restoredConfig))
	}

	t.Log("✅ Complete AI workflow test passed - code successfully restored!")
}

// TestBranchWorkflow tests branch creation and switching workflow
func TestBranchWorkflow(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Create base project structure
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	mainFile := filepath.Join(srcDir, "main.go")
	testFile := filepath.Join(srcDir, "main_test.go")

	baseMain := `package main

import "fmt"

func main() {
    fmt.Println("Hello World")
}

func add(a, b int) int {
    return a + b
}`

	baseTest := `package main

import "testing"

func TestAdd(t *testing.T) {
    result := add(2, 3)
    if result != 5 {
        t.Errorf("Expected 5, got %d", result)
    }
}`

	if err := os.WriteFile(mainFile, []byte(baseMain), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}
	if err := os.WriteFile(testFile, []byte(baseTest), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create initial snapshot on main
	err := gitManager.CreateSnapshot("Initial project setup")
	if err != nil {
		t.Fatalf("Failed to create initial snapshot: %v", err)
	}

	// Get initial snapshots
	mainSnapshots, err := gitManager.ListSnapshots(10, "")
	if err != nil {
		t.Fatalf("Failed to list main snapshots: %v", err)
	}
	mainSnapshotCount := len(mainSnapshots)
	t.Logf("Main branch has %d snapshots", mainSnapshotCount)

	// Create feature branch for new functionality
	cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", "feature/calculator")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add calculator functionality
	calculatorMain := `package main

import "fmt"

func main() {
    fmt.Println("Calculator App")
    fmt.Printf("2 + 3 = %d\n", add(2, 3))
    fmt.Printf("10 - 4 = %d\n", subtract(10, 4))
    fmt.Printf("5 * 6 = %d\n", multiply(5, 6))
}

func add(a, b int) int {
    return a + b
}

func subtract(a, b int) int {
    return a - b
}

func multiply(a, b int) int {
    return a * b
}`

	calculatorTest := `package main

import "testing"

func TestAdd(t *testing.T) {
    result := add(2, 3)
    if result != 5 {
        t.Errorf("Expected 5, got %d", result)
    }
}

func TestSubtract(t *testing.T) {
    result := subtract(10, 4)
    if result != 6 {
        t.Errorf("Expected 6, got %d", result)
    }
}

func TestMultiply(t *testing.T) {
    result := multiply(5, 6)
    if result != 30 {
        t.Errorf("Expected 30, got %d", result)
    }
}`

	if err := os.WriteFile(mainFile, []byte(calculatorMain), 0644); err != nil {
		t.Fatalf("Failed to update main file: %v", err)
	}
	if err := os.WriteFile(testFile, []byte(calculatorTest), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Create snapshot on feature branch
	err = gitManager.CreateSnapshot("Add calculator functionality")
	if err != nil {
		t.Fatalf("Failed to create feature snapshot: %v", err)
	}

	// Switch back to main - handle both master and main branch names
	currentBranch, err := gitManager.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	t.Logf("Currently on branch: %s", currentBranch)

	// If we're not on master/main, try to switch
	if currentBranch != "master" && currentBranch != "main" {
		cmd = exec.Command("git", "-C", tempDir, "checkout", "master")
		if err := cmd.Run(); err != nil {
			// Try "main" if "master" doesn't exist
			cmd = exec.Command("git", "-C", tempDir, "checkout", "main")
			if err := cmd.Run(); err != nil {
				t.Logf("Could not switch to main branch, staying on %s: %v", currentBranch, err)
			}
		}
	}

	// Create another change on main
	hotfixMain := `package main

import "fmt"

func main() {
    fmt.Println("Hello World - Fixed bug")
}

func add(a, b int) int {
    // Fixed: handle negative numbers correctly
    return a + b
}`

	if err := os.WriteFile(mainFile, []byte(hotfixMain), 0644); err != nil {
		t.Fatalf("Failed to create hotfix: %v", err)
	}

	err = gitManager.CreateSnapshot("Hotfix: Handle edge cases in add function")
	if err != nil {
		t.Fatalf("Failed to create hotfix snapshot: %v", err)
	}

	// Test branch-specific snapshot listing
	allSnapshots, err := gitManager.ListSnapshots(20, "")
	if err != nil {
		t.Fatalf("Failed to list all snapshots: %v", err)
	}

	t.Logf("Total snapshots across branches: %d", len(allSnapshots))

	// Verify branch detection in messages
	branchSwitchCount := 0
	for _, snapshot := range allSnapshots {
		if strings.Contains(snapshot.Message, "→") || strings.Contains(snapshot.Message, "BRANCH SWITCH") {
			branchSwitchCount++
			t.Logf("Branch switch detected: %s", snapshot.Message)
		}
	}

	if len(allSnapshots) < 3 {
		t.Errorf("Expected at least 3 snapshots, got %d", len(allSnapshots))
	}

	t.Logf("✅ Branch workflow completed: %d total snapshots, %d branch switches detected",
		len(allSnapshots), branchSwitchCount)
}

// TestLongRunningSession simulates a long development session
func TestLongRunningSession(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Simulate a long session with many small changes
	const sessionSnapshots = 15

	baseFile := filepath.Join(tempDir, "session.txt")
	branches := []string{"main", "feature/auth", "feature/api", "bugfix/crash", "main"}

	snapshotCount := 0

	for i := 0; i < sessionSnapshots; i++ {
		// Occasionally switch branches
		if i > 0 && i%4 == 0 {
			branchIndex := i / 4
			if branchIndex < len(branches) {
				targetBranch := branches[branchIndex]
				if targetBranch != "main" {
					// Create branch if it doesn't exist
					cmd := exec.Command("git", "-C", tempDir, "checkout", "-b", targetBranch)
					_ = cmd.Run() // Ignore error if branch exists
				} else {
					cmd := exec.Command("git", "-C", tempDir, "checkout", "master")
					if err := cmd.Run(); err != nil {
						cmd = exec.Command("git", "-C", tempDir, "checkout", "main")
						_ = cmd.Run()
					}
				}
				t.Logf("Session: switched to branch %s", targetBranch)
			}
		}

		// Make a change
		content := fmt.Sprintf("Session snapshot %d\nTimestamp: %s\nIteration: %d\n",
			i+1, time.Now().Format(time.RFC3339), i)

		if err := os.WriteFile(baseFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write session file %d: %v", i, err)
		}

		// Create snapshot
		message := fmt.Sprintf("Session work - iteration %d", i+1)
		if i%5 == 0 {
			message = fmt.Sprintf("Major milestone - iteration %d", i+1)
		}

		err := gitManager.CreateSnapshot(message)
		if err != nil {
			t.Fatalf("Failed to create session snapshot %d: %v", i, err)
		}

		snapshotCount++

		// Short delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all snapshots were created
	allSnapshots, err := gitManager.ListSnapshots(sessionSnapshots+5, "")
	if err != nil {
		t.Fatalf("Failed to list session snapshots: %v", err)
	}

	if len(allSnapshots) < sessionSnapshots {
		t.Errorf("Expected at least %d snapshots, got %d", sessionSnapshots, len(allSnapshots))
	}

	// Check for branch switches
	switchCount := 0
	milestoneCount := 0
	for _, snapshot := range allSnapshots {
		if strings.Contains(snapshot.Message, "→") || strings.Contains(snapshot.Message, "BRANCH SWITCH") {
			switchCount++
		}
		if strings.Contains(snapshot.Message, "Major milestone") {
			milestoneCount++
		}
	}

	t.Logf("✅ Long session completed: %d snapshots, %d switches, %d milestones",
		len(allSnapshots), switchCount, milestoneCount)

	// Test cleanup - list only recent snapshots
	recentSnapshots, err := gitManager.ListSnapshots(5, "")
	if err != nil {
		t.Fatalf("Failed to list recent snapshots: %v", err)
	}

	if len(recentSnapshots) > 5 {
		t.Errorf("Expected max 5 recent snapshots, got %d", len(recentSnapshots))
	}

	t.Log("✅ Session snapshot limiting works correctly")
}

// TestErrorRecoveryWorkflow tests recovery from various error conditions
func TestErrorRecoveryWorkflow(t *testing.T) {
	tempDir, _, gitManager := setupTestRepo(t)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Test 1: Recovery from file permission issues
	restrictedFile := filepath.Join(tempDir, "restricted.txt")
	if err := os.WriteFile(restrictedFile, []byte("restricted content"), 0644); err != nil {
		t.Fatalf("Failed to create restricted file: %v", err)
	}

	// Create snapshot with restricted file
	err := gitManager.CreateSnapshot("Before permission changes")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	snapshots, err := gitManager.ListSnapshots(1, "")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatal("No snapshots created")
	}

	originalHash := snapshots[0].Hash

	// Change file permissions (this might not work on all systems)
	if err := os.Chmod(restrictedFile, 0000); err != nil {
		t.Logf("Could not restrict file permissions (may be expected): %v", err)
	}

	// Try to restore - should handle permission issues gracefully
	err = gitManager.RestoreSnapshot(originalHash, []string{"restricted.txt"})
	if err != nil {
		t.Logf("Restore failed due to permissions (expected): %v", err)
	}

	// Restore permissions
	if err := os.Chmod(restrictedFile, 0644); err != nil {
		t.Logf("Could not restore file permissions: %v", err)
	}

	// Test 2: Recovery from disk space issues (simulated)
	// Create a very large file to simulate space issues
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := strings.Repeat("A", 1024*1024) // 1MB of 'A's
	if err := os.WriteFile(largeFile, []byte(largeContent), 0644); err != nil {
		t.Logf("Could not create large file: %v", err)
	}

	// Try to create snapshot with large file
	err = gitManager.CreateSnapshot("Large file test")
	if err != nil {
		t.Logf("Large file snapshot failed (may be expected): %v", err)
	} else {
		t.Log("Large file snapshot succeeded")
	}

	// Test 3: Recovery from corrupted working directory
	// Remove the large file to simulate corruption
	if err := os.Remove(largeFile); err != nil {
		t.Logf("Could not remove large file: %v", err)
	}

	// Create a new snapshot - should handle missing files gracefully
	newFile := filepath.Join(tempDir, "recovery.txt")
	if err := os.WriteFile(newFile, []byte("recovery content"), 0644); err != nil {
		t.Fatalf("Failed to create recovery file: %v", err)
	}

	err = gitManager.CreateSnapshot("Recovery after issues")
	if err != nil {
		t.Fatalf("Failed to create recovery snapshot: %v", err)
	}

	// Verify system is still functional
	finalSnapshots, err := gitManager.ListSnapshots(5, "")
	if err != nil {
		t.Fatalf("Failed to list final snapshots: %v", err)
	}

	if len(finalSnapshots) == 0 {
		t.Error("No snapshots available after recovery")
	}

	t.Logf("✅ Error recovery workflow completed: %d snapshots available", len(finalSnapshots))
}
