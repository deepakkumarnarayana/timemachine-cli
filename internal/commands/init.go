package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// InitCmd creates the init command
func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize Time Machine in the current Git repository",
		Long: `Initialize Time Machine by creating a shadow repository for snapshots.

This command:
- Creates a shadow repository at .git/timemachine_snapshots/
- Updates .gitignore to exclude the shadow repository
- Installs a post-push hook for automatic cleanup
- Creates an initial snapshot`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	fmt.Println("🔧 Initializing Time Machine...")

	// Check if already initialized
	if state.IsInitialized {
		color.Green("✅ Time Machine is already initialized!")
		fmt.Printf("   Shadow repository exists at: %s\n", state.ShadowRepoDir)
		return nil
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Step 1: Create shadow repository
	fmt.Print("  Creating shadow repository... ")
	if err := gitManager.InitializeShadowRepo(); err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to create shadow repository: %w", err)
	}
	color.Green("✅")

	// Step 2: Update .gitignore
	fmt.Print("  Updating .gitignore... ")
	if err := updateGitignore(state.ProjectRoot); err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}
	color.Green("✅")

	// Step 3: Install post-push hook
	fmt.Print("  Installing auto-cleanup hook... ")
	if err := installPostPushHook(state.GitDir); err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to install post-push hook: %w", err)
	}
	color.Green("✅")

	// Step 4: Create initial snapshot
	fmt.Print("  Creating initial snapshot... ")
	if err := gitManager.CreateSnapshot("Initial Time Machine snapshot"); err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to create initial snapshot: %w", err)
	}
	color.Green("✅")

	// Success message
	fmt.Println()
	color.Green("✨ Time Machine initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  • Run 'timemachine start' to begin watching for changes")
	fmt.Println("  • Run 'timemachine list' to see snapshots")
	fmt.Println("  • Run 'timemachine restore <hash>' to restore a snapshot")

	return nil
}

// updateGitignore adds the timemachine_snapshots directory to .gitignore
// MUST preserve existing content and only append if not already present
func updateGitignore(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	
	// Read existing .gitignore content
	var existingContent []string
	var timemachineFound bool
	
	if file, err := os.Open(gitignorePath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		
		for scanner.Scan() {
			line := scanner.Text()
			existingContent = append(existingContent, line)
			
			// Check if already contains timemachine_snapshots
			if strings.Contains(line, "timemachine_snapshots") {
				timemachineFound = true
			}
		}
		
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read .gitignore: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	
	// If already contains timemachine_snapshots, nothing to do
	if timemachineFound {
		return nil
	}
	
	// Append Time Machine exclusion
	timemachineSection := []string{
		"",
		"# Time Machine shadow repository",
		".git/timemachine_snapshots/",
	}
	
	// Write updated .gitignore
	file, err := os.Create(gitignorePath)
	if err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	defer file.Close()
	
	writer := bufio.NewWriter(file)
	
	// Write existing content
	for _, line := range existingContent {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write existing content: %w", err)
		}
	}
	
	// Write Time Machine section
	for _, line := range timemachineSection {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write Time Machine section: %w", err)
		}
	}
	
	return writer.Flush()
}

// installPostPushHook installs or updates the post-push hook for automatic cleanup
// MUST preserve existing hook content and only append if not already present
func installPostPushHook(gitDir string) error {
	hookPath := filepath.Join(gitDir, "hooks", "post-push")
	
	// Create hooks directory if it doesn't exist
	hooksDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}
	
	// Read existing hook content
	var existingContent []string
	var timemachineFound bool
	
	if file, err := os.Open(hookPath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		
		for scanner.Scan() {
			line := scanner.Text()
			existingContent = append(existingContent, line)
			
			// Check if already contains timemachine command
			if strings.Contains(line, "timemachine clean") {
				timemachineFound = true
			}
		}
		
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read existing hook: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to open existing hook: %w", err)
	}
	
	// If already contains timemachine cleanup, nothing to do
	if timemachineFound {
		return nil
	}
	
	// Time Machine hook content
	timemachineHook := []string{
		"",
		"# Time Machine auto-cleanup",
		"if command -v timemachine >/dev/null 2>&1; then",
		"    timemachine clean --auto --quiet",
		"fi",
	}
	
	// Create or update the hook
	file, err := os.Create(hookPath)
	if err != nil {
		return fmt.Errorf("failed to create hook file: %w", err)
	}
	defer file.Close()
	
	writer := bufio.NewWriter(file)
	
	// If no existing content, add shebang
	if len(existingContent) == 0 {
		if _, err := writer.WriteString("#!/bin/sh\n"); err != nil {
			return fmt.Errorf("failed to write shebang: %w", err)
		}
	}
	
	// Write existing content
	for _, line := range existingContent {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write existing hook content: %w", err)
		}
	}
	
	// Write Time Machine hook
	for _, line := range timemachineHook {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write Time Machine hook: %w", err)
		}
	}
	
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush hook file: %w", err)
	}
	
	// Make hook executable
	if err := os.Chmod(hookPath, 0755); err != nil {
		return fmt.Errorf("failed to make hook executable: %w", err)
	}
	
	return nil
}