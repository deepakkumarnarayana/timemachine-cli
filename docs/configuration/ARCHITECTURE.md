# TimeMachine CLI Configuration Architecture

## Overview

TimeMachine CLI implements a sophisticated, security-first configuration management system built on Viper with enterprise-grade validation and multi-source hierarchy support. The system is designed for production environments requiring robust configuration management, comprehensive validation, and security hardening.

## Design Principles

### 1. Security-First Architecture
- **Explicit Environment Variable Binding**: Removed `AutomaticEnv()` vulnerability, only whitelisted environment variables are processed
- **Path Traversal Prevention**: Comprehensive validation against directory traversal attacks
- **Input Sanitization**: All configuration values undergo strict validation
- **Secure File Permissions**: Configuration files use 0600 permissions (owner read/write only)

### 2. Hierarchical Configuration Sources
Configuration is loaded with clear precedence order (highest to lowest priority):
1. **CLI Flags** - Highest priority, immediate overrides
2. **Environment Variables** - `TIMEMACHINE_*` prefixed variables
3. **Configuration Files** - Multiple location search hierarchy
4. **Built-in Defaults** - Fallback values ensuring system functionality

### 3. Comprehensive Validation
- **Type Safety**: Strong type validation for all configuration sections
- **Range Validation**: Numeric limits and duration bounds
- **Enum Validation**: Restricted value sets for categorical options
- **Cross-field Validation**: Logical consistency checks (e.g., cleanup_threshold < max_commits)
- **Security Validation**: Path sanitization and injection prevention

## Architecture Components

### Core Components

```
internal/config/
├── config.go           # Configuration management and loading logic
├── validator.go        # Security-first validation system
├── config_test.go      # Configuration loading tests
└── validator_test.go   # Comprehensive validation tests

internal/commands/
└── config.go          # CLI integration and subcommands

timemachine.yaml       # Example configuration file
```

### Configuration Manager (`config.go`)

**Manager Struct**:
```go
type Manager struct {
    config    *Config          // Loaded configuration
    viper     *viper.Viper     # Viper instance for file/env handling
    validator *Validator       # Validation engine
}
```

**Key Responsibilities**:
- Multi-source configuration loading with precedence handling
- File system path resolution and search hierarchy
- Environment variable binding (security-hardened)
- Configuration validation orchestration
- Default value management

### Validation Engine (`validator.go`)

**Security-First Validation**:
- **Path Traversal Prevention**: Multiple attack vector detection (encoded paths, Windows patterns, relative sequences)
- **Safe Directory Enforcement**: Absolute paths restricted to safe locations
- **Type System Integration**: Deep validation of nested configuration structures
- **Field-Specific Rules**: Granular validation for each configuration property

**Validation Coverage**:
- Log configuration: Level/format validation, file path security
- Watcher configuration: Performance limits, pattern safety
- Cache configuration: Memory/performance bounds
- Git configuration: Operational limits, consistency checks
- UI configuration: Format validation, option constraints

### Configuration Structure

The system manages 5 main configuration sections:

```yaml
log:                    # Logging behavior
  level: info           # debug|info|warn|error
  format: text          # text|json
  file: ""             # optional log file path

watcher:                # File watching behavior
  debounce_delay: 2s    # 100ms-10s range
  max_watched_files: 100000  # 1K-1M range
  ignore_patterns: []   # additional ignore patterns
  batch_size: 100       # 1-1000 range
  enable_recursive: true

cache:                  # Caching behavior
  max_entries: 10000    # 1K-100K range
  max_memory_mb: 50     # 10-1024 MB range
  ttl: 1h              # 1m-24h range
  enable_lru: true

git:                    # Git operations
  cleanup_threshold: 100 # 10-10K range
  auto_gc: true
  max_commits: 1000     # 50-50K range
  use_shallow_clone: false

ui:                     # User interface
  progress_indicators: true
  color_output: true
  pager: auto           # auto|always|never
  table_format: table   # table|json|yaml
```

## Configuration File Search Hierarchy

The system searches for configuration files in the following order:

### 1. Project-Specific Configuration (Highest Priority)
- `./timemachine.yaml` (project root)
- `./.timemachine/timemachine.yaml` (project subdirectory)

### 2. User Configuration
- `~/.config/timemachine/timemachine.yaml` (XDG config directory)
- `~/timemachine.yaml` (home directory fallback)

