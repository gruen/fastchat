package compose

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
