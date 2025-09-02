# TimeMachine CLI Configuration Reference

## Table of Contents
- [Configuration Overview](#configuration-overview)
- [Configuration Sources and Precedence](#configuration-sources-and-precedence)
- [Configuration Sections](#configuration-sections)
- [Environment Variables](#environment-variables)
- [Configuration File Locations](#configuration-file-locations)
- [Examples](#examples)
- [CLI Commands](#cli-commands)

## Configuration Overview

TimeMachine CLI uses a hierarchical configuration system that allows you to customize behavior through multiple sources. All configuration options have sensible defaults, making the system functional out of the box while providing extensive customization capabilities.

### Configuration Format

TimeMachine supports YAML configuration files with the following structure:

```yaml
# Example timemachine.yaml
log:
  level: info
  format: text
  file: ""

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  ignore_patterns: []
  batch_size: 100
  enable_recursive: true

cache:
  max_entries: 10000
  max_memory_mb: 50
  ttl: 1h
  enable_lru: true

git:
  cleanup_threshold: 100
  auto_gc: true
  max_commits: 1000
  use_shallow_clone: false

ui:
  progress_indicators: true
  color_output: true
  pager: auto
  table_format: table
```

## Configuration Sources and Precedence

Configuration is loaded from multiple sources with the following precedence order (highest to lowest):

1. **Command-line flags** (highest priority)
2. **Environment variables** (`TIMEMACHINE_*`)
3. **Configuration files** (searched in order)
4. **Built-in defaults** (lowest priority)

### Configuration File Search Order

1. Project-specific configuration:
   - `./timemachine.yaml`
   - `./.timemachine/timemachine.yaml`

2. User configuration:
   - `~/.config/timemachine/timemachine.yaml` (Linux/macOS)
   - `~/timemachine.yaml` (fallback)

3. System configuration:
   - `/etc/timemachine/timemachine.yaml`

## Configuration Sections

### Log Configuration

Controls logging behavior throughout the application.

| Setting | Type | Default | Valid Values | Description |
|---------|------|---------|--------------|-------------|
| `log.level` | string | `info` | `debug`, `info`, `warn`, `error` | Logging level |
| `log.format` | string | `text` | `text`, `json` | Log output format |
| `log.file` | string | `""` | Valid file path | Optional log file (empty = stdout) |

**Examples:**
```yaml
log:
  level: debug           # Enable debug logging
  format: json          # JSON formatted logs
  file: /tmp/timemachine.log  # Log to file

# Environment variable equivalents:
# TIMEMACHINE_LOG_LEVEL=debug
# TIMEMACHINE_LOG_FORMAT=json
# TIMEMACHINE_LOG_FILE=/tmp/timemachine.log
```

### Watcher Configuration

Controls file system watching behavior and snapshot creation.

| Setting | Type | Default | Valid Range | Description |
|---------|------|---------|-------------|-------------|
| `watcher.debounce_delay` | duration | `2s` | 100ms - 10s | Delay before creating snapshot after changes |
| `watcher.max_watched_files` | int | `100000` | 1,000 - 1,000,000 | Maximum number of files to watch |
| `watcher.ignore_patterns` | []string | `[]` | Valid patterns | Additional ignore patterns beyond `.timemachine-ignore` |
| `watcher.batch_size` | int | `100` | 1 - 1,000 | Number of files to process in a single batch |
| `watcher.enable_recursive` | bool | `true` | true/false | Enable recursive directory watching |

**Performance Notes:**
- `debounce_delay`: Higher values reduce snapshot frequency but increase latency
- `max_watched_files`: System-dependent; adjust based on available file descriptors
- `batch_size`: Larger batches improve I/O efficiency but use more memory

**Examples:**
```yaml
watcher:
  debounce_delay: 5s        # Wait 5 seconds before snapshotting
  max_watched_files: 50000  # Reduce for resource-constrained systems
  ignore_patterns:
    - "*.tmp"               # Ignore temporary files
    - "build/**"            # Ignore build directory
    - "*.log"               # Ignore log files
  batch_size: 50            # Smaller batches for memory efficiency
  enable_recursive: false   # Only watch root directory

# Environment variable equivalents:
# TIMEMACHINE_WATCHER_DEBOUNCE=5s
# TIMEMACHINE_WATCHER_MAX_FILES=50000
```

### Cache Configuration

Controls internal caching for performance optimization.

| Setting | Type | Default | Valid Range | Description |
|---------|------|---------|-------------|-------------|
| `cache.max_entries` | int | `10000` | 1,000 - 100,000 | Maximum cache entries |
| `cache.max_memory_mb` | int | `50` | 10 - 1,024 | Maximum cache memory usage (MB) |
| `cache.ttl` | duration | `1h` | 1m - 24h | Cache entry time-to-live |
| `cache.enable_lru` | bool | `true` | true/false | Use LRU (Least Recently Used) eviction |

**Memory Management:**
- `max_entries` × average entry size ≈ total memory usage
- `max_memory_mb` provides hard memory limit
- `ttl` prevents stale data accumulation

**Examples:**
```yaml
cache:
  max_entries: 5000      # Reduce for memory-constrained systems
  max_memory_mb: 25      # Limit cache to 25MB
  ttl: 30m              # Shorter TTL for frequently changing data
  enable_lru: false     # Use FIFO eviction instead

# Environment variable equivalents:
# TIMEMACHINE_CACHE_MAX_ENTRIES=5000
# TIMEMACHINE_CACHE_MAX_MEMORY=25
# TIMEMACHINE_CACHE_TTL=30m
```

### Git Configuration

Controls Git operations and repository management.

| Setting | Type | Default | Valid Range | Description |
|---------|------|---------|-------------|-------------|
| `git.cleanup_threshold` | int | `100` | 10 - 10,000 | Number of snapshots before cleanup |
| `git.auto_gc` | bool | `true` | true/false | Automatically run git garbage collection |
| `git.max_commits` | int | `1000` | 50 - 50,000 | Maximum snapshots to keep |
| `git.use_shallow_clone` | bool | `false` | true/false | Use shallow cloning for performance |

**Important Constraints:**
- `cleanup_threshold` must be less than `max_commits`
- `auto_gc` recommended for long-running sessions
- `use_shallow_clone` reduces disk usage but may affect some Git operations

**Examples:**
```yaml
git:
  cleanup_threshold: 50   # Clean up more frequently
  auto_gc: false         # Disable automatic garbage collection
  max_commits: 500       # Keep fewer snapshots
  use_shallow_clone: true # Enable for large repositories

# Environment variable equivalents:
# TIMEMACHINE_GIT_CLEANUP_THRESHOLD=50
# TIMEMACHINE_GIT_AUTO_GC=false
```

### UI Configuration

Controls user interface behavior and output formatting.

| Setting | Type | Default | Valid Values | Description |
|---------|------|---------|--------------|-------------|
| `ui.progress_indicators` | bool | `true` | true/false | Show progress bars and spinners |
| `ui.color_output` | bool | `true` | true/false | Colorize output |
| `ui.pager` | string | `auto` | `auto`, `always`, `never` | When to use pager for output |
| `ui.table_format` | string | `table` | `table`, `json`, `yaml` | Default output format for tables |

**Pager Behavior:**
- `auto`: Use pager for long output if stdout is a terminal
- `always`: Always use pager
- `never`: Never use pager

**Examples:**
```yaml
ui:
  progress_indicators: false  # Disable for CI/CD environments
  color_output: false        # Plain text output
  pager: never              # Never use pager
  table_format: json        # JSON output by default

# Environment variable equivalents:
# TIMEMACHINE_UI_COLOR=false
# TIMEMACHINE_UI_PAGER=never
```

## Environment Variables

All configuration options can be overridden using environment variables with the `TIMEMACHINE_` prefix:

### Complete Environment Variable Reference

```bash
# Log Configuration
TIMEMACHINE_LOG_LEVEL=info|debug|warn|error
TIMEMACHINE_LOG_FORMAT=text|json
TIMEMACHINE_LOG_FILE=/path/to/logfile

# Watcher Configuration
TIMEMACHINE_WATCHER_DEBOUNCE=2s
TIMEMACHINE_WATCHER_MAX_FILES=100000

# Cache Configuration  
TIMEMACHINE_CACHE_MAX_ENTRIES=10000
TIMEMACHINE_CACHE_MAX_MEMORY=50
TIMEMACHINE_CACHE_TTL=1h

# Git Configuration
TIMEMACHINE_GIT_CLEANUP_THRESHOLD=100
TIMEMACHINE_GIT_AUTO_GC=true

# UI Configuration
TIMEMACHINE_UI_COLOR=true
TIMEMACHINE_UI_PAGER=auto
```

### Environment Variable Examples

```bash
# Development environment with debug logging
export TIMEMACHINE_LOG_LEVEL=debug
export TIMEMACHINE_LOG_FORMAT=json

# Production environment with conservative settings
export TIMEMACHINE_WATCHER_DEBOUNCE=5s
export TIMEMACHINE_CACHE_MAX_MEMORY=25
export TIMEMACHINE_GIT_CLEANUP_THRESHOLD=50

# CI/CD environment
export TIMEMACHINE_UI_COLOR=false
export TIMEMACHINE_UI_PAGER=never
export TIMEMACHINE_LOG_FORMAT=json
```

## Configuration File Locations

### Project Configuration
Create project-specific configuration that applies only to the current repository:

```bash
# Primary project configuration
./timemachine.yaml

# Alternative project configuration
./.timemachine/timemachine.yaml
```

### User Configuration
Create user-specific configuration that applies to all projects:

```bash
# Linux/macOS (preferred)
~/.config/timemachine/timemachine.yaml

# Fallback location
~/timemachine.yaml
```

### System Configuration
System-wide configuration (requires administrator privileges):

```bash
# System-wide configuration
/etc/timemachine/timemachine.yaml
```

## Examples

### Basic Configuration

```yaml
# timemachine.yaml - Basic configuration
log:
  level: info
  format: text

watcher:
  debounce_delay: 2s
  ignore_patterns:
    - "node_modules/**"
    - "*.log"
    - ".DS_Store"

git:
  cleanup_threshold: 100
  max_commits: 1000

ui:
  color_output: true
  table_format: table
```

### Development Environment

```yaml
# Development configuration with debug logging
log:
  level: debug
  format: json
  file: "debug.log"

watcher:
  debounce_delay: 1s       # Faster snapshots for development
  max_watched_files: 50000
  ignore_patterns:
    - "node_modules/**"
    - "dist/**"
    - "build/**"
    - "*.tmp"
    - "*.log"

cache:
  max_entries: 5000        # Smaller cache for development
  max_memory_mb: 25
  ttl: 30m

git:
  cleanup_threshold: 25    # More frequent cleanup
  max_commits: 500
  auto_gc: true

ui:
  progress_indicators: true
  color_output: true
  pager: auto
  table_format: table
```

### Production Environment

```yaml
# Production configuration with performance optimization
log:
  level: warn             # Reduce log noise
  format: json           # Structured logging
  file: "/var/log/timemachine/app.log"

watcher:
  debounce_delay: 5s      # Longer delay for stability
  max_watched_files: 200000
  batch_size: 200        # Larger batches for efficiency
  ignore_patterns:
    - "node_modules/**"
    - "vendor/**"
    - "*.log"
    - "*.tmp"
    - "*.cache"

cache:
  max_entries: 20000      # Larger cache for performance
  max_memory_mb: 100
  ttl: 2h                # Longer TTL for stability
  enable_lru: true

git:
  cleanup_threshold: 200  # Less frequent cleanup
  max_commits: 2000      # Keep more history
  auto_gc: true
  use_shallow_clone: false

ui:
  progress_indicators: false  # Disable for automated environments
  color_output: false        # Plain output for logs
  pager: never
  table_format: json
```

### CI/CD Environment

```yaml
# CI/CD optimized configuration
log:
  level: info
  format: json          # Machine-readable logs
  file: ""             # Output to stdout for log aggregation

watcher:
  debounce_delay: 3s
  max_watched_files: 10000  # Smaller for CI containers
  batch_size: 50

cache:
  max_entries: 1000     # Minimal cache for short-lived containers
  max_memory_mb: 10
  ttl: 5m

git:
  cleanup_threshold: 10  # Aggressive cleanup for containers
  max_commits: 50
  auto_gc: false        # Manual GC in CI

ui:
  progress_indicators: false  # No interactive elements
  color_output: false        # No colors in CI logs
  pager: never              # Never use pager
  table_format: json        # Machine-readable output
```

## CLI Commands

### Initialize Configuration

```bash
# Create project configuration
timemachine config init

# Create global user configuration  
timemachine config init --global

# Overwrite existing configuration
timemachine config init --force
```

### View Configuration

```bash
# Show current configuration (YAML format)
timemachine config show

# Show configuration in JSON format
timemachine config show --format json

# Get specific configuration value
timemachine config get log.level
timemachine config get watcher.debounce_delay
timemachine config get git.max_commits
```

### Modify Configuration

```bash
# Set configuration values (not yet implemented)
timemachine config set log.level debug
timemachine config set watcher.debounce_delay 3s

# Set global configuration
timemachine config set log.level info --global
```

### Validate Configuration

```bash
# Validate current configuration
timemachine config validate

# Shows:
# - Configuration validation status
# - Active configuration file location
# - Environment variable overrides
# - Validation errors (if any)
```

This comprehensive reference covers all configuration options, validation rules, and usage examples for TimeMachine CLI's configuration system.