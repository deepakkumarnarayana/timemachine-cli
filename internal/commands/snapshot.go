package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// SnapshotCmd creates the snapshot command for manual snapshot creation
func SnapshotCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Create a manual snapshot of the current state",
		Long: `Create a manual snapshot of the current working directory state.

This creates an immediate snapshot without waiting for file changes or debounce delays.
Useful for capturing important milestones or before making risky changes.

The snapshot will include all current files and their state, similar to automatic
snapshots created by the file watcher.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSnapshot(message)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&message, "message", "m", "", "Custom message for the snapshot")

	return cmd
}

func runSnapshot(message string) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	// Check if initialized
	if !state.IsInitialized {
		color.Red("❌ Time Machine is not initialized!")
		fmt.Println("Run 'timemachine init' to get started.")
		return fmt.Errorf("time machine not initialized")
	}

	// Ensure valid branch state with graceful degradation
	if err := state.EnsureValidBranchState(); err != nil {
		color.Yellow("⚠️  Warning: Branch state validation failed: %v", err)
		fmt.Println("   Continuing with snapshot creation. Some branch-specific features may not work correctly.")
		fmt.Println("   Try 'timemachine branch --reset' to reset branch state if issues persist.")
		fmt.Println()
	}

	// Get current branch context for enhanced messaging
	currentBranch, _, _, err := state.GetBranchContext()
	if err != nil {
		currentBranch = "unknown"
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Auto-sync working tree before creating snapshot (solves Issue 1: Branch conflicts)
	if err := gitManager.EnsureCleanWorkingTree(); err != nil {
		color.Yellow("⚠️  Warning: Auto-sync failed: %v", err)
		fmt.Println("   Continuing with snapshot creation. Some branch operations may conflict.")
	}

	// Create the snapshot
	fmt.Printf("📸 Creating manual snapshot on branch '%s'... ", currentBranch)
	
	if err := gitManager.CreateSnapshot(message); err != nil {
		color.Red("❌ Failed to create snapshot: %v", err)
		return fmt.Errorf("snapshot creation failed: %w", err)
	}

	// Get the latest snapshot to show details
	snapshots, err := gitManager.ListSnapshots(1, "")
	if err == nil && len(snapshots) > 0 {
		latest := snapshots[0]
		color.Green("✅ Success!")
		fmt.Printf("   Hash: %s\n", latest.Hash[:8])
		fmt.Printf("   Time: %s\n", latest.Time)
		if message != "" {
			fmt.Printf("   Message: %s\n", latest.Message)
		}
	} else {
		color.Green("✅ Snapshot created successfully!")
	}

	fmt.Println()
	fmt.Println("💡 Use 'timemachine list' to see all snapshots")
	fmt.Println("   Use 'timemachine show <hash>' for details")

	return nil
}