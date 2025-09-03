package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// BranchCmd creates the branch command for managing TimeMachine branch state (Phase 4: CLI Safety)
func BranchCmd() *cobra.Command {
	var verbose bool
	var reset bool

	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Show TimeMachine branch status and context",
		Long: `Show detailed information about TimeMachine's shadow branching system.

This command displays:
- Current Git branch and shadow branch mapping
- Branch synchronization status  
- Shadow repository branch structure
- Recent branch switches and their impact

Use --verbose for detailed shadow repository information.
Use --reset to force reset branch state when stuck after failed branch operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBranchStatus(verbose, reset)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed branch and repository information")
	cmd.Flags().BoolVar(&reset, "reset", false, "Force reset branch state (use when stuck after failed branch operations)")

	return cmd
}

func runBranchStatus(verbose bool, reset bool) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	// Check if initialized
	if !state.IsInitialized {
		color.Red("❌ TimeMachine is not initialized!")
		fmt.Println("Run 'timemachine init' to get started.")
		return nil
	}

	// Handle reset flag - force reset branch state
	if reset {
		fmt.Println("🔄 Resetting branch state...")
		state.ForceResetBranchState()
		color.Green("✅ Branch state reset successfully!")
		fmt.Println("   You can now retry TimeMachine operations.")
		fmt.Println("   Next command will automatically resynchronize the branch state.")
		return nil
	}

	// Ensure valid branch state with error recovery guidance
	if err := state.EnsureValidBranchState(); err != nil {
		color.Red("❌ Branch state error: %v", err)
		fmt.Println()
		fmt.Println("💡 Try these troubleshooting steps:")
		fmt.Println("   1. Run 'timemachine branch --reset' to reset the branch state")
		fmt.Println("   2. Run 'git status' to check your repository state")
		fmt.Println("   3. Ensure you're in a valid Git repository")
		fmt.Println("   4. Check that TimeMachine was properly initialized")
		return nil
	}

	// Get branch context
	currentBranch, shadowBranch, synced, err := state.GetBranchContext()
	if err != nil {
		return fmt.Errorf("failed to get branch context: %w", err)
	}

	// Display header
	fmt.Println("🌿 TimeMachine Branch Status")
	fmt.Println()

	// Current branch information
	color.Green("📋 Current Context:")
	fmt.Printf("   Git Branch:    %s\n", currentBranch)
	fmt.Printf("   Shadow Branch: %s\n", shadowBranch)
	
	if synced {
		color.Green("   Status:        ✅ Synchronized")
		fmt.Println("   Your TimeMachine snapshots are properly isolated for this branch.")
	} else {
		color.Yellow("   Status:        🔄 Synchronizing...")
		fmt.Println("   TimeMachine is creating/switching to the shadow branch for this Git branch.")
	}

	fmt.Println()

	// Create Git manager for additional information
	gitManager := core.NewGitManager(state)

	// Current branch snapshots
	snapshots, err := gitManager.ListSnapshots(5, "")
	if err != nil {
		color.Yellow("⚠️  Could not get snapshot information: %v", err)
	} else {
		color.Cyan("📸 Recent Snapshots (current branch):")
		if len(snapshots) == 0 {
			fmt.Println("   No snapshots found on this branch")
			fmt.Println("   Start watching with 'timemachine start' to create snapshots")
		} else {
			for i, snapshot := range snapshots {
				if i >= 3 && !verbose { // Show only 3 unless verbose
					break
				}
				fmt.Printf("   • %s  %s  %s\n", 
					snapshot.Hash[:8], 
					truncateMessage(snapshot.Message, 40), 
					snapshot.Time)
			}
			if len(snapshots) > 3 && !verbose {
				fmt.Printf("   ... and %d more snapshots (use --verbose to see all)\n", len(snapshots)-3)
			}
		}
	}

	fmt.Println()

	// Show helpful commands
	color.Cyan("💡 Branch-Related Commands:")
	fmt.Println("   timemachine list                    # Show snapshots for current branch")
	fmt.Println("   timemachine list --all-branches     # Show snapshots from all branches")
	fmt.Println("   timemachine status                  # General TimeMachine status")
	fmt.Println("   timemachine start                   # Start watching (detects branch switches)")

	// Verbose information
	if verbose {
		fmt.Println()
		color.Cyan("🔧 Technical Details:")
		fmt.Printf("   Project Root:      %s\n", state.ProjectRoot)
		fmt.Printf("   Git Directory:     %s\n", state.GitDir)
		fmt.Printf("   Shadow Repository: %s\n", state.ShadowRepoDir)
		
		// Note: Branch cache information is internal and not exposed for security
	}

	return nil
}

// truncateMessage truncates a commit message to specified length with ellipsis
func truncateMessage(message string, maxLen int) string {
	if len(message) <= maxLen {
		return message
	}
	return message[:maxLen-3] + "..."
}