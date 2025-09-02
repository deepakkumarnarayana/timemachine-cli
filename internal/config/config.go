package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Log     LogConfig     `mapstructure:"log" yaml:"log" validate:"dive"`
	Watcher WatcherConfig `mapstructure:"watcher" yaml:"watcher" validate:"dive"`
	Cache   CacheConfig   `mapstructure:"cache" yaml:"cache" validate:"dive"`
	Git     GitConfig     `mapstructure:"git" yaml:"git" validate:"dive"`
	UI      UIConfig      `mapstructure:"ui" yaml:"ui" validate:"dive"`
}

// LogConfig controls logging behavior
type LogConfig struct {
	Level  string `mapstructure:"level" yaml:"level" validate:"oneof=debug info warn error" default:"info"`
	Format string `mapstructure:"format" yaml:"format" validate:"oneof=text json" default:"text"`
	File   string `mapstructure:"file" yaml:"file" default:""`
}

// WatcherConfig controls file watching behavior
type WatcherConfig struct {
	DebounceDelay    time.Duration `mapstructure:"debounce_delay" yaml:"debounce_delay" validate:"min=100ms,max=10s" default:"2s"`
	MaxWatchedFiles  int           `mapstructure:"max_watched_files" yaml:"max_watched_files" validate:"min=1000,max=1000000" default:"100000"`
	IgnorePatterns   []string      `mapstructure:"ignore_patterns" yaml:"ignore_patterns" default:"[]"`
	BatchSize        int           `mapstructure:"batch_size" yaml:"batch_size" validate:"min=1,max=1000" default:"100"`
	EnableRecursive  bool          `mapstructure:"enable_recursive" yaml:"enable_recursive" default:"true"`
}

// CacheConfig controls caching behavior
type CacheConfig struct {
	MaxEntries   int           `mapstructure:"max_entries" yaml:"max_entries" validate:"min=1000,max=100000" default:"10000"`
	MaxMemoryMB  int           `mapstructure:"max_memory_mb" yaml:"max_memory_mb" validate:"min=10,max=1024" default:"50"`
	TTL          time.Duration `mapstructure:"ttl" yaml:"ttl" validate:"min=1m,max=24h" default:"1h"`
	EnableLRU    bool          `mapstructure:"enable_lru" yaml:"enable_lru" default:"true"`
}

// GitConfig controls Git operations
type GitConfig struct {
	CleanupThreshold int  `mapstructure:"cleanup_threshold" yaml:"cleanup_threshold" validate:"min=10,max=10000" default:"100"`
	AutoGC           bool `mapstructure:"auto_gc" yaml:"auto_gc" default:"true"`
	MaxCommits       int  `mapstructure:"max_commits" yaml:"max_commits" validate:"min=50,max=50000" default:"1000"`
	UseShallowClone  bool `mapstructure:"use_shallow_clone" yaml:"use_shallow_clone" default:"false"`
}

// UIConfig controls user interface behavior
type UIConfig struct {
	ProgressIndicators bool   `mapstructure:"progress_indicators" yaml:"progress_indicators" default:"true"`
	ColorOutput        bool   `mapstructure:"color_output" yaml:"color_output" default:"true"`
	Pager              string `mapstructure:"pager" yaml:"pager" validate:"oneof=auto always never" default:"auto"`
	TableFormat        string `mapstructure:"table_format" yaml:"table_format" validate:"oneof=table json yaml" default:"table"`
}

// Manager handles configuration loading and management
type Manager struct {
	config    *Config
	viper     *viper.Viper
	validator *Validator
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()
	
	// Set configuration defaults
	setDefaults(v)
	
	return &Manager{
		config:    &Config{},
		viper:     v,
		validator: NewValidator(),
	}
}

// Load loads configuration from multiple sources in precedence order:
// 1. CLI flags (highest priority)
// 2. Environment variables
// 3. Configuration files
// 4. Defaults (lowest priority)
func (m *Manager) Load(projectRoot string) error {
	// Set up configuration file locations and names
	if err := m.setupConfigPaths(projectRoot); err != nil {
		return fmt.Errorf("failed to setup config paths: %w", err)
	}
	
	// Set up environment variable handling
	m.setupEnvironmentVariables()
	
	// Read configuration files (doesn't error if file doesn't exist)
	if err := m.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}
	
	// Unmarshal configuration into struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate configuration
	if err := m.validator.Validate(m.config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	return m.config
}

// GetViper returns the underlying viper instance for CLI integration
func (m *Manager) GetViper() *viper.Viper {
	return m.viper
}

