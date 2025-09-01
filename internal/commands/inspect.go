package commands

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// validateGitHash ensures git hash is safe for use in commands
func validateGitHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("empty hash not allowed")
	}
	// Only allow alphanumeric characters and ensure reasonable length (4-40 chars for git hashes)
	matched, err := regexp.MatchString("^[a-fA-F0-9]{4,40}$", hash)
	if err != nil {
		return fmt.Errorf("regex validation failed: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid git hash format: must be 4-40 hexadecimal characters")
	}
	return nil
}

// sanitizeFilePath prevents path traversal attacks
func sanitizeFilePath(path string) (string, error) {
	if path == "" {
		return "", nil // Empty path is allowed for no filter
	}
	
	// Prevent directory traversal
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}
	
	// Prevent absolute paths
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths not allowed")
	}
	
	// Clean and normalize the path
	cleaned := filepath.Clean(path)
	
	// Additional safety: ensure it doesn't start with / after cleaning
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("path must be relative")
	}
	
	return cleaned, nil
}

// InspectCmd creates the inspect command
func InspectCmd() *cobra.Command {
	var (
		showDiff   bool
		showStats  bool
		fileFilter string
		verbose    bool
		searchAll  bool
	)

	cmd := &cobra.Command{
		Use:   "inspect [snapshot-hash]",
		Short: "Inspect what changed in snapshots",
		Long: `Inspect and analyze snapshot contents to see exactly what files were changed.

Examples:
  timemachine inspect                    # Show latest snapshot changes
  timemachine inspect abc123def         # Show specific snapshot by hash
  timemachine inspect --diff            # Show detailed line-by-line changes
  timemachine inspect --stats           # Show repository statistics
  timemachine inspect --file=main.go    # Show changes only for specific file
  timemachine inspect --verbose         # Show comprehensive analysis
  timemachine inspect --search-all --file=main.go  # Search all snapshots for changes to main.go`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(cmd, args, showDiff, showStats, fileFilter, verbose, searchAll)
		},
	}

	cmd.Flags().BoolVarP(&showDiff, "diff", "d", false, "Show detailed line-by-line differences")
	cmd.Flags().BoolVarP(&showStats, "stats", "s", false, "Show repository storage statistics")
	cmd.Flags().StringVarP(&fileFilter, "file", "f", "", "Filter changes to specific file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show comprehensive analysis")
	cmd.Flags().BoolVarP(&searchAll, "search-all", "a", false, "Search all snapshots for file changes")

	return cmd
}

func runInspect(cmd *cobra.Command, args []string, showDiff, showStats bool, fileFilter string, verbose, searchAll bool) error {
	// Validate and sanitize file filter input
	sanitizedFileFilter, err := sanitizeFilePath(fileFilter)
	if err != nil {
		return fmt.Errorf("invalid file filter: %w", err)
	}
	fileFilter = sanitizedFileFilter

	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	if !state.IsInitialized {
		color.Red("❌ Time Machine is not initialized")
		fmt.Println("Run 'timemachine init' first to initialize the shadow repository.")
		return nil
	}

	// Create Git manager
	gitManager := core.NewGitManager(state)

	// Show repository statistics if requested
	if showStats {
		if err := showRepositoryStats(state); err != nil {
			return fmt.Errorf("failed to show repository stats: %w", err)
		}
		fmt.Println()
	}

	// Handle search-all mode
	if searchAll {
		return runSearchAllSnapshots(state, fileFilter, showDiff, verbose)
	}

	// Determine which snapshot to inspect
	var targetHash string
	if len(args) > 0 {
		targetHash = args[0]
		// Validate user-provided hash for security
		if err := validateGitHash(targetHash); err != nil {
			return fmt.Errorf("invalid snapshot hash: %w", err)
		}
	} else {
		// Get latest snapshot
		snapshots, err := gitManager.ListSnapshots(1, "")
		if err != nil {
			return fmt.Errorf("failed to get snapshots: %w", err)
		}
		if len(snapshots) == 0 {
			color.Yellow("📝 No snapshots found")
			return nil
		}
		targetHash = snapshots[0].Hash
		// Internal hashes from ListSnapshots are trusted, but validate anyway for defense in depth
		if err := validateGitHash(targetHash); err != nil {
			return fmt.Errorf("internal hash validation failed: %w", err)
		}
	}

	// Validate the hash exists
	if !isValidHash(state, targetHash) {
		return fmt.Errorf("snapshot hash '%s' not found", targetHash)
	}

	// Show snapshot overview
	if err := showSnapshotOverview(state, targetHash); err != nil {
		return fmt.Errorf("failed to show snapshot overview: %w", err)
	}

	// Show file changes
	if err := showFileChanges(state, targetHash, fileFilter); err != nil {
		return fmt.Errorf("failed to show file changes: %w", err)
	}

	// Show deleted file contents if any
	if err := showDeletedFiles(state, targetHash, fileFilter); err != nil {
		return fmt.Errorf("failed to show deleted files: %w", err)
	}

	// Show detailed diff if requested
	if showDiff || verbose {
		if err := showDetailedDiff(state, targetHash, fileFilter); err != nil {
			return fmt.Errorf("failed to show detailed diff: %w", err)
		}
	}

	// Show comprehensive analysis if verbose
	if verbose {
		if err := showComprehensiveAnalysis(state, targetHash); err != nil {
			return fmt.Errorf("failed to show comprehensive analysis: %w", err)
		}
	}

	return nil
}

