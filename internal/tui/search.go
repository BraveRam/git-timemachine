package tui

import (
	"fmt"
	"strings"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchModel manages interactive search filtering of commits.
type SearchModel struct {
	input       textinput.Model
	commits     []*git.Commit
	matches     []int // indices into commits
	selectedIdx int
	active      bool
	theme       Theme
	width       int
}

// NewSearchModel creates a new search modal.
func NewSearchModel(theme Theme) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search message, author, or hash..."
	ti.CharLimit = 64
	ti.Width = 40

	return SearchModel{
		input: ti,
		theme: theme,
	}
}

// Activate opens the search input.
func (s *SearchModel) Activate(commits []*git.Commit) tea.Cmd {
	s.commits = commits
	s.active = true
	s.input.Reset()
	s.input.Focus()
	s.matches = nil
	s.selectedIdx = 0
	return textinput.Blink
}

// Deactivate closes search mode.
func (s *SearchModel) Deactivate() {
	s.active = false
	s.input.Blur()
}

// IsActive returns whether search mode is open.
func (s SearchModel) IsActive() bool {
	return s.active
}

// Update handles key events when search is open.
func (s *SearchModel) Update(msg tea.Msg) (*int, tea.Cmd) {
	if !s.active {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			s.Deactivate()
			return nil, nil
		case "enter":
			if len(s.matches) > 0 {
				target := s.matches[s.selectedIdx]
				s.Deactivate()
				return &target, nil
			}
			s.Deactivate()
			return nil, nil
		case "up":
			if len(s.matches) > 0 && s.selectedIdx > 0 {
				s.selectedIdx--
			}
			return nil, nil
		case "down":
			if len(s.matches) > 0 && s.selectedIdx < len(s.matches)-1 {
				s.selectedIdx++
			}
			return nil, nil
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	s.filterCommits()
	return nil, cmd
}

func (s *SearchModel) filterCommits() {
	query := strings.ToLower(strings.TrimSpace(s.input.Value()))
	if query == "" {
		s.matches = nil
		s.selectedIdx = 0
		return
	}

	var matches []int
	for i, c := range s.commits {
		if strings.Contains(strings.ToLower(c.Subject), query) ||
			strings.Contains(strings.ToLower(c.AuthorName), query) ||
			strings.HasPrefix(strings.ToLower(c.Hash), query) ||
			strings.HasPrefix(strings.ToLower(c.ShortHash), query) {
			matches = append(matches, i)
		}
	}

	s.matches = matches
	if s.selectedIdx >= len(s.matches) {
		s.selectedIdx = 0
	}
}

// View renders the centered search bar overlay.
func (s SearchModel) View(width int) string {
	if !s.active {
		return ""
	}

	w := width - 10
	if w > 70 {
		w = 70
	}
	if w < 30 {
		w = 30
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(s.theme.Cursor)).
		Render("Search History")

	countText := lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Muted)).
		Render(fmt.Sprintf("(%d matches)", len(s.matches)))

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Muted)).
		Render("[Enter] Jump to commit • [↑/↓] Select • [Esc] Cancel")

	var previewLines []string
	if len(s.matches) > 0 {
		maxPreviews := 5
		start := max(0, s.selectedIdx-maxPreviews+1)
		for i := start; i < len(s.matches) && i < start+maxPreviews; i++ {
			idx := s.matches[i]
			c := s.commits[idx]
			prefix := "  "
			if i == s.selectedIdx {
				prefix = "▶ "
			}

			line := fmt.Sprintf("%s[%s] %s (%s)", prefix, c.ShortHash, c.Subject, c.AuthorName)
			if i == s.selectedIdx {
				line = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(s.theme.Cursor)).
					Render(line)
			} else {
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color(s.theme.Foreground)).
					Render(line)
			}
			previewLines = append(previewLines, line)
		}
	}

	body := fmt.Sprintf("%s  %s\n\n%s\n\n%s", title, countText, s.input.View(), strings.Join(previewLines, "\n"))
	if len(previewLines) > 0 {
		body += "\n\n"
	} else {
		body += "\n"
	}
	body += hint

	return lipgloss.NewStyle().
		Width(w).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(s.theme.BorderFocus)).
		Background(lipgloss.Color(s.theme.Background)).
		Padding(1, 2).
		Render(body)
}
