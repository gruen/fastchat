package compose

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
)

// Message types for compose view
type StreamChunkMsg struct {
	Content string
	Done    bool
}

type StreamErrMsg struct {
	Err error
}

type StreamStartedMsg struct {
	Cancel context.CancelFunc
}

type SessionCreatedMsg struct {
	Session *db.Session
}

type MessageSavedMsg struct{}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func createSessionCmd(database *db.DB, provider llm.Provider) tea.Cmd {
	return func() tea.Msg {
		now := time.Now()
		s := &db.Session{
			ID:        newUUID(),
			Provider:  provider.Name(),
			CreatedAt: now,
			UpdatedAt: now,
		}
		database.CreateSession(s)
		return SessionCreatedMsg{Session: s}
	}
}

func saveMessageCmd(database *db.DB, sessionID, role, content string) tea.Cmd {
	return func() tea.Msg {
		m := &db.Message{
			SessionID: sessionID,
			Role:      role,
			Content:   content,
			CreatedAt: time.Now(),
		}
		database.AddMessage(m)
		return MessageSavedMsg{}
	}
}

func updateTitleCmd(database *db.DB, sessionID, title string) tea.Cmd {
	return func() tea.Msg {
		database.UpdateSessionTitle(sessionID, title)
		return MessageSavedMsg{}
	}
}

func streamCmd(provider llm.Provider, msgs []llm.ChatMessage, p *tea.Program) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())

		ch, err := provider.Stream(ctx, msgs)
		if err != nil {
			cancel()
			return StreamErrMsg{Err: err}
		}

		go func() {
			for chunk := range ch {
				if chunk.Error != nil {
					p.Send(StreamErrMsg{Err: chunk.Error})
					return
				}
				p.Send(StreamChunkMsg{Content: chunk.Content, Done: chunk.Done})
			}
		}()

		return StreamStartedMsg{Cancel: cancel}
	}
}