func showRepositoryStats(state *core.AppState) error {
	color.Cyan("🗄️  Repository Statistics")
	color.Cyan("========================")

	// Repository size
	cmd := exec.Command("du", "-sh", state.ShadowRepoDir)
	if sizeOutput, err := cmd.Output(); err == nil {
		fmt.Printf("Repository size: %s", string(sizeOutput))
	}

	// Object count and storage details
	cmd = exec.Command("git", "--git-dir="+state.ShadowRepoDir, "count-objects", "-v")
	if objectOutput, err := cmd.Output(); err == nil {
		lines := strings.Split(string(objectOutput), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Total commits
	cmd = exec.Command("git", "--git-dir="+state.ShadowRepoDir, "rev-list", "--count", "HEAD")
	if countOutput, err := cmd.Output(); err == nil {
		count := strings.TrimSpace(string(countOutput))
		fmt.Printf("  total-snapshots: %s\n", count)
	}

	return nil
}

func showSnapshotOverview(state *core.AppState, hash string) error {
	color.Green("🔍 Snapshot Overview")
	fmt.Printf("Hash: %s\n", hash)

	// Get commit info
	cmd := exec.Command("git", "--git-dir="+state.ShadowRepoDir, "--work-tree="+state.ProjectRoot,
		"show", "--no-patch", "--format=%an%n%ad%n%s", hash)
	
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) >= 3 {
			fmt.Printf("Author: %s\n", lines[0])
			fmt.Printf("Date: %s\n", lines[1])
			fmt.Printf("Message: %s\n", lines[2])
		}
	}
	fmt.Println()

	return nil
}

func showFileChanges(state *core.AppState, hash string, fileFilter string) error {
	color.Blue("📝 File Changes")
	color.Blue("===============")

	// Build command args
	args := []string{"--git-dir=" + state.ShadowRepoDir, "--work-tree=" + state.ProjectRoot,
		"show", "--name-status", hash}
	
	if fileFilter != "" {
		args = append(args, "--", fileFilter)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get file changes: %w", err)
	}

	// Parse and display file changes
	lines := strings.Split(string(output), "\n")
	fileCount := 0
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "commit ") || 
		   strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") ||
		   strings.Contains(line, "Snapshot at") {
			continue
		}

		// Parse status and filename
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			status := parts[0]
			filename := parts[1]
			fileCount++

			// Color-code the status
			var statusColor *color.Color
			var statusText string
			switch status {
			case "A":
				statusColor = color.New(color.FgGreen)
				statusText = "Added"
			case "M":
				statusColor = color.New(color.FgYellow) 
				statusText = "Modified"
			case "D":
				statusColor = color.New(color.FgRed)
				statusText = "Deleted"
			case "R":
				statusColor = color.New(color.FgBlue)
				statusText = "Renamed"
			default:
				statusColor = color.New(color.FgWhite)
				statusText = status
			}

			statusColor.Printf("  %s", statusText)
			fmt.Printf("\t%s\n", filename)
		}
	}

	if fileCount == 0 {
		color.Yellow("  No file changes found")
		if fileFilter != "" {
			fmt.Printf("  (filtered for: %s)\n", fileFilter)
		}
	} else {
		fmt.Printf("\nTotal files changed: %d\n", fileCount)
	}
	fmt.Println()

	return nil
}

