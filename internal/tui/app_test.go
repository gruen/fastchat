package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
)

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
	m := NewAppModel(testConfig(), nil, nil)

	if m.activeView != ComposeView {
		t.Errorf("expected activeView to be ComposeView, got %v", m.activeView)
	}
}

func TestAppModel_CtrlD_SetsQuittingAndReturnsQuit(t *testing.T) {
	m := NewAppModel(testConfig(), nil, nil)

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
	m := NewAppModel(testConfig(), nil, nil)

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
	m := NewAppModel(testConfig(), nil, nil)

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
	m := NewAppModel(testConfig(), nil, nil)

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
