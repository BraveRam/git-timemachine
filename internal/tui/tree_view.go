package tui

import (
	"fmt"
	"strings"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/lipgloss"
)

// TreeViewModel manages rendering the Unicode commit history graph.
type TreeViewModel struct {
	width        int
	height       int
	commits      []*git.Commit
	currentIndex int
	scrollOffset int
	focused      bool
	theme        Theme
}

// NewTreeViewModel creates a new TreeViewModel.
func NewTreeViewModel(theme Theme) TreeViewModel {
	return TreeViewModel{
		theme:   theme,
		focused: true,
	}
}

// SetSize updates pane dimensions.
func (t *TreeViewModel) SetSize(w, h int) {
	t.width = w
	t.height = h
}

// SetFocus sets whether this pane has keyboard focus.
func (t *TreeViewModel) SetFocus(focused bool) {
	t.focused = focused
}

// SetState updates commits, active index, and theme.
func (t *TreeViewModel) SetState(commits []*git.Commit, idx int, theme Theme) {
	t.commits = commits
	t.currentIndex = idx
	t.theme = theme
	t.ensureVisible()
}

// ensureVisible adjusts scrollOffset so the active commit is centered or visible.
func (t *TreeViewModel) ensureVisible() {
	if t.height <= 2 || len(t.commits) == 0 {
		return
	}

	visibleRows := t.height - 2 // account for borders

	// Keep active commit roughly in middle of visible rows
	mid := visibleRows / 2
	targetOffset := t.currentIndex - mid

	if targetOffset < 0 {
		targetOffset = 0
	}
	maxOffset := len(t.commits) - visibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if targetOffset > maxOffset {
		targetOffset = maxOffset
	}

	t.scrollOffset = targetOffset
}

// View renders the tree view.
func (t TreeViewModel) View() string {
	w := t.width
	h := t.height
	if w <= 0 {
		w = 40
	}
	if h <= 0 {
		h = 20
	}

	titleText := " Commit Tree "
	if t.focused {
		titleText = " ● Commit Tree "
	}

	borderStyle := t.theme.BorderStyle(t.focused).
		Width(w - 2).
		Height(h - 2)

	if len(t.commits) == 0 {
		return borderStyle.Render(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.theme.Muted)).
				Render("No commits in history."),
		)
	}

	visibleRows := h - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	contentWidth := w - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	var rows []string
	startIdx := t.scrollOffset
	endIdx := startIdx + visibleRows
	if endIdx > len(t.commits) {
		endIdx = len(t.commits)
	}

	for i := startIdx; i < endIdx; i++ {
		c := t.commits[i]
		isActive := (i == t.currentIndex)
		isPast := (i < t.currentIndex)
		isFuture := (i > t.currentIndex)

		row := t.renderCommitRow(c, isActive, isPast, isFuture, contentWidth)
		rows = append(rows, row)
	}

	// Pad remaining rows if fewer than visibleRows
	for len(rows) < visibleRows {
		rows = append(rows, strings.Repeat(" ", contentWidth))
	}

	innerContent := strings.Join(rows, "\n")
	header := lipgloss.NewStyle().
		Bold(t.focused).
		Foreground(lipgloss.Color(t.theme.BorderFocus)).
		Render(titleText)

	// Combine into pane with header title
	return borderStyle.Render(header + "\n" + innerContent)
}

func (t TreeViewModel) renderCommitRow(c *git.Commit, isActive, isPast, isFuture bool, width int) string {
	// 1. Lane color
	laneCol := t.theme.GetLaneColor(c.Node.Lane)

	// 2. Format graph prefix
	prefixStyle := lipgloss.NewStyle().Foreground(laneCol)
	if isFuture {
		prefixStyle = prefixStyle.Foreground(lipgloss.Color(t.theme.Muted))
	}
	prefix := prefixStyle.Render(c.Node.Prefix)

	// 3. Short hash
	hashStyle := lipgloss.NewStyle().Foreground(laneCol)
	if isFuture {
		hashStyle = hashStyle.Foreground(lipgloss.Color(t.theme.Muted))
	}
	shortHash := hashStyle.Render(c.ShortHash)

	// 4. Ref badges (HEAD, branch, tags)
	var badges []string
	for _, r := range c.Refs {
		if r.Type == git.RefTag {
			b := lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.theme.Warning)).
				Render("◆ " + r.Name)
			badges = append(badges, b)
		} else if r.Type == git.RefHead {
			b := lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.theme.Danger)).
				Bold(true).
				Render("⚑ HEAD")
			badges = append(badges, b)
		} else if r.Type == git.RefBranch {
			b := lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.theme.Success)).
				Render(" " + r.Name)
			badges = append(badges, b)
		}
	}
	badgesStr := strings.Join(badges, " ")

	// 5. Subject
	subj := c.Subject
	subjStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.theme.Foreground))
	if isFuture {
		subjStyle = subjStyle.Foreground(lipgloss.Color(t.theme.Muted))
	} else if isPast {
		subjStyle = subjStyle.Foreground(lipgloss.Color(t.theme.Foreground))
	}

	cursorPrefix := "  "
	if isActive {
		cursorPrefix = "▶ "
	}

	leftPart := fmt.Sprintf("%s%s %s ", cursorPrefix, prefix, shortHash)
	if badgesStr != "" {
		leftPart += badgesStr + " "
	}

	leftLen := lipgloss.Width(leftPart)
	availSubj := width - leftLen - 1
	if availSubj > 0 {
		if len(subj) > availSubj {
			if availSubj > 3 {
				subj = subj[:availSubj-3] + "..."
			} else {
				subj = subj[:availSubj]
			}
		}
	} else {
		subj = ""
	}

	rowText := leftPart + subjStyle.Render(subj)
	pad := width - lipgloss.Width(rowText)
	if pad > 0 {
		rowText += strings.Repeat(" ", pad)
	}

	if isActive {
		return lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color(t.theme.Border)).
			Foreground(lipgloss.Color(t.theme.Cursor)).
			Render(rowText)
	}

	return rowText
}
