# TimeMachine CLI Configuration Developer Guide

## Table of Contents
- [Integration Overview](#integration-overview)
- [Getting Started](#getting-started)
- [Configuration Loading](#configuration-loading)
- [Adding New Configuration Options](#adding-new-configuration-options)
- [Validation System](#validation-system)
- [Testing](#testing)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)
- [Migration Guide](#migration-guide)

## Integration Overview

The TimeMachine CLI configuration system is designed for easy integration with existing application components. The system provides:

- **Clean Dependency Injection**: Configuration is loaded once and passed to components
- **Type-Safe Access**: Strongly typed configuration structs
- **Hot Reloading Support**: Architecture supports runtime configuration updates
- **Comprehensive Testing**: Full test coverage for all integration scenarios

### Key Components

```go
// Core configuration types
type Config struct {
    Log     LogConfig     `mapstructure:"log"`
    Watcher WatcherConfig `mapstructure:"watcher"`
    Cache   CacheConfig   `mapstructure:"cache"`
    Git     GitConfig     `mapstructure:"git"`
    UI      UIConfig      `mapstructure:"ui"`
}

// Configuration manager
type Manager struct {
    config    *Config
    viper     *viper.Viper
    validator *Validator
}
```

## Getting Started

### Basic Integration

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/deepakkumarnarayana/timemachine-cli/internal/config"
    "github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

func main() {
    // Method 1: Use AppState (Recommended)
    state, err := core.NewAppState()
    if err != nil {
        log.Fatalf("Failed to initialize app state: %v", err)
    }
    
    // Access configuration through AppState
    logLevel := state.Config.Log.Level
    debounceDelay := state.Config.Watcher.DebounceDelay
    
    // Method 2: Direct configuration manager usage
    manager := config.NewManager()
    if err := manager.Load("/path/to/project"); err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    cfg := manager.Get()
    fmt.Printf("Log level: %s\n", cfg.Log.Level)
}
```

### AppState Integration

The recommended approach is using `AppState` which handles configuration loading automatically:

```go
// internal/core/state.go integration
type AppState struct {
    ProjectRoot     string
    GitRoot         string
    ShadowRepo      string
    ConfigManager   *config.Manager
    Config          *config.Config
    GitManager      *GitManager
    // ... other fields
}

func NewAppState() (*AppState, error) {
    // Configuration is loaded automatically
    configManager := config.NewManager()
    
    // Project root discovery and configuration loading
    projectRoot, err := discoverProjectRoot()
    if err != nil {
        return nil, err
    }
    
    if err := configManager.Load(projectRoot); err != nil {
        return nil, fmt.Errorf("failed to load configuration: %w", err)
    }
    
    return &AppState{
        ProjectRoot:   projectRoot,
        ConfigManager: configManager,
        Config:        configManager.Get(),
        // ... initialize other components
    }, nil
}
```

## Configuration Loading

### Loading Process

The configuration loading follows a specific sequence:

```go
func (m *Manager) Load(projectRoot string) error {
    // 1. Setup configuration file search paths
    if err := m.setupConfigPaths(projectRoot); err != nil {
        return fmt.Errorf("failed to setup config paths: %w", err)
    }
    
    // 2. Setup environment variables (security-hardened)
    m.setupEnvironmentVariables()
    
    // 3. Read configuration files (graceful if missing)
    if err := m.viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("failed to read config file: %w", err)
        }
    }
    
    // 4. Unmarshal into typed struct
    if err := m.viper.Unmarshal(m.config); err != nil {
        return fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // 5. Validate configuration
    if err := m.validator.Validate(m.config); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }
    
    return nil
}
```

### Error Handling

```go
// Proper error handling in your application
state, err := core.NewAppState()
if err != nil {
    switch {
    case strings.Contains(err.Error(), "configuration validation failed"):
        // Handle validation errors
        fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
        fmt.Fprintf(os.Stderr, "Run 'timemachine config validate' for details\n")
        os.Exit(1)
    case strings.Contains(err.Error(), "failed to read config file"):
        // Handle file reading errors
        fmt.Fprintf(os.Stderr, "Configuration file error: %v\n", err)
        fmt.Fprintf(os.Stderr, "Run 'timemachine config init' to create default configuration\n")
        os.Exit(1)
    default:
        // Handle other initialization errors
        log.Fatalf("Failed to initialize application: %v", err)
    }
}
```

## Adding New Configuration Options

### Step 1: Extend Configuration Structs

```go
// Add new field to appropriate config struct
type WatcherConfig struct {
    DebounceDelay    time.Duration `mapstructure:"debounce_delay" yaml:"debounce_delay" validate:"min=100ms,max=10s" default:"2s"`
    MaxWatchedFiles  int           `mapstructure:"max_watched_files" yaml:"max_watched_files" validate:"min=1000,max=1000000" default:"100000"`
    
    // New field example
    EnableCompression bool `mapstructure:"enable_compression" yaml:"enable_compression" default:"false"`
}
```

### Step 2: Add Default Values

```go
// Update setDefaults function in config.go
func setDefaults(v *viper.Viper) {
    // Existing defaults...
    v.SetDefault("watcher.debounce_delay", "2s")
    v.SetDefault("watcher.max_watched_files", 100000)
    
    // New default
    v.SetDefault("watcher.enable_compression", false)
}
```

### Step 3: Add Environment Variable Support

```go
// Update setupEnvironmentVariables in config.go
func (m *Manager) setupEnvironmentVariables() {
    allowedEnvVars := map[string]string{
        // Existing variables...
        "TIMEMACHINE_WATCHER_DEBOUNCE":     "watcher.debounce_delay",
        "TIMEMACHINE_WATCHER_MAX_FILES":    "watcher.max_watched_files",
        
        // New environment variable
        "TIMEMACHINE_WATCHER_COMPRESSION":  "watcher.enable_compression",
    }
    
    for env, key := range allowedEnvVars {
        m.viper.BindEnv(key, env)
    }
}
```

### Step 4: Add Validation

```go
// Update validateWatcherConfig in validator.go
func (v *Validator) validateWatcherConfig(config *WatcherConfig) error {
    var errors []string
    
    // Existing validations...
    
    // New validation (if needed)
    if config.EnableCompression {
        // Add any specific validation for compression
        if config.MaxWatchedFiles > 500000 {
            errors = append(errors, "compression not recommended with max_watched_files > 500000")
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("%s", strings.Join(errors, "; "))
    }
    return nil
}
```

### Step 5: Update Documentation

```yaml
# Update default configuration template
watcher:
  debounce_delay: 2s
  max_watched_files: 100000
  enable_compression: false  # New option with comment
```

### Step 6: Add Tests

```go
func TestWatcherCompressionConfig(t *testing.T) {
    tests := []struct {
        name        string
        config      WatcherConfig
        expectError bool
    }{
        {
            name: "compression enabled",
            config: WatcherConfig{
                EnableCompression: true,
                MaxWatchedFiles:   100000,
                DebounceDelay:     2 * time.Second,
                BatchSize:         100,
            },
            expectError: false,
        },
        {
            name: "compression with too many files",
            config: WatcherConfig{
                EnableCompression: true,
                MaxWatchedFiles:   600000,
                DebounceDelay:     2 * time.Second,
                BatchSize:         100,
            },
            expectError: true,
        },
    }
    
    validator := NewValidator()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.validateWatcherConfig(&tt.config)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Validation System

### Custom Validation Rules

```go
// Add custom validation to ValidateUpdate method
func (v *Validator) ValidateUpdate(field string, value interface{}) error {
    switch field {
    case "watcher.enable_compression":
        if enabled, ok := value.(bool); ok && enabled {
            // Custom validation logic
            return v.validateCompressionRequirements()
        }
        
    // Handle other fields...
    default:
        return v.validateType(field, value)
    }
    
    return nil
}

func (v *Validator) validateCompressionRequirements() error {
    // Implement custom validation logic
    return nil
}
```

### Cross-Field Validation

```go
func (v *Validator) Validate(config *Config) error {
    // Individual section validation
    if err := v.validateLogConfig(&config.Log); err != nil {
        return fmt.Errorf("log config: %v", err)
    }
    
    // Cross-field validation example
    if config.Watcher.EnableCompression && config.Cache.MaxMemoryMB < 100 {
        return fmt.Errorf("compression requires cache.max_memory_mb >= 100")
    }
    
    return nil
}
```

## Testing

### Configuration Loading Tests

```go
func TestConfigurationLoading(t *testing.T) {
    // Create temporary directory structure
    tmpDir := t.TempDir()
    
    // Create test configuration file
    configContent := `
log:
  level: debug
watcher:
  debounce_delay: 5s
`
    configPath := filepath.Join(tmpDir, "timemachine.yaml")
    require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))
    
    // Test configuration loading
    manager := config.NewManager()
    err := manager.Load(tmpDir)
    require.NoError(t, err)
    
    cfg := manager.Get()
    assert.Equal(t, "debug", cfg.Log.Level)
    assert.Equal(t, 5*time.Second, cfg.Watcher.DebounceDelay)
}
```

### Environment Variable Tests

```go
func TestEnvironmentVariableOverrides(t *testing.T) {
    // Set environment variables
    t.Setenv("TIMEMACHINE_LOG_LEVEL", "warn")
    t.Setenv("TIMEMACHINE_WATCHER_DEBOUNCE", "10s")
    
    manager := config.NewManager()
    err := manager.Load("")  // No config file
    require.NoError(t, err)
    
    cfg := manager.Get()
    assert.Equal(t, "warn", cfg.Log.Level)
    assert.Equal(t, 10*time.Second, cfg.Watcher.DebounceDelay)
}
```

### Validation Tests

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name        string
        config      Config
        expectError bool
        errorMatch  string
    }{
        {
            name: "valid configuration",
            config: Config{
                Log: LogConfig{Level: "info", Format: "text"},
                Watcher: WatcherConfig{
                    DebounceDelay:   2 * time.Second,
                    MaxWatchedFiles: 100000,
                    BatchSize:       100,
                },
                // ... other valid sections
            },
            expectError: false,
        },
        {
            name: "invalid log level",
            config: Config{
                Log: LogConfig{Level: "invalid", Format: "text"},
            },
            expectError: true,
            errorMatch:  "invalid log level",
        },
    }
    
    validator := config.NewValidator()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.Validate(&tt.config)
            if tt.expectError {
                require.Error(t, err)
                if tt.errorMatch != "" {
                    assert.Contains(t, err.Error(), tt.errorMatch)
                }
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

## Best Practices

### 1. Configuration Access Patterns

```go
// ✅ Good: Access through AppState
func NewWatcher(state *core.AppState) *Watcher {
    return &Watcher{
        debounceDelay: state.Config.Watcher.DebounceDelay,
        maxFiles:      state.Config.Watcher.MaxWatchedFiles,
        // ...
    }
}

// ❌ Avoid: Direct config manager access in components
func NewWatcher(manager *config.Manager) *Watcher {
    cfg := manager.Get()  // Avoid this pattern
    // ...
}
```

### 2. Configuration Validation

```go
// ✅ Good: Validate early
func NewComponent(config ComponentConfig) (*Component, error) {
    if config.Timeout < time.Second {
        return nil, fmt.Errorf("timeout must be at least 1 second")
    }
    return &Component{timeout: config.Timeout}, nil
}

// ❌ Avoid: Runtime validation failures
func (c *Component) Process() error {
    if c.timeout < time.Second {  // Too late!
        return fmt.Errorf("invalid timeout")
    }
    // ...
}
```

### 3. Environment Variable Naming

```go
// ✅ Good: Consistent naming with hierarchy
"TIMEMACHINE_WATCHER_DEBOUNCE"     -> "watcher.debounce_delay"
"TIMEMACHINE_CACHE_MAX_MEMORY"     -> "cache.max_memory_mb"

// ❌ Avoid: Inconsistent or flat naming
"TM_DEBOUNCE"                      // Too abbreviated
"TIMEMACHINE_WATCHER_DEBOUNCE_DELAY_MS"  // Too verbose
```

### 4. Default Values

```go
// ✅ Good: Reasonable defaults for all environments
v.SetDefault("watcher.debounce_delay", "2s")    // Works for most cases
v.SetDefault("cache.max_memory_mb", 50)         // Conservative memory usage

// ❌ Avoid: Extreme defaults
v.SetDefault("watcher.debounce_delay", "100ms") // Too aggressive
v.SetDefault("cache.max_memory_mb", 1024)       // Too much memory
```

### 5. Configuration Documentation

```go
type WatcherConfig struct {
    // Good: Clear documentation with examples
    DebounceDelay time.Duration `mapstructure:"debounce_delay" yaml:"debounce_delay" validate:"min=100ms,max=10s" default:"2s"`
    // Delay before creating snapshot after file changes.
    // Higher values reduce snapshot frequency but increase latency.
    // Examples: "500ms", "2s", "5s"
}
```

## Common Patterns

### Configuration-Driven Component Initialization

```go
// Pattern: Factory functions that accept configuration
func NewGitManager(cfg *config.Config, projectRoot string) (*GitManager, error) {
    return &GitManager{
        projectRoot:       projectRoot,
        cleanupThreshold: cfg.Git.CleanupThreshold,
        maxCommits:       cfg.Git.MaxCommits,
        autoGC:           cfg.Git.AutoGC,
    }, nil
}

// Usage in AppState initialization
func NewAppState() (*AppState, error) {
    configManager := config.NewManager()
    if err := configManager.Load(projectRoot); err != nil {
        return nil, err
    }
    
    cfg := configManager.Get()
    
    gitManager, err := NewGitManager(cfg, projectRoot)
    if err != nil {
        return nil, err
    }
    
    return &AppState{
        Config:     cfg,
        GitManager: gitManager,
    }, nil
}
```

### Configuration Hot Reloading

```go
// Pattern: Configuration update notification
type ConfigurableComponent interface {
    UpdateConfig(config *config.Config) error
}

func (m *Manager) Reload() error {
    newConfig := &Config{}
    if err := m.viper.Unmarshal(newConfig); err != nil {
        return err
    }
    
    if err := m.validator.Validate(newConfig); err != nil {
        return err
    }
    
    m.config = newConfig
    return nil
}
```

### Configuration Debugging

```go
// Pattern: Configuration introspection for debugging
func (m *Manager) GetConfigSources() map[string]interface{} {
    sources := make(map[string]interface{})
    
    // Configuration file used
    if file := m.viper.ConfigFileUsed(); file != "" {
        sources["config_file"] = file
    }
    
    // Environment variables
    envVars := make(map[string]string)
    for _, env := range os.Environ() {
        if strings.HasPrefix(env, "TIMEMACHINE_") {
            parts := strings.SplitN(env, "=", 2)
            if len(parts) == 2 {
                envVars[parts[0]] = parts[1]
            }
        }
    }
    if len(envVars) > 0 {
        sources["environment_variables"] = envVars
    }
    
    return sources
}
```

## Migration Guide

### Migrating from Direct Viper Usage

```go
// Old pattern: Direct viper usage
func oldInitialization() {
    viper.SetConfigName("config")
    viper.AddConfigPath(".")
    viper.ReadInConfig()
    
    logLevel := viper.GetString("log.level")
    // ... manual configuration handling
}

// New pattern: Use configuration manager
func newInitialization() {
    state, err := core.NewAppState()
    if err != nil {
        log.Fatalf("Configuration error: %v", err)
    }
    
    logLevel := state.Config.Log.Level
    // ... typed configuration access
}
```

### Adding Backward Compatibility

```go
// Support old configuration keys
func (m *Manager) Load(projectRoot string) error {
    // ... existing loading logic
    
    // Migration: Support old configuration keys
    if oldValue := m.viper.GetString("old_log_level"); oldValue != "" {
        m.viper.Set("log.level", oldValue)
    }
    
    // ... continue with validation
}
```

### Configuration Schema Evolution

```go
// Version-aware configuration loading
type Config struct {
    Version int           `mapstructure:"version" yaml:"version" default:"1"`
    Log     LogConfig     `mapstructure:"log" yaml:"log"`
    // ... other sections
}

func (m *Manager) migrateConfiguration(config *Config) error {
    switch config.Version {
    case 0, 1:
        // Current version, no migration needed
        return nil
    default:
        return fmt.Errorf("unsupported configuration version: %d", config.Version)
    }
}
```

This developer guide provides comprehensive information for integrating with and extending TimeMachine CLI's configuration system.