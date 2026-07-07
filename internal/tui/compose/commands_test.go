package compose

import (
	"context"
	"testing"

	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
)

// stubProvider is a minimal llm.Provider used to test createSessionCmd in
// isolation. It returns fixed Name()/Model() values and a no-op Stream().
type stubProvider struct {
	name  string
	model string
}

func (s *stubProvider) Stream(ctx context.Context, messages []llm.ChatMessage) (<-chan llm.StreamChunk, error) {
	return nil, nil
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Model() string { return s.model }

// WithModel returns a copy of s scoped to the given model.
func (s *stubProvider) WithModel(model string) llm.Provider {
	cp := *s
	cp.model = model
	return &cp
}

// TestCreateSessionCmd_PersistsProviderModel proves that createSessionCmd sets
// the Session.Model from provider.Model() (regression for the bug where Model
// was left as the zero-value "").
func TestCreateSessionCmd_PersistsProviderModel(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer database.Close()

	provider := &stubProvider{name: "claude", model: "X"}

	cmd := createSessionCmd(database, provider)
	msg := cmd()

	created, ok := msg.(SessionCreatedMsg)
	if !ok {
		t.Fatalf("expected SessionCreatedMsg, got %T", msg)
	}
	if created.Session == nil {
		t.Fatal("expected non-nil Session")
	}
	if created.Session.Model != "X" {
		t.Errorf("expected Session.Model %q, got %q", "X", created.Session.Model)
	}
	if created.Session.Provider != "claude" {
		t.Errorf("expected Session.Provider %q, got %q", "claude", created.Session.Provider)
	}

	// The model must also be the value actually persisted to SQLite.
	persisted, err := database.GetSession(created.Session.ID)
	if err != nil {
		t.Fatalf("failed to load persisted session: %v", err)
	}
	if persisted.Model != "X" {
		t.Errorf("expected persisted Model %q, got %q", "X", persisted.Model)
	}
}

// TestCreateSessionCmd_BlankModelWhenProviderHasNone ensures that when the
// provider reports an empty model, that empty value is what gets persisted
// (i.e. we always read from provider.Model() rather than fabricating one).
func TestCreateSessionCmd_BlankModelWhenProviderHasNone(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer database.Close()

	provider := &stubProvider{name: "openai", model: ""}

	msg := createSessionCmd(database, provider)()
	created, ok := msg.(SessionCreatedMsg)
	if !ok {
		t.Fatalf("expected SessionCreatedMsg, got %T", msg)
	}
	if created.Session.Model != "" {
		t.Errorf("expected empty Model, got %q", created.Session.Model)
	}
}
