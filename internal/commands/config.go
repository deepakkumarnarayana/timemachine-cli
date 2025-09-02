package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/deepakkumarnarayana/timemachine-cli/internal/core"
)

// ConfigCmd creates the config command with subcommands
func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage TimeMachine configuration",
		Long: `Manage TimeMachine configuration settings.

Configuration is loaded from multiple sources in order of precedence:
1. Command-line flags (highest priority)
2. Environment variables (TIMEMACHINE_*)
3. Configuration files:
   - Project: ./timemachine.yaml or ./.timemachine/timemachine.yaml
   - User: ~/.config/timemachine/timemachine.yaml
   - System: /etc/timemachine/timemachine.yaml
4. Built-in defaults (lowest priority)`,
	}

	// Add subcommands
	cmd.AddCommand(configInitCmd())
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configValidateCmd())

	return cmd
}

// configInitCmd creates a default configuration file
func configInitCmd() *cobra.Command {
	var (
		global bool
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a default configuration file",
		Long: `Create a default TimeMachine configuration file with all available options and comments.

By default, creates a project-specific configuration file (timemachine.yaml) in the current directory.
Use --global to create a user-specific configuration file instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if global {
				return initGlobalConfig(force)
			}
			return initProjectConfig(force)
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Create global user configuration instead of project configuration")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration file")

	return cmd
}

// configShowCmd shows the current configuration
func configShowCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current configuration with values from all sources merged.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig(format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "yaml", "Output format (yaml, json)")

	return cmd
}

// configGetCmd gets a specific configuration value
func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  "Get a specific configuration value by key (e.g., 'log.level', 'watcher.debounce_delay')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getConfigValue(args[0])
		},
	}
}

// configSetCmd sets a configuration value
func configSetCmd() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Set a configuration value in the configuration file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setConfigValue(args[0], args[1], global)
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Set value in global user configuration")

	return cmd
}

// configValidateCmd validates the current configuration
func configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  "Validate the current configuration and show any errors or warnings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateConfig()
		},
	}
}

// Implementation functions

func initProjectConfig(force bool) error {
	// Create application state to get project root
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	configPath := filepath.Join(state.ProjectRoot, "timemachine.yaml")

	// Check if file already exists
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			color.Yellow("Configuration file already exists at: %s", configPath)
			fmt.Println("Use --force to overwrite")
			return nil
		}
	}

	// Create default configuration file
	if err := state.ConfigManager.CreateDefaultConfigFile(state.ProjectRoot); err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}

	color.Green("✅ Created configuration file: %s", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("• Edit the configuration file to customize settings")
	fmt.Println("• Run 'timemachine config validate' to check your configuration")
	fmt.Println("• Use environment variables (TIMEMACHINE_*) to override settings")

	return nil
}

func initGlobalConfig(force bool) error {
	// Get user config directory
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config directory: %w", err)
	}

	configDir := filepath.Join(userConfigDir, "timemachine")
	configPath := filepath.Join(configDir, "timemachine.yaml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file already exists
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			color.Yellow("Global configuration file already exists at: %s", configPath)
			fmt.Println("Use --force to overwrite")
			return nil
		}
	}

	// Create configuration manager and default file
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}
	if err := state.ConfigManager.CreateDefaultConfigFile(configDir); err != nil {
		return fmt.Errorf("failed to create global configuration file: %w", err)
	}

	color.Green("✅ Created global configuration file: %s", configPath)
	fmt.Println("\nThis configuration will be used for all TimeMachine projects.")
	fmt.Println("Project-specific configuration files will override global settings.")

	return nil
}

func showConfig(format string) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	switch format {
	case "yaml":
		// Use viper to output YAML to stdout
		if err := state.ConfigManager.GetViper().WriteConfig(); err != nil {
			// Fallback to manual YAML output
			fmt.Printf(`log:
  level: %s
  format: %s
  file: "%s"

watcher:
  debounce_delay: %s
  max_watched_files: %d
  ignore_patterns: %v
  batch_size: %d
  enable_recursive: %t

cache:
  max_entries: %d
  max_memory_mb: %d
  ttl: %s
  enable_lru: %t

git:
  cleanup_threshold: %d
  auto_gc: %t
  max_commits: %d
  use_shallow_clone: %t

ui:
  progress_indicators: %t
  color_output: %t
  pager: %s
  table_format: %s
`,
				state.Config.Log.Level, state.Config.Log.Format, state.Config.Log.File,
				state.Config.Watcher.DebounceDelay, state.Config.Watcher.MaxWatchedFiles, state.Config.Watcher.IgnorePatterns,
				state.Config.Watcher.BatchSize, state.Config.Watcher.EnableRecursive,
				state.Config.Cache.MaxEntries, state.Config.Cache.MaxMemoryMB, state.Config.Cache.TTL, state.Config.Cache.EnableLRU,
				state.Config.Git.CleanupThreshold, state.Config.Git.AutoGC, state.Config.Git.MaxCommits, state.Config.Git.UseShallowClone,
				state.Config.UI.ProgressIndicators, state.Config.UI.ColorOutput, state.Config.UI.Pager, state.Config.UI.TableFormat)
		}
	case "json":
		// Convert to JSON (simplified version)
		fmt.Printf(`{
  "log": {
    "level": "%s",
    "format": "%s",
    "file": "%s"
  },
  "watcher": {
    "debounce_delay": "%s",
    "max_watched_files": %d,
    "ignore_patterns": %v,
    "batch_size": %d,
    "enable_recursive": %t
  },
  "cache": {
    "max_entries": %d,
    "max_memory_mb": %d,
    "ttl": "%s",
    "enable_lru": %t
  },
  "git": {
    "cleanup_threshold": %d,
    "auto_gc": %t,
    "max_commits": %d,
    "use_shallow_clone": %t
  },
  "ui": {
    "progress_indicators": %t,
    "color_output": %t,
    "pager": "%s",
    "table_format": "%s"
  }
}`,
			state.Config.Log.Level, state.Config.Log.Format, state.Config.Log.File,
			state.Config.Watcher.DebounceDelay, state.Config.Watcher.MaxWatchedFiles, state.Config.Watcher.IgnorePatterns,
			state.Config.Watcher.BatchSize, state.Config.Watcher.EnableRecursive,
			state.Config.Cache.MaxEntries, state.Config.Cache.MaxMemoryMB, state.Config.Cache.TTL, state.Config.Cache.EnableLRU,
			state.Config.Git.CleanupThreshold, state.Config.Git.AutoGC, state.Config.Git.MaxCommits, state.Config.Git.UseShallowClone,
			state.Config.UI.ProgressIndicators, state.Config.UI.ColorOutput, state.Config.UI.Pager, state.Config.UI.TableFormat)
	default:
		return fmt.Errorf("unsupported format: %s (use 'yaml' or 'json')", format)
	}

	return nil
}

func getConfigValue(key string) error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	// Get value from viper
	value := state.ConfigManager.GetViper().Get(key)
	if value == nil {
		return fmt.Errorf("configuration key '%s' not found", key)
	}

	fmt.Printf("%v\n", value)
	return nil
}

func setConfigValue(key, value string, global bool) error {
	// This is a simplified implementation
	// In a full implementation, you'd want to:
	// 1. Parse the value according to the expected type
	// 2. Validate the value
	// 3. Update the appropriate configuration file
	// 4. Reload the configuration

	fmt.Printf("Setting %s = %s", key, value)
	if global {
		fmt.Print(" (global)")
	}
	fmt.Println()

	color.Yellow("⚠️  Configuration modification not yet implemented")
	fmt.Println("For now, please edit the configuration file directly:")
	
	if global {
		userConfigDir, _ := os.UserConfigDir()
		fmt.Printf("  %s/timemachine/timemachine.yaml\n", userConfigDir)
	} else {
		state, err := core.NewAppState()
		if err == nil {
			fmt.Printf("  %s/timemachine.yaml\n", state.ProjectRoot)
		}
	}

	return nil
}

func validateConfig() error {
	// Create application state
	state, err := core.NewAppState()
	if err != nil {
		return fmt.Errorf("failed to initialize app state: %w", err)
	}

	// Validation is performed during config loading
	// If we reach here, configuration is valid
	color.Green("✅ Configuration is valid")

	// Show configuration source information
	fmt.Println("\nConfiguration sources:")
	if configFile := state.ConfigManager.GetViper().ConfigFileUsed(); configFile != "" {
		fmt.Printf("• Configuration file: %s\n", configFile)
	} else {
		fmt.Println("• No configuration file found (using defaults)")
	}

	// Show environment variable overrides
	envVars := []string{
		"TIMEMACHINE_LOG_LEVEL", "TIMEMACHINE_LOG_FORMAT", "TIMEMACHINE_LOG_FILE",
		"TIMEMACHINE_WATCHER_DEBOUNCE", "TIMEMACHINE_WATCHER_MAX_FILES",
		"TIMEMACHINE_CACHE_MAX_ENTRIES", "TIMEMACHINE_CACHE_MAX_MEMORY", "TIMEMACHINE_CACHE_TTL",
		"TIMEMACHINE_GIT_CLEANUP_THRESHOLD", "TIMEMACHINE_GIT_AUTO_GC",
		"TIMEMACHINE_UI_COLOR", "TIMEMACHINE_UI_PAGER",
	}

	envOverrides := []string{}
	for _, envVar := range envVars {
		if os.Getenv(envVar) != "" {
			envOverrides = append(envOverrides, envVar)
		}
	}

	if len(envOverrides) > 0 {
		fmt.Println("• Environment variable overrides:")
		for _, env := range envOverrides {
			fmt.Printf("  - %s=%s\n", env, os.Getenv(env))
		}
	}

	return nil
}