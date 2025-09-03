package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// StartCmd creates the start command
func StartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start watching for file changes and creating automatic snapshots",
		Long: `Start the Time Machine file watcher to automatically create snapshots
when files change. This runs in the foreground and will continue until
you press Ctrl+C.

The watcher:
- Monitors all files in the project recursively
- Ignores common build/cache directories (node_modules, dist, .git, etc.)
- Groups rapid changes together to prevent snapshot spam
- Creates snapshots with 500ms debounce delay`,
		RunE: runStart,
	}
}

func runStart(cmd *cobra.Command, args []string) error {
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

	// Phase 3B: Pre-flight branch state validation with recovery guidance
	fmt.Print("🔍 Validating branch state... ")
	if err := state.EnsureValidBranchState(); err != nil {
		color.Red("❌")
		fmt.Println()
		color.Red("Branch state validation failed: %v", err)
		fmt.Println()
		fmt.Println("💡 To fix this issue:")
		fmt.Println("   1. Run 'timemachine branch --reset' to reset the branch state")
		fmt.Println("   2. Then retry 'timemachine start'")
		fmt.Println("   3. If the problem persists, check 'git status' and ensure you're in a valid Git repository")
		return nil
	}
	color.Green("✅")

	// Display current branch context for user awareness
	currentBranch, shadowBranch, synced, err := state.GetBranchContext()
	if err != nil {
		fmt.Printf("Warning: failed to get branch context: %v\n", err)
	} else {
		if synced {
			color.Cyan("📋 Branch: %s (synchronized)", currentBranch)
		} else {
			color.Yellow("📋 Branch: %s → %s (synchronizing...)", currentBranch, shadowBranch)
		}
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Create watcher
	watcher, err := core.NewWatcher(state, gitManager)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watcher in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := watcher.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("\n🛑 Received %v signal, stopping watcher...\n", sig)
		watcher.Stop()
		fmt.Println("✅ Time Machine stopped gracefully")
		return nil
		
	case err := <-errChan:
		watcher.Stop()
		return fmt.Errorf("watcher error: %w", err)
	}
}