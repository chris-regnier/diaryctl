package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ShellConfig holds shell integration configuration.
type ShellConfig struct {
	CacheTTL    string `mapstructure:"cache_ttl"`
	TodayIcon   string `mapstructure:"today_icon"`
	NoTodayIcon string `mapstructure:"no_today_icon"`
	StreakIcon  string `mapstructure:"streak_icon"`
	ShowContext bool   `mapstructure:"show_context"`
	ShowBackend bool   `mapstructure:"show_backend"`
}

// ThemeConfig holds TUI and markdown theme configuration.
type ThemeConfig struct {
	Preset        string `mapstructure:"preset"`
	Primary       string `mapstructure:"primary"`
	Secondary     string `mapstructure:"secondary"`
	Accent        string `mapstructure:"accent"`
	Muted         string `mapstructure:"muted"`
	Danger        string `mapstructure:"danger"`
	Background    string `mapstructure:"background"`
	MarkdownStyle string `mapstructure:"markdown_style"`
}

// DictateConfig holds speech-to-text configuration.
type DictateConfig struct {
	Recorder string            `mapstructure:"recorder"` // "sox", "arecord", "ffmpeg", or path
	Engine   string            `mapstructure:"engine"`   // "parakeet", "whisper", "command"
	Command  string            `mapstructure:"command"`  // Custom transcription command template
	Model    string            `mapstructure:"model"`    // Model name or path
	Language string            `mapstructure:"language"` // Language hint (e.g., "en")
	Options  map[string]string `mapstructure:"options"`  // Engine-specific options
}

// Config holds the application configuration.
type Config struct {
	Storage          string        `mapstructure:"storage"`
	DataDir          string        `mapstructure:"data_dir"`
	Editor           string        `mapstructure:"editor"`
	DefaultTemplate  string        `mapstructure:"default_template"`
	MaxWidth         int           `mapstructure:"max_width"`
	ContextProviders []string      `mapstructure:"context_providers"`
	ContextResolvers []string      `mapstructure:"context_resolvers"`
	Shell            ShellConfig   `mapstructure:"shell"`
	Theme            ThemeConfig   `mapstructure:"theme"`
	Dictate          DictateConfig `mapstructure:"dictate"`
}

// DefaultDataDir returns the default data directory (~/.diaryctl/).
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".diaryctl")
	}
	return filepath.Join(home, ".diaryctl")
}

// Load reads configuration from file, environment variables, and defaults.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("storage", "markdown")
	v.SetDefault("data_dir", DefaultDataDir())
	v.SetDefault("editor", "")
	v.SetDefault("default_template", "")
	v.SetDefault("max_width", 100)
	v.SetDefault("context_providers", []string{})
	v.SetDefault("context_resolvers", []string{})
	v.SetDefault("shell.cache_ttl", "5m")
	v.SetDefault("shell.today_icon", "✓")
	v.SetDefault("shell.no_today_icon", "✗")
	v.SetDefault("shell.streak_icon", "🔥")
	v.SetDefault("shell.show_context", true)
	v.SetDefault("shell.show_backend", false)
	v.SetDefault("theme.preset", "default-dark")
	v.SetDefault("dictate.recorder", "")
	v.SetDefault("dictate.engine", "")
	v.SetDefault("dictate.command", "")
	v.SetDefault("dictate.model", "base")
	v.SetDefault("dictate.language", "en")

	// Config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// XDG support
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			v.AddConfigPath(filepath.Join(xdg, "diaryctl"))
		}
		v.AddConfigPath(filepath.Join(DefaultDataDir()))
		v.SetConfigName("config")
		v.SetConfigType("toml")
	}

	// Environment variables: DIARYCTL_STORAGE, DIARYCTL_DATA_DIR, etc.
	v.SetEnvPrefix("DIARYCTL")
	v.AutomaticEnv()

	// Read config file (ignore not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not a "file not found" error
			if configPath != "" {
				return nil, err
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
