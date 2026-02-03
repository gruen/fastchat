package history

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/export"
)

func loadSessionsCmd(database *db.DB, includeArchived bool) tea.Cmd {
	return func() tea.Msg {
		sessions, err := database.ListSessions(includeArchived)
		if err != nil {
			return SessionsLoadedMsg{Sessions: nil}
		}
		return SessionsLoadedMsg{Sessions: sessions}
	}
}

func archiveSessionCmd(database *db.DB, sessionID string) tea.Cmd {
	return func() tea.Msg {
		database.ArchiveSession(sessionID)
		return SessionArchivedMsg{SessionID: sessionID}
	}
}

func resumeSessionCmd(database *db.DB, session db.Session) tea.Cmd {
	return func() tea.Msg {
		messages, _ := database.GetSessionMessages(session.ID)
		return ResumeSessionMsg{Session: session, Messages: messages}
	}
}

func exportSessionCmd(database *db.DB, session db.Session, notesDir string) tea.Cmd {
	return func() tea.Msg {
		messages, _ := database.GetSessionMessages(session.ID)
		path, _ := export.ToMarkdown(session, messages, notesDir)
		return SessionExportedMsg{Path: path}
	}
}
