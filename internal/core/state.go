package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	
	"github.com/deepakkumarnarayana/timemachine-cli/internal/config"
)

// AppState contains the application state and paths
type AppState struct {
	ProjectRoot     string          // Absolute path to project root (parent of .git)
	GitDir          string          // Path to .git directory
	ShadowRepoDir   string          // Path to .git/timemachine_snapshots
	IsInitialized   bool            // Whether shadow repo exists and is valid
	Config          *config.Config  // Application configuration
	ConfigManager   *config.Manager // Configuration manager
	CurrentBranch   string          // Current Git branch name
	ShadowBranch    string          // Current shadow repository branch
	BranchSynced    bool            // Whether shadow branch matches main branch
	
	// Phase 3A: Lifecycle Management
	branchCacheTime time.Time     // When branch state was last refreshed
	branchCacheTTL  time.Duration // How long branch cache is valid (default: 30s)
	stateMutex      sync.RWMutex  // Protects branch state operations
}

// NewAppState creates a new AppState by finding the Git repository
// and checking if the shadow repository is initialized
func NewAppState() (*AppState, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Walk up directory tree looking for .git directory
	gitDir := findGitDir(cwd)
	if gitDir == "" {
		return nil, errors.New("not in a Git repository (or any parent directory)")
	}

	// Set ProjectRoot to parent of .git
	projectRoot := filepath.Dir(gitDir)
	
	// Set ShadowRepoDir to .git/timemachine_snapshots
	shadowRepoDir := filepath.Join(gitDir, "timemachine_snapshots")
	
	// Check if shadow repo exists by looking for HEAD file
	headFile := filepath.Join(shadowRepoDir, "HEAD")
	isInitialized := false
	if _, err := os.Stat(headFile); err == nil {
		isInitialized = true
	}

	// Initialize configuration manager
	configManager := config.NewManager()
	
	// Load configuration (don't fail if config doesn't exist)
	if err := configManager.Load(projectRoot); err != nil {
		// Log warning but continue - config is optional
		fmt.Printf("Warning: failed to load configuration: %v\n", err)
	}

	return &AppState{
		ProjectRoot:     projectRoot,
		GitDir:          gitDir,
		ShadowRepoDir:   shadowRepoDir,
		IsInitialized:   isInitialized,
		Config:          configManager.Get(),
		ConfigManager:   configManager,
		CurrentBranch:   "",  // Will be populated by UpdateBranchState()
		ShadowBranch:    "",  // Will be populated by UpdateBranchState()
		BranchSynced:    false, // Will be populated by UpdateBranchState()
		
		// Phase 3A: Initialize lifecycle management
		branchCacheTime: time.Time{}, // Zero time indicates no cache
		branchCacheTTL:  30 * time.Second, // 30 second cache TTL
	}, nil
}

// NewAppStateWithConfig creates a new AppState with custom configuration
// This is useful for testing or when configuration should be loaded differently
func NewAppStateWithConfig(configManager *config.Manager) (*AppState, error) {
	state, err := NewAppState()
	if err != nil {
		return nil, err
	}
	
	// Override configuration
	state.ConfigManager = configManager
	state.Config = configManager.Get()
	
	return state, nil
}

// UpdateBranchState updates the current branch information (Phase 3A: Enhanced with caching)
// DEPRECATED: Use RefreshBranchState() for new code - this is kept for backward compatibility
func (s *AppState) UpdateBranchState() error {
	// Delegate to new RefreshBranchState method
	return s.RefreshBranchState()
}

// EnsureBranchSync ensures the shadow repository branch matches the main repository branch
func (s *AppState) EnsureBranchSync() error {
	if !s.IsInitialized {
		return fmt.Errorf("shadow repository not initialized")
	}
	
	// Update branch state first
	if err := s.UpdateBranchState(); err != nil {
		return fmt.Errorf("failed to update branch state: %w", err)
	}
	
	// If already synced, nothing to do
	if s.BranchSynced {
		return nil
	}
	
	// Create GitManager and switch to correct shadow branch
	gitManager := NewGitManager(s)
	if err := gitManager.SwitchOrCreateShadowBranch(s.CurrentBranch); err != nil {
		return fmt.Errorf("failed to sync shadow branch: %w", err)
	}
	
	// Update state
	s.ShadowBranch = s.CurrentBranch
	s.BranchSynced = true
	
	return nil
}

