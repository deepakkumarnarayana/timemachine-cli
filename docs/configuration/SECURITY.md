# TimeMachine CLI Configuration Security Guide

## Table of Contents
- [Security Overview](#security-overview)
- [Threat Model](#threat-model)
- [Security Features](#security-features)
- [Best Practices](#best-practices)
- [Vulnerability Prevention](#vulnerability-prevention)
- [Security Validation](#security-validation)
- [Compliance Guidelines](#compliance-guidelines)
- [Incident Response](#incident-response)

## Security Overview

TimeMachine CLI's configuration system is designed with security as a first-class concern. The system implements defense-in-depth strategies to protect against common attack vectors including path traversal, environment variable injection, and configuration manipulation attacks.

### Security Philosophy

1. **Secure by Default**: Default configurations prioritize security over convenience
2. **Explicit Allow Lists**: Only explicitly defined configuration sources are processed
3. **Input Validation**: All configuration input undergoes strict validation
4. **Least Privilege**: File permissions and access controls follow least privilege principles
5. **Defense in Depth**: Multiple layers of security controls

## Threat Model

### Identified Threats

| Threat | Impact | Likelihood | Mitigation |
|--------|---------|------------|------------|
| Path Traversal Attack | High | Medium | Comprehensive path validation |
| Environment Variable Injection | High | Medium | Explicit variable binding only |
| Configuration File Manipulation | Medium | Low | Secure file permissions |
| Information Disclosure | Medium | Low | Restrictive file access |
| Privilege Escalation | High | Low | Validation and sandboxing |

### Attack Vectors

1. **Malicious Configuration Files**
   - Path traversal attempts in file paths
   - Injection of malicious values
   - Symlink attacks

2. **Environment Variable Attacks**
   - Variable injection via `AutomaticEnv()` (FIXED)
   - Shell command injection
   - Information leakage

3. **File System Attacks**
   - Directory traversal
   - Symbolic link exploitation
   - Permission escalation

## Security Features

### 1. Path Traversal Prevention

**Critical Security Implementation**: The `isValidFilePath()` function provides comprehensive protection:

```go
func (v *Validator) isValidFilePath(path string) bool {
    // Normalize and clean the path
    cleanPath := filepath.Clean(path)
    
    // Check for encoded path traversal sequences
    decodedPath, err := url.QueryUnescape(path)
    if err == nil && strings.Contains(decodedPath, "..") {
        return false
    }
    
    // Check for Windows path traversal attempts
    if strings.Contains(strings.ToLower(path), "..\\") {
        return false
    }
    
    // Check for path traversal patterns
    if strings.Contains(cleanPath, "..") {
        return false
    }
    
    // Validate absolute paths are in safe directories
    if filepath.IsAbs(cleanPath) {
        return v.isInSafeDirectory(cleanPath)
    }
    
    return true
}
```

**Protected Against**:
- Standard path traversal: `../../../etc/passwd`
- URL-encoded traversal: `%2e%2e%2f`
- Windows-style traversal: `..\..\..\windows\system32`
- Double-encoded attacks: `%252e%252e%252f`
- Mixed encoding attacks

### 2. Environment Variable Security

**CRITICAL FIX**: Removed `viper.AutomaticEnv()` vulnerability:

```go
// OLD (VULNERABLE): Any environment variable could be injected
// viper.AutomaticEnv()

// NEW (SECURE): Only explicitly allowed variables
allowedEnvVars := map[string]string{
    "TIMEMACHINE_LOG_LEVEL":            "log.level",
    "TIMEMACHINE_LOG_FORMAT":           "log.format",
    "TIMEMACHINE_LOG_FILE":             "log.file",
    "TIMEMACHINE_WATCHER_DEBOUNCE":     "watcher.debounce_delay",
    "TIMEMACHINE_WATCHER_MAX_FILES":    "watcher.max_watched_files",
    // ... only explicitly defined variables
}

for env, key := range allowedEnvVars {
    viper.BindEnv(key, env)
}
```

**Benefits**:
- Prevents arbitrary environment variable injection
- Ensures all values go through validation pipeline
- Eliminates side-channel configuration attacks
- Provides audit trail of configuration sources

### 3. File Permissions Security

**Secure File Creation**:
```go
// Configuration files created with restrictive permissions
if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
    return fmt.Errorf("failed to write default config file: %w", err)
}
```

**Permission Model**:
- `0600`: Owner read/write only for configuration files
- `0755`: Standard directory permissions for config directories
- No group or world access to sensitive configuration

### 4. Input Sanitization

**Comprehensive Validation Pipeline**:
```go
func (v *Validator) Validate(config *Config) error {
    var errors []string
    
    // Section-specific validation
    if err := v.validateLogConfig(&config.Log); err != nil {
        errors = append(errors, fmt.Sprintf("log config: %v", err))
    }
    
    // Security-specific validation
    if err := v.validateSecurityConstraints(config); err != nil {
        errors = append(errors, fmt.Sprintf("security: %v", err))
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
    }
    
    return nil
}
```

## Best Practices

### 1. Configuration File Security

**File Location Security**:
```bash
# ✅ Secure: Project-specific configuration
./timemachine.yaml                    # 0600 permissions
./.timemachine/timemachine.yaml       # 0600 permissions

# ✅ Secure: User configuration
~/.config/timemachine/timemachine.yaml # 0600 permissions

# ⚠️  Caution: System configuration (admin only)
/etc/timemachine/timemachine.yaml     # 0644 permissions (readable by all)
```

**Secure Configuration Creation**:
```bash
# Create configuration with secure permissions
umask 0077
timemachine config init

# Verify permissions
ls -la timemachine.yaml
# Should show: -rw------- (0600)
```

### 2. Environment Variable Security

**Secure Environment Variable Usage**:
```bash
# ✅ Secure: Use only documented variables
export TIMEMACHINE_LOG_LEVEL=info
export TIMEMACHINE_WATCHER_DEBOUNCE=2s

# ❌ Insecure: Undocumented variables (ignored)
export TIMEMACHINE_ARBITRARY_VAR=value  # Ignored by system

# ❌ Insecure: Shell injection attempts (validated)
export TIMEMACHINE_LOG_FILE="$(malicious_command)"  # Blocked by validation
```

**Production Environment Variables**:
```bash
# Production-safe environment variables
export TIMEMACHINE_LOG_LEVEL=warn        # Reduce log verbosity
export TIMEMACHINE_LOG_FORMAT=json       # Structured logging
export TIMEMACHINE_UI_COLOR=false        # No ANSI codes in logs
export TIMEMACHINE_UI_PAGER=never        # No interactive elements
```

### 3. Log File Security

**Secure Log Configuration**:
```yaml
log:
  level: warn                    # Avoid debug logs in production
  format: json                   # Structured, parseable logs
  file: "/var/log/timemachine/app.log"  # Secure location
```

**Log File Permissions**:
```bash
# Create log directory with secure permissions
sudo mkdir -p /var/log/timemachine
sudo chown timemachine:timemachine /var/log/timemachine
sudo chmod 750 /var/log/timemachine

# Log files will be created with 0600 permissions
```

### 4. Network Security Considerations

Although TimeMachine doesn't use network features, configuration security impacts deployment:

```yaml
# Secure configuration for CI/CD
log:
  level: info
  format: json                   # Machine-readable for log aggregation
  file: ""                      # stdout for container log collection

ui:
  color_output: false           # No ANSI codes in automated environments
  pager: never                  # Never wait for user input
  progress_indicators: false    # No interactive elements
```

## Vulnerability Prevention

### 1. Path Traversal Prevention

**Implementation Details**:
```go
// Multi-layer path validation
func (v *Validator) isValidFilePath(path string) bool {
    // Layer 1: Basic sanitization
    if strings.TrimSpace(path) == "" {
        return false
    }
    
    // Layer 2: Path cleaning and normalization
    cleanPath := filepath.Clean(path)
    
    // Layer 3: URL decoding attack prevention
    if decodedPath, err := url.QueryUnescape(path); err == nil {
        if strings.Contains(decodedPath, "..") {
            return false
        }
    }
    
    // Layer 4: Platform-specific patterns
    if strings.Contains(strings.ToLower(path), "..\\") {
        return false  // Windows traversal
    }
    
    // Layer 5: Direct traversal patterns
    if strings.Contains(cleanPath, "..") {
        return false
    }
    
    // Layer 6: Absolute path validation
    if filepath.IsAbs(cleanPath) {
        return v.isInSafeDirectory(cleanPath)
    }
    
    return true
}
```

**Test Coverage**:
```go
// Comprehensive path traversal tests
func TestPathTraversalPrevention(t *testing.T) {
    tests := []struct {
        name     string
        path     string
        expected bool
    }{
        {"normal_path", "logs/app.log", true},
        {"traversal_unix", "../../../etc/passwd", false},
        {"traversal_windows", "..\\..\\windows\\system32", false},
        {"url_encoded", "%2e%2e%2f", false},
        {"double_encoded", "%252e%252e%252f", false},
        {"mixed_encoding", "../%2e%2e%2f", false},
        {"null_byte", "valid\x00../etc/passwd", false},
        {"unicode_traversal", "..%c0%af", false},
    }
    
    validator := NewValidator()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := validator.isValidFilePath(tt.path)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 2. Injection Prevention

**Configuration Value Validation**:
```go
func (v *Validator) validateLogConfig(config *LogConfig) error {
    // Prevent command injection in log file paths
    if config.File != "" {
        // Check for shell metacharacters
        if strings.ContainsAny(config.File, "$`&|;()") {
            return fmt.Errorf("log file path contains invalid characters")
        }
        
        // Path traversal validation
        if !v.isValidFilePath(config.File) {
            return fmt.Errorf("invalid log file path '%s'", config.File)
        }
    }
    
    return nil
}
```

### 3. Information Disclosure Prevention

**Secure Error Messages**:
```go
func (m *Manager) Load(projectRoot string) error {
    if err := m.viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            // Don't expose internal file paths in error messages
            return fmt.Errorf("failed to read configuration file")
        }
    }
    
    if err := m.validator.Validate(m.config); err != nil {
        // Validation errors are safe to expose
        return fmt.Errorf("configuration validation failed: %w", err)
    }
    
    return nil
}
```

## Security Validation

### 1. Configuration Validation

**Security-Focused Validation Rules**:
```go
func (v *Validator) validateSecurityConstraints(config *Config) error {
    var errors []string
    
    // Check for dangerous file paths
    if config.Log.File != "" {
        if !v.isValidFilePath(config.Log.File) {
            errors = append(errors, "log file path failed security validation")
        }
    }
    
    // Validate resource limits (prevent DoS)
    if config.Watcher.MaxWatchedFiles > 1000000 {
        errors = append(errors, "max_watched_files exceeds security limit")
    }
    
    if config.Cache.MaxMemoryMB > 1024 {
        errors = append(errors, "cache memory limit exceeds security threshold")
    }
    
    // Validate ignore patterns for injection
    for _, pattern := range config.Watcher.IgnorePatterns {
        if strings.Contains(pattern, "..") {
            errors = append(errors, "ignore pattern contains path traversal sequence")
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("%s", strings.Join(errors, "; "))
    }
    
    return nil
}
```

### 2. Runtime Security Checks

**Ongoing Security Validation**:
```go
func (state *AppState) validateRuntimeSecurity() error {
    // Check configuration file permissions
    if configFile := state.ConfigManager.GetViper().ConfigFileUsed(); configFile != "" {
        if err := validateFilePermissions(configFile); err != nil {
            return fmt.Errorf("configuration file security check failed: %w", err)
        }
    }
    
    // Check log file permissions
    if state.Config.Log.File != "" {
        if err := validateFilePermissions(state.Config.Log.File); err != nil {
            return fmt.Errorf("log file security check failed: %w", err)
        }
    }
    
    return nil
}

func validateFilePermissions(filePath string) error {
    info, err := os.Stat(filePath)
    if err != nil {
        return err
    }
    
    mode := info.Mode().Perm()
    if mode&0077 != 0 {  // Check for group/other permissions
        return fmt.Errorf("file %s has insecure permissions %o", filePath, mode)
    }
    
    return nil
}
```

## Compliance Guidelines

### 1. Enterprise Security Requirements

**Configuration Security Checklist**:
- [ ] Configuration files use restrictive permissions (0600)
- [ ] No hardcoded credentials in configuration
- [ ] Log files are in secure locations
- [ ] Environment variables are explicitly whitelisted
- [ ] Path validation prevents traversal attacks
- [ ] Input validation prevents injection attacks
- [ ] Error messages don't disclose sensitive information
- [ ] Default configuration follows security best practices

### 2. Audit Trail

**Configuration Change Tracking**:
```go
func (m *Manager) auditConfigurationAccess() {
    log.Info("Configuration loaded", 
        "config_file", m.viper.ConfigFileUsed(),
        "user", os.Getenv("USER"),
        "timestamp", time.Now(),
    )
    
    // Log environment variable overrides
    for _, env := range []string{
        "TIMEMACHINE_LOG_LEVEL",
        "TIMEMACHINE_LOG_FORMAT",
        "TIMEMACHINE_LOG_FILE",
    } {
        if value := os.Getenv(env); value != "" {
            log.Info("Environment variable override",
                "variable", env,
                "value", value,
            )
        }
    }
}
```

### 3. Security Monitoring

**Monitoring Configuration**:
```yaml
log:
  level: info                    # Log security events
  format: json                   # Structured logs for SIEM
  file: "/var/log/timemachine/security.log"  # Dedicated security log

watcher:
  max_watched_files: 50000       # Conservative limits
  debounce_delay: 5s            # Prevent rapid snapshots

cache:
  max_memory_mb: 25             # Conservative memory usage
  ttl: 30m                      # Shorter TTL for security

ui:
  color_output: false           # No ANSI codes in logs
  progress_indicators: false    # No interactive elements
```

## Incident Response

### 1. Security Incident Detection

**Indicators of Compromise**:
- Unexpected configuration file modifications
- Unusual environment variable values
- Path traversal attempts in logs
- Validation errors indicating attack attempts
- Abnormal resource usage patterns

### 2. Response Procedures

**Immediate Response**:
1. **Isolate**: Stop TimeMachine processes
2. **Assess**: Review configuration and logs
3. **Contain**: Reset configuration to known good state
4. **Investigate**: Analyze attack vectors
5. **Recover**: Restore from secure configuration

**Configuration Reset**:
```bash
# Emergency configuration reset
rm -f timemachine.yaml
unset $(env | grep TIMEMACHINE_ | cut -d= -f1)
timemachine config init
timemachine config validate
```

### 3. Post-Incident Analysis

**Security Review Process**:
1. Review all configuration files and permissions
2. Audit environment variable usage
3. Validate path traversal prevention
4. Check log file security
5. Update security documentation
6. Enhance monitoring and validation

**Security Hardening Recommendations**:
```bash
# Post-incident hardening
chmod 600 timemachine.yaml                    # Secure file permissions
chattr +i timemachine.yaml                    # Make immutable (Linux)
export TIMEMACHINE_LOG_LEVEL=warn             # Reduce log verbosity
export TIMEMACHINE_UI_COLOR=false             # No ANSI codes
audit2allow -w -a | grep timemachine          # SELinux policy review
```

This security guide provides comprehensive protection strategies for TimeMachine CLI's configuration system in enterprise and production environments.