package config

import (
	"strings"
	"testing"
)

// TestLoad_NormalizesModelFields proves that the Model (singular) and Models
// (list) fields are kept consistent after Load: whenever at least one is
// given, both end up populated. This keeps backward compatibility with
// configs that only set `model` while enabling new configs that only set
// `models`.
func TestLoad_NormalizesModelFields(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantModel     string
		wantModels    []string
		wantModelSet  bool
		wantModelsSet bool
	}{
		{
			name: "only model set derives models list",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "test"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
`,
			wantModel:     "gpt-4",
			wantModels:    []string{"gpt-4"},
			wantModelSet:  true,
			wantModelsSet: true,
		},
		{
			name: "only models set derives singular model as first",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "test"
base_url = "https://api.openai.com/v1"
models = ["gpt-4o", "gpt-4o-mini"]
`,
			wantModel:     "gpt-4o",
			wantModels:    []string{"gpt-4o", "gpt-4o-mini"},
			wantModelSet:  true,
			wantModelsSet: true,
		},
		{
			name: "both set are left as-is when both non-empty",
			content: `
default_provider = "openai"

[providers.openai]
api_key = "test"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
models = ["gpt-4o", "gpt-4o-mini"]
`,
			wantModel:     "gpt-4",
			wantModels:    []string{"gpt-4o", "gpt-4o-mini"},
			wantModelSet:  true,
			wantModelsSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.content)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			p := cfg.Providers["openai"]
			if tt.wantModelSet && p.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", p.Model, tt.wantModel)
			}
			if tt.wantModelsSet {
				if len(p.Models) != len(tt.wantModels) {
					t.Errorf("Models = %v, want %v", p.Models, tt.wantModels)
				} else {
					for i, m := range tt.wantModels {
						if p.Models[i] != m {
							t.Errorf("Models[%d] = %q, want %q", i, p.Models[i], m)
						}
					}
				}
			}
		})
	}
}

// TestLoad_RequiresAtLeastOneModel proves validation rejects a provider that
// specifies neither `model` nor `models`.
func TestLoad_RequiresAtLeastOneModel(t *testing.T) {
	content := `
default_provider = "openai"

[providers.openai]
api_key = "test"
base_url = "https://api.openai.com/v1"
`
	path := writeTempConfig(t, content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for provider with no model, got nil")
	}
	if !strings.Contains(err.Error(), "must have at least one model") {
		t.Errorf("expected error mentioning 'must have at least one model', got: %v", err)
	}
	if !strings.Contains(err.Error(), "openai") {
		t.Errorf("expected error to name the offending provider 'openai', got: %v", err)
	}
}
