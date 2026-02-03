package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DefaultProvider string              `toml:"default_provider"`
	Providers       map[string]Provider `toml:"providers"`
	Storage         Storage             `toml:"storage"`
	UI              UI                  `toml:"ui"`
}

type Provider struct {
	APIKey       string `toml:"api_key"`
	BaseURL      string `toml:"base_url"`
	Model        string `toml:"model"`
	SystemPrompt string `toml:"system_prompt"`
	MaxTokens    int    `toml:"max_tokens"`
}

type Storage struct {
	DBPath   string `toml:"db_path"`
	NotesDir string `toml:"notes_dir"`
}

type UI struct {
	ShowTokens bool `toml:"show_tokens"`
	MaxWidth   int  `toml:"max_width"`
}

// DefaultPath returns ~/.config/ai-tui/config.toml
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ai-tui", "config.toml")
}

// Load reads and parses the TOML config file, expands env vars and ~, validates, applies defaults
func Load(path string) (*Config, error) {
	var cfg Config

	// Parse TOML file
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	// Expand environment variables and home directories
	expandConfig(&cfg)

	// Validate
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	// Apply MaxTokens default
	for name, provider := range cfg.Providers {
		if provider.MaxTokens == 0 {
			provider.MaxTokens = 4096
			cfg.Providers[name] = provider
		}
	}

	// Apply MaxWidth default
	if cfg.UI.MaxWidth == 0 {
		cfg.UI.MaxWidth = 100
	}

	// Apply DBPath default
	if cfg.Storage.DBPath == "" {
		cfg.Storage.DBPath = "~/.local/share/ai-tui/ai-tui.db"
	}

	// Apply NotesDir default
	if cfg.Storage.NotesDir == "" {
		cfg.Storage.NotesDir = "~/ai-notes/"
	}
}

func expandConfig(cfg *Config) {
	// Expand environment variables in API keys
	for name, provider := range cfg.Providers {
		if strings.HasPrefix(provider.APIKey, "$") {
			envVar := provider.APIKey[1:]
			if val := os.Getenv(envVar); val != "" {
				provider.APIKey = val
				cfg.Providers[name] = provider
			}
			// If env var is empty, leave as-is (don't error)
		}
	}

	// Expand ~ in storage paths
	cfg.Storage.DBPath = expandHome(cfg.Storage.DBPath)
	cfg.Storage.NotesDir = expandHome(cfg.Storage.NotesDir)
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}

func validate(cfg *Config) error {
	// At least one provider must be defined
	if len(cfg.Providers) == 0 {
		return fmt.Errorf("at least one provider must be defined")
	}

	// DefaultProvider must exist in Providers map
	if cfg.DefaultProvider == "" {
		return fmt.Errorf("default_provider must be set")
	}

	if _, ok := cfg.Providers[cfg.DefaultProvider]; !ok {
		return fmt.Errorf("default_provider '%s' not found in providers", cfg.DefaultProvider)
	}

	return nil
}
