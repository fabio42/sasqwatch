package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// SasqTheme defines the color palette used throughout sasqwatch's UI.
// Use DefaultTheme to get the standard terminal-color theme.
type SasqTheme struct {
	StatusRunColor    color.Color // running (▶) state indicator
	StatusStopColor   color.Color // paused (■) state indicator
	StatusOptionColor color.Color // counters and mode indicators
	StatusBgColor     color.Color // status bar background
	StatusFgColor     color.Color // status bar foreground
	StatusModeFgColor color.Color // foreground for the run/stop mode block
	DiffColor         color.Color // background highlight for diff insertions
	OptionSeparator   string      // separates mode tokens in the status bar
}

// DefaultTheme returns the default sasqwatch color theme using standard ANSI terminal colors.
func DefaultTheme() SasqTheme {
	return SasqTheme{
		StatusRunColor:    lipgloss.Color("2"), // green
		StatusStopColor:   lipgloss.Color("1"), // red
		StatusOptionColor: lipgloss.Color("3"), // yellow
		StatusBgColor:     lipgloss.Color("0"), // black
		StatusFgColor:     lipgloss.Color("7"), // white
		StatusModeFgColor: lipgloss.Color("0"), // black — readable on green/red backgrounds
		DiffColor:         lipgloss.Color("1"), // red
		OptionSeparator:   "| ",
	}
}
