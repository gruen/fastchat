package compose

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
)

// Local styles — do NOT import from internal/tui to avoid import cycle
var (
	userStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	assistantStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	errorStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// DisplayMessage holds a rendered conversation message.
type DisplayMessage struct {
	Role    string
	Content string
}

// Model is the compose view for chatting with an LLM.
type Model struct {
	textarea  textarea.Model
	viewport  viewport.Model
	messages  []DisplayMessage
	streaming bool
	streamBuf *strings.Builder
	session   *db.Session
	db        *db.DB
	provider  llm.Provider
	program   *tea.Program
	cancelFn  context.CancelFunc
	err       error
	width     int
	height    int
}

// New creates a new compose view model.
func New(database *db.DB, provider llm.Provider) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(3)

	vp := viewport.New(80, 20)

	return Model{
		textarea:  ta,
		viewport:  vp,
		db:        database,
		provider:  provider,
		streamBuf: &strings.Builder{},
	}
}

// SetProgram sets the tea.Program reference for streaming.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// SetSize updates the dimensions of the compose view.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	taHeight := 3
	helpHeight := 1
	vpHeight := h - taHeight - helpHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.Width = w
	m.viewport.Height = vpHeight
	m.textarea.SetWidth(w)
	m.textarea.SetHeight(taHeight)
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the compose view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if !m.streaming && strings.TrimSpace(m.textarea.Value()) != "" {
				text := strings.TrimSpace(m.textarea.Value())
				m.textarea.Reset()
				m.messages = append(m.messages, DisplayMessage{Role: "user", Content: text})
				m.streaming = true
				m.err = nil

				// Build chat messages for LLM
				var chatMsgs []llm.ChatMessage
				for _, dm := range m.messages {
					chatMsgs = append(chatMsgs, llm.ChatMessage{Role: dm.Role, Content: dm.Content})
				}

				if m.session == nil && m.db != nil {
					cmds = append(cmds, createSessionCmd(m.db, m.provider))
				}
				if m.session != nil && m.db != nil {
					cmds = append(cmds, saveMessageCmd(m.db, m.session.ID, "user", text))
				}
				if m.provider != nil && m.program != nil {
					cmds = append(cmds, streamCmd(m.provider, chatMsgs, m.program))
				}

				m.updateViewport()
				return m, tea.Batch(cmds...)
			}
			// If streaming or empty, pass to textarea
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd

		case tea.KeyEsc, tea.KeyCtrlC:
			if m.streaming {
				if m.cancelFn != nil {
					m.cancelFn()
				}
				m.streaming = false
				if m.streamBuf.Len() > 0 {
					m.messages = append(m.messages, DisplayMessage{Role: "assistant", Content: m.streamBuf.String()})
					m.streamBuf.Reset()
				}
				m.updateViewport()
				return m, nil
			}

		default:
			if !m.streaming {
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		}

	case StreamStartedMsg:
		m.cancelFn = msg.Cancel
		return m, nil

	case SessionCreatedMsg:
		m.session = msg.Session
		// Save the first user message that was deferred
		if m.db != nil && len(m.messages) > 0 {
			firstMsg := m.messages[0]
			cmds = append(cmds, saveMessageCmd(m.db, m.session.ID, firstMsg.Role, firstMsg.Content))
			title := firstMsg.Content
			if len(title) > 60 {
				title = title[:60] + "..."
			}
			cmds = append(cmds, updateTitleCmd(m.db, m.session.ID, title))
		}
		return m, tea.Batch(cmds...)

	case StreamChunkMsg:
		m.streamBuf.WriteString(msg.Content)
		if msg.Done {
			m.streaming = false
			content := m.streamBuf.String()
			m.messages = append(m.messages, DisplayMessage{Role: "assistant", Content: content})
			m.streamBuf.Reset()
			if m.session != nil && m.db != nil {
				cmds = append(cmds, saveMessageCmd(m.db, m.session.ID, "assistant", content))
			}
		}
		m.updateViewport()
		return m, tea.Batch(cmds...)

	case StreamErrMsg:
		m.streaming = false
		m.err = msg.Err
		m.streamBuf.Reset()
		m.updateViewport()
		return m, nil

	case MessageSavedMsg:
		return m, nil
	}

	return m, nil
}

// View renders the compose view.
func (m Model) View() string {
	var parts []string
	parts = append(parts, m.viewport.View())

	if m.streaming {
		parts = append(parts, helpStyle.Render("Generating... (esc: stop | ctrl+d: quit)"))
	} else {
		parts = append(parts, m.textarea.View())
		parts = append(parts, helpStyle.Render("enter: send | ctrl+h: history | ctrl+d: quit"))
	}

	return strings.Join(parts, "\n")
}

func (m *Model) updateViewport() {
	var sb strings.Builder
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(userStyle.Render("You:"))
			sb.WriteString("\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString(assistantStyle.Render("Assistant:"))
			sb.WriteString("\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}
	if m.streaming && m.streamBuf.Len() > 0 {
		sb.WriteString(assistantStyle.Render("Assistant:"))
		sb.WriteString("\n")
		sb.WriteString(m.streamBuf.String())
		sb.WriteString("\n")
	}
	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		sb.WriteString("\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}
