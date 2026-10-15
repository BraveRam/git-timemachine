package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme encapsulates color tokens and styling functions for git-timemachine.
type Theme struct {
	Name        string
	Background  string
	Foreground  string
	Cursor      string
	Muted       string
	Accent      string
	Success     string
	Warning     string
	Danger      string
	Border      string
	BorderFocus string
	Lanes       []string

	// Diff colors
	DiffAddBg    string
	DiffAddFg    string
	DiffDelBg    string
	DiffDelFg    string
	DiffHeaderFg string
	DiffChunkFg  string
}

// Available Themes
var (
	ThemeCatppuccinMocha = Theme{
		Name:        "Catppuccin Mocha",
		Background:  "#1e1e2e",
		Foreground:  "#cdd6f4",
		Cursor:      "#cba6f7", // Mauve
		Muted:       "#6c7086", // Overlay0
		Accent:      "#89b4fa", // Blue
		Success:     "#a6e3a1", // Green
		Warning:     "#f9e2af", // Yellow
		Danger:      "#f38ba8", // Red
		Border:      "#313244", // Surface0
		BorderFocus: "#cba6f7", // Mauve
		Lanes: []string{
			"#cba6f7", // Mauve
			"#89b4fa", // Blue
			"#a6e3a1", // Green
			"#f9e2af", // Yellow
			"#fab387", // Peach
			"#94e2d5", // Teal
			"#f5c2e7", // Pink
			"#74c7ec", // Sapphire
		},
		DiffAddBg:    "#1e3a2f",
		DiffAddFg:    "#a6e3a1",
		DiffDelBg:    "#3e1e28",
		DiffDelFg:    "#f38ba8",
		DiffHeaderFg: "#89b4fa",
		DiffChunkFg:  "#cba6f7",
	}

	ThemeTokyoNight = Theme{
		Name:        "Tokyo Night",
		Background:  "#1a1b26",
		Foreground:  "#c0caf5",
		Cursor:      "#7aa2f7",
		Muted:       "#565f89",
		Accent:      "#bb9af7",
		Success:     "#9ece6a",
		Warning:     "#e0af68",
		Danger:      "#f7768e",
		Border:      "#24283b",
		BorderFocus: "#7aa2f7",
		Lanes: []string{
			"#7aa2f7", // Blue
			"#bb9af7", // Purple
			"#7dcfff", // Cyan
			"#9ece6a", // Green
			"#e0af68", // Orange
			"#f7768e", // Red
			"#2ac3de", // Teal
			"#b4f9f8", // Light Cyan
		},
		DiffAddBg:    "#1a302a",
		DiffAddFg:    "#9ece6a",
		DiffDelBg:    "#371d28",
		DiffDelFg:    "#f7768e",
		DiffHeaderFg: "#7aa2f7",
		DiffChunkFg:  "#bb9af7",
	}

	ThemeNord = Theme{
		Name:        "Nord",
		Background:  "#2e3440",
		Foreground:  "#eceff4",
		Cursor:      "#88c0d0",
		Muted:       "#4c566a",
		Accent:      "#81a1c1",
		Success:     "#a3be8c",
		Warning:     "#ebcb8b",
		Danger:      "#bf616a",
		Border:      "#3b4252",
		BorderFocus: "#88c0d0",
		Lanes: []string{
			"#88c0d0", // Nord8 Frost Cyan
			"#81a1c1", // Nord9 Frost Blue
			"#a3be8c", // Nord14 Aurora Green
			"#ebcb8b", // Nord13 Aurora Yellow
			"#d08770", // Nord12 Aurora Orange
			"#bf616a", // Nord11 Aurora Red
			"#b48ead", // Nord15 Aurora Purple
			"#5e81ac", // Nord10 Deep Frost
		},
		DiffAddBg:    "#253c30",
		DiffAddFg:    "#a3be8c",
		DiffDelBg:    "#3d242a",
		DiffDelFg:    "#bf616a",
		DiffHeaderFg: "#88c0d0",
		DiffChunkFg:  "#b48ead",
	}

	ThemeMonokaiPro = Theme{
		Name:        "Monokai Pro",
		Background:  "#2d2a2e",
		Foreground:  "#fcfcfa",
		Cursor:      "#ffd866",
		Muted:       "#727072",
		Accent:      "#78dce8",
		Success:     "#a9dc76",
		Warning:     "#ffd866",
		Danger:      "#ff6188",
		Border:      "#403e41",
		BorderFocus: "#ffd866",
		Lanes: []string{
			"#ffd866", // Yellow
			"#78dce8", // Cyan
			"#ff6188", // Red/Pink
			"#a9dc76", // Green
			"#ab9df2", // Purple
			"#fc9867", // Orange
			"#fcfcfa", // White
			"#727072", // Grey
		},
		DiffAddBg:    "#283820",
		DiffAddFg:    "#a9dc76",
		DiffDelBg:    "#382028",
		DiffDelFg:    "#ff6188",
		DiffHeaderFg: "#78dce8",
		DiffChunkFg:  "#ab9df2",
	}
)

var AllThemes = []Theme{
	ThemeCatppuccinMocha,
	ThemeTokyoNight,
	ThemeNord,
	ThemeMonokaiPro,
}

// GetLaneColor returns a lipgloss.Color for a given lane index.
func (t Theme) GetLaneColor(lane int) lipgloss.Color {
	if len(t.Lanes) == 0 {
		return lipgloss.Color(t.Accent)
	}
	return lipgloss.Color(t.Lanes[lane%len(t.Lanes)])
}

// BorderStyle returns border styling based on focus.
func (t Theme) BorderStyle(focused bool) lipgloss.Style {
	if focused {
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderFocus))
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border))
}
