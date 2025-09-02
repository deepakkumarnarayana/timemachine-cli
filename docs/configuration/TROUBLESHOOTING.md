# TimeMachine CLI Configuration Troubleshooting Guide

## Table of Contents
- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
- [Configuration Loading Problems](#configuration-loading-problems)
- [Validation Errors](#validation-errors)
- [Environment Variable Issues](#environment-variable-issues)
- [File Permission Problems](#file-permission-problems)
- [Performance Issues](#performance-issues)
- [Migration Issues](#migration-issues)
- [Advanced Debugging](#advanced-debugging)

## Quick Diagnostics

### First Steps for Any Configuration Issue

1. **Validate Current Configuration**
   ```bash
   timemachine config validate
   ```

2. **Check Configuration Sources**
   ```bash
   timemachine config show
   ```

3. **Verify Environment Variables**
   ```bash
   env | grep TIMEMACHINE_
   ```

4. **Check File Permissions**
   ```bash
   ls -la timemachine.yaml
   ls -la ~/.config/timemachine/
   ```

### Configuration Health Check Script

```bash
#!/bin/bash
# config-healthcheck.sh - Configuration diagnostic script

echo "=== TimeMachine Configuration Health Check ==="

# Check for configuration files
echo "1. Configuration file locations:"
for path in "./timemachine.yaml" "./.timemachine/timemachine.yaml" "$HOME/.config/timemachine/timemachine.yaml"; do
    if [ -f "$path" ]; then
        echo "  ✅ Found: $path"
        ls -la "$path"
    else
        echo "  ❌ Not found: $path"
    fi
done

# Check environment variables
echo "2. Environment variables:"
env | grep TIMEMACHINE_ | while read var; do
    echo "  ✅ $var"
done

# Validate configuration
echo "3. Configuration validation:"
if timemachine config validate 2>/dev/null; then
    echo "  ✅ Configuration is valid"
else
    echo "  ❌ Configuration validation failed"
    timemachine config validate
fi

# Check file permissions
echo "4. File permissions:"
if [ -f "timemachine.yaml" ]; then
    perms=$(stat -c "%a" timemachine.yaml 2>/dev/null || stat -f "%Lp" timemachine.yaml)
    if [ "$perms" = "600" ]; then
        echo "  ✅ Configuration file permissions: $perms"
    else
        echo "  ⚠️  Configuration file permissions: $perms (recommended: 600)"
    fi
fi

echo "=== Health Check Complete ==="
```

## Common Issues

### Issue 1: "Configuration file not found"

**Symptoms:**
```
Warning: No configuration file found, using defaults
```

**Causes:**
- No configuration file exists
- Configuration file in wrong location
- Incorrect file name

**Solutions:**

```bash
# Solution 1: Create default configuration
timemachine config init

# Solution 2: Create global configuration
timemachine config init --global

# Solution 3: Check expected locations
echo "TimeMachine looks for configuration in these locations:"
echo "1. ./timemachine.yaml"
echo "2. ./.timemachine/timemachine.yaml"
echo "3. ~/.config/timemachine/timemachine.yaml"
echo "4. ~/timemachine.yaml"
echo "5. /etc/timemachine/timemachine.yaml"
```

### Issue 2: "Configuration validation failed"

**Symptoms:**
```
Error: configuration validation failed: log config: invalid log level 'verbose'
```

**Causes:**
- Invalid configuration values
- Typos in configuration keys
- Wrong data types

**Solutions:**

```bash
# Solution 1: Check validation rules
timemachine config validate

# Solution 2: View current configuration
timemachine config show

# Solution 3: Reset to defaults
mv timemachine.yaml timemachine.yaml.backup
timemachine config init

# Solution 4: Fix specific validation errors
# See validation error reference below
```

### Issue 3: "Permission denied"

**Symptoms:**
```
Error: failed to create configuration file: permission denied
```

**Causes:**
- Insufficient file system permissions
- Directory doesn't exist
- File is read-only

**Solutions:**

```bash
# Solution 1: Check current permissions
ls -la timemachine.yaml

# Solution 2: Fix permissions
chmod 600 timemachine.yaml

# Solution 3: Create directory if needed
mkdir -p ~/.config/timemachine
timemachine config init --global

# Solution 4: Check directory permissions
ls -la ~/.config/
ls -la .
```

### Issue 4: Environment variables not working

**Symptoms:**
```
Environment variables are set but not affecting configuration
```

**Causes:**
- Incorrect environment variable names
- Wrong data format
- Variables not exported

**Solutions:**

```bash
# Solution 1: Check variable names (must match exactly)
export TIMEMACHINE_LOG_LEVEL=debug          # ✅ Correct
export TIMEMACHINE_LOGLEVEL=debug           # ❌ Incorrect

# Solution 2: Export variables
TIMEMACHINE_LOG_LEVEL=debug                 # ❌ Not exported
export TIMEMACHINE_LOG_LEVEL=debug          # ✅ Exported

# Solution 3: Check supported variables
echo "Supported environment variables:"
echo "TIMEMACHINE_LOG_LEVEL, TIMEMACHINE_LOG_FORMAT, TIMEMACHINE_LOG_FILE"
echo "TIMEMACHINE_WATCHER_DEBOUNCE, TIMEMACHINE_WATCHER_MAX_FILES"
echo "TIMEMACHINE_CACHE_MAX_ENTRIES, TIMEMACHINE_CACHE_MAX_MEMORY, TIMEMACHINE_CACHE_TTL"
echo "TIMEMACHINE_GIT_CLEANUP_THRESHOLD, TIMEMACHINE_GIT_AUTO_GC"
echo "TIMEMACHINE_UI_COLOR, TIMEMACHINE_UI_PAGER"
```

## Configuration Loading Problems

### Problem: Configuration not loading in correct order

**Debugging Steps:**

1. **Check precedence understanding**
   ```
   Priority (highest to lowest):
   1. Command-line flags
   2. Environment variables  
   3. Configuration files
   4. Default values
   ```

2. **Verify which configuration file is being used**
   ```bash
   timemachine config validate  # Shows active config file
   ```

3. **Test each source individually**
   ```bash
   # Test with no config file or env vars
   mv timemachine.yaml timemachine.yaml.bak
   unset $(env | grep TIMEMACHINE_ | cut -d= -f1)
   timemachine config show
   
   # Test with config file only
   mv timemachine.yaml.bak timemachine.yaml
   timemachine config show
   
   # Test with environment variables
   export TIMEMACHINE_LOG_LEVEL=debug
   timemachine config show
   ```

### Problem: Configuration file found but not parsed

**Common YAML Syntax Issues:**

```yaml
# ❌ YAML syntax errors

# Wrong indentation
log:
level: info        # Should be indented

# Missing colons
log
  level info       # Should be "level: info"

# Wrong quotes
log:
  level: 'debug"   # Mismatched quotes

# Invalid characters
log:
  level: info      # Tab character instead of spaces
```

```yaml
# ✅ Correct YAML syntax

log:
  level: info
  format: text
  file: ""

watcher:
  debounce_delay: 2s
  max_watched_files: 100000
```

**Validation Tools:**
```bash
# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('timemachine.yaml'))"

# Or using yq
yq eval . timemachine.yaml

# Or using online validator
# Copy configuration to: https://yamlvalidator.com/
```

## Validation Errors

### Complete Validation Error Reference

#### Log Configuration Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid log level 'verbose'` | Invalid log level | Use: `debug`, `info`, `warn`, `error` |
| `invalid log format 'plain'` | Invalid log format | Use: `text`, `json` |
| `invalid log file path` | Path traversal attempt | Use safe paths, avoid `..` sequences |

```yaml
# ✅ Valid log configuration
log:
  level: debug    # debug|info|warn|error
  format: json    # text|json
  file: "/tmp/app.log"  # valid path only
```

#### Watcher Configuration Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `debounce_delay must be at least 100ms` | Value too small | Minimum: `100ms` |
| `debounce_delay must be at most 10s` | Value too large | Maximum: `10s` |
| `max_watched_files must be at least 1000` | Value too small | Minimum: `1000` |
| `max_watched_files must be at most 1000000` | Value too large | Maximum: `1000000` |
| `batch_size must be at least 1` | Value too small | Minimum: `1` |
| `batch_size must be at most 1000` | Value too large | Maximum: `1000` |

```yaml
# ✅ Valid watcher configuration
watcher:
  debounce_delay: 2s        # 100ms - 10s
  max_watched_files: 100000 # 1,000 - 1,000,000
  batch_size: 100          # 1 - 1,000
  ignore_patterns: []      # no '..' sequences
  enable_recursive: true
```

#### Cache Configuration Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `max_entries must be at least 1000` | Value too small | Minimum: `1000` |
| `max_entries must be at most 100000` | Value too large | Maximum: `100000` |
| `max_memory_mb must be at least 10` | Value too small | Minimum: `10` |
| `max_memory_mb must be at most 1024` | Value too large | Maximum: `1024` |
| `ttl must be at least 1m` | Value too small | Minimum: `1m` |
| `ttl must be at most 24h` | Value too large | Maximum: `24h` |

```yaml
# ✅ Valid cache configuration
cache:
  max_entries: 10000    # 1,000 - 100,000
  max_memory_mb: 50     # 10 - 1,024
  ttl: 1h              # 1m - 24h
  enable_lru: true
```

#### Git Configuration Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `cleanup_threshold must be at least 10` | Value too small | Minimum: `10` |
| `cleanup_threshold must be at most 10000` | Value too large | Maximum: `10000` |
| `max_commits must be at least 50` | Value too small | Minimum: `50` |
| `max_commits must be at most 50000` | Value too large | Maximum: `50000` |
| `cleanup_threshold must be less than max_commits` | Logical error | Ensure cleanup < max |

```yaml
# ✅ Valid git configuration
git:
  cleanup_threshold: 100    # 10 - 10,000 (< max_commits)
  auto_gc: true
  max_commits: 1000        # 50 - 50,000 (> cleanup_threshold)
  use_shallow_clone: false
```

#### UI Configuration Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid pager setting 'sometimes'` | Invalid pager value | Use: `auto`, `always`, `never` |
| `invalid table_format 'csv'` | Invalid table format | Use: `table`, `json`, `yaml` |

```yaml
# ✅ Valid UI configuration
ui:
  progress_indicators: true
  color_output: true
  pager: auto           # auto|always|never
  table_format: table   # table|json|yaml
```

## Environment Variable Issues

### Issue: Environment variables have wrong format

**Duration Values:**
```bash
# ❌ Wrong format
export TIMEMACHINE_WATCHER_DEBOUNCE=2000        # Missing unit
export TIMEMACHINE_CACHE_TTL=3600               # Missing unit

# ✅ Correct format
export TIMEMACHINE_WATCHER_DEBOUNCE=2s          # With unit
export TIMEMACHINE_CACHE_TTL=1h                 # With unit
```

**Boolean Values:**
```bash
# ❌ Wrong format
export TIMEMACHINE_GIT_AUTO_GC=1               # Use true/false
export TIMEMACHINE_UI_COLOR=yes                # Use true/false

# ✅ Correct format
export TIMEMACHINE_GIT_AUTO_GC=true            # Boolean true
export TIMEMACHINE_UI_COLOR=false              # Boolean false
```

**Numeric Values:**
```bash
# ❌ Wrong format
export TIMEMACHINE_CACHE_MAX_MEMORY="50 MB"    # No units in value
export TIMEMACHINE_WATCHER_MAX_FILES="100k"    # No suffixes

# ✅ Correct format
export TIMEMACHINE_CACHE_MAX_MEMORY=50          # Plain number
export TIMEMACHINE_WATCHER_MAX_FILES=100000    # Plain number
```

### Issue: Environment variables not taking effect

**Debugging Steps:**

1. **Check variable names exactly**
   ```bash
   # Case sensitive and must match exactly
   export TIMEMACHINE_LOG_LEVEL=debug     # ✅
   export timemachine_log_level=debug     # ❌ Wrong case
   export TM_LOG_LEVEL=debug              # ❌ Wrong prefix
   ```

2. **Verify variables are exported**
   ```bash
   # Check if variable is in environment
   env | grep TIMEMACHINE_LOG_LEVEL
   
   # If not found, export it
   export TIMEMACHINE_LOG_LEVEL=debug
   ```

3. **Test variable isolation**
   ```bash
   # Test with single variable
   unset $(env | grep TIMEMACHINE_ | cut -d= -f1)
   export TIMEMACHINE_LOG_LEVEL=debug
   timemachine config show | grep "level: debug"
   ```

## File Permission Problems

### Issue: Cannot create configuration file

**Error Symptoms:**
```
Error: failed to write default config file: permission denied
```

**Solutions:**

```bash
# Check current directory permissions
ls -la .
ls -la ~/.config/

# Create directories if needed
mkdir -p ~/.config/timemachine
chmod 755 ~/.config/timemachine

# Fix directory permissions
sudo chown $USER:$USER ~/.config/timemachine

# For system-wide configuration
sudo mkdir -p /etc/timemachine
sudo timemachine config init --global
```

### Issue: Configuration file has wrong permissions

**Security Warning:**
```
Warning: Configuration file has insecure permissions 644
```

**Fix Permissions:**
```bash
# Fix configuration file permissions
chmod 600 timemachine.yaml
chmod 600 ~/.config/timemachine/timemachine.yaml

# Verify permissions
ls -la timemachine.yaml
# Should show: -rw------- (600)
```

### Issue: Cannot read configuration file

**Error:**
```
Error: failed to read config file: permission denied
```

**Solutions:**
```bash
# Check file ownership
ls -la timemachine.yaml

# Fix ownership if needed
sudo chown $USER:$USER timemachine.yaml

# Check parent directory permissions
ls -la .
ls -la ~/.config/
```

## Performance Issues

### Issue: Configuration loading is slow

**Symptoms:**
- Application startup takes several seconds
- Configuration validation is slow

**Debugging:**
```bash
# Time configuration loading
time timemachine config validate

# Check file system performance
time ls -la timemachine.yaml

# Check configuration file size
du -h timemachine.yaml
```

**Solutions:**

1. **Reduce configuration file size**
   ```yaml
   # Remove unnecessary comments and whitespace
   # Use minimal configuration with defaults
   
   log:
     level: info
   watcher:
     debounce_delay: 2s
   ```

2. **Optimize file system access**
   ```bash
   # Move configuration to faster storage
   # Avoid network file systems for config files
   ```

3. **Check for file system issues**
   ```bash
   # Check disk space
   df -h .
   
   # Check inode usage
   df -i .
   
   # Check for file system errors
   sudo fsck /dev/sda1  # Replace with your device
   ```

### Issue: High memory usage during configuration loading

**Symptoms:**
- Application uses excessive memory
- Out of memory errors during startup

**Debugging:**
```bash
# Monitor memory usage
top -p $(pgrep timemachine)

# Check configuration values
timemachine config get cache.max_memory_mb
timemachine config get cache.max_entries
timemachine config get watcher.max_watched_files
```

**Solutions:**

```yaml
# Reduce memory-intensive settings
cache:
  max_entries: 5000      # Reduce from default 10000
  max_memory_mb: 25      # Reduce from default 50

watcher:
  max_watched_files: 50000  # Reduce from default 100000
  batch_size: 50           # Reduce from default 100
```

## Migration Issues

### Issue: Upgrading from older TimeMachine versions

**Pre-migration Checklist:**
```bash
# Backup current configuration
cp timemachine.yaml timemachine.yaml.backup

# Check current version
timemachine --version

# Validate current configuration
timemachine config validate
```

### Issue: Configuration format changes

**Symptoms:**
```
Warning: Unknown configuration key 'old_key_name'
Error: Required configuration key 'new_key_name' missing
```

**Migration Steps:**

1. **Check for deprecated keys**
   ```bash
   # Review configuration for deprecated keys
   grep -E "(old_key|deprecated)" timemachine.yaml
   ```

2. **Use migration script** (if available)
   ```bash
   # Run migration utility
   timemachine config migrate --from-version 1.0 --to-version 2.0
   ```

3. **Manual migration**
   ```yaml
   # Old format (deprecated)
   log_level: debug
   watcher_delay: 2000ms
   
   # New format (current)
   log:
     level: debug
   watcher:
     debounce_delay: 2s
   ```

### Issue: Environment variable format changes

**Old vs New Format:**
```bash
# Old format (if applicable)
export TM_LOG_LEVEL=debug
export TM_DEBOUNCE=2s

# New format (current)
export TIMEMACHINE_LOG_LEVEL=debug  
export TIMEMACHINE_WATCHER_DEBOUNCE=2s
```

## Advanced Debugging

### Enable Debug Logging

```bash
# Enable maximum verbosity
export TIMEMACHINE_LOG_LEVEL=debug
export TIMEMACHINE_LOG_FORMAT=json
export TIMEMACHINE_LOG_FILE=/tmp/debug.log

# Run with debug output
timemachine config validate
timemachine config show

# Review debug log
cat /tmp/debug.log | jq '.'
```

### Configuration Loading Trace

```bash
# Trace configuration loading
strace -e trace=openat,read timemachine config show 2>&1 | grep -E "(timemachine|config)"

# On macOS
dtruss -n timemachine 2>&1 | grep -E "(timemachine|config)"
```

### Viper Debug Mode

```go
// Enable in development
package main

import (
    "github.com/spf13/viper"
    "github.com/spf13/cobra"
)

func init() {
    // Enable viper debug mode
    viper.SetEnvPrefix("TIMEMACHINE")
    viper.Debug()  // Enables debug output
}
```

### Memory Profiling

```bash
# Profile memory usage
go tool pprof -http=:8080 timemachine /tmp/mem.prof

# Enable memory profiling in development
export TIMEMACHINE_ENABLE_PROFILING=true
timemachine config validate
```

### Configuration State Dump

```bash
# Create comprehensive state dump
echo "=== Configuration State Dump ===" > config-debug.txt
echo "Date: $(date)" >> config-debug.txt
echo "User: $(whoami)" >> config-debug.txt
echo "Working Directory: $(pwd)" >> config-debug.txt
echo >> config-debug.txt

echo "Environment Variables:" >> config-debug.txt
env | grep TIMEMACHINE_ >> config-debug.txt
echo >> config-debug.txt

echo "Configuration Files:" >> config-debug.txt
find . -name "timemachine.yaml" -exec ls -la {} \; >> config-debug.txt
find ~/.config -name "timemachine.yaml" -exec ls -la {} \; 2>/dev/null >> config-debug.txt
echo >> config-debug.txt

echo "Configuration Validation:" >> config-debug.txt
timemachine config validate 2>&1 >> config-debug.txt
echo >> config-debug.txt

echo "Current Configuration:" >> config-debug.txt
timemachine config show 2>&1 >> config-debug.txt

echo "=== End Debug Dump ===" >> config-debug.txt
```

This troubleshooting guide covers the most common configuration issues and provides comprehensive solutions for diagnosing and fixing configuration problems in TimeMachine CLI.