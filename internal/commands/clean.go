package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/utils"
)

// CleanCmd creates the clean command
func CleanCmd() *cobra.Command {
	var (
		auto    bool
		quiet   bool
		keep    int
		olderThan string
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up snapshots to save disk space",
		Long: `Clean up Time Machine snapshots to save disk space.

By default, removes all snapshots after confirmation.
Use --keep to retain the N most recent snapshots.
Use --older-than to remove snapshots older than specified duration (e.g., "7d", "2w", "1m").

Examples:
  timemachine clean                    # Remove all snapshots (with confirmation)
  timemachine clean --auto            # Remove all snapshots (no confirmation)
  timemachine clean --keep 10         # Keep 10 most recent snapshots
  timemachine clean --older-than 1w   # Remove snapshots older than 1 week
  timemachine clean --auto --quiet    # Silent cleanup (used by post-push hook)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(auto, quiet, keep, olderThan)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&auto, "auto", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress output (useful for automation)")
	cmd.Flags().IntVar(&keep, "keep", 0, "Keep N most recent snapshots (0 = remove all)")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "Remove snapshots older than duration (e.g., 7d, 2w, 1m)")

	return cmd
}

func runClean(auto, quiet bool, keep int, olderThan string) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		if !quiet {
			return fmt.Errorf("failed to initialize app state: %w", err)
		}
		return nil // Silently fail in quiet mode
	}

	// Check if initialized
	if !state.IsInitialized {
		if !quiet {
			color.Red("❌ Time Machine is not initialized!")
			fmt.Println("Nothing to clean.")
		}
		return nil
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Get current snapshots before cleaning
	snapshots, err := gitManager.ListSnapshots(0, "")
	if err != nil {
		if !quiet {
			return fmt.Errorf("failed to list snapshots: %w", err)
		}
		return nil
	}

	if len(snapshots) == 0 {
		if !quiet {
			fmt.Println("📸 No snapshots found. Nothing to clean.")
		}
		return nil
	}

	// Determine what to clean
	var snapshotsToRemove []core.Snapshot
	var keepCount int

	if olderThan != "" {
		// Clean based on age
		snapshotsToRemove, keepCount, err = filterByAge(snapshots, olderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than format: %w", err)
		}
	} else if keep > 0 {
		// Keep N most recent
		if len(snapshots) > keep {
			snapshotsToRemove = snapshots[keep:] // Keep first N (most recent)
			keepCount = keep
		} else {
			keepCount = len(snapshots)
		}
	} else {
		// Remove all
		snapshotsToRemove = snapshots
		keepCount = 0
	}

	if len(snapshotsToRemove) == 0 {
		if !quiet {
			fmt.Printf("📸 All %d snapshots are within retention policy. Nothing to clean.\n", len(snapshots))
		}
		return nil
	}

	// Show what will be cleaned
	if !quiet {
		fmt.Println("🧹 Time Machine Cleanup")
		fmt.Println()
		fmt.Printf("Total snapshots: %d\n", len(snapshots))
		fmt.Printf("Will remove: %d snapshots\n", len(snapshotsToRemove))
		fmt.Printf("Will keep: %d snapshots\n", keepCount)

		if len(snapshotsToRemove) <= 5 {
			// Show all snapshots to be removed if not too many
			fmt.Println("\nSnapshots to remove:")
			for _, snapshot := range snapshotsToRemove {
				fmt.Printf("  • %s  %s  %s\n", 
					snapshot.Hash[:8], 
					utils.TruncateString(snapshot.Message, 40), 
					snapshot.Time)
			}
		} else {
			// Show sample if many snapshots
			fmt.Printf("\nOldest snapshots to remove (showing first 3 of %d):\n", len(snapshotsToRemove))
			for i, snapshot := range snapshotsToRemove[:3] {
				if i >= 3 {
					break
				}
				fmt.Printf("  • %s  %s  %s\n", 
					snapshot.Hash[:8], 
					utils.TruncateString(snapshot.Message, 40), 
					snapshot.Time)
			}
		}
		fmt.Println()
	}

	// Ask for confirmation unless --auto
	if !auto && !quiet {
		fmt.Print("Do you want to continue? (y/N): ")
		
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cleanup cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Perform cleanup
	if !quiet {
		fmt.Print("🧹 Cleaning up snapshots... ")
	}

	if keep == 0 && olderThan == "" {
		// Remove entire shadow repository for complete cleanup
		err = os.RemoveAll(state.ShadowRepoDir)
		if err != nil {
			if !quiet {
				color.Red("❌")
			}
			return fmt.Errorf("failed to remove shadow repository: %w", err)
		}
		
		// Update state
		state.IsInitialized = false
	} else {
		// Remove specific commits (more complex, but preserves repository)
		// For now, we'll use the simple approach of recreating with kept snapshots
		err = cleanupSelectiveSnapshots(gitManager, snapshotsToRemove, keepCount)
		if err != nil {
			if !quiet {
				color.Red("❌")
			}
			return fmt.Errorf("failed to cleanup snapshots: %w", err)
		}
	}

	if !quiet {
		color.Green("✅")
		fmt.Println()
		
		if keep == 0 && olderThan == "" {
			color.Green("✨ All snapshots removed successfully!")
			fmt.Println("   Run 'timemachine init' to reinitialize if needed.")
		} else {
			color.Green("✨ Cleanup completed successfully!")
			fmt.Printf("   Removed %d snapshots, kept %d snapshots.\n", len(snapshotsToRemove), keepCount)
		}
	}

	return nil
}

// filterByAge filters snapshots based on age
func filterByAge(snapshots []core.Snapshot, olderThan string) ([]core.Snapshot, int, error) {
	// Parse duration (simplified - could be enhanced)
	duration, err := parseDuration(olderThan)
	if err != nil {
		return nil, 0, err
	}
	
	// For now, use simple heuristic based on relative time
	// In a real implementation, we'd parse the actual commit timestamps
	var toRemove []core.Snapshot
	var toKeep int
	
	for _, snapshot := range snapshots {
		// Simple heuristic: if relative time suggests it's old, remove it
		if isOlderThan(snapshot.Time, duration) {
			toRemove = append(toRemove, snapshot)
		} else {
			toKeep++
		}
	}
	
	return toRemove, toKeep, nil
}

// parseDuration parses duration strings like "7d", "2w", "1m"
func parseDuration(s string) (int, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short")
	}
	
	numStr := s[:len(s)-1]
	unit := s[len(s)-1:]
	
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}
	
	switch unit {
	case "d":
		return num, nil // days
	case "w":
		return num * 7, nil // weeks to days
	case "m":
		return num * 30, nil // months to days (approximate)
	default:
		return 0, fmt.Errorf("unsupported unit: %s (use d, w, or m)", unit)
	}
}

// isOlderThan checks if a relative time string suggests the snapshot is older than specified days
func isOlderThan(timeStr string, days int) bool {
	// Simple heuristic based on common time formats
	if strings.Contains(timeStr, "month") || strings.Contains(timeStr, "year") {
		return days <= 30 // If looking for anything older than a month, include month+ old items
	}
	if strings.Contains(timeStr, "week") && days <= 7 {
		return true
	}
	if strings.Contains(timeStr, "day") && days <= 1 {
		return true
	}
	return false
}

// cleanupSelectiveSnapshots removes specific snapshots while preserving others
func cleanupSelectiveSnapshots(gitManager *core.GitManager, toRemove []core.Snapshot, keepCount int) error {
	// For now, implement simple approach - in production might use git rebase/filter-branch
	// This is a placeholder for more sophisticated selective cleanup
	
	if keepCount == 0 {
		// If keeping nothing, just remove the whole repository
		return os.RemoveAll(gitManager.State.ShadowRepoDir)
	}
	
	// For selective removal, we'd need more complex Git operations
	// For MVP, we'll just warn that this is not yet implemented
	return fmt.Errorf("selective snapshot cleanup not yet implemented - use --keep 0 for complete cleanup")
}

