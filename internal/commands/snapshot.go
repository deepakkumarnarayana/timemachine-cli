package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// SnapshotCmd creates the snapshot command for manual snapshot creation
func SnapshotCmd() *cobra.Command {
	var messageFlag string

	cmd := &cobra.Command{
		Use:   "snapshot [message]",
		Short: "Create a manual snapshot of the current working directory",
		Long: `Create a manual snapshot of the current working directory state.

This command captures the current state of all files and creates a snapshot
in the shadow repository with the specified message. If no message is provided,
a timestamp-based message will be generated.

You can provide the message in two ways:
- As a positional argument: timemachine snapshot "message"
- Using the -m flag: timemachine snapshot -m "message"

Examples:
  timemachine snapshot "Fixed login bug"
  timemachine snapshot -m "Added validation logic" 
  timemachine snapshot   # Uses timestamp-based message`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := messageFlag
			
			// Positional argument takes precedence over flag
			if len(args) > 0 {
				message = args[0]
			}
			
			return runSnapshot(message)
		},
	}

	cmd.Flags().StringVarP(&messageFlag, "message", "m", "", "Snapshot message (alternative to positional argument)")

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
		return nil
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Show what we're about to snapshot
	fmt.Print("📸 Creating snapshot... ")

	// Create the snapshot
	err = gitManager.CreateSnapshot(message)
	if err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	color.Green("✅")
	fmt.Println()

	// Get current branch to show in success message
	currentBranch, _ := gitManager.GetCurrentBranch()
	color.Green("✨ Snapshot created successfully!")
	fmt.Printf("   Branch context: %s\n", currentBranch)
	
	if message != "" {
		fmt.Printf("   Message: %s\n", message)
	}
	
	fmt.Println()
	fmt.Println("💡 Use 'timemachine list' to see your snapshots")
	fmt.Println("   Use 'timemachine restore <hash>' to restore if needed")

	return nil
}