package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CalculateDirectorySize calculates the total size of all files in a directory
func CalculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// FormatBytes formats bytes in human-readable format
func FormatBytes(bytes int64) string {
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

// CountProjectFiles counts files and directories in a project, excluding ignored patterns
func CountProjectFiles(rootPath string) (fileCount, dirCount int) {
	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || 
			   name == "build" || name == "__pycache__" {
				return filepath.SkipDir
			}
			dirCount++
		} else {
			fileCount++
		}
		return nil
	})
	
	return fileCount, dirCount
}