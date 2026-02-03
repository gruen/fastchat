package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global key bindings available across all views
type GlobalKeyMap struct {
	History key.Binding // ctrl+h - view conversation history
	NewChat key.Binding // ctrl+n - start a new chat
	Quit    key.Binding // ctrl+d - quit the application
}

// ShortHelp returns the key bindings to show in the help bar
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.History, k.NewChat, k.Quit}
}

// GlobalKeys is the global key map instance
var GlobalKeys = GlobalKeyMap{
	History: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("ctrl+h", "history"),
	),
	NewChat: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new chat"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "quit"),
	),
}
