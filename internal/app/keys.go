package app

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	ToggleFocus key.Binding
	Quit        key.Binding
}

var Keys = KeyMap{
	ToggleFocus: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "toggle sidebar/panel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}
