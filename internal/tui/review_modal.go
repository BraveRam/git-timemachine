package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type reviewChoice struct {
	label, path, hash string
	line              int
}

type reviewModal struct {
	kind, title string
	input       textinput.Model
	rows        []reviewChoice
	selected    int
	active      bool
}

func newReviewModal(kind, title string, rows []reviewChoice) reviewModal {
	input := textinput.New()
	input.Placeholder = "Type to filter…"
	if kind == "query" {
		input.Placeholder = "Find text added or removed in history…"
	}
	input.CharLimit = 256
	input.Focus()
	return reviewModal{kind: kind, title: title, input: input, rows: rows, active: true}
}

func (p reviewModal) matches() []reviewChoice {
	query := strings.ToLower(p.input.Value())
	var rows []reviewChoice
	for _, row := range p.rows {
		if strings.Contains(strings.ToLower(row.label), query) {
			rows = append(rows, row)
		}
	}
	return rows
}

func (p *reviewModal) update(msg tea.Msg) (*reviewChoice, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		rows := p.matches()
		switch key.String() {
		case "esc":
			p.active = false
			return nil, nil
		case "up", "ctrl+p":
			p.selected = max(0, p.selected-1)
			return nil, nil
		case "down", "ctrl+n":
			p.selected = min(max(0, len(rows)-1), p.selected+1)
			return nil, nil
		case "enter":
			if p.kind == "query" {
				value := strings.TrimSpace(p.input.Value())
				if value != "" {
					p.active = false
					return &reviewChoice{label: value}, nil
				}
			} else if len(rows) > 0 {
				p.active = false
				choice := rows[min(p.selected, len(rows)-1)]
				return &choice, nil
			}
			return nil, nil
		}
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.selected = 0
	return nil, cmd
}

func (p reviewModal) view(width, height int, theme Theme) string {
	w, h := max(1, width-4), max(1, height-2)
	w = min(w, 100)
	p.input.Width = max(1, w-4)
	p.input.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Foreground))
	p.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	rows := p.matches()
	lines := []string{p.title, p.input.View(), ""}
	visible := max(1, h-7)
	start := max(0, p.selected-visible+1)
	for i := start; i < len(rows) && i < start+visible; i++ {
		line := "  " + rows[i].label
		if i == p.selected {
			line = "▶ " + rows[i].label
			line = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Cursor)).Bold(true).Render(line)
		}
		lines = append(lines, ansi.Truncate(line, max(1, w-2), "…"))
	}
	if p.kind != "query" && len(rows) == 0 {
		lines = append(lines, "No matching items.")
	}
	lines = append(lines, "", fmt.Sprintf("%d items • ↑/↓ Select • Enter Open • Esc Cancel", len(rows)))
	pane := renderPane(theme, true, w, h, strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, pane)
}
