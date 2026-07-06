package compose

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/db"
)

func TestNewModel(t *testing.T) {
	m := New(nil, nil)
	if m.streaming {
		t.Error("new model should not be streaming")
	}
	if len(m.messages) != 0 {
		t.Error("new model should have no messages")
	}
}

func TestStreamChunkMsg(t *testing.T) {
	m := New(nil, nil)
	m.streaming = true

	m, _ = m.Update(StreamChunkMsg{Content: "Hello", Done: false})
	if m.streamBuf.String() != "Hello" {
		t.Errorf("expected streamBuf 'Hello', got '%s'", m.streamBuf.String())
	}
	if !m.streaming {
		t.Error("should still be streaming")
	}
}

func TestStreamDone(t *testing.T) {
	m := New(nil, nil)
	m.streaming = true
	m.streamBuf = &strings.Builder{}
	m.streamBuf.WriteString("Hello ")

	m, _ = m.Update(StreamChunkMsg{Content: "world", Done: true})
	if m.streaming {
		t.Error("should not be streaming after Done")
	}
	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Content != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", m.messages[0].Content)
	}
	if m.messages[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", m.messages[0].Role)
	}
}

func TestEscCancelsStream(t *testing.T) {
	m := New(nil, nil)
	m.streaming = true
	cancelled := false
	m.cancelFn = func() { cancelled = true }
	m.streamBuf = &strings.Builder{}
	m.streamBuf.WriteString("partial")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.streaming {
		t.Error("should not be streaming after Esc")
	}
	if !cancelled {
		t.Error("cancelFn should have been called")
	}
	if len(m.messages) != 1 || m.messages[0].Content != "partial" {
		t.Error("partial content should be saved as message")
	}
}

func TestStreamErrMsg(t *testing.T) {
	m := New(nil, nil)
	m.streaming = true

	m, _ = m.Update(StreamErrMsg{Err: fmt.Errorf("test error")})
	if m.streaming {
		t.Error("should not be streaming after error")
	}
	if m.err == nil {
		t.Error("err should be set")
	}
}

func TestHelpBarContent(t *testing.T) {
	m := New(nil, nil)
	view := m.View()
	if !strings.Contains(view, "enter: send") {
		t.Error("normal view should show 'enter: send'")
	}

	m.streaming = true
	view = m.View()
	if !strings.Contains(view, "esc: stop") {
		t.Error("streaming view should show 'esc: stop'")
	}
}

func TestLoadSession_SetsSessionAndMessages(t *testing.T) {
	m := New(nil, nil)
	session := &db.Session{ID: "sess-123", Provider: "openai"}
	msgs := []db.Message{
		{SessionID: "sess-123", Role: "user", Content: "Hello"},
		{SessionID: "sess-123", Role: "assistant", Content: "Hi there"},
		{SessionID: "sess-123", Role: "user", Content: "How are you?"},
	}

	m.LoadSession(session, msgs)

	if m.SessionID() != "sess-123" {
		t.Errorf("expected session id 'sess-123', got %q", m.SessionID())
	}
	if m.session == nil || m.session.ID != "sess-123" {
		t.Errorf("expected m.session.ID 'sess-123', got %v", m.session)
	}
	if m.MessageCount() != 3 {
		t.Fatalf("expected 3 messages, got %d", m.MessageCount())
	}
	if m.messages[0].Role != "user" || m.messages[0].Content != "Hello" {
		t.Errorf("unexpected first message: %+v", m.messages[0])
	}
	if m.messages[1].Role != "assistant" || m.messages[1].Content != "Hi there" {
		t.Errorf("unexpected second message: %+v", m.messages[1])
	}
	if m.messages[2].Content != "How are you?" {
		t.Errorf("unexpected last message: %+v", m.messages[2])
	}
}

