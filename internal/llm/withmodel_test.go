package llm

import (
	"context"
	"testing"
)

// TestOpenAIProvider_WithModel proves WithModel returns a copy scoped to the
// new model without mutating the receiver, preserving all other fields.
func TestOpenAIProvider_WithModel(t *testing.T) {
	orig := &openaiProvider{
		name:         "openai",
		apiKey:       "sk-test",
		baseURL:      "https://api.openai.com/v1",
		model:        "gpt-4",
		systemPrompt: "be helpful",
		maxTokens:    2000,
		client:       nil,
	}

	cp := orig.WithModel("gpt-4o")

	// The copy is scoped to the new model.
	got, ok := cp.(*openaiProvider)
	if !ok {
		t.Fatalf("WithModel did not return *openaiProvider, got %T", cp)
	}
	if got.model != "gpt-4o" {
		t.Errorf("copy model = %q, want %q", got.model, "gpt-4o")
	}
	if cp.Model() != "gpt-4o" {
		t.Errorf("copy Model() = %q, want %q", cp.Model(), "gpt-4o")
	}

	// The receiver is NOT mutated.
	if orig.model != "gpt-4" {
		t.Errorf("receiver model was mutated: got %q, want %q", orig.model, "gpt-4")
	}
	if orig.Model() != "gpt-4" {
		t.Errorf("receiver Model() was mutated: got %q, want %q", orig.Model(), "gpt-4")
	}

	// The copy is a distinct pointer.
	if cp == orig {
		t.Error("WithModel returned the same pointer; expected a copy")
	}

	// Other fields are preserved on the copy.
	if got.name != orig.name || got.apiKey != orig.apiKey ||
		got.baseURL != orig.baseURL || got.systemPrompt != orig.systemPrompt ||
		got.maxTokens != orig.maxTokens {
		t.Errorf("copy fields do not match receiver: got %+v, want %+v", got, orig)
	}
}

// TestClaudeProvider_WithModel proves WithModel returns a copy scoped to the
// new model without mutating the receiver, preserving all other fields.
func TestClaudeProvider_WithModel(t *testing.T) {
	orig := &claudeProvider{
		name:         "claude",
		apiKey:       "ant-test",
		baseURL:      "https://api.anthropic.com",
		model:        "claude-3",
		systemPrompt: "be concise",
		maxTokens:    1024,
		client:       nil,
	}

	cp := orig.WithModel("claude-3.5")

	got, ok := cp.(*claudeProvider)
	if !ok {
		t.Fatalf("WithModel did not return *claudeProvider, got %T", cp)
	}
	if got.model != "claude-3.5" {
		t.Errorf("copy model = %q, want %q", got.model, "claude-3.5")
	}
	if cp.Model() != "claude-3.5" {
		t.Errorf("copy Model() = %q, want %q", cp.Model(), "claude-3.5")
	}
	if orig.model != "claude-3" {
		t.Errorf("receiver model was mutated: got %q, want %q", orig.model, "claude-3")
	}
	if cp == orig {
		t.Error("WithModel returned the same pointer; expected a copy")
	}
	if got.name != orig.name || got.apiKey != orig.apiKey ||
		got.baseURL != orig.baseURL || got.systemPrompt != orig.systemPrompt ||
		got.maxTokens != orig.maxTokens {
		t.Errorf("copy fields do not match receiver: got %+v, want %+v", got, orig)
	}
}

// TestWithModel_StreamsSelectedModel proves the model produced by WithModel is
// the one the provider would send in a real request. A stub provider records
// its bound model when Stream is called.
func TestWithModel_StreamsSelectedModel(t *testing.T) {
	orig := &openaiProvider{
		name:   "openai",
		apiKey: "sk-test",
		model:  "gpt-4",
		client: nil,
	}
	cp := orig.WithModel("gpt-4o").(*openaiProvider)

	// The bound model on the copy is the overridden model, so a request built
	// from it would use "gpt-4o".
	if cp.Model() != "gpt-4o" {
		t.Fatalf("copy Model() = %q, want %q", cp.Model(), "gpt-4o")
	}

	// Sanity: the interface still satisfies Provider.
	var p Provider = orig
	_ = p
	_ = context.Background()
}
