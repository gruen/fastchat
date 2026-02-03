package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
	"github.com/mg/ai-tui/internal/tui/compose"
	"github.com/mg/ai-tui/internal/tui/history"
)

// View represents the currently active view
type View int

const (
	ComposeView View = iota
	HistoryView
)

// AppModel is the root model for the TUI application
type AppModel struct {
	activeView View
	compose    compose.Model
	history    history.Model
	cfg        *config.Config
	db         *db.DB
	providers  map[string]llm.Provider
	width      int
	height     int
	quitting   bool
	program    *tea.Program
	help       help.Model
}

// NewAppModel creates a new root application model
func NewAppModel(cfg *config.Config, database *db.DB, providers map[string]llm.Provider) AppModel {
	return AppModel{
		activeView: ComposeView,
		compose:    compose.New(database, providers[cfg.DefaultProvider]),
		history:    history.New(database, cfg.Storage.NotesDir),
		cfg:        cfg,
		db:         database,
		providers:  providers,
		help:       help.New(),
	}
}

// SetProgram sets the tea.Program reference for sending messages
func (m *AppModel) SetProgram(p *tea.Program) {
	m.program = p
	m.compose.SetProgram(p)
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages for the root model
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case history.ResumeSessionMsg:
		m.activeView = ComposeView
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for status bar (1 line) and help bar (1 line)
		contentHeight := msg.Height - 2
		m.compose.SetSize(msg.Width, contentHeight)
		m.history.SetSize(msg.Width, contentHeight)

		return m, nil

	case tea.KeyMsg:
		// Handle global key bindings
		switch {
		case key.Matches(msg, GlobalKeys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, GlobalKeys.History):
			m.activeView = HistoryView
			cmd := m.history.Init()
			return m, cmd

		case key.Matches(msg, GlobalKeys.NewChat):
			m.activeView = ComposeView
			m.compose = compose.New(m.db, m.providers[m.cfg.DefaultProvider])
			m.compose.SetProgram(m.program)
			m.compose.SetSize(m.width, m.height-2)
			return m, nil

		default:
			// Delegate to active view
			var cmd tea.Cmd
			switch m.activeView {
			case ComposeView:
				m.compose, cmd = m.compose.Update(msg)
			case HistoryView:
				m.history, cmd = m.history.Update(msg)
			}
			return m, cmd
		}
	}

	// Delegate other messages to active view
	var cmd tea.Cmd
	switch m.activeView {
	case ComposeView:
		m.compose, cmd = m.compose.Update(msg)
	case HistoryView:
		m.history, cmd = m.history.Update(msg)
	}
	return m, cmd
}

// View renders the application UI
func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	var content string
	switch m.activeView {
	case ComposeView:
		content = m.compose.View()
	case HistoryView:
		content = m.history.View()
	}

	// Build status bar
	providerName := m.cfg.DefaultProvider
	modelName := ""
	if provider, ok := m.cfg.Providers[providerName]; ok {
		modelName = provider.Model
	}
	statusBar := StatusBarStyle.Render(fmt.Sprintf("Provider: %s | Model: %s", providerName, modelName))

	// Build help bar
	helpView := m.help.ShortHelpView(GlobalKeys.ShortHelp())
	helpBar := HelpBarStyle.Render(helpView)

	// Combine all parts
	parts := []string{content, statusBar, helpBar}
	return strings.Join(parts, "\n")
}
