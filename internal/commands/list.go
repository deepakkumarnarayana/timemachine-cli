package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/utils"
)

// ListCmd creates the list command
func ListCmd() *cobra.Command {
	var (
		filePath    string
		limit       int
		allBranches bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent snapshots from current branch",
		Long: `List recent snapshots from the Time Machine shadow repository.

By default, shows snapshots from the current branch only. Use --all-branches
to see snapshots from all branches.

You can filter snapshots by file and limit the number of results.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(filePath, limit, allBranches)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Filter snapshots by file path")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Limit number of snapshots to show")
	cmd.Flags().BoolVar(&allBranches, "all-branches", false, "Show snapshots from all branches")

	return cmd
}

func runList(filePath string, limit int, allBranches bool) error {
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

	// Phase 3B: Ensure valid branch state
	if err := state.EnsureValidBranchState(); err != nil {
		return fmt.Errorf("branch state validation failed: %w", err)
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Get snapshots
	snapshots, err := gitManager.ListSnapshots(limit, filePath)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Get branch context for display
	currentBranch, _, _, err := state.GetBranchContext()
	if err != nil {
		currentBranch = "unknown"
	}

	// Handle empty results
	if len(snapshots) == 0 {
		if allBranches {
			fmt.Println("📸 No snapshots found across all branches.")
		} else {
			fmt.Printf("📸 No snapshots found on branch '%s'.\n", currentBranch)
		}
		if filePath != "" {
			fmt.Printf("   Try without the --file filter or check if '%s' exists.\n", filePath)
		} else {
			fmt.Println("   Create your first snapshot by making changes to files.")
		}
		return nil
	}

	// Display header with branch context
	if allBranches {
		fmt.Println("📸 Recent snapshots (all branches):")
	} else {
		fmt.Printf("📸 Recent snapshots (branch: %s):\n", currentBranch)
	}
	fmt.Println()

	// Simple table output without tablewriter for now
	for _, snapshot := range snapshots {
		// Truncate hash to 8 characters for display
		shortHash := snapshot.Hash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}
		
		// Format with consistent spacing
		fmt.Printf("%-10s  %-50s  %s\n", 
			shortHash, 
			utils.TruncateString(snapshot.Message, 50), 
			snapshot.Time,
		)
	}
	
	// Display summary
	fmt.Println()
	if filePath != "" {
		if allBranches {
			fmt.Printf("Total: %d snapshots for '%s' (all branches)\n", len(snapshots), filePath)
		} else {
			fmt.Printf("Total: %d snapshots for '%s' (branch: %s)\n", len(snapshots), filePath, currentBranch)
		}
	} else {
		if allBranches {
			fmt.Printf("Total: %d snapshots (all branches)\n", len(snapshots))
		} else {
			fmt.Printf("Total: %d snapshots (branch: %s)\n", len(snapshots), currentBranch)
		}
	}
	fmt.Println()
	fmt.Println("Use 'timemachine show <hash>' to see details")
	fmt.Println("Use 'timemachine restore <hash>' to restore a snapshot")
	
	// Show helpful hint about other branches (Phase 3C: Enhanced UX)
	if !allBranches {
		fmt.Println("Use 'timemachine list --all-branches' to see snapshots from all branches")
		if len(snapshots) == 0 {
			fmt.Println()
			color.Cyan("💡 Tip: If you recently switched branches, snapshots may be on other branches.")
			fmt.Println("   Try 'timemachine list --all-branches' to see your complete snapshot history.")
		}
	}

	return nil
}