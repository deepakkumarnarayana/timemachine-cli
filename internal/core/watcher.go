package core

import (
	"fmt"
	"os"
	"path/filepath"
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
}

// NewWatcher creates a new file system watcher
func NewWatcher(state *AppState, gitManager *GitManager) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Create debouncer with 500ms delay (critical for npm install, etc.)
	debouncer := NewDebouncer(500 * time.Millisecond)

	// Create enhanced ignore manager with .timemachine-ignore support
	ignoreManager := NewEnhancedIgnoreManager(state.ProjectRoot)

	return &Watcher{
		fsWatcher:     fsWatcher,
		gitManager:    gitManager,
		debouncer:     debouncer,
		stopChan:      make(chan bool),
		state:         state,
		ignoreManager: ignoreManager,
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

		// Skip ignored directories using new IgnoreManager
		if w.ignoreManager.ShouldIgnoreDirectory(path) {
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

// shouldIgnoreDirectory checks if a directory should be ignored (DEPRECATED - use IgnoreManager)
func (w *Watcher) shouldIgnoreDirectory(path string) bool {
	// Delegate to new IgnoreManager for backward compatibility
	return w.ignoreManager.ShouldIgnoreDirectory(path)
}

// shouldIgnoreFile checks if a file should be ignored (DEPRECATED - use IgnoreManager)
func (w *Watcher) shouldIgnoreFile(path string) bool {
	// Delegate to new IgnoreManager for backward compatibility
	return w.ignoreManager.ShouldIgnoreFile(path)
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
			if !w.ignoreManager.ShouldIgnoreDirectory(event.Name) {
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