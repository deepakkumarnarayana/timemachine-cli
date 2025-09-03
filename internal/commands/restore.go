package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// RestoreCmd creates the restore command
func RestoreCmd() *cobra.Command {
	var (
		files []string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "restore <hash>",
		Short: "Restore files from a snapshot",
		Long: `Restore files from a specific snapshot to the working directory.

By default, this restores all files from the snapshot. You can specify
specific files to restore using the --files flag.

IMPORTANT: This only affects the working directory, not the Git staging area.
Your Git history and staged changes are preserved.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args[0], files, force)
		},
	}

	// Add flags
	cmd.Flags().StringSliceVar(&files, "files", []string{}, "Specific files to restore (comma-separated)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runRestore(hash string, files []string, force bool) error {
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

	// Phase 3B: Ensure valid branch state with graceful degradation
	if err := state.EnsureValidBranchState(); err != nil {
		color.Yellow("⚠️  Warning: Branch state validation failed: %v", err)
		fmt.Println("   Continuing with restore operation. Some branch-specific features may not work correctly.")
		fmt.Println("   Try 'timemachine branch --reset' to reset branch state if issues persist.")
		fmt.Println()
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Verify the hash exists
	_, err = gitManager.RunCommand("rev-parse", "--verify", hash+"^{commit}")
	if err != nil {
		color.Red("❌ Snapshot not found!")
		fmt.Printf("   Hash '%s' does not exist.\n", hash)
		fmt.Println("   Use 'timemachine list' to see available snapshots.")
		return nil
	}

	// Get snapshot details for confirmation
	snapshots, err := gitManager.ListSnapshots(0, "")
	if err != nil {
		return fmt.Errorf("failed to get snapshot info: %w", err)
	}

	var targetSnapshot *core.Snapshot
	for _, snapshot := range snapshots {
		if strings.HasPrefix(snapshot.Hash, hash) || snapshot.Hash == hash {
			targetSnapshot = &snapshot
			break
		}
	}

	if targetSnapshot == nil {
		color.Red("❌ Could not find snapshot details!")
		fmt.Println()
		color.Yellow("💡 Tip: This snapshot may be from a different branch.")
		fmt.Println("   Try 'timemachine list --all-branches' to see snapshots from all branches.")
		return nil
	}

	// Phase 4: Cross-branch safety check
	if err := performCrossBranchSafetyCheck(state, gitManager, hash); err != nil {
		color.Yellow("⚠️  Cross-Branch Safety Warning:")
		fmt.Printf("   %v\n", err)
		fmt.Println()
	}

	// Get current branch context for display
	currentBranch, _, _, err := state.GetBranchContext()
	if err != nil {
		currentBranch = "unknown"
	}

	// Show what will be restored with enhanced branch context (Phase 3C: Enhanced UX)
	fmt.Println("📸 Restore Snapshot")
	fmt.Println()
	fmt.Printf("Hash:    %s\n", targetSnapshot.Hash[:8])
	fmt.Printf("Message: %s\n", targetSnapshot.Message)
	fmt.Printf("Time:    %s\n", targetSnapshot.Time)
	fmt.Printf("Branch:  %s\n", currentBranch)
	fmt.Println()

	// Enhanced branch context warning
	_, shadowBranch, synced, err := state.GetBranchContext()
	if err == nil && !synced {
		color.Yellow("⚠️  Branch Context Warning:")
		fmt.Printf("   You're restoring while branches are synchronizing (%s → %s)\n", currentBranch, shadowBranch)
		fmt.Println("   Consider waiting for synchronization to complete for best results.")
		fmt.Println()
	}

	if len(files) == 0 {
		color.Yellow("⚠️  This will restore ALL files from this snapshot")
		fmt.Println("   Any uncommitted changes in your working directory will be lost!")
	} else {
		color.Yellow("⚠️  This will restore the following files:")
		for _, file := range files {
			fmt.Printf("   • %s\n", file)
		}
		fmt.Println("   Any uncommitted changes to these files will be lost!")
	}

	fmt.Println()
	color.Cyan("ℹ️  Note: This only affects your working directory.")
	fmt.Println("   Your Git staging area and commit history remain unchanged.")

	// Ask for confirmation unless --force is used
	if !force {
		fmt.Println()
		fmt.Print("Do you want to continue? (y/N): ")
		
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Perform the restore
	fmt.Println()
	fmt.Print("🔄 Restoring files... ")
	
	err = gitManager.RestoreSnapshot(targetSnapshot.Hash, files)
	if err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}
	
	color.Green("✅")
	fmt.Println()
	
	if len(files) == 0 {
		color.Green("✨ All files restored successfully!")
	} else {
		color.Green("✨ Files restored successfully!")
	}
	
	fmt.Println()
	fmt.Println("📝 Reminder:")
	fmt.Println("   • Changes are in your working directory only")
	fmt.Println("   • Use 'git add' and 'git commit' if you want to save these changes")
	fmt.Println("   • Use 'git status' to see what changed")

	return nil
}

// performCrossBranchSafetyCheck checks if restoring might have cross-branch implications (Phase 4: CLI Safety)
func performCrossBranchSafetyCheck(state *core.AppState, gitManager *core.GitManager, hash string) error {
	// Get current branch
	currentBranch, _, _, err := state.GetBranchContext()
	if err != nil {
		return nil // Skip check if we can't determine branch context
	}

	// Check if snapshot exists on current branch
	snapshots, err := gitManager.ListSnapshots(0, "")
	if err != nil {
		return nil // Skip check if we can't list snapshots
	}

	// Look for the hash in current branch snapshots
	foundOnCurrentBranch := false
	for _, snapshot := range snapshots {
		if strings.HasPrefix(snapshot.Hash, hash) || snapshot.Hash == hash {
			foundOnCurrentBranch = true
			break
		}
	}

	if !foundOnCurrentBranch {
		return fmt.Errorf("snapshot '%s' may be from a different branch than '%s'.\n"+
			"   This could restore files that don't belong to your current branch context.\n"+
			"   Consider using 'timemachine list --all-branches' to verify the snapshot's origin", 
			hash[:8], currentBranch)
	}

	return nil
}