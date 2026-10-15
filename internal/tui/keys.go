package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines the keybindings for git-timemachine.
type KeyMap struct {
	ScrubLeft      key.Binding
	ScrubRight     key.Binding
	JumpBack10     key.Binding
	JumpForward10  key.Binding
	JumpFirst      key.Binding
	JumpLast       key.Binding
	TogglePlay     key.Binding
	SpeedDown      key.Binding
	SpeedUp        key.Binding
	Tab            key.Binding
	ShiftTab       key.Binding
	ToggleDiffFull key.Binding
	Search         key.Binding
	Theme          key.Binding
	Help           key.Binding
	Quit           key.Binding
	Up             key.Binding
	Down           key.Binding
	Esc            key.Binding
}

// DefaultKeyMap returns the default keybindings for the application.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		ScrubLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "scrub backward"),
		),
		ScrubRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "scrub forward"),
		),
		JumpBack10: key.NewBinding(
			key.WithKeys("shift+left", "J"),
			key.WithHelp("J/⇧←", "jump -10"),
		),
		JumpForward10: key.NewBinding(
			key.WithKeys("shift+right", "K"),
			key.WithHelp("K/⇧→", "jump +10"),
		),
		JumpFirst: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/Home", "initial commit"),
		),
		JumpLast: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/End", "latest commit"),
		),
		TogglePlay: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("Space", "play/pause timelapse"),
		),
		SpeedDown: key.NewBinding(
			key.WithKeys("[", "-"),
			key.WithHelp("[/-", "slower speed"),
		),
		SpeedUp: key.NewBinding(
			key.WithKeys("]", "+", "="),
			key.WithHelp("]/+", "faster speed"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("⇧Tab", "prev pane"),
		),
		ToggleDiffFull: key.NewBinding(
			key.WithKeys("f", "enter"),
			key.WithHelp("f/Enter", "fullscreen diff"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search commits"),
		),
		Theme: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "cycle theme"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "back/close"),
		),
	}
}
