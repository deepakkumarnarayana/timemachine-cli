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
	fsWatcher  *fsnotify.Watcher
	gitManager *GitManager
	debouncer  *Debouncer
	stopChan   chan bool
	wg         sync.WaitGroup
	state      *AppState
}

// NewWatcher creates a new file system watcher
func NewWatcher(state *AppState, gitManager *GitManager) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Create debouncer with 500ms delay (critical for npm install, etc.)
	debouncer := NewDebouncer(500 * time.Millisecond)

	return &Watcher{
		fsWatcher:  fsWatcher,
		gitManager: gitManager,
		debouncer:  debouncer,
		stopChan:   make(chan bool),
		state:      state,
	}, nil
}

// Start begins monitoring file changes
func (w *Watcher) Start() error {
	// Add project root and subdirectories to watch
	if err := w.addDirectoryRecursive(w.state.ProjectRoot); err != nil {
		return fmt.Errorf("failed to add directories to watch: %w", err)
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