func showDeletedFiles(state *core.AppState, hash string, fileFilter string) error {
	// Get list of deleted files
	args := []string{"--git-dir=" + state.ShadowRepoDir, "--work-tree=" + state.ProjectRoot,
		"show", "--name-status", hash}
	
	if fileFilter != "" {
		args = append(args, "--", fileFilter)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get file changes: %w", err)
	}

	// Find deleted files
	deletedFiles := []string{}
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "commit ") || 
		   strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") ||
		   strings.Contains(line, "Snapshot at") {
			continue
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 && parts[0] == "D" {
			deletedFiles = append(deletedFiles, parts[1])
		}
	}

	if len(deletedFiles) == 0 {
		return nil
	}

	color.Red("🗑️  Deleted File Contents")
	color.Red("========================")

	for _, filename := range deletedFiles {
		// Get the parent commit to show what the file contained before deletion
		parentCmd := exec.Command("git", "--git-dir="+state.ShadowRepoDir, "show", "--format=%P", "--no-patch", hash)
		parentOutput, err := parentCmd.Output()
		if err != nil {
			continue
		}
		
		parent := strings.TrimSpace(string(parentOutput))
		if parent == "" {
			continue
		}

		// Show file content from parent commit
		fileCmd := exec.Command("git", "--git-dir="+state.ShadowRepoDir, "show", parent+":"+filename)
		fileContent, err := fileCmd.Output()
		if err != nil {
			color.Yellow(fmt.Sprintf("⚠️  Could not retrieve content of deleted file: %s", filename))
			continue
		}

		color.Cyan(fmt.Sprintf("📄 File: %s (before deletion)", filename))
		color.Cyan(strings.Repeat("-", len(filename)+25))
		
		// Show file contents with line numbers
		contentLines := strings.Split(string(fileContent), "\n")
		for i, contentLine := range contentLines {
			if i < len(contentLines)-1 || contentLine != "" { // Skip last empty line
				color.New(color.FgYellow).Printf("%4d: ", i+1)
				fmt.Println(contentLine)
			}
		}
		fmt.Println()
	}

	return nil
}

func showDetailedDiff(state *core.AppState, hash string, fileFilter string) error {
	color.Magenta("📋 Detailed Changes")
	color.Magenta("===================")

	// Build command args
	args := []string{"--git-dir=" + state.ShadowRepoDir, "--work-tree=" + state.ProjectRoot,
		"show", hash}
	
	if fileFilter != "" {
		args = append(args, "--", fileFilter)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get detailed diff: %w", err)
	}

	// Format the diff output
	lines := strings.Split(string(output), "\n")
	inDiffSection := false
	currentFile := ""
	isDeletedFile := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			inDiffSection = true
			// Extract filename from diff header
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentFile = strings.TrimPrefix(parts[2], "a/")
			}
			color.Cyan(line)
		} else if strings.HasPrefix(line, "deleted file mode") {
			isDeletedFile = true
			color.Red("🗑️  " + line + " - File was completely removed")
		} else if strings.HasPrefix(line, "new file mode") {
			isDeletedFile = false
			color.Green("📄 " + line + " - New file was added")
		} else if strings.HasPrefix(line, "@@") {
			if isDeletedFile && currentFile != "" {
				color.Yellow(fmt.Sprintf("📖 Contents of deleted file '%s':", currentFile))
			}
			color.Blue(line)
		} else if strings.HasPrefix(line, "+") {
			color.Green(line)
		} else if strings.HasPrefix(line, "-") {
			if isDeletedFile {
				// Highlight deleted file content differently
				color.New(color.FgRed, color.BgBlack).Print("- ")
				color.New(color.FgWhite).Println(line[1:])
			} else {
				color.Red(line)
			}
		} else if inDiffSection {
			fmt.Println(line)
		}
		
		// Reset flags when moving to next file
		if strings.HasPrefix(line, "diff --git") && inDiffSection {
			isDeletedFile = false
		}
	}

	return nil
}

