package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
	"github.com/mg/ai-tui/internal/tui/compose"
	"github.com/mg/ai-tui/internal/tui/history"
	"github.com/mg/ai-tui/internal/tui/selector"
)

// fakeProvider is a stub llm.Provider used only for testing provider binding.
// WithModel returns a copy with the model set so tests can assert the model
// selected by the UI is the one actually used for requests; Stream records
// the bound model so a test can assert which model would be sent.
type fakeProvider struct {
	name        string
	model       string
	streamModel string // model captured at Stream call time
}

func (f *fakeProvider) Stream(ctx context.Context, messages []llm.ChatMessage) (<-chan llm.StreamChunk, error) {
	f.streamModel = f.model
	return nil, nil
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Model() string { return f.model }

// WithModel returns a SHALLOW COPY of f with model set to the argument. The
// receiver is not mutated, matching the real providers' semantics.
func (f *fakeProvider) WithModel(model string) llm.Provider {
	cp := *f
	cp.model = model
	return &cp
}

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

// multiModelConfig builds a config with a single provider "test" that lists
// two models ("alpha", "beta") so the selector/model-override path can be
// exercised under one provider.
func multiModelConfig() *config.Config {
	return &config.Config{
		DefaultProvider: "test",
		Providers: map[string]config.Provider{
			"test": {
				Model:  "alpha",
				Models: []string{"alpha", "beta"},
			},
		},
	}
}

// stripANSI removes ANSI escape sequences so status-bar assertions can match
// the plain text a user would see.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			// skip escape sequence: ESC '[' ... letter
			i++
			for i < len(s) && !isASCIILetter(s[i]) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isASCIILetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// typeIntoCompose sends runes through the AppModel to populate the compose
// textarea, mimicking a user typing.
func typeIntoCompose(t *testing.T, m AppModel, text string) AppModel {
	t.Helper()
	for _, r := range text {
		um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = um.(AppModel)
	}
	return m
}

// sendFirstMessage submits the textarea content. For a fresh compose (no
// session yet) with a db and provider but no program, the only dispatched
// command is createSessionCmd; tea.Batch returns it directly so we can run it
// synchronously and read the created session.
func sendFirstMessage(t *testing.T, m AppModel) (*db.Session, AppModel) {
	t.Helper()
	um, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = um.(AppModel)
	if cmd == nil {
		t.Fatal("expected a createSession command to be dispatched on send")
	}
	msg := cmd()
	created, ok := msg.(compose.SessionCreatedMsg)
	if !ok {
		t.Fatalf("expected compose.SessionCreatedMsg, got %T", msg)
	}
	if created.Session == nil {
		t.Fatal("expected non-nil Session")
	}
	return created.Session, m
}

// TestModelSelectedMsg_ScopesProviderToSelectedModel proves that after
// selecting a model, the compose's bound provider is scoped to that model —
// the core fix for the bug where ModelName was ignored.
func TestModelSelectedMsg_ScopesProviderToSelectedModel(t *testing.T) {
	prov := &fakeProvider{name: "test", model: "alpha"}
	providers := map[string]llm.Provider{"test": prov}
	m := NewAppModel(multiModelConfig(), nil, providers)

	um, _ := m.Update(selector.ModelSelectedMsg{ProviderName: "test", ModelName: "beta"})
	updated := um.(AppModel)

	if updated.compose.Provider() == nil {
		t.Fatal("expected compose to have a bound provider")
	}
	if got := updated.compose.Provider().Model(); got != "beta" {
		t.Errorf("expected bound provider model 'beta', got %q", got)
	}
	if updated.ActiveProvider() != "test" {
		t.Errorf("expected active provider 'test', got %q", updated.ActiveProvider())
	}

	// The original provider in the map is untouched (no mutation).
	if prov.Model() != "alpha" {
		t.Errorf("original provider mutated: model = %q, want %q", prov.Model(), "alpha")
	}
}

// TestModelSelectedMsg_DifferentModelsProduceDifferentSessions proves the
// acceptance criterion: selecting two different models under the SAME
// provider produces sessions whose Model differs.
func TestModelSelectedMsg_DifferentModelsProduceDifferentSessions(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer database.Close()

	providers := map[string]llm.Provider{"test": &fakeProvider{name: "test", model: "alpha"}}

	// Select "alpha" and create a session.
	m := NewAppModel(multiModelConfig(), database, providers)
	m = typeIntoCompose(t, m, "hi")
	sessAlpha, m := sendFirstMessage(t, m)
	if sessAlpha.Model != "alpha" {
		t.Errorf("alpha session Model = %q, want %q", sessAlpha.Model, "alpha")
	}

	// Select "beta" and create a fresh session.
	um, _ := m.Update(selector.ModelSelectedMsg{ProviderName: "test", ModelName: "beta"})
	m = um.(AppModel)
	m = typeIntoCompose(t, m, "hi")
	sessBeta, _ := sendFirstMessage(t, m)
	if sessBeta.Model != "beta" {
		t.Errorf("beta session Model = %q, want %q", sessBeta.Model, "beta")
	}

	if sessAlpha.Model == sessBeta.Model {
		t.Errorf("expected different models, both were %q", sessAlpha.Model)
	}

	// Both rows persisted to SQLite with distinct Model values.
	persistedAlpha, err := database.GetSession(sessAlpha.ID)
	if err != nil {
		t.Fatalf("failed to load alpha session: %v", err)
	}
	persistedBeta, err := database.GetSession(sessBeta.ID)
	if err != nil {
		t.Fatalf("failed to load beta session: %v", err)
	}
	if persistedAlpha.Model != "alpha" || persistedBeta.Model != "beta" {
		t.Errorf("persisted models alpha=%q beta=%q", persistedAlpha.Model, persistedBeta.Model)
	}
}

// TestModelSelectedMsg_SelectedModelUsedByProvider proves the selected model is
// the one the provider would send in a request. The fakeProvider records its
// bound model when Stream is invoked; after selecting "beta", a Stream on the
// compose's bound provider must record "beta".
func TestModelSelectedMsg_SelectedModelUsedByProvider(t *testing.T) {
	providers := map[string]llm.Provider{"test": &fakeProvider{name: "test", model: "alpha"}}
	m := NewAppModel(multiModelConfig(), nil, providers)

	um, _ := m.Update(selector.ModelSelectedMsg{ProviderName: "test", ModelName: "beta"})
	updated := um.(AppModel)

	p := updated.compose.Provider()
	if p == nil {
		t.Fatal("expected compose to have a bound provider")
	}
	fp, ok := p.(*fakeProvider)
	if !ok {
		t.Fatalf("expected *fakeProvider, got %T", p)
	}
	if _, err := fp.Stream(context.Background(), nil); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if fp.streamModel != "beta" {
		t.Errorf("provider would send model %q, want %q", fp.streamModel, "beta")
	}
}

// TestModelSelectedMsg_StatusBarReflectsSelectedModel proves the status bar
// shows the actually-selected model (not the provider's configured model) —
// the second part of the bug fix.
func TestModelSelectedMsg_StatusBarReflectsSelectedModel(t *testing.T) {
	providers := map[string]llm.Provider{"test": &fakeProvider{name: "test", model: "alpha"}}
	m := NewAppModel(multiModelConfig(), nil, providers)
	// Give the compose a size so View() is well-formed.
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(AppModel)

	// Before selection, the status bar reflects the configured model "alpha".
	view := stripANSI(m.View())
	if !strings.Contains(view, "alpha") {
		t.Errorf("pre-select status bar should contain 'alpha', got: %s", view)
	}

	// Select "beta".
	um, _ = m.Update(selector.ModelSelectedMsg{ProviderName: "test", ModelName: "beta"})
	updated := um.(AppModel)

	view = stripANSI(updated.View())
	if !strings.Contains(view, "beta") {
		t.Errorf("status bar should contain selected model 'beta', got: %s", view)
	}
	// The other model from the same provider must NOT appear: proving the bar
	// reflects the override rather than the configured default.
	// ("alpha" still appears nowhere else in an empty compose view.)
	if strings.Contains(view, "alpha") {
		t.Errorf("status bar should NOT contain 'alpha' after selecting 'beta', got: %s", view)
	}
}

// TestResumeSessionMsg_ScopesProviderToSessionModel proves a resumed session
// created with a model override keeps using that model on follow-ups: the
// compose's bound provider is scoped to the session's Model.
func TestResumeSessionMsg_ScopesProviderToSessionModel(t *testing.T) {
	providers := map[string]llm.Provider{
		"test": &fakeProvider{name: "test", model: "alpha"},
	}
	cfg := &config.Config{
		DefaultProvider: "test",
		Providers: map[string]config.Provider{
			"test": {Model: "alpha", Models: []string{"alpha", "beta"}},
		},
	}
	m := NewAppModel(cfg, nil, providers)

	// Resume a session whose stored Model is an override ("beta") differing
	// from the provider's configured default ("alpha").
	session := db.Session{ID: "s1", Provider: "test", Model: "beta"}
	um, _ := m.Update(history.ResumeSessionMsg{Session: session, Messages: nil})
	updated := um.(AppModel)

	p := updated.compose.Provider()
	if p == nil {
		t.Fatal("expected compose to have a bound provider")
	}
	if got := p.Model(); got != "beta" {
		t.Errorf("expected resumed provider scoped to 'beta', got %q", got)
	}
}
