package main

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

func main() {
	// Test with the current directory
	currentDir, _ := os.Getwd()
	fmt.Printf("Testing with project root: %s\n", currentDir)
	
	manager := core.NewEnhancedIgnoreManager(currentDir)
	fmt.Printf("Loaded %d patterns\n", manager.GetPatternsCount())
	
	// Test paths that should be ignored
	testPaths := []string{
		"dk/test.txt",
		"dk/testdir",
		"dk/testdir/",
		"dk/testdir/somefile.txt", 
		"dk/testdir/subdir/file.txt",
		filepath.Join(currentDir, "dk", "test.txt"),
		filepath.Join(currentDir, "dk", "testdir"),
		filepath.Join(currentDir, "dk", "testdir", "somefile.txt"),
	}
	
	for _, testPath := range testPaths {
		result := manager.ShouldIgnore(testPath)
		fmt.Printf("ShouldIgnore(%-40s) = %v\n", testPath, result)
	}
	
	// Test directory-specific method
	fmt.Println("\nTesting ShouldIgnoreDirectory:")
	dirPaths := []string{
		"dk/testdir",
		filepath.Join(currentDir, "dk", "testdir"),
	}
	
	for _, dirPath := range dirPaths {
		result := manager.ShouldIgnoreDirectory(dirPath)
		fmt.Printf("ShouldIgnoreDirectory(%-40s) = %v\n", dirPath, result)
	}
}