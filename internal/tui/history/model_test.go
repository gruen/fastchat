package history

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/db"
)

func TestNewModel(t *testing.T) {
	m := New(nil, "/tmp/notes")
	if m.showArchived {
		t.Error("showArchived should be false initially")
	}
	if m.notesDir != "/tmp/notes" {
		t.Errorf("expected notesDir '/tmp/notes', got '%s'", m.notesDir)
	}
}

func TestSessionsLoaded(t *testing.T) {
	m := New(nil, "/tmp/notes")
	sessions := []db.Session{
		{ID: "1", Title: "First", Provider: "claude", Model: "sonnet", CreatedAt: time.Now()},
		{ID: "2", Title: "Second", Provider: "openai", Model: "gpt-4o", CreatedAt: time.Now()},
	}

	m, _ = m.Update(SessionsLoadedMsg{Sessions: sessions})
	if len(m.sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(m.sessions))
	}
	if len(m.list.Items()) != 2 {
		t.Errorf("expected 2 list items, got %d", len(m.list.Items()))
	}
}

func TestArchiveToggle(t *testing.T) {
	m := New(nil, "/tmp/notes")
	if m.showArchived {
		t.Error("should start with showArchived=false")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !m.showArchived {
		t.Error("should be showArchived=true after 'a'")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.showArchived {
		t.Error("should be showArchived=false after second 'a'")
	}
}

func TestSessionExportedMsg(t *testing.T) {
	m := New(nil, "/tmp/notes")
	m, _ = m.Update(SessionExportedMsg{Path: "/tmp/notes/test.md"})
	if m.statusMsg == "" {
		t.Error("statusMsg should be set after export")
	}
}

func TestSessionArchivedMsg(t *testing.T) {
	m := New(nil, "/tmp/notes")
	m, _ = m.Update(SessionArchivedMsg{SessionID: "123"})
	if m.statusMsg == "" {
		t.Error("statusMsg should be set after archive")
	}
}
