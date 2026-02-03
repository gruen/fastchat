package history

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mg/ai-tui/internal/db"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// sessionItem implements list.Item
type sessionItem struct {
	session db.Session
}

func (i sessionItem) Title() string {
	if i.session.Title == "" {
		return "Untitled"
	}
	return i.session.Title
}

func (i sessionItem) Description() string {
	return fmt.Sprintf("%s | %s | %s", i.session.Provider, i.session.Model, i.session.CreatedAt.Format("Jan 2 15:04"))
}

func (i sessionItem) FilterValue() string { return i.Title() }

// Message types
type SessionsLoadedMsg struct{ Sessions []db.Session }
type SessionArchivedMsg struct{ SessionID string }
type SessionExportedMsg struct{ Path string }
type ResumeSessionMsg struct {
	Session  db.Session
	Messages []db.Message
}

// Model is the history view for browsing past sessions.
type Model struct {
	list         list.Model
	sessions     []db.Session
	db           *db.DB
	notesDir     string
	showArchived bool
	width        int
	height       int
	statusMsg    string
}

// New creates a new history view model.
func New(database *db.DB, notesDir string) Model {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "Chat History"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)

	return Model{
		list:     l,
		db:       database,
		notesDir: notesDir,
	}
}

// SetSize updates the dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h-1)
}

// Init returns the initial command to load sessions.
func (m Model) Init() tea.Cmd {
	if m.db != nil {
		return loadSessionsCmd(m.db, m.showArchived)
	}
	return nil
}

// Update handles messages for the history view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SessionsLoadedMsg:
		m.sessions = msg.Sessions
		items := make([]list.Item, len(msg.Sessions))
		for i, s := range msg.Sessions {
			items[i] = sessionItem{session: s}
		}
		m.list.SetItems(items)
		return m, nil

	case SessionArchivedMsg:
		m.statusMsg = "Session archived"
		if m.db != nil {
			return m, loadSessionsCmd(m.db, m.showArchived)
		}
		return m, nil

	case SessionExportedMsg:
		m.statusMsg = fmt.Sprintf("Exported to %s", msg.Path)
		return m, nil

	case tea.KeyMsg:
		m.statusMsg = ""

		switch msg.String() {
		case "enter", "l":
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				if m.db != nil {
					return m, resumeSessionCmd(m.db, item.session)
				}
			}
			return m, nil

		case "s":
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				if m.db != nil {
					return m, exportSessionCmd(m.db, item.session, m.notesDir)
				}
			}
			return m, nil

		case "d":
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				if m.db != nil {
					return m, archiveSessionCmd(m.db, item.session.ID)
				}
			}
			return m, nil

		case "a":
			m.showArchived = !m.showArchived
			if m.db != nil {
				return m, loadSessionsCmd(m.db, m.showArchived)
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the history view.
func (m Model) View() string {
	var parts []string
	parts = append(parts, m.list.View())

	if m.statusMsg != "" {
		parts = append(parts, m.statusMsg)
	}

	help := "enter: open | s: save | d: archive | a: show archived | ctrl+n: new | ctrl+d: quit"
	if m.showArchived {
		help = "enter: open | s: save | d: archive | a: hide archived | ctrl+n: new | ctrl+d: quit"
	}
	parts = append(parts, helpStyle.Render(help))

	return strings.Join(parts, "\n")
}