func TestLoadSession_ViewShowsLoadedMessages(t *testing.T) {
	m := New(nil, nil)
	session := &db.Session{ID: "sess-1"}
	msgs := []db.Message{
		{SessionID: "sess-1", Role: "user", Content: "Hello world"},
		{SessionID: "sess-1", Role: "assistant", Content: "Greetings"},
	}

	m.LoadSession(session, msgs)
	view := m.View()

	if !strings.Contains(view, "Hello world") {
		t.Error("view should contain the loaded user message content")
	}
	if !strings.Contains(view, "Greetings") {
		t.Error("view should contain the loaded assistant message content")
	}
}

func TestLoadSession_ResetsStreamingState(t *testing.T) {
	m := New(nil, nil)
	m.streaming = true
	m.streamBuf.WriteString("partial")
	m.err = fmt.Errorf("stale error")

	m.LoadSession(&db.Session{ID: "s1"}, []db.Message{{Role: "user", Content: "x"}})

	if m.streaming {
		t.Error("streaming should be reset to false")
	}
	if m.streamBuf.Len() != 0 {
		t.Errorf("streamBuf should be reset, got %q", m.streamBuf.String())
	}
	if m.err != nil {
		t.Errorf("err should be cleared, got %v", m.err)
	}
	if m.cancelFn != nil {
		t.Error("cancelFn should be nil")
	}
}

func TestLoadSession_WithNoMessages(t *testing.T) {
	m := New(nil, nil)
	m.messages = append(m.messages, DisplayMessage{Role: "user", Content: "stale"})

	m.LoadSession(&db.Session{ID: "empty"}, nil)

	if m.SessionID() != "empty" {
		t.Errorf("expected session id 'empty', got %q", m.SessionID())
	}
	if m.MessageCount() != 0 {
		t.Errorf("expected 0 messages, got %d", m.MessageCount())
	}
}

// TestLoadSession_FollowUpSavesToSameSession proves that once a session is
// loaded, sending a new message persists to the SAME session id (no new
// sessions row). It uses a real in-memory SQLite database.
func TestLoadSession_FollowUpSavesToSameSession(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer database.Close()

	now := time.Now()
	session := &db.Session{
		ID:        "resume-1",
		Provider:  "openai",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := database.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// provider/program are nil so streamCmd is not dispatched; the only
	// command produced on send is saveMessageCmd, which tea.Batch returns
	// directly (single non-nil cmd) and thus runs synchronously when invoked.
	m := New(database, nil)
	m.LoadSession(session, []db.Message{
		{SessionID: "resume-1", Role: "user", Content: "hi"},
		{SessionID: "resume-1", Role: "assistant", Content: "hello"},
	})
	if m.SessionID() != "resume-1" {
		t.Fatalf("expected resumed session id 'resume-1', got %q", m.SessionID())
	}

	// Type a follow-up and send it.
	m.textarea.SetValue("follow up")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a save command to be dispatched")
	}
	// Execute the save command synchronously.
	cmd()

	// No new session should have been created.
	sessions, err := database.ListSessions(false)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session (no new row), got %d", len(sessions))
	} else if sessions[0].ID != "resume-1" {
		t.Errorf("expected session id 'resume-1', got %q", sessions[0].ID)
	}

	// The follow-up user message should be appended to the same session id.
	saved, err := database.GetSessionMessages("resume-1")
	if err != nil {
		t.Fatalf("failed to get session messages: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 saved follow-up message, got %d", len(saved))
	}
	if saved[0].SessionID != "resume-1" {
		t.Errorf("expected session id 'resume-1', got %q", saved[0].SessionID)
	}
	if saved[0].Role != "user" || saved[0].Content != "follow up" {
		t.Errorf("unexpected saved message: %+v", saved[0])
	}

	// The in-memory display list should also reflect the appended message.
	if m.MessageCount() != 3 {
		t.Errorf("expected 3 display messages (2 loaded + 1 sent), got %d", m.MessageCount())
	}
}
