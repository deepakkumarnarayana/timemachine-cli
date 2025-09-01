package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// ListCmd creates the list command
func ListCmd() *cobra.Command {
	var (
		filePath string
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent snapshots",
		Long: `List recent snapshots from the Time Machine shadow repository.

You can filter snapshots by file and limit the number of results.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(filePath, limit)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Filter snapshots by file path")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Limit number of snapshots to show")

	return cmd
}

func runList(filePath string, limit int) error {
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
			truncateString(snapshot.Message, 50), 
			snapshot.Time,
		)
	}
	
	// Display summary
	fmt.Println()
	if filePath != "" {
		fmt.Printf("Total: %d snapshots for '%s'\n", len(snapshots), filePath)
	} else {
		fmt.Printf("Total: %d snapshots\n", len(snapshots))
	}
	fmt.Println()
	fmt.Println("Use 'timemachine show <hash>' to see details")
	fmt.Println("Use 'timemachine restore <hash>' to restore a snapshot")

	return nil
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}