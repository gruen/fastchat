package history

import tea "github.com/charmbracelet/bubbletea"

type Model struct {
	width, height int
}

func New() Model { return Model{} }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) { return m, nil }

func (m Model) View() string { return "history view (placeholder)" }

func (m *Model) SetSize(w, h int) { m.width = w; m.height = h }
