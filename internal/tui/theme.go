package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme encapsulates color tokens and styling functions for git-timemachine.
type Theme struct {
	ID          string
	Light       bool
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
		ID:          "catppuccin",
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
		ID:          "tokyonight",
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
		ID:          "nord",
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
		ID:          "monokai",
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

// Additional palettes use darker accents on light surfaces and brighter accents
// on dark surfaces. The eight lane colors also supply semantic UI colors.
func palette(id, name, background, foreground, muted, border string, light bool, lanes [8]string) Theme {
	return Theme{
		ID: id, Name: name, Light: light,
		Background: background, Foreground: foreground, Muted: muted, Border: border,
		Cursor: lanes[0], BorderFocus: lanes[0], Accent: lanes[1],
		Success: lanes[2], Warning: lanes[3], Danger: lanes[4], Lanes: lanes[:],
		DiffAddBg: background, DiffAddFg: lanes[2], DiffDelBg: background,
		DiffDelFg: lanes[4], DiffHeaderFg: lanes[1], DiffChunkFg: lanes[5],
	}
}

var (
	ThemePaperWhite   = palette("paper", "Paper White", "#ffffff", "#24292f", "#57606a", "#d0d7de", true, [8]string{"#6639ba", "#0550ae", "#116329", "#7d4e00", "#a40e26", "#8250df", "#0a6575", "#953800"})
	ThemeArcticLight  = palette("arctic", "Arctic Light", "#f7fafc", "#243449", "#526579", "#cbd5e1", true, [8]string{"#245c99", "#00677a", "#24613a", "#855400", "#ad2445", "#6946a2", "#8a450f", "#4257a1"})
	ThemeRoseQuartz   = palette("rose", "Rose Quartz", "#fff8fb", "#3c2538", "#795b70", "#e4cad8", true, [8]string{"#973865", "#6f46a8", "#286443", "#855100", "#b42b49", "#8051a5", "#126877", "#914323"})
	ThemeMidnightBlue = palette("midnight", "Midnight Blue", "#0b1020", "#dce7ff", "#8999bd", "#263553", false, [8]string{"#82aaff", "#89ddff", "#b4e88d", "#ffcc80", "#ff738a", "#c792ea", "#f78c6c", "#80cbc4"})
	ThemeEmber        = palette("ember", "Ember", "#211713", "#f4e5da", "#b49a88", "#50382b", false, [8]string{"#ffb86b", "#f6d080", "#b7cf85", "#f0c674", "#ff8578", "#d8a0df", "#80c4b7", "#dbb899"})
	ThemeForest       = palette("forest", "Forest", "#12201b", "#dfebdf", "#8fae9c", "#304a3c", false, [8]string{"#99d98c", "#7cd4c5", "#b5df8a", "#e6c779", "#f48c8c", "#c6a0df", "#83b8df", "#e3ad81"})
	ThemeCyberpunk    = palette("cyberpunk", "Cyberpunk", "#170d27", "#f1e7ff", "#ac91bf", "#432750", false, [8]string{"#f38bff", "#67e8f9", "#a3ed83", "#ffe477", "#ff7597", "#b7a0ff", "#ffaf75", "#82b8ff"})
	ThemeDeepOcean    = palette("ocean", "Deep Ocean", "#081e2b", "#d9edf4", "#86a7b8", "#234453", false, [8]string{"#58d6d1", "#7fbeff", "#98dca0", "#e9ce83", "#ff8d91", "#b6a5ef", "#85d4ef", "#f0b283"})
	ThemeAmethyst     = palette("amethyst", "Amethyst", "#21182d", "#eee4f5", "#b09cbd", "#493652", false, [8]string{"#d4a3ff", "#a6b9ff", "#a5d6a7", "#f1d18b", "#f995b2", "#e5b4ef", "#8dd8cd", "#efb18a"})
	ThemeSlate        = palette("slate", "Slate", "#181c22", "#e2e8f0", "#98a6b8", "#35404e", false, [8]string{"#a5b4fc", "#7dd3fc", "#86efac", "#fde68a", "#fda4af", "#d8b4fe", "#5eead4", "#fdba74"})
)

var AllThemes = []Theme{
	ThemeCatppuccinMocha,
	ThemeTokyoNight,
	ThemeNord,
	ThemeMonokaiPro,
	ThemePaperWhite,
	ThemeArcticLight,
	ThemeRoseQuartz,
	ThemeMidnightBlue,
	ThemeEmber,
	ThemeForest,
	ThemeCyberpunk,
	ThemeDeepOcean,
	ThemeAmethyst,
	ThemeSlate,
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

// ThemeByName accepts CLI IDs, display names, and the original short aliases.
func ThemeByName(name string) (Theme, bool) {
	normalize := func(s string) string {
		return strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(strings.TrimSpace(s)))
	}
	name = normalize(name)
	switch name {
	case "mocha":
		name = "catppuccin"
	case "tokyo":
		name = "tokyonight"
	}
	for _, theme := range AllThemes {
		if name == normalize(theme.ID) || name == normalize(theme.Name) {
			return theme, true
		}
	}
	return Theme{}, false
}

// ThemeIDs lists the names accepted by the CLI.
func ThemeIDs() []string {
	ids := make([]string, len(AllThemes))
	for i, theme := range AllThemes {
		ids[i] = theme.ID
	}
	return ids
}
