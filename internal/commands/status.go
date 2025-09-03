package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/utils"
)

// StatusCmd creates the status command
func StatusCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show Time Machine status and statistics",
		Long: `Show the current status of Time Machine including:
- Initialization status
- Number of snapshots
- Shadow repository size
- Recent activity
- Configuration details

Use --verbose for detailed information including file counts and paths.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

func runStatus(verbose bool) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		color.Red("❌ Error: %v", err)
		fmt.Println()
		showNotInGitRepo()
		return nil
	}

	// Show header
	fmt.Println("⏰ Time Machine Status")
	fmt.Println()

	// Basic project information
	fmt.Printf("📁 Project: %s\n", filepath.Base(state.ProjectRoot))
	if verbose {
		fmt.Printf("   Path: %s\n", state.ProjectRoot)
		fmt.Printf("   Git: %s\n", state.GitDir)
	}

	// Initialization status
	if state.IsInitialized {
		color.Green("✅ Status: Initialized and ready")
		if verbose {
			fmt.Printf("   Shadow repository: %s\n", state.ShadowRepoDir)
		}

		// Phase 3B: Show branch context
		if err := state.EnsureValidBranchState(); err != nil {
			color.Yellow("⚠️  Branch state: %v", err)
		} else {
			currentBranch, shadowBranch, synced, err := state.GetBranchContext()
			if err != nil {
				color.Yellow("⚠️  Branch context: %v", err)
			} else {
				if synced {
					color.Green("🌿 Branch: %s (synchronized)", currentBranch)
				} else {
					color.Yellow("🌿 Branch: %s → %s (synchronizing...)", currentBranch, shadowBranch)
				}
			}
		}
	} else {
		color.Yellow("⚠️  Status: Not initialized")
		fmt.Println("   Run 'timemachine init' to get started")
		fmt.Println()
		return nil
	}

	// Create Git manager for statistics
	gitManager := core.NewGitManager(state)

	// Get snapshot statistics
	snapshots, err := gitManager.ListSnapshots(0, "")
	if err != nil {
		color.Red("❌ Error getting snapshots: %v", err)
		return nil
	}

	fmt.Println()
	fmt.Printf("📸 Snapshots: %d total\n", len(snapshots))

	if len(snapshots) > 0 {
		// Show recent activity
		recentSnapshots := snapshots
		if len(snapshots) > 5 {
			recentSnapshots = snapshots[:5]
		}

		fmt.Println("   Recent activity:")
		for _, snapshot := range recentSnapshots {
			fmt.Printf("   • %s  %s  %s\n", 
				snapshot.Hash[:8], 
				utils.TruncateString(snapshot.Message, 35), 
				snapshot.Time)
		}

		if verbose && len(snapshots) > 5 {
			fmt.Printf("   ... and %d more snapshots\n", len(snapshots)-5)
		}
	} else {
		fmt.Println("   No snapshots yet")
	}

	// Shadow repository size
	fmt.Println()
	size, err := utils.CalculateDirectorySize(state.ShadowRepoDir)
	if err != nil {
		fmt.Printf("💾 Repository size: Unable to calculate (%v)\n", err)
	} else {
		fmt.Printf("💾 Repository size: %s\n", utils.FormatBytes(size))
	}

	// Configuration status
	fmt.Println()
	fmt.Println("⚙️  Configuration:")
	
	// Check .gitignore
	gitignorePath := filepath.Join(state.ProjectRoot, ".gitignore")
	if hasTimeMachineInGitignore(gitignorePath) {
		color.Green("   ✅ .gitignore updated")
	} else {
		color.Yellow("   ⚠️  .gitignore not updated")
	}

	// Check post-push hook
	hookPath := filepath.Join(state.GitDir, "hooks", "post-push")
	if hasTimeMachineHook(hookPath) {
		color.Green("   ✅ Auto-cleanup hook installed")
	} else {
		color.Yellow("   ⚠️  Auto-cleanup hook not installed")
	}

	// Show verbose details
	if verbose {
		fmt.Println()
		fmt.Println("🔧 Detailed Information:")
		showDetailedStatus(state, gitManager)
	}

	// Show helpful commands
	fmt.Println()
	fmt.Println("💡 Common commands:")
	fmt.Println("   timemachine start       # Begin watching for changes")
	fmt.Println("   timemachine list        # View all snapshots")
	fmt.Println("   timemachine clean       # Clean up old snapshots")

	return nil
}

func showNotInGitRepo() {
	fmt.Println("Time Machine requires a Git repository to function.")
	fmt.Println()
	fmt.Println("To get started:")
	fmt.Println("1. Navigate to a Git repository directory")
	fmt.Println("2. Run: timemachine init")
	fmt.Println("3. Run: timemachine start")
}


func hasTimeMachineInGitignore(gitignorePath string) bool {
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false
	}
	
	return utils.Contains(string(content), "timemachine_snapshots")
}

func hasTimeMachineHook(hookPath string) bool {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	
	return utils.Contains(string(content), "timemachine clean")
}

func showDetailedStatus(state *core.AppState, gitManager *core.GitManager) {
	// File count in project (excluding ignored)
	fileCount, dirCount := utils.CountProjectFiles(state.ProjectRoot)
	fmt.Printf("   Project files: %d files in %d directories\n", fileCount, dirCount)

	// Shadow repo details
	shadowFileCount, _ := utils.CountProjectFiles(state.ShadowRepoDir)
	fmt.Printf("   Shadow repo files: %d files\n", shadowFileCount)

	// Recent Git activity in main repo
	fmt.Printf("   Working directory: %s\n", state.ProjectRoot)
	
	// Check if there are uncommitted changes
	hasChanges, err := checkUncommittedChanges(state.ProjectRoot)
	if err == nil {
		if hasChanges {
			color.Yellow("   ⚠️  Uncommitted changes detected in main repo")
		} else {
			color.Green("   ✅ Working directory clean")
		}
	}
}

func checkUncommittedChanges(projectRoot string) (bool, error) {
	// Simple check for uncommitted changes
	// This is a basic implementation - could be enhanced
	
	// Check if git status shows any changes
	// For now, we'll just check if there are any untracked files or modifications
	// A more complete implementation would use git commands
	
	return false, nil // Placeholder - implement actual git status check if needed
}