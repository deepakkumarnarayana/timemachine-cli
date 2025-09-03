package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors file system changes and creates snapshots
type Watcher struct {
	fsWatcher     *fsnotify.Watcher
	gitManager    *GitManager
	debouncer     *Debouncer
	stopChan      chan bool
	wg            sync.WaitGroup
	state         *AppState
	ignoreManager *EnhancedIgnoreManager
	lastBranch    string // Track last known branch for change detection
	branchMutex   sync.RWMutex // Protect branch state access
}

// NewWatcher creates a new file system watcher
func NewWatcher(state *AppState, gitManager *GitManager) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Use configuration for debounce delay if available
	debounceDelay := 2000 * time.Millisecond
	if state.Config != nil {
		debounceDelay = state.Config.Watcher.DebounceDelay
	}
	debouncer := NewDebouncer(debounceDelay)

	// Create enhanced ignore manager with .timemachine-ignore support
	ignoreManager := NewEnhancedIgnoreManager(state.ProjectRoot)

	// Get initial branch state for tracking
	initialBranch, err := gitManager.GetCurrentBranch()
	if err != nil {
		// Don't fail completely, just log warning and continue
		fmt.Printf("Warning: failed to get initial branch: %v\n", err)
		initialBranch = ""
	}

	return &Watcher{
		fsWatcher:     fsWatcher,
		gitManager:    gitManager,
		debouncer:     debouncer,
		stopChan:      make(chan bool),
		state:         state,
		ignoreManager: ignoreManager,
		lastBranch:    initialBranch,
	}, nil
}

// Start begins monitoring file changes
func (w *Watcher) Start() error {
	// Add project root and subdirectories to watch
	if err := w.addDirectoryRecursive(w.state.ProjectRoot); err != nil {
		return fmt.Errorf("failed to add directories to watch: %w", err)
	}

	// Add .git/HEAD to watch for branch changes (Phase 2: Real-time branch awareness)
	gitHeadPath := filepath.Join(w.state.GitDir, "HEAD")
	if err := w.fsWatcher.Add(gitHeadPath); err != nil {
		fmt.Printf("Warning: couldn't watch Git HEAD file for branch changes: %v\n", err)
		// Don't fail completely - branch watching is enhancement, not critical
	}

	// Ensure initial branch sync before creating snapshot
	if err := w.state.EnsureBranchSync(); err != nil {
		fmt.Printf("Warning: failed to sync initial branch state: %v\n", err)
	}

	// Create initial snapshot
	fmt.Print("✅ Creating initial snapshot... ")
	if err := w.gitManager.CreateSnapshot(""); err != nil {
		color.Red("❌")
		return fmt.Errorf("failed to create initial snapshot: %w", err)
	}
	color.Green("Done!")

	// Start event loop
	w.wg.Add(1)
	go w.eventLoop()

	// Print status
	color.Green("🚀 Time Machine is watching for changes...")
	fmt.Println("   Press Ctrl+C to stop")

	return nil
}

// Stop stops the file watcher
func (w *Watcher) Stop() {
	close(w.stopChan)
	w.debouncer.Cancel()
	w.fsWatcher.Close()
	w.wg.Wait()
}

// addDirectoryRecursive adds a directory and all its subdirectories to the watcher
func (w *Watcher) addDirectoryRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't read
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		// Skip ignored directories
		if w.shouldIgnoreDirectory(path) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := w.fsWatcher.Add(path); err != nil {
			// Log but don't fail - some directories might not be accessible
			fmt.Printf("Warning: couldn't watch directory %s: %v\n", path, err)
		}

		return nil
	})
}