func showComprehensiveAnalysis(state *core.AppState, hash string) error {
	fmt.Println()
	color.Cyan("📊 Comprehensive Analysis")
	color.Cyan("=========================")

	// Show diff stats
	cmd := exec.Command("git", "--git-dir="+state.ShadowRepoDir, "--work-tree="+state.ProjectRoot,
		"show", "--stat", hash)
	
	if output, err := cmd.Output(); err == nil {
		fmt.Println("Statistics:")
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "|") || strings.Contains(line, "changed") {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Show parent commit (what it was based on)
	cmd = exec.Command("git", "--git-dir="+state.ShadowRepoDir, "show", "--format=%P", "--no-patch", hash)
	if output, err := cmd.Output(); err == nil {
		parent := strings.TrimSpace(string(output))
		if parent != "" {
			fmt.Printf("\nParent commit: %s\n", parent[:8])
		}
	}

	// Show object information
	cmd = exec.Command("git", "--git-dir="+state.ShadowRepoDir, "cat-file", "-s", hash)
	if output, err := cmd.Output(); err == nil {
		size := strings.TrimSpace(string(output))
		if sizeInt, err := strconv.Atoi(size); err == nil {
			fmt.Printf("Commit object size: %s bytes\n", formatBytes(int64(sizeInt)))
		}
	}

	return nil
}

func isValidHash(state *core.AppState, hash string) bool {
	cmd := exec.Command("git", "--git-dir="+state.ShadowRepoDir, "cat-file", "-e", hash)
	return cmd.Run() == nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func runSearchAllSnapshots(state *core.AppState, fileFilter string, showDiff, verbose bool) error {
	// File filter is already validated in runInspect, but validate again for defense in depth
	if _, err := sanitizeFilePath(fileFilter); err != nil {
		return fmt.Errorf("invalid file filter in search-all: %w", err)
	}
	color.Green("🔍 Searching All Snapshots")
	if fileFilter != "" {
		color.Cyan(fmt.Sprintf("📁 File History: %s", fileFilter))
	} else {
		color.Cyan("📊 All Snapshots")
	}
	fmt.Println()

	// Use Git's native --follow command for efficient file history
	var args []string
	if fileFilter != "" {
		// Use git log --follow for file-specific history (most efficient)
		args = []string{"--git-dir=" + state.ShadowRepoDir, "--work-tree=" + state.ProjectRoot,
			"log", "--follow", "--oneline", "--date=short", "--format=%H|%ad|%s", "--", fileFilter}
	} else {
		// Show all snapshots
		args = []string{"--git-dir=" + state.ShadowRepoDir, "--work-tree=" + state.ProjectRoot,
			"log", "--oneline", "--date=short", "--format=%H|%ad|%s"}
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get file history: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		color.Yellow("📝 No snapshots found")
		if fileFilter != "" {
			fmt.Printf("   (no history found for: %s)\n", fileFilter)
		}
		return nil
	}

	color.Green(fmt.Sprintf("📸 Found %d snapshot(s)\n", len(lines)))

	for i, line := range lines {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		
		hash := parts[0]
		date := parts[1] 
		message := parts[2]

		color.Cyan(fmt.Sprintf("📸 Snapshot %d/%d - %s", i+1, len(lines), hash[:8]))
		fmt.Printf("📅 %s - %s\n", date, message)

		// Show what files changed in this snapshot
		if showDiff || verbose {
			if err := showDetailedDiff(state, hash, fileFilter); err == nil {
				fmt.Println()
			}
		} else {
			// Show just the file changes summary
			if err := showFileChanges(state, hash, fileFilter); err == nil {
				fmt.Println()
			}
		}
		
		fmt.Println(strings.Repeat("-", 60))
	}

	// Show additional file operations if specific file requested
	if fileFilter != "" && (showDiff || verbose) {
		if err := showFileOperationsHistory(state, fileFilter); err != nil {
			color.Yellow(fmt.Sprintf("⚠️  Could not show operation history: %v", err))
		}
	}

	return nil
}

func showFileOperationsHistory(state *core.AppState, filename string) error {
	fmt.Println()
	color.Magenta("📋 File Operations History")
	color.Magenta("==========================")

	// Show renames/moves using --follow --name-status
	args := []string{"--git-dir=" + state.ShadowRepoDir, "--work-tree=" + state.ProjectRoot,
		"log", "--follow", "--name-status", "--format=%H|%ad|%s", "--date=short", "--", filename}
	
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	currentCommit := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.Contains(line, "|") {
			// Commit info line
			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 {
				currentCommit = parts[0][:8]
				fmt.Printf("\n📸 %s (%s): %s\n", currentCommit, parts[1], parts[2])
			}
		} else {
			// File operation line
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				status := parts[0]
				file := parts[1]
				
				switch status {
				case "A":
					color.Green(fmt.Sprintf("  ✅ Added: %s", file))
				case "M":
					color.Yellow(fmt.Sprintf("  ✏️  Modified: %s", file))
				case "D":
					color.Red(fmt.Sprintf("  🗑️  Deleted: %s", file))
				case "R100":
					color.Blue(fmt.Sprintf("  📝 Renamed: %s", file))
				default:
					fmt.Printf("  %s: %s\n", status, file)
				}
			}
		}
	}

	return nil
}