package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/commands"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

const Version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:     "timemachine",
	Version: Version,
	Short:   "Time Machine - Automatic Git snapshots for AI-assisted development",
	Long: `⏰ Time Machine - AI-Safe Development Tool

Time Machine creates automatic Git snapshots in a shadow repository to provide
instant rollback capabilities during AI-assisted coding sessions. It monitors
your codebase and creates snapshots without affecting your main Git workflow.

🔧 QUICK START:
  timemachine init     # Initialize in your Git repository
  timemachine start    # Start watching for changes
  timemachine list     # View recent snapshots
  timemachine restore  # Rollback when needed

💡 KEY FEATURES:
  • Shadow repository isolation (no main Git interference)  
  • Real-time file watching with smart debouncing
  • Cross-platform path security and validation
  • Production-ready configuration management
  • Comprehensive snapshot inspection and recovery

📚 COMMON WORKFLOWS:
  1. AI Coding Session:
     timemachine start → AI makes changes → timemachine restore <hash>
  
  2. Configuration Management:  
     timemachine config show → timemachine config set log.level debug
     
  3. Snapshot Analysis:
     timemachine list → timemachine inspect <hash> --diff --verbose`,
	Run: func(cmd *cobra.Command, args []string) {
		if version, _ := cmd.Flags().GetBool("version"); version {
			fmt.Printf("Time Machine CLI v%s\n", Version)
			return
		}
		
		// Show enhanced help with current status
		state, err := core.NewAppState()
		if err != nil {
			fmt.Printf("⚠️  Warning: %v\n", err)
			fmt.Println("   Some commands may not work outside of a Git repository.")
		} else {
			fmt.Printf("📂 Git Repository: %s\n", state.ProjectRoot)
			if state.IsInitialized {
				fmt.Println("✅ Time Machine: Initialized and ready")
			} else {
				fmt.Println("❌ Time Machine: Not initialized")
				fmt.Println("   Run 'timemachine init' to get started")
			}
			fmt.Println()
		}
		
		fmt.Println("Use 'timemachine --help' for detailed command information")
		fmt.Println("Use 'timemachine <command> --help' for specific command help")
	},
}

func init() {
	// Add version flag
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")
	
	// Add commands in logical order
	rootCmd.AddCommand(commands.InitCmd())      // Setup
	rootCmd.AddCommand(commands.ConfigCmd())    // Configuration  
	rootCmd.AddCommand(commands.StartCmd())     // Core functionality
	rootCmd.AddCommand(commands.ListCmd())      // Inspection
	rootCmd.AddCommand(commands.ShowCmd())      // Inspection
	rootCmd.AddCommand(commands.InspectCmd())   // Inspection
	rootCmd.AddCommand(commands.RestoreCmd())   // Recovery
	rootCmd.AddCommand(commands.StatusCmd())    // Status
	rootCmd.AddCommand(commands.BranchCmd())    // Branch management (Phase 4)
	rootCmd.AddCommand(commands.CleanCmd())     // Maintenance
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}