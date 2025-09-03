package core

import (
	"os"
	"testing"
	"time"
)

// setupTestRepoWithLifecycle sets up a test repo with lifecycle management initialized
func setupTestRepoWithLifecycle(t *testing.T) (string, *AppState, *GitManager) {
	tempDir, state, gitManager := setupTestRepo(t)
	
	// Initialize lifecycle fields since setupTestRepo doesn't use NewAppState
	state.branchCacheTTL = 30 * time.Second
	
	return tempDir, state, gitManager
}

func TestAppStateLifecycleManagement(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Test initial state
	if !state.branchCacheTime.IsZero() {
		t.Error("Branch cache time should be zero initially (indicating no cache)")
	}

	if state.branchCacheTTL == 0 {
		t.Error("Branch cache TTL should be set")
	}

	// Test RefreshBranchState
	err := state.RefreshBranchState()
	if err != nil {
		t.Fatalf("RefreshBranchState failed: %v", err)
	}

	if state.CurrentBranch == "" {
		t.Error("CurrentBranch should be set after refresh")
	}

	if state.ShadowBranch == "" {
		t.Error("ShadowBranch should be set after refresh")
	}
}

func TestBranchStateValidation(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Test validation with empty cache
	state.InvalidateBranchCache()
	err := state.ValidateBranchState()
	if err == nil {
		t.Error("ValidateBranchState should fail with empty cache")
	}

	// Refresh and test validation
	err = state.RefreshBranchState()
	if err != nil {
		t.Fatalf("RefreshBranchState failed: %v", err)
	}

	err = state.ValidateBranchState()
	if err != nil {
		t.Errorf("ValidateBranchState should pass after refresh: %v", err)
	}
}

func TestBranchStateCacheExpiry(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Set short TTL for testing
	state.branchCacheTTL = 1 * time.Millisecond

	// Refresh state
	err := state.RefreshBranchState()
	if err != nil {
		t.Fatalf("RefreshBranchState failed: %v", err)
	}

	// Should be valid immediately
	err = state.ValidateBranchState()
	if err != nil {
		t.Errorf("ValidateBranchState should pass immediately after refresh: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	// Should now be invalid due to expiry
	err = state.ValidateBranchState()
	if err == nil {
		t.Error("ValidateBranchState should fail after cache expiry")
	}
}

func TestEnsureValidBranchState(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Test with fresh state (should automatically refresh)
	state.InvalidateBranchCache()
	err := state.EnsureValidBranchState()
	if err != nil {
		t.Errorf("EnsureValidBranchState should handle empty cache: %v", err)
	}

	// Verify state is now valid
	err = state.ValidateBranchState()
	if err != nil {
		t.Errorf("Branch state should be valid after EnsureValidBranchState: %v", err)
	}

	// Test with expired cache
	state.branchCacheTTL = 1 * time.Millisecond
	time.Sleep(5 * time.Millisecond)

	err = state.EnsureValidBranchState()
	if err != nil {
		t.Errorf("EnsureValidBranchState should handle expired cache: %v", err)
	}
}

func TestGetBranchContext(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Ensure state is valid
	err := state.EnsureValidBranchState()
	if err != nil {
		t.Fatalf("EnsureValidBranchState failed: %v", err)
	}

	// Test GetBranchContext
	currentBranch, shadowBranch, synced, err := state.GetBranchContext()
	if err != nil {
		t.Fatalf("GetBranchContext failed: %v", err)
	}

	if currentBranch == "" {
		t.Error("CurrentBranch should not be empty")
	}

	if shadowBranch == "" {
		t.Error("ShadowBranch should not be empty")
	}

	if currentBranch != shadowBranch && synced {
		t.Error("Synced should be false when branches differ")
	}

	if currentBranch == shadowBranch && !synced {
		t.Error("Synced should be true when branches match")
	}
}

func TestInvalidateBranchCache(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Refresh state
	err := state.RefreshBranchState()
	if err != nil {
		t.Fatalf("RefreshBranchState failed: %v", err)
	}

	// Should be valid
	err = state.ValidateBranchState()
	if err != nil {
		t.Errorf("ValidateBranchState should pass: %v", err)
	}

	// Invalidate cache
	state.InvalidateBranchCache()

	// Should now be invalid
	err = state.ValidateBranchState()
	if err == nil {
		t.Error("ValidateBranchState should fail after invalidation")
	}
}

func TestConcurrentBranchStateAccess(t *testing.T) {
	// Create test environment
	tempDir, state, _ := setupTestRepoWithLifecycle(t)
	defer os.RemoveAll(tempDir)

	// Ensure initial state
	err := state.EnsureValidBranchState()
	if err != nil {
		t.Fatalf("EnsureValidBranchState failed: %v", err)
	}

	// Test concurrent read access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			_, _, _, err := state.GetBranchContext()
			if err != nil {
				t.Errorf("GetBranchContext failed in goroutine: %v", err)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent write access (should be serialized)
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			
			state.InvalidateBranchCache()
			_ = state.RefreshBranchState()
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Final state should be valid
	err = state.ValidateBranchState()
	if err != nil {
		t.Errorf("Final state should be valid after concurrent operations: %v", err)
	}
}