// setupConfigPaths configures where to look for configuration files
func (m *Manager) setupConfigPaths(projectRoot string) error {
	m.viper.SetConfigName("timemachine")
	m.viper.SetConfigType("yaml")
	
	// Configuration file search order (highest to lowest priority):
	// 1. Project root (.timemachine.yaml)
	// 2. Project .timemachine/ directory  
	// 3. User config directory (~/.config/timemachine/)
	// 4. System config directory (/etc/timemachine/)
	
	// Project-specific config (highest priority)
	if projectRoot != "" {
		m.viper.AddConfigPath(projectRoot)
		
		// Also check .timemachine/ subdirectory
		timemachineDir := filepath.Join(projectRoot, ".timemachine")
		if _, err := os.Stat(timemachineDir); err == nil {
			m.viper.AddConfigPath(timemachineDir)
		}
	}
	
	// User config directory
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		m.viper.AddConfigPath(filepath.Join(userConfigDir, "timemachine"))
	}
	
	// User home directory (fallback)
	if homeDir, err := os.UserHomeDir(); err == nil {
		m.viper.AddConfigPath(homeDir)
	}
	
	// System config directory
	m.viper.AddConfigPath("/etc/timemachine")
	
	return nil
}

// setupEnvironmentVariables configures environment variable handling
// SECURITY: Only explicitly defined environment variables are bound to prevent injection attacks
func (m *Manager) setupEnvironmentVariables() {
	// REMOVED: AutomaticEnv() - this was a security vulnerability that allowed
	// arbitrary environment variable injection. Now only explicitly defined
	// variables are processed, ensuring all values go through validation.
	
	// Only these specific environment variables are allowed
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
	
	// Bind only explicitly defined environment variables
	// This ensures all values go through the normal validation pipeline
	for env, key := range allowedEnvVars {
		m.viper.BindEnv(key, env)
	}
}

// setDefaults sets all default configuration values
func setDefaults(v *viper.Viper) {
	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.file", "")
	
	// Watcher defaults
	v.SetDefault("watcher.debounce_delay", "2s")
	v.SetDefault("watcher.max_watched_files", 100000)
	v.SetDefault("watcher.ignore_patterns", []string{})
	v.SetDefault("watcher.batch_size", 100)
	v.SetDefault("watcher.enable_recursive", true)
	
	// Cache defaults
	v.SetDefault("cache.max_entries", 10000)
	v.SetDefault("cache.max_memory_mb", 50)
	v.SetDefault("cache.ttl", "1h")
	v.SetDefault("cache.enable_lru", true)
	
	// Git defaults
	v.SetDefault("git.cleanup_threshold", 100)
	v.SetDefault("git.auto_gc", true)
	v.SetDefault("git.max_commits", 1000)
	v.SetDefault("git.use_shallow_clone", false)
	
	// UI defaults
	v.SetDefault("ui.progress_indicators", true)
	v.SetDefault("ui.color_output", true)
	v.SetDefault("ui.pager", "auto")
	v.SetDefault("ui.table_format", "table")
}

// CreateDefaultConfigFile creates a default configuration file in the project root
func (m *Manager) CreateDefaultConfigFile(projectRoot string) error {
	configPath := filepath.Join(projectRoot, "timemachine.yaml")
	
	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}
	
	// Create default configuration with comments
	defaultConfig := `# TimeMachine CLI Configuration
# This file contains configuration options for TimeMachine CLI
# All settings have sensible defaults and can be overridden via:
#   - Command line flags (highest priority)
#   - Environment variables (TIMEMACHINE_*)
#   - This configuration file
#   - Built-in defaults (lowest priority)

log:
  level: info          # debug, info, warn, error
  format: text         # text, json  
  file: ""            # optional log file path (empty = stdout)

watcher:
  debounce_delay: 2s           # delay before creating snapshot after changes
  max_watched_files: 100000    # maximum number of files to watch
  ignore_patterns: []          # additional patterns to ignore
  batch_size: 100             # number of files to process in batch
  enable_recursive: true      # recursively watch subdirectories

cache:
  max_entries: 10000      # maximum cache entries
  max_memory_mb: 50       # maximum cache memory usage
  ttl: 1h                # cache entry time-to-live
  enable_lru: true       # use LRU eviction policy

git:
  cleanup_threshold: 100      # number of snapshots before cleanup
  auto_gc: true              # automatically run git gc
  max_commits: 1000          # maximum snapshots to keep
  use_shallow_clone: false   # use shallow cloning for performance

ui:
  progress_indicators: true   # show progress bars and spinners
  color_output: true         # colorize output
  pager: auto               # auto, always, never
  table_format: table       # table, json, yaml
`
	
	// Write the default configuration with secure permissions (0600 = owner read/write only)
	// SECURITY: Use restrictive permissions to prevent other users from reading configuration
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}
	
	return nil
}