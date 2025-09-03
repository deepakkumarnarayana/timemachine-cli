package core

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

// Constants based on real-world analysis and Git's approach
const (
	MaxIgnoreFileSize   = 10 * 1024 * 1024 // 10MB (Git allows 100MB, but we're more conservative)
	MaxIgnoreLines      = 10000             // Maximum lines in ignore file
	MaxPatternLength    = 4096              // 4KB per pattern (very generous)
	MaxPatterns         = 1000              // More than any real project needs
	MaxPathCacheEntries = 10000             // Cache for file path results
	MaxCacheMemoryMB    = 50                // Memory limit for cache (50MB)
	DefaultIgnoreFile   = ".timemachine-ignore"
)

// IgnorePattern represents a parsed ignore pattern with optimizations
type IgnorePattern struct {
	Original    string // Original pattern text
	Pattern     string // Processed pattern (without ! or /)
	IsNegation  bool   // Pattern starts with !
	IsDirectory bool   // Pattern ends with /
	IsAbsolute  bool   // Pattern starts with /
	IsSimple    bool   // No wildcards (fast path)
}

// EnhancedIgnoreManager provides high-performance ignore pattern matching
// with Git-inspired optimizations and thread-safe caching
type EnhancedIgnoreManager struct {
	// Core data
	patterns    []IgnorePattern
	projectRoot string
	ignoreFile  string

	// Performance cache (thread-safe)
	pathCache   map[string]bool
	cacheMutex  sync.RWMutex
	cacheMemory int64

	// Statistics (for monitoring/debugging)
	cacheHits   int64
	cacheMisses int64
	totalChecks int64
}

// NewEnhancedIgnoreManager creates a new enhanced ignore manager with caching
func NewEnhancedIgnoreManager(projectRoot string) *EnhancedIgnoreManager {
	manager := &EnhancedIgnoreManager{
		projectRoot: projectRoot,
		ignoreFile:  filepath.Join(projectRoot, DefaultIgnoreFile),
		pathCache:   make(map[string]bool),
	}

	// Load patterns from .timemachine-ignore file
	if err := manager.loadIgnoreFile(); err != nil {
		log.Printf("Warning: Failed to load ignore patterns: %v", err)
	}

	return manager
}

// loadIgnoreFile loads and parses the .timemachine-ignore file
func (eim *EnhancedIgnoreManager) loadIgnoreFile() error {
	file, err := os.Open(eim.ignoreFile)
	if os.IsNotExist(err) {
		log.Printf("Info: No %s file found, using no custom ignore patterns", DefaultIgnoreFile)
		return nil // No file is okay
	}
	if err != nil {
		return fmt.Errorf("failed to open ignore file: %w", err)
	}
	defer file.Close()

	// Security: Check file size before reading
	if stat, err := file.Stat(); err == nil {
		if stat.Size() > MaxIgnoreFileSize {
			return fmt.Errorf("ignore file too large: %d bytes (max %d bytes)", 
				stat.Size(), MaxIgnoreFileSize)
		}
	}

	scanner := bufio.NewScanner(file)
	
	// Set buffer size for long lines
	buf := make([]byte, MaxPatternLength)
	scanner.Buffer(buf, MaxPatternLength)

	lineCount := 0
	patternCount := 0

	for scanner.Scan() {
		lineCount++
		
		// Security: Limit total lines
		if lineCount > MaxIgnoreLines {
			log.Printf("Warning: Ignore file has too many lines (%d), truncating at %d", 
				lineCount, MaxIgnoreLines)
			break
		}

		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Security: Check pattern length
		if len(line) > MaxPatternLength {
			log.Printf("Warning: Pattern too long, skipping: %.50s...", line)
			continue
		}

		// Parse pattern
		pattern, err := eim.parsePattern(line)
		if err != nil {
			log.Printf("Warning: Invalid pattern '%s': %v", line, err)
			continue
		}

		// Security: Limit total patterns
		if patternCount >= MaxPatterns {
			log.Printf("Warning: Too many patterns (%d), ignoring remaining", MaxPatterns)
			break
		}

		eim.patterns = append(eim.patterns, pattern)
		patternCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read ignore file: %w", err)
	}

	log.Printf("Loaded %d ignore patterns from %s", len(eim.patterns), DefaultIgnoreFile)
	return nil
}