// shouldIgnoreDirectory checks if a directory should be ignored
func (w *Watcher) shouldIgnoreDirectory(path string) bool {
	// Get relative path from project root
	relPath, err := filepath.Rel(w.state.ProjectRoot, path)
	if err != nil {
		return false
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Ignore patterns
	ignorePatterns := []string{
		".git",
		"node_modules",
		"dist",
		"build",
		"__pycache__",
		".next",
		".nuxt",
		"target", // Rust
		"bin",    // Go
		"obj",    // .NET
		".vscode",
		".idea",
		"coverage",
		".nyc_output",
		".cache",
	}

	for _, pattern := range ignorePatterns {
		if strings.HasPrefix(relPath, pattern+"/") || relPath == pattern {
			return true
		}
	}

	return false
}

// shouldIgnoreFile checks if a file should be ignored
func (w *Watcher) shouldIgnoreFile(path string) bool {
	filename := filepath.Base(path)
	
	// Ignore common temporary/swap files
	ignorePatterns := []string{
		".swp", ".swo", ".swn", // Vim
		"~",                    // Backup files
		".tmp", ".temp",        // Temporary files
		".DS_Store",           // macOS
		"Thumbs.db",           // Windows
		".log",                // Log files
	}

	for _, pattern := range ignorePatterns {
		if strings.HasSuffix(filename, pattern) {
			return true
		}
	}

	// Ignore files in timemachine_snapshots
	if strings.Contains(path, "timemachine_snapshots") {
		return true
	}

	return false
}

// eventLoop processes file system events
func (w *Watcher) eventLoop() {
	defer w.wg.Done()

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			w.handleEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("File watcher error: %v\n", err)

		case <-w.stopChan:
			return
		}
	}
}

// handleEvent processes a single file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Check if this is a Git HEAD change (branch switch detection)
	gitHeadPath := filepath.Join(w.state.GitDir, "HEAD")
	if event.Name == gitHeadPath && (event.Op&fsnotify.Write == fsnotify.Write) {
		w.handleBranchChange()
		return // Branch changes don't trigger regular snapshots
	}

	// Ignore if file should be ignored
	if w.shouldIgnoreFile(event.Name) {
		return
	}

	// If a new directory was created, add it to watch list
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if !w.shouldIgnoreDirectory(event.Name) {
				if err := w.addDirectoryRecursive(event.Name); err != nil {
					fmt.Printf("Warning: couldn't watch new directory %s: %v\n", event.Name, err)
				}
			}
		}
	}

	// Debounce snapshot creation
	w.debouncer.Trigger(w.createSnapshot)
}

// handleBranchChange processes Git branch changes (Phase 2: Real-time branch awareness)
func (w *Watcher) handleBranchChange() {
	// Use mutex to prevent race conditions during branch changes
	w.branchMutex.Lock()
	defer w.branchMutex.Unlock()

	// Get current branch
	currentBranch, err := w.gitManager.GetCurrentBranch()
	if err != nil {
		fmt.Printf("Warning: failed to get current branch after HEAD change: %v\n", err)
		return
	}

	// Check if branch actually changed
	if currentBranch == w.lastBranch {
		return // No change, ignore
	}

	// Log branch change
	if w.lastBranch != "" {
		color.Cyan("🌿 Branch changed: %s → %s", w.lastBranch, currentBranch)
	} else {
		color.Cyan("🌿 Branch detected: %s", currentBranch)
	}

	// Update tracking
	w.lastBranch = currentBranch
	w.state.CurrentBranch = currentBranch
	w.state.BranchSynced = false

	// Sync shadow repository to new branch
	fmt.Print("🔄 Syncing shadow repository... ")
	if err := w.state.EnsureBranchSync(); err != nil {
		color.Red("❌ Failed to sync shadow branch: %v", err)
		return
	}
	color.Green("✅ Done!")
}

// createSnapshot creates a snapshot (called after debounce delay)
func (w *Watcher) createSnapshot() {
	fmt.Print("📸 Creating snapshot... ")
	
	if err := w.gitManager.CreateSnapshot(""); err != nil {
		color.Red("❌ Error: %v", err)
		return
	}
	
	// Get latest snapshot for display
	snapshots, err := w.gitManager.ListSnapshots(1, "")
	if err == nil && len(snapshots) > 0 {
		color.Green("✅ Done! (Latest: %s)", snapshots[0].Time)
	} else {
		color.Green("✅ Done!")
	}
}