### 3. System Configuration (Lowest Priority)
- `/etc/timemachine/timemachine.yaml`

## Environment Variable System

### Explicit Binding (Security Enhancement)

**CRITICAL SECURITY FIX**: Removed `viper.AutomaticEnv()` which allowed arbitrary environment variable injection. Now only explicitly defined variables are processed:

```go
allowedEnvVars := map[string]string{
    "TIMEMACHINE_LOG_LEVEL":            "log.level",
    "TIMEMACHINE_LOG_FORMAT":           "log.format", 
    "TIMEMACHINE_LOG_FILE":             "log.file",
    "TIMEMACHINE_WATCHER_DEBOUNCE":     "watcher.debounce_delay",
    "TIMEMACHINE_WATCHER_MAX_FILES":    "watcher.max_watched_files",
    "TIMEMACHINE_CACHE_MAX_ENTRIES":    "cache.max_entries",
    "TIMEMACHINE_CACHE_MAX_MEMORY":     "cache.max_memory_mb",
    "TIMEMACHINE_CACHE_TTL":            "cache.ttl",
    "TIMEMACHINE_GIT_CLEANUP_THRESHOLD": "git.cleanup_threshold",
    "TIMEMACHINE_GIT_AUTO_GC":          "git.auto_gc",
    "TIMEMACHINE_UI_COLOR":             "ui.color_output",
    "TIMEMACHINE_UI_PAGER":             "ui.pager",
}
```

## CLI Integration

### Configuration Commands (`internal/commands/config.go`)

Complete CLI integration with subcommands:

```bash
timemachine config init [--global] [--force]    # Create default config
timemachine config show [--format yaml|json]    # Display current config
timemachine config get <key>                    # Get specific value
timemachine config set <key> <value> [--global] # Set configuration value
timemachine config validate                     # Validate configuration
```

### AppState Integration

The configuration system integrates seamlessly with the application's `AppState`:

```go
type AppState struct {
    ConfigManager *config.Manager
    Config        *config.Config
    // ... other fields
}
```

## Performance Characteristics

### Memory Efficiency
- Lazy loading of configuration files
- Efficient struct mapping with `mapstructure` tags
- Minimal memory footprint for validation rules

### Startup Performance
- Fast configuration loading with minimal file I/O
- Cached validation rules
- Optimized path resolution

### Validation Performance
- O(1) lookup for enum validations
- Efficient regex-based path security checks
- Early termination on validation failures

## Error Handling Strategy

### Graceful Degradation
- Missing configuration files don't cause failures
- Invalid values fall back to validated defaults
- Detailed error messages for troubleshooting

### Comprehensive Error Reporting
```go
// Multi-error collection with context
var errors []string
if err := v.validateLogConfig(&config.Log); err != nil {
    errors = append(errors, fmt.Sprintf("log config: %v", err))
}
```

## Security Model

### Attack Surface Reduction
- **No Dynamic Environment Variables**: Prevents injection attacks
- **Path Traversal Protection**: Multiple validation layers
- **Type Safety**: Strong typing prevents malformed data
- **Permission Enforcement**: Secure file creation and access

### Validation Security
- Input sanitization at configuration load time
- URL decoding attack prevention
- Windows/Unix path traversal detection
- Safe directory whitelist enforcement

## Testing Strategy

### Comprehensive Test Coverage
- **Configuration Loading Tests** (`config_test.go`): 368+ lines covering all loading scenarios
- **Validation Tests** (`validator_test.go`): 593+ lines with 100+ test cases
- **Security Tests**: Path traversal, injection prevention
- **Integration Tests**: CLI command testing
- **Edge Case Coverage**: Error conditions, malformed inputs

### Test Categories
1. **Unit Tests**: Individual component validation
2. **Integration Tests**: Full configuration loading workflow
3. **Security Tests**: Attack scenario validation
4. **Performance Tests**: Load time and memory usage
5. **CLI Tests**: Command-line interface validation

## Future Enhancements

### Planned Features
- Configuration schema validation with JSON Schema
- Hot configuration reloading
- Configuration diff and merge utilities
- Remote configuration source support
- Configuration templating system

### Extensibility Points
- Plugin-based validation rules
- Custom configuration sources
- Dynamic configuration updates
- Configuration change notifications

This architecture provides a robust, secure, and maintainable foundation for TimeMachine CLI's configuration management, suitable for enterprise production environments.