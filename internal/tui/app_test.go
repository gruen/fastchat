package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
	"github.com/mg/ai-tui/internal/tui/history"
)

// fakeProvider is a stub llm.Provider used only for testing provider binding.
type fakeProvider struct {
	name  string
	model string
}

func (f *fakeProvider) Stream(ctx context.Context, messages []llm.ChatMessage) (<-chan llm.StreamChunk, error) {
	return nil, nil
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Model() string { return f.model }

// Helper function to create a minimal test config
func testConfig() *config.Config {
	return &config.Config{
		DefaultProvider: "test",
		Providers: map[string]config.Provider{
			"test": {
				Model: "test-model",
			},
		},
	}
}

func TestNewAppModel_StartsInComposeView(t *testing.T) {
	m := NewAppModel(testConfig(), nil, map[string]llm.Provider{})

	if m.activeView != ComposeView {
		t.Errorf("expected activeView to be ComposeView, got %v", m.activeView)
	}
}

func TestAppModel_CtrlD_SetsQuittingAndReturnsQuit(t *testing.T) {
	m := NewAppModel(testConfig(), nil, map[string]llm.Provider{})

	// Send ctrl+d
	msg := tea.KeyMsg{Type: tea.KeyCtrlD}
	updatedModel, cmd := m.Update(msg)

	// Type assert back to AppModel
	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}

	if !updated.quitting {
		t.Error("expected quitting to be true after ctrl+d")
	}

	if cmd == nil {
		t.Error("expected cmd to be non-nil (tea.Quit)")
	}
}

func TestAppModel_CtrlH_SwitchesToHistoryView(t *testing.T) {
	m := NewAppModel(testConfig(), nil, map[string]llm.Provider{})

	// Verify we start in ComposeView
	if m.activeView != ComposeView {
		t.Fatalf("expected to start in ComposeView")
	}

	// Send ctrl+h
	msg := tea.KeyMsg{Type: tea.KeyCtrlH}
	updatedModel, _ := m.Update(msg)

	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}

	if updated.activeView != HistoryView {
		t.Errorf("expected activeView to be HistoryView, got %v", updated.activeView)
	}
}

func TestAppModel_CtrlN_SwitchesToComposeView(t *testing.T) {
	m := NewAppModel(testConfig(), nil, map[string]llm.Provider{})

	// First switch to HistoryView
	msg := tea.KeyMsg{Type: tea.KeyCtrlH}
	updatedModel, _ := m.Update(msg)
	m, _ = updatedModel.(AppModel)

	// Verify we're in HistoryView
	if m.activeView != HistoryView {
		t.Fatalf("expected to be in HistoryView")
	}

	// Send ctrl+n
	msg = tea.KeyMsg{Type: tea.KeyCtrlN}
	updatedModel, _ = m.Update(msg)

	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}

	if updated.activeView != ComposeView {
		t.Errorf("expected activeView to be ComposeView, got %v", updated.activeView)
	}
}

func TestAppModel_WindowSizeMsg_UpdatesDimensions(t *testing.T) {
	m := NewAppModel(testConfig(), nil, map[string]llm.Provider{})

	// Send window size message
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := m.Update(msg)

	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}

	if updated.width != 120 {
		t.Errorf("expected width to be 120, got %d", updated.width)
	}

	if updated.height != 40 {
		t.Errorf("expected height to be 40, got %d", updated.height)
	}
}

func TestAppModel_ResumeSessionMsg_LoadsMessagesIntoCompose(t *testing.T) {
	m := NewAppModel(testConfig(), nil, map[string]llm.Provider{})

	msgs := []db.Message{
		{SessionID: "abc", Role: "user", Content: "Hello"},
		{SessionID: "abc", Role: "assistant", Content: "Hi there"},
		{SessionID: "abc", Role: "user", Content: "How are you?"},
	}
	session := db.Session{ID: "abc", Provider: "test"}

	updatedModel, _ := m.Update(history.ResumeSessionMsg{Session: session, Messages: msgs})
	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}

	if updated.activeView != ComposeView {
		t.Errorf("expected activeView to be ComposeView, got %v", updated.activeView)
	}
	if updated.compose.SessionID() != "abc" {
		t.Errorf("expected compose session id 'abc', got %q", updated.compose.SessionID())
	}
	if got := updated.compose.MessageCount(); got != 3 {
		t.Errorf("expected compose to show 3 messages, got %d", got)
	}
}

func TestAppModel_ResumeSessionMsg_BindsSessionProvider(t *testing.T) {
	openaiProv := &fakeProvider{name: "openai"}
	anthropicProv := &fakeProvider{name: "anthropic"}
	providers := map[string]llm.Provider{
		"openai":    openaiProv,
		"anthropic": anthropicProv,
	}
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"openai":    {Model: "gpt-4"},
			"anthropic": {Model: "claude-3"},
		},
	}

	m := NewAppModel(cfg, nil, providers)

	// Sanity: compose starts bound to the active (anthropic) provider.
	if m.compose.Provider() == nil || m.compose.Provider().Name() != "anthropic" {
		t.Fatalf("expected initial provider 'anthropic', got %v", m.compose.Provider())
	}

	// Resume a session that originally used openai while the active provider
	// is anthropic. Compose must be rebound to the session's original provider.
	session := db.Session{ID: "sess-oai", Provider: "openai"}
	updatedModel, _ := m.Update(history.ResumeSessionMsg{Session: session, Messages: nil})
	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}

	if updated.compose.Provider() == nil {
		t.Fatal("expected compose to have a bound provider")
	}
	if updated.compose.Provider().Name() != "openai" {
		t.Errorf("expected compose bound to session's provider 'openai', got %q",
			updated.compose.Provider().Name())
	}
	// Active provider follows the resumed session so the status bar is consistent.
	if updated.ActiveProvider() != "openai" {
		t.Errorf("expected active provider 'openai', got %q", updated.ActiveProvider())
	}
}
