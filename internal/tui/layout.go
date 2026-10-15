package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// fitContent clips by terminal cells, preserving ANSI styles and UTF-8 text.
func fitContent(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}

// renderPane includes the border in its assigned dimensions.
func renderPane(theme Theme, focused bool, width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if width < 3 || height < 3 {
		return lipgloss.NewStyle().Width(width).Height(height).Render(fitContent(content, width, height))
	}
	return theme.BorderStyle(focused).
		Width(width - 2).Height(height - 2).
		Render(fitContent(content, width-2, height-2))
}
