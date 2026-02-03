package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Fatal("DefaultPath returned empty string")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, ".config", "ai-tui", "config.toml")
	if path != expected {
		t.Errorf("DefaultPath() = %q, want %q", path, expected)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		envVars map[string]string
		wantErr bool
		errMsg  string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config with all fields",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "sk-test123"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
system_prompt = "You are a helpful assistant"
max_tokens = 2000

[providers.anthropic]
api_key = "ant-test456"
base_url = "https://api.anthropic.com/v1"
model = "claude-3-opus"
max_tokens = 0

[storage]
db_path = "/tmp/test.db"
notes_dir = "/tmp/notes"

[ui]
show_tokens = true
max_width = 120
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.DefaultProvider != "openai" {
					t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "openai")
				}
				if len(cfg.Providers) != 2 {
					t.Errorf("len(Providers) = %d, want 2", len(cfg.Providers))
				}
				
				openai := cfg.Providers["openai"]
				if openai.APIKey != "sk-test123" {
					t.Errorf("openai.APIKey = %q, want %q", openai.APIKey, "sk-test123")
				}
				if openai.MaxTokens != 2000 {
					t.Errorf("openai.MaxTokens = %d, want 2000", openai.MaxTokens)
				}

				anthropic := cfg.Providers["anthropic"]
				if anthropic.MaxTokens != 4096 {
					t.Errorf("anthropic.MaxTokens = %d, want 4096 (default)", anthropic.MaxTokens)
				}

				if cfg.Storage.DBPath != "/tmp/test.db" {
					t.Errorf("Storage.DBPath = %q, want %q", cfg.Storage.DBPath, "/tmp/test.db")
				}
				if cfg.UI.MaxWidth != 120 {
					t.Errorf("UI.MaxWidth = %d, want 120", cfg.UI.MaxWidth)
				}
				if !cfg.UI.ShowTokens {
					t.Error("UI.ShowTokens = false, want true")
				}
			},
		},
		{
			name: "env var expansion",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "$TEST_API_KEY"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
`,
			envVars: map[string]string{
				"TEST_API_KEY": "resolved-key-123",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				openai := cfg.Providers["openai"]
				if openai.APIKey != "resolved-key-123" {
					t.Errorf("openai.APIKey = %q, want %q (env var expanded)", openai.APIKey, "resolved-key-123")
				}
			},
		},
		{
			name: "env var not set leaves as-is",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "$NONEXISTENT_VAR"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				openai := cfg.Providers["openai"]
				if openai.APIKey != "$NONEXISTENT_VAR" {
					t.Errorf("openai.APIKey = %q, want %q (env var left as-is)", openai.APIKey, "$NONEXISTENT_VAR")
				}
			},
		},
		{
			name: "tilde expansion in paths",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "test"
model = "gpt-4"

[storage]
db_path = "~/.local/share/ai-tui/test.db"
notes_dir = "~/notes/"
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				home, _ := os.UserHomeDir()
				
				expectedDB := filepath.Join(home, ".local/share/ai-tui/test.db")
				if cfg.Storage.DBPath != expectedDB {
					t.Errorf("Storage.DBPath = %q, want %q (~ expanded)", cfg.Storage.DBPath, expectedDB)
				}

				expectedNotes := filepath.Join(home, "notes/")
				if cfg.Storage.NotesDir != expectedNotes {
					t.Errorf("Storage.NotesDir = %q, want %q (~ expanded)", cfg.Storage.NotesDir, expectedNotes)
				}
			},
		},
		{
			name: "defaults applied",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "test"
model = "gpt-4"
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				openai := cfg.Providers["openai"]
				if openai.MaxTokens != 4096 {
					t.Errorf("openai.MaxTokens = %d, want 4096 (default)", openai.MaxTokens)
				}

				if cfg.UI.MaxWidth != 100 {
					t.Errorf("UI.MaxWidth = %d, want 100 (default)", cfg.UI.MaxWidth)
				}

				home, _ := os.UserHomeDir()
				expectedDB := filepath.Join(home, ".local/share/ai-tui/ai-tui.db")
				if cfg.Storage.DBPath != expectedDB {
					t.Errorf("Storage.DBPath = %q, want %q (default)", cfg.Storage.DBPath, expectedDB)
				}

				expectedNotes := filepath.Join(home, "ai-notes/")
				if cfg.Storage.NotesDir != expectedNotes {
					t.Errorf("Storage.NotesDir = %q, want %q (default)", cfg.Storage.NotesDir, expectedNotes)
				}
			},
		},
		{
			name: "missing providers error",
			content: `
default_provider = "openai"
`,
			wantErr: true,
			errMsg:  "at least one provider must be defined",
		},
		{
			name: "default_provider not in map error",
			content: `
default_provider = "nonexistent"

[providers.openai]
api_key = "test"
model = "gpt-4"
`,
			wantErr: true,
			errMsg:  "default_provider 'nonexistent' not found in providers",
		},
		{
			name: "missing default_provider error",
			content: `
[providers.openai]
api_key = "test"
model = "gpt-4"
`,
			wantErr: true,
			errMsg:  "default_provider must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, val := range tt.envVars {
				t.Setenv(key, val)
			}

			// Create temp config file
			path := writeTempConfig(t, tt.content)

			// Load config
			cfg, err := Load(path)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, wantErr %v", tt.wantErr)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}

			// Run validation
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/to/config.toml")
	if err == nil {
		t.Fatal("Load() with nonexistent file should return error")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := writeTempConfig(t, "this is not valid toml {{{")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() with invalid TOML should return error")
	}
}

// writeTempConfig creates a temporary TOML file and returns its path
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	return path
}
