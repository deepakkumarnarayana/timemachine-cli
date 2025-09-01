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
	Long: `Time Machine creates automatic Git snapshots in a shadow repository
to provide instant rollback capabilities during AI-assisted coding sessions.
It watches for file changes and creates snapshots without affecting your
main Git repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		if version, _ := cmd.Flags().GetBool("version"); version {
			fmt.Printf("Time Machine CLI v%s\n", Version)
			return
		}
		fmt.Println("⏰ Time Machine CLI")
		fmt.Println("Use 'timemachine --help' for available commands")
	},
}

func init() {
	// Add version flag
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")
	
	// Add commands
	rootCmd.AddCommand(commands.InitCmd())
	rootCmd.AddCommand(commands.StartCmd())
	rootCmd.AddCommand(commands.ListCmd())
	rootCmd.AddCommand(commands.ShowCmd())
	rootCmd.AddCommand(commands.RestoreCmd())
	rootCmd.AddCommand(commands.StatusCmd())
	rootCmd.AddCommand(commands.CleanCmd())
}

func main() {
	// Test our core functionality
	state, err := core.NewAppState()
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Some commands may not work outside of a Git repository.")
	} else {
		fmt.Printf("Found Git repository at: %s\n", state.ProjectRoot)
		if state.IsInitialized {
			fmt.Println("Time Machine is initialized ✅")
		} else {
			fmt.Println("Time Machine not initialized. Run 'timemachine init' to get started.")
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}