// parsePattern parses a single ignore pattern line
func (eim *EnhancedIgnoreManager) parsePattern(line string) (IgnorePattern, error) {
	if line == "" {
		return IgnorePattern{}, fmt.Errorf("empty pattern")
	}

	pattern := IgnorePattern{
		Original: line,
		Pattern:  line,
	}

	// Handle negation (!)
	if strings.HasPrefix(line, "!") {
		pattern.IsNegation = true
		pattern.Pattern = line[1:]
		if pattern.Pattern == "" {
			return IgnorePattern{}, fmt.Errorf("empty negation pattern")
		}
	}

	// Handle directory patterns (/)
	if strings.HasSuffix(pattern.Pattern, "/") {
		pattern.IsDirectory = true
		pattern.Pattern = strings.TrimSuffix(pattern.Pattern, "/")
	}

	// Handle absolute patterns (leading /)
	if strings.HasPrefix(pattern.Pattern, "/") {
		pattern.IsAbsolute = true
		pattern.Pattern = strings.TrimPrefix(pattern.Pattern, "/")
	}

	// Check if pattern is simple (no wildcards) for fast path
	pattern.IsSimple = !strings.ContainsAny(pattern.Pattern, "*?[]")

	// Basic validation
	if pattern.Pattern == "" {
		return IgnorePattern{}, fmt.Errorf("empty pattern after processing")
	}

	return pattern, nil
}

// ShouldIgnore determines if a file path should be ignored
// This is the main entry point called by the watcher
func (eim *EnhancedIgnoreManager) ShouldIgnore(path string) bool {
	// Convert to relative path
	relPath, err := filepath.Rel(eim.projectRoot, path)
	if err != nil {
		relPath = path // Fallback to absolute path
	}
	relPath = filepath.ToSlash(relPath) // Normalize path separators

	// Check cache first (thread-safe read)
	eim.cacheMutex.RLock()
	result, exists := eim.pathCache[relPath]
	eim.cacheMutex.RUnlock()

	if exists {
		// Thread-safe counter increment
		eim.cacheMutex.Lock()
		eim.cacheHits++
		eim.totalChecks++
		eim.cacheMutex.Unlock()
		return result
	}

	// Compute result
	result = eim.matchPatterns(relPath)

	// Cache result and update stats (thread-safe)
	eim.cacheMutex.Lock()
	eim.cacheMisses++
	eim.totalChecks++
	eim.cacheMutex.Unlock()
	
	eim.addToCache(relPath, result)

	return result
}

// matchPatterns checks if a path matches any ignore patterns
func (eim *EnhancedIgnoreManager) matchPatterns(relPath string) bool {
	filename := filepath.Base(relPath)
	dirname := filepath.Dir(relPath)
	
	// Process patterns in order (later patterns can override earlier ones)
	ignored := false
	
	for _, pattern := range eim.patterns {
		var matched bool
		
		if pattern.IsDirectory {
			// Directory pattern: check against directory components
			matched = eim.matchDirectoryPattern(pattern, relPath, dirname)
		} else {
			// File pattern: check against filename or full path
			matched = eim.matchFilePattern(pattern, relPath, filename)
		}
		
		if matched {
			ignored = !pattern.IsNegation // Negation patterns un-ignore
		}
	}
	
	return ignored
}

// matchFilePattern matches a file pattern against a path
func (eim *EnhancedIgnoreManager) matchFilePattern(pattern IgnorePattern, relPath, filename string) bool {
	if pattern.IsAbsolute {
		// For absolute patterns, match against the path from root
		if pattern.IsSimple {
			// Check if path starts with pattern (for directories) or equals pattern (for files)
			return strings.HasPrefix(relPath, pattern.Pattern+"/") || relPath == pattern.Pattern
		}
		matched, err := filepath.Match(pattern.Pattern, relPath)
		return err == nil && matched
	}

	// For non-absolute patterns, match against filename or check if file is within pattern directory
	if pattern.IsSimple {
		// If pattern contains slash, it should match path components
		if strings.Contains(pattern.Pattern, "/") {
			// Check if the relative path starts with this pattern (for directories)
			// or equals this pattern (for exact file matches)
			return strings.HasPrefix(relPath, pattern.Pattern+"/") || relPath == pattern.Pattern
		}
		// Fast path: exact string matching against filename only
		return filename == pattern.Pattern
	}

	// Use Go's filepath.Match for wildcard patterns
	// If pattern contains slash, match against full relative path, otherwise just filename
	matchTarget := filename
	if strings.Contains(pattern.Pattern, "/") {
		matchTarget = relPath
	}
	matched, err := filepath.Match(pattern.Pattern, matchTarget)
	return err == nil && matched
}

