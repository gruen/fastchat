package selector

import (
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
)

// ModelItem implements list.Item for provider/model pairs
type ModelItem struct {
	ProviderName string
	ModelName    string
}

func (i ModelItem) Title() string {
	return i.ProviderName + " > " + i.ModelName
}

func (i ModelItem) Description() string {
	return ""
}

func (i ModelItem) FilterValue() string {
	return i.Title()
}

// ModelSelectedMsg is sent when a model is selected
type ModelSelectedMsg struct {
	ProviderName string
	ModelName    string
}

// Model is the model selector overlay
type Model struct {
	list   list.Model
	active bool
	width  int
	height int
}

// New creates a new model selector from the provider config
func New(providers map[string]config.Provider) Model {
	// Sort provider names alphabetically
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build list items
	items := make([]list.Item, 0, len(providers))
	for _, name := range names {
		provider := providers[name]
		items = append(items, ModelItem{
			ProviderName: name,
			ModelName:    provider.Model,
		})
	}

	// Create list
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 80, 20)
	l.Title = "Select model"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	return Model{
		list:   l,
		active: false,
	}
}

// SetSize updates the dimensions
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

// Toggle shows or hides the selector and resets filter when opening
func (m *Model) Toggle() {
	m.active = !m.active
	if m.active {
		m.list.ResetFilter()
	}
}

// IsActive returns whether the selector is currently shown
func (m *Model) IsActive() bool {
	return m.active
}

// Update handles messages for the model selector
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.active = false
			return m, nil

		case "enter":
			if item, ok := m.list.SelectedItem().(ModelItem); ok {
				m.active = false
				return m, func() tea.Msg {
					return ModelSelectedMsg{
						ProviderName: item.ProviderName,
						ModelName:    item.ModelName,
					}
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the model selector
func (m Model) View() string {
	if !m.active {
		return ""
	}
	return m.list.View()
}
