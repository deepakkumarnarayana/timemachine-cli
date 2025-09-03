package commands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// ShowCmd creates the show command
func ShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <hash>",
		Short: "Show detailed information about a snapshot",
		Long: `Show detailed information about a specific snapshot including:
- Full commit hash
- Commit message  
- Author and timestamp
- Changed files`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args[0])
		},
	}
}

func runShow(hash string) error {
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

	// Get detailed commit information
	commitInfo, err := gitManager.RunCommand("show", "--pretty=fuller", "--name-status", hash)
	if err != nil {
		if strings.Contains(err.Error(), "bad object") || strings.Contains(err.Error(), "bad revision") {
			color.Red("❌ Snapshot not found!")
			fmt.Printf("   Hash '%s' does not exist.\n", hash)
			fmt.Println("   Use 'timemachine list' to see available snapshots.")
			return nil
		}
		return fmt.Errorf("failed to show snapshot details: %w", err)
	}

	// Get current branch context for display
	currentBranch, _, _, err := state.GetBranchContext()
	if err != nil {
		currentBranch = "unknown"
	}

	// Display the information with nice formatting
	fmt.Printf("📸 Snapshot Details (branch: %s)\n", currentBranch)
	fmt.Println()
	
	// Parse and format the git show output
	lines := strings.Split(commitInfo, "\n")
	inFileList := false
	
	for _, line := range lines {
		// Handle commit info section
		if strings.HasPrefix(line, "commit ") {
			color.Yellow("Commit:    %s", strings.TrimPrefix(line, "commit "))
		} else if strings.HasPrefix(line, "Author: ") {
			fmt.Printf("Author:    %s\n", strings.TrimPrefix(line, "Author: "))
		} else if strings.HasPrefix(line, "AuthorDate: ") {
			fmt.Printf("Date:      %s\n", strings.TrimPrefix(line, "AuthorDate: "))
		} else if strings.HasPrefix(line, "Commit: ") {
			fmt.Printf("Committer: %s\n", strings.TrimPrefix(line, "Commit: "))
		} else if strings.HasPrefix(line, "CommitDate: ") {
			fmt.Printf("Committed: %s\n", strings.TrimPrefix(line, "CommitDate: "))
		} else if line == "" && !inFileList {
			// Empty line before commit message
			fmt.Println()
		} else if !inFileList && !strings.HasPrefix(line, "commit ") && 
				  !strings.HasPrefix(line, "Author") && 
				  !strings.HasPrefix(line, "Commit") && 
				  !strings.HasPrefix(line, "    ") && 
				  line != "" {
			// This is likely the start of file status
			inFileList = true
			fmt.Println()
			color.Cyan("Changed Files:")
			formatFileStatus(line)
		} else if inFileList {
			if line == "" {
				continue
			}
			formatFileStatus(line)
		} else if strings.HasPrefix(line, "    ") {
			// Commit message (indented)
			message := strings.TrimPrefix(line, "    ")
			if message != "" {
				color.Green("Message:   %s", message)
				fmt.Println()
			}
		}
	}
	
	fmt.Println()
	fmt.Printf("Use 'timemachine restore %s' to restore this snapshot\n", hash)

	return nil
}

// formatFileStatus formats the file status output from git show --name-status
func formatFileStatus(line string) {
	if line == "" {
		return
	}
	
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return
	}
	
	status := parts[0]
	filename := strings.Join(parts[1:], " ")
	
	switch status {
	case "A":
		color.Green("  + %s (added)", filename)
	case "M":
		color.Yellow("  ~ %s (modified)", filename)
	case "D":
		color.Red("  - %s (deleted)", filename)
	case "R":
		if len(parts) >= 3 {
			color.Blue("  → %s → %s (renamed)", parts[1], parts[2])
		}
	case "C":
		if len(parts) >= 3 {
			color.Cyan("  ≈ %s → %s (copied)", parts[1], parts[2])
		}
	default:
		fmt.Printf("  %s %s\n", status, filename)
	}
}