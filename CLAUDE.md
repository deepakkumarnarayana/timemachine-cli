# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Time Machine CLI is an innovative tool for AI-assisted development that creates automatic Git snapshots in a **shadow repository** without affecting the main Git workflow. It solves the critical problem of AI assistants breaking working code by providing instant, safe rollback capabilities.

**Core Innovation:** Uses a separate Git repository at `.git/timemachine_snapshots/` that shares the same working tree but maintains independent history from the main repository.

## Quick Start Commands

```bash
# Build the application
make build
# or: go build -o timemachine ./cmd/timemachine

# Run all tests
make test
# or: go test -v ./...

# Run tests with coverage
make test-coverage

# Run specific test package
go test -v ./internal/core
go test -v ./internal/core -run TestDebouncer

# Development mode with race detection
make dev
# or: go run -race ./cmd/timemachine

# Format code
make fmt

# Clean build artifacts
make clean
```

## Architecture Overview

### Shadow Repository System (Core Innovation)

**Critical Design Principle:** All Git operations use `--git-dir=.git/timemachine_snapshots --work-tree=.` to ensure complete isolation from the main repository.

- **AppState (`internal/core/state.go`)**: Manages application state and Git repository discovery
- **GitManager (`internal/core/git.go`)**: Handles all Git operations with shadow repo isolation
- **Shadow Repository Location**: `.git/timemachine_snapshots/`
- **Isolation Method**: Every Git command uses `--git-dir` and `--work-tree` flags

### File Watching System

- **Watcher (`internal/core/watcher.go`)**: Monitors filesystem changes using `fsnotify`
- **EnhancedIgnoreManager (`internal/core/ignore.go`)**: Thread-safe ignore pattern matching with `.timemachine-ignore` support
- **Debouncer (`internal/core/debouncer.go`)**: Groups rapid changes (500ms delay) to prevent snapshot spam
- **Ignore Patterns**: Uses `.timemachine-ignore` file with gitignore-compatible syntax, includes `.fuse_hidden*` for FUSE filesystems
- **Recursive Monitoring**: Automatically adds new directories to watch list

### Command Structure

All CLI commands are in `internal/commands/`:
- **init**: Creates shadow repo, updates `.gitignore`, creates default `.timemachine-ignore`, installs cleanup hooks
- **start**: Launches file watcher with signal handling (Ctrl+C)
- **list**: Shows snapshots with filtering and pagination
- **show**: Displays detailed snapshot information with file changes
- **inspect**: Advanced snapshot analysis with deleted file recovery, search-all functionality, and security-hardened input validation
- **restore**: Safely restores files using `git restore --worktree` (never affects staging)

## Critical Implementation Rules

### Git Operations Safety
- **NEVER use `git checkout` or `git reset`** - they affect the staging area
- **ALWAYS use `git restore --source=<hash> --worktree`** for restoration
- **NEVER modify the main .git directory** except for hooks in `.git/hooks/`
- **ALWAYS use `git add -A`** to capture all changes including deletions

### Shadow Repository Isolation
- Every `GitManager.RunCommand()` uses `--git-dir` and `--work-tree` flags
- Shadow repo operations are completely independent of main Git workflow
- All snapshots are stored in `.git/timemachine_snapshots/` which is auto-ignored

### File Watching Best Practices
- Debounce delay: minimum 500ms to handle bulk operations (npm install, etc.)
- Recursive directory watching with automatic new directory detection
- Comprehensive ignore patterns for build artifacts and temporary files
- Proper signal handling for graceful shutdown

### Security Implementation
- **Input Validation**: All Git hashes validated with regex `^[a-fA-F0-9]{4,40}$`
- **Path Sanitization**: File filter paths checked for traversal attacks (`..` patterns blocked)
- **Command Injection Prevention**: No user input directly passed to shell commands
- **Relative Path Enforcement**: Absolute paths blocked in file filters

## Key Files and Their Purpose

### Core Logic
- `internal/core/state.go`: Git repository discovery and application state
- `internal/core/git.go`: Shadow repository operations and snapshot management
- `internal/core/watcher.go`: File system monitoring and event handling with enhanced ignore manager
- `internal/core/ignore.go`: Thread-safe ignore pattern matching with comprehensive gitignore-compatible syntax
- `internal/core/debouncer.go`: Change grouping to prevent snapshot spam

### Testing
- Comprehensive test coverage in `*_test.go` files
- Tests cover shadow repo isolation, debouncing, file operations, and edge cases
- Integration tests verify complete workflow: init → snapshot → restore

### Entry Point
- `cmd/timemachine/main.go`: CLI setup using Cobra framework
- Commands are modular and imported from `internal/commands/`

## Development Workflow

1. **Make changes to core logic** in `internal/core/`
2. **Add corresponding tests** in `*_test.go` files
3. **Update command implementations** in `internal/commands/`
4. **Run tests**: `make test` or `go test -v ./...`
5. **Test manually**: `make build && ./timemachine init && ./timemachine start`
6. **Verify shadow repo isolation**: Check that main Git workflow is unaffected

## Shadow Repository Verification

To verify shadow repo is working correctly:
```bash
# Check shadow repo exists
ls -la .git/timemachine_snapshots/

# View shadow repo commits (should be separate from main)
git --git-dir=.git/timemachine_snapshots log --oneline

# Verify main repo is unaffected
git status  # Should not show timemachine changes
```

## Dependencies

- **fsnotify**: File system event monitoring
- **cobra**: CLI framework and command structure
- **color**: Terminal output coloring
- Standard Go libraries for Git operations and file handling

## Target Users

AI-assisted developers using Claude, ChatGPT, and other coding assistants who need instant rollback capabilities when AI breaks working code.