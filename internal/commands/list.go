package commands

import (
	"fmt"

	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ListCmd creates the list command
func ListCmd() *cobra.Command {
	var (
		filePath   string
		limit      int
		enhanced   bool
		compact    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent snapshots with enhanced formatting",
		Long: `List recent snapshots from the Time Machine shadow repository with rich statistics.

The enhanced format shows file counts, line changes, and color-coded change magnitude.
You can filter snapshots by file and limit the number of results.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(filePath, limit, enhanced, compact)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Filter snapshots by file path")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Limit number of snapshots to show")
	cmd.Flags().BoolVar(&enhanced, "enhanced", true, "Use enhanced table format with statistics")
	cmd.Flags().BoolVar(&compact, "compact", false, "Use compact format (legacy)")

	return cmd
}

func runList(filePath string, limit int, enhanced bool, compact bool) error {
	// Handle flag conflicts
	if compact {
		enhanced = false
	}
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

	// Get snapshots
	snapshots, err := gitManager.ListSnapshots(limit, filePath)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Handle empty results
	if len(snapshots) == 0 {
		fmt.Println("📸 No snapshots found.")
		if filePath != "" {
			fmt.Printf("   Try without the --file filter or check if '%s' exists.\n", filePath)
		} else {
			fmt.Println("   Create your first snapshot by making changes to files.")
		}
		return nil
	}

	// Display header
	fmt.Println("📸 Recent snapshots:")
	fmt.Println()

	// Choose display format based on flags
	if enhanced && len(snapshots) > 0 {
		// Enhanced table output with rich formatting
		fmt.Printf("%-10s %-40s %-8s %-12s %-15s\n",
			"Hash", "Message", "Files", "Changes", "Time")
		fmt.Printf("%-10s %-40s %-8s %-12s %-15s\n",
			"────────", "──────────────────────────────────────", "─────", "──────────", "─────────────")

		for _, snapshot := range snapshots {
			// Truncate hash to 8 characters for display
			shortHash := snapshot.Hash
			if len(shortHash) > 8 {
				shortHash = shortHash[:8]
			}

			// Format changes as +added/-removed
			changes := fmt.Sprintf("+%d/-%d", snapshot.LinesAdded, snapshot.LinesRemoved)
			if snapshot.LinesAdded == 0 && snapshot.LinesRemoved == 0 {
				changes = "no changes"
			}

			// Colorize based on change magnitude
			var coloredChanges string
			totalChanges := snapshot.LinesAdded + snapshot.LinesRemoved
			if totalChanges > 500 {
				coloredChanges = color.RedString(changes) // Large changes
			} else if totalChanges > 100 {
				coloredChanges = color.YellowString(changes) // Medium changes
			} else {
				coloredChanges = color.GreenString(changes) // Small changes
			}

			fmt.Printf("%-10s %-40s %-8d %-12s %-15s\n",
				color.CyanString(shortHash),
				utils.TruncateString(snapshot.Message, 40),
				snapshot.FilesChanged,
				coloredChanges,
				snapshot.Time,
			)
		}
	} else if len(snapshots) > 0 {
		// Compact/legacy format for backwards compatibility
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
	}

	// Display summary with statistics
	fmt.Println()
	if len(snapshots) > 0 {
		if enhanced {
			// Calculate summary statistics for enhanced mode
			totalFiles := 0
			totalAdded := 0
			totalRemoved := 0
			for _, snapshot := range snapshots {
				totalFiles += snapshot.FilesChanged
				totalAdded += snapshot.LinesAdded
				totalRemoved += snapshot.LinesRemoved
			}

			if filePath != "" {
				fmt.Printf("Summary: %d snapshots for '%s' | %d files changed | +%d/-%d lines\n",
					len(snapshots), filePath, totalFiles, totalAdded, totalRemoved)
			} else {
				fmt.Printf("Summary: %d snapshots | %d files changed | +%d/-%d lines\n",
					len(snapshots), totalFiles, totalAdded, totalRemoved)
			}
		} else {
			// Simple summary for compact mode
			if filePath != "" {
				fmt.Printf("Total: %d snapshots for '%s'\n", len(snapshots), filePath)
			} else {
				fmt.Printf("Total: %d snapshots\n", len(snapshots))
			}
		}
	}
	fmt.Println()

	if enhanced {
		fmt.Println("💡 Tips:")
		fmt.Println("   • Use 'timemachine show <hash>' to see details")
		fmt.Println("   • Use 'timemachine restore <hash>' to restore a snapshot")
		fmt.Println("   • Use 'timemachine restore <hash> --interactive' for selective restore")
		fmt.Println("   • Use '--compact' for legacy format")
	} else {
		fmt.Println("Use 'timemachine show <hash>' to see details")
		fmt.Println("Use 'timemachine restore <hash>' to restore a snapshot")
	}

	return nil
}
