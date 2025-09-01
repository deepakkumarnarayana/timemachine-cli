# Time Machine CLI ⏰

**Automatic Git snapshots for AI-assisted development**

Time Machine creates automatic Git snapshots in a shadow repository without affecting your main Git workflow. Perfect for AI-assisted coding sessions where you need instant rollback capabilities when AI breaks working code.

## 🚀 Quick Start

```bash
# 1. Initialize Time Machine in your Git repository
timemachine init

# 2. Start automatic file watching
timemachine start

# 3. Work normally - all changes are automatically captured
# Make changes to your files...

# 4. If something breaks, instantly rollback
timemachine list
timemachine restore <hash>
```

## 💡 Core Innovation

Time Machine uses a **shadow repository** (`.git/timemachine_snapshots/`) that shares your working tree but maintains completely separate history from your main Git repository. This means:

- ✅ **Zero interference** with your main Git workflow
- ✅ **Instant snapshots** without affecting staging area or commits
- ✅ **Safe restoration** using `git restore --worktree`
- ✅ **Automatic cleanup** via post-push hooks

## 📋 Commands

### `timemachine init`
Initialize Time Machine in your Git repository
- Creates shadow repository at `.git/timemachine_snapshots/`
- Updates `.gitignore` to exclude shadow repository
- Installs auto-cleanup post-push hook
- Creates initial snapshot

### `timemachine start`
Start watching for file changes (press Ctrl+C to stop)
- Monitors all files recursively
- Ignores build directories (`node_modules/`, `dist/`, etc.)
- Groups rapid changes with 500ms debounce delay
- Creates automatic snapshots with timestamps

### `timemachine list`
List recent snapshots
```bash
timemachine list                    # Show 20 most recent
timemachine list --limit 50        # Show 50 most recent
timemachine list --file src/app.js # Filter by specific file
```

### `timemachine show <hash>`
Show detailed snapshot information
- Full commit details and timestamp
- Color-coded file changes (added/modified/deleted)
- Helpful restoration command

### `timemachine restore <hash>`
Restore files from a snapshot
```bash
timemachine restore abc12345                           # Restore everything
timemachine restore abc12345 --files src/app.js       # Restore specific file
timemachine restore abc12345 --force                  # Skip confirmation
```

### `timemachine status`
Show current status and statistics
```bash
timemachine status           # Basic status
timemachine status --verbose # Detailed information
```

### `timemachine clean`
Clean up snapshots to save disk space
```bash
timemachine clean                    # Remove all (with confirmation)
timemachine clean --auto            # Remove all (no confirmation) 
timemachine clean --keep 10         # Keep 10 most recent
timemachine clean --older-than 1w   # Remove older than 1 week
timemachine clean --auto --quiet    # Silent cleanup (for automation)
```

## 🔧 Installation

### From Source
```bash
git clone https://github.com/deepakkumarnarayana/timemachine-cli.git
cd timemachine-cli/timemachine
go build -o timemachine ./cmd/timemachine
# Move binary to your PATH
```

### Build Requirements
- Go 1.21+
- Git installed and available in PATH

## 🎯 Perfect For

**AI-Assisted Development:**
- Using Claude Code, ChatGPT, Copilot, etc.
- Rapid prototyping where AI might break working code
- Learning new frameworks with AI guidance
- Experimenting with AI-generated code changes

**General Development:**
- Complex refactoring sessions
- Trying multiple approaches to a problem
- Working in unfamiliar codebases
- Backup before major changes

## 🏗️ How It Works

1. **Shadow Repository:** Creates `.git/timemachine_snapshots/` - a separate Git repo sharing your working tree
2. **File Watching:** Uses `fsnotify` for efficient recursive directory monitoring
3. **Debouncing:** Groups rapid changes (500ms delay) to prevent snapshot spam during `npm install`, etc.
4. **Isolation:** All operations use `--git-dir` and `--work-tree` flags for complete separation
5. **Safe Restoration:** Always uses `git restore --worktree` to preserve your main Git state

## 📊 Example Workflow

```bash
# Start a coding session
$ timemachine init
✨ Time Machine initialized successfully!

$ timemachine start
🚀 Time Machine is watching for changes...

# Work normally - changes are captured automatically
# Edit files, add features, etc.
📸 Creating snapshot... ✅ Done! (Latest: 2 seconds ago)

# Something breaks? Instant recovery!
$ timemachine list
📸 Recent snapshots:

abc12345  Added user authentication     2 minutes ago  
def67890  Fixed CSS styling issues      5 minutes ago
ghi09876  Initial working homepage      8 minutes ago

$ timemachine restore def67890
✨ Files restored successfully!
```

## 🔍 Advanced Usage

### File Filtering
```bash
# See all changes to a specific file
timemachine list --file src/components/Header.js
timemachine show abc12345  # Shows what changed in that snapshot
```

### Cleanup Automation
```bash
# Clean up old snapshots weekly (add to cron)
timemachine clean --older-than 1w --auto --quiet
```

### Status Monitoring
```bash
# Check repository health
timemachine status --verbose
```

## 🛠️ Development

```bash
# Build
make build

# Test
make test

# Development mode
make dev

# Format code
make fmt
```

## ⚠️ Important Notes

- **Git Repository Required:** Only works inside Git repositories
- **Shadow Repository Size:** Grows over time - use `timemachine clean` periodically
- **File Permissions:** Preserves original file permissions on restoration
- **Large Files:** Works with binary files but will increase repository size
- **No Network:** Everything is local - no data sent anywhere

## 🤝 Contributing

Contributions welcome! This tool is designed to make AI-assisted development safer and more productive.

## 📜 License

MIT License - See LICENSE file for details.

## 🙏 Acknowledgments

Built for the AI-assisted development community. Special thanks to:
- The Go team for excellent tooling
- `fsnotify` for reliable file system watching  
- `cobra` for CLI framework
- The Git team for the foundation this builds upon

---

**Make AI coding sessions fearless! 🚀**