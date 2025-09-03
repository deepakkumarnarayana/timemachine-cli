package commands

import (
	"os"
	"path/filepath"
	"testing"
	"os/exec"
	
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// setupBenchmarkRepo creates a test repository for benchmark testing
func setupBenchmarkRepo(b *testing.B) string {
	tempDir, err := os.MkdirTemp("", "timemachine_benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to init git: %v", err)
	}
	
	// Set git config
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	cmd.Run()
	
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	cmd.Run()
	
	// Create initial file and commit
	testFile := filepath.Join(tempDir, "README.md")
	os.WriteFile(testFile, []byte("Test repo"), 0644)
	
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tempDir
	cmd.Run()
	
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	cmd.Run()
	
	// Initialize TimeMachine
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	
	state, err := core.NewAppState()
	if err == nil {
		gitManager := core.NewGitManager(state)
		gitManager.InitializeShadowRepo()
	}
	
	os.Chdir(originalDir)
	
	return tempDir
}

func BenchmarkSnapshotCreation(b *testing.B) {
	// Create test environment
	tempDir := setupBenchmarkRepo(b)
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	originalDir, _ := os.Getwd()
	err := os.Chdir(tempDir)
	if err != nil {
		b.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)
	
	// Create initial test file
	testFile := filepath.Join(tempDir, "benchmark_test.txt")
	err = os.WriteFile(testFile, []byte("benchmark test content"), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Modify file content to ensure there are changes to commit
		content := []byte("benchmark content iteration " + string(rune(i%26+'A')))
		err = os.WriteFile(testFile, content, 0644)
		if err != nil {
			b.Fatalf("Failed to update test file: %v", err)
		}
		
		// Run snapshot creation
		err = runSnapshot("Benchmark snapshot")
		if err != nil {
			b.Fatalf("Snapshot creation failed: %v", err)
		}
	}
}

func BenchmarkSnapshotWithMultipleFiles(b *testing.B) {
	// Create test environment
	tempDir := setupBenchmarkRepo(b)
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	originalDir, _ := os.Getwd()
	err := os.Chdir(tempDir)
	if err != nil {
		b.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)
	
	// Create multiple test files
	numFiles := 5
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		fileName := "benchmark_file_" + string(rune(i%26+'A')) + ".txt"
		files[i] = filepath.Join(tempDir, fileName)
		err = os.WriteFile(files[i], []byte("initial content"), 0644)
		if err != nil {
			b.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Modify multiple files
		for j, file := range files {
			content := []byte("benchmark content iteration " + string(rune((i+j)%26+'A')))
			err = os.WriteFile(file, content, 0644)
			if err != nil {
				b.Fatalf("Failed to update test file: %v", err)
			}
		}
		
		// Create snapshot with all changes
		err = runSnapshot("Benchmark snapshot with multiple files")
		if err != nil {
			b.Fatalf("Snapshot creation with multiple files failed: %v", err)
		}
	}
}

func BenchmarkSnapshotMessageProcessing(b *testing.B) {
	// Benchmark message processing overhead
	tempDir := setupBenchmarkRepo(b)
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	originalDir, _ := os.Getwd()
	err := os.Chdir(tempDir)
	if err != nil {
		b.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)
	
	// Create test file once
	testFile := filepath.Join(tempDir, "message_benchmark.txt")
	
	messages := []string{
		"Short message",
		"This is a medium length message that tests typical usage patterns",
		"This is a very long message that simulates detailed commit descriptions with lots of context about changes",
		"Message with special characters: émojis 🚀 symbols @#$%^&*()",
		"", // Empty message (tests automatic generation)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Modify file to ensure changes exist
		content := []byte("benchmark iteration " + string(rune(i%26+'A')))
		err = os.WriteFile(testFile, content, 0644)
		if err != nil {
			b.Fatalf("Failed to update test file: %v", err)
		}
		
		// Use different messages to test processing overhead
		message := messages[i%len(messages)]
		err = runSnapshot(message)
		if err != nil {
			b.Fatalf("Snapshot creation failed: %v", err)
		}
	}
}