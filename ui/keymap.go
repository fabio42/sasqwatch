package ui

import "github.com/charmbracelet/bubbles/key"

const (
	helpHeight     = 1
	helpFullHeight = 4
)

var (
	km = keymap{
		pause: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "pause/unpause")),
		run:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "trigger command")),
		prev:  key.NewBinding(key.WithKeys("[", "{"), key.WithHelp("[", "previous record")),
		next:  key.NewBinding(key.WithKeys("]", "}"), key.WithHelp("]", "next record")),
		diff:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "change diff mode")),
		quit:  key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("ctrl+c/q", "quit")),
		copy:  key.NewBinding(key.WithKeys("y", "c"), key.WithHelp("y/c", "copy to clipboard")),
		help:  key.NewBinding(key.WithKeys("?", "h"), key.WithHelp("?/h", "help")),
		nav:   key.NewBinding(key.WithKeys(""), key.WithHelp("↑↓←→", "Pager navigation")),
	}
)

type keymap struct {
	pause key.Binding
	run   key.Binding
	prev  key.Binding
	next  key.Binding
	quit  key.Binding
	diff  key.Binding
	copy  key.Binding
	help  key.Binding
	nav   key.Binding
}

func (m *Model) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.diff,
		m.keymap.help,
		m.keymap.quit,
	})
}

func (m *Model) helpFullView() string {
	return "\n" + m.help.FullHelpView([][]key.Binding{
		{
			m.keymap.pause,
			m.keymap.run,
			m.keymap.prev,
			m.keymap.next,
		},
		{
			m.keymap.diff,
			m.keymap.copy,
			m.keymap.nav,
		},
		{
			m.keymap.help,
			m.keymap.quit,
		},
	})
}