// matchDirectoryPattern matches a directory pattern against a path
func (eim *EnhancedIgnoreManager) matchDirectoryPattern(pattern IgnorePattern, relPath, dirname string) bool {
	if pattern.IsAbsolute {
		// For absolute directory patterns, match against path from root
		if pattern.IsSimple {
			return strings.HasPrefix(relPath, pattern.Pattern+"/") || 
			       dirname == pattern.Pattern ||
			       relPath == pattern.Pattern
		}
		matched, err := filepath.Match(pattern.Pattern, dirname)
		return err == nil && matched
	}

	// For non-absolute directory patterns, match against any directory component
	if pattern.IsSimple {
		// Check if any part of the path contains this directory
		return strings.Contains(relPath, "/"+pattern.Pattern+"/") || 
		       strings.HasPrefix(relPath, pattern.Pattern+"/") ||
		       dirname == pattern.Pattern ||
		       relPath == pattern.Pattern  // Match the directory name itself
	}

	// Check each directory component with wildcards
	dirs := strings.Split(dirname, "/")
	for _, dir := range dirs {
		if matched, err := filepath.Match(pattern.Pattern, dir); err == nil && matched {
			return true
		}
	}

	return false
}

// addToCache adds a result to the cache with memory management
func (eim *EnhancedIgnoreManager) addToCache(path string, result bool) {
	eim.cacheMutex.Lock()
	defer eim.cacheMutex.Unlock()

	// Memory management: check cache size
	if len(eim.pathCache) >= MaxPathCacheEntries {
		// Simple eviction: clear oldest half of cache
		// This is more predictable than LRU for our use case
		eim.clearOldestCacheEntries()
	}

	// Estimate memory usage (rough calculation)
	entrySize := int64(len(path) + 1) // path + bool
	if eim.cacheMemory+entrySize > MaxCacheMemoryMB*1024*1024 {
		eim.clearOldestCacheEntries()
		eim.cacheMemory = 0 // Reset counter after clear
	}

	eim.pathCache[path] = result
	eim.cacheMemory += entrySize
}

// clearOldestCacheEntries clears half the cache (simple eviction strategy)
func (eim *EnhancedIgnoreManager) clearOldestCacheEntries() {
	targetSize := len(eim.pathCache) / 2
	count := 0
	
	// Clear entries until we reach target size
	for path := range eim.pathCache {
		delete(eim.pathCache, path)
		count++
		if count >= targetSize {
			break
		}
	}
	
	// Reset memory counter (rough estimate)
	eim.cacheMemory = eim.cacheMemory / 2
}

// ClearCache clears the entire cache (useful for testing or memory pressure)
func (eim *EnhancedIgnoreManager) ClearCache() {
	eim.cacheMutex.Lock()
	defer eim.cacheMutex.Unlock()
	
	eim.pathCache = make(map[string]bool)
	eim.cacheMemory = 0
	eim.cacheHits = 0
	eim.cacheMisses = 0
}

// GetStats returns cache performance statistics
func (eim *EnhancedIgnoreManager) GetStats() (hits, misses, total int64, hitRate float64) {
	eim.cacheMutex.RLock()
	defer eim.cacheMutex.RUnlock()
	
	hits = eim.cacheHits
	misses = eim.cacheMisses
	total = eim.totalChecks
	
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	
	return
}

// ReloadIgnoreFile reloads the ignore file (useful for dynamic updates)
func (eim *EnhancedIgnoreManager) ReloadIgnoreFile() error {
	// Clear existing patterns and cache
	eim.patterns = nil
	eim.ClearCache()
	
	// Reload from file
	return eim.loadIgnoreFile()
}

// GetPatternsCount returns the number of loaded patterns
func (eim *EnhancedIgnoreManager) GetPatternsCount() int {
	return len(eim.patterns)
}

// EstimateMemoryUsage returns estimated memory usage in bytes
func (eim *EnhancedIgnoreManager) EstimateMemoryUsage() int64 {
	eim.cacheMutex.RLock()
	defer eim.cacheMutex.RUnlock()
	
	// Rough calculation: patterns + cache
	patternsMemory := int64(len(eim.patterns) * int(unsafe.Sizeof(IgnorePattern{})))
	cacheMemory := eim.cacheMemory
	
	return patternsMemory + cacheMemory
}

// Legacy compatibility methods (for drop-in replacement)

// ShouldIgnoreFile determines if a file should be ignored
func (eim *EnhancedIgnoreManager) ShouldIgnoreFile(path string) bool {
	return eim.ShouldIgnore(path)
}

// ShouldIgnoreDirectory determines if a directory should be ignored  
func (eim *EnhancedIgnoreManager) ShouldIgnoreDirectory(path string) bool {
	// For directories, append / to match directory patterns correctly
	dirPath := path
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}
	return eim.ShouldIgnore(dirPath)
}