// RefreshBranchState refreshes branch information from Git (Phase 3A: Lifecycle Management)
func (s *AppState) RefreshBranchState() error {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	if !s.IsInitialized {
		return fmt.Errorf("shadow repository not initialized")
	}

	// Create a temporary GitManager to get branch info
	gitManager := NewGitManager(s)

	// Get current main branch
	currentBranch, err := gitManager.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get current shadow branch
	shadowBranch, err := gitManager.GetCurrentShadowBranch()
	if err != nil {
		return fmt.Errorf("failed to get shadow branch: %w", err)
	}

	// Update state
	s.CurrentBranch = currentBranch
	s.ShadowBranch = shadowBranch
	s.BranchSynced = (currentBranch == shadowBranch)
	s.branchCacheTime = time.Now()

	return nil
}

// ValidateBranchState checks if current branch state is valid and recent (Phase 3A: Lifecycle Management)
func (s *AppState) ValidateBranchState() error {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()

	if !s.IsInitialized {
		return fmt.Errorf("shadow repository not initialized")
	}

	// Check if we have cached branch state
	if s.branchCacheTime.IsZero() {
		return fmt.Errorf("branch state not initialized")
	}

	// Check if cache has expired
	if time.Since(s.branchCacheTime) > s.branchCacheTTL {
		return fmt.Errorf("branch state cache expired")
	}

	// Validate branch state consistency
	if s.CurrentBranch == "" {
		return fmt.Errorf("current branch not set")
	}

	if s.ShadowBranch == "" {
		return fmt.Errorf("shadow branch not set")
	}

	return nil
}

// EnsureValidBranchState ensures branch state is current and valid (Phase 3A: Lifecycle Management)
// This is the main entry point for commands to validate their branch context
func (s *AppState) EnsureValidBranchState() error {
	// First check if current state is valid
	if err := s.ValidateBranchState(); err != nil {
		// State is invalid or stale, refresh it
		if refreshErr := s.RefreshBranchState(); refreshErr != nil {
			return fmt.Errorf("failed to refresh branch state: %w (validation error: %v)", refreshErr, err)
		}
		
		// Try validation again after refresh
		if err := s.ValidateBranchState(); err != nil {
			return fmt.Errorf("branch state still invalid after refresh: %w", err)
		}
	}

	// Ensure branches are synchronized
	if !s.BranchSynced {
		if err := s.EnsureBranchSync(); err != nil {
			return fmt.Errorf("failed to synchronize branches: %w", err)
		}
	}

	return nil
}

// GetBranchContext returns current branch context in a thread-safe manner (Phase 3A: Lifecycle Management)
func (s *AppState) GetBranchContext() (currentBranch, shadowBranch string, synced bool, err error) {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()

	if err := s.ValidateBranchState(); err != nil {
		return "", "", false, fmt.Errorf("invalid branch state: %w", err)
	}

	return s.CurrentBranch, s.ShadowBranch, s.BranchSynced, nil
}

// InvalidateBranchCache forces the next operation to refresh branch state (Phase 3A: Lifecycle Management)
func (s *AppState) InvalidateBranchCache() {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	
	s.branchCacheTime = time.Time{} // Zero time indicates no cache
}

// findGitDir searches for a .git directory starting from the given directory
// and walking up the directory tree until it finds one or reaches the filesystem root
func findGitDir(startDir string) string {
	currentDir := startDir
	
	for {
		// Check for .git directory in current directory
		gitPath := filepath.Join(currentDir, ".git")
		
		// Check if .git exists and is a directory (not a file, which could be a submodule)
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return gitPath
		}
		
		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		
		// Stop if we've reached the filesystem root
		if parentDir == currentDir {
			break
		}
		
		currentDir = parentDir
	}
	
	// Not found
	return ""
}