package tui

import (
	"fmt"
	"strings"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// DetailsViewModel renders the active commit metadata card.
type DetailsViewModel struct {
	width       int
	height      int
	commit      *git.Commit
	diffData    *git.DiffData
	focused     bool
	theme       Theme
	workingMode string
}

// NewDetailsViewModel creates a new DetailsViewModel.
func NewDetailsViewModel(theme Theme) DetailsViewModel {
	return DetailsViewModel{
		theme: theme,
	}
}

// SetSize updates pane dimensions.
func (d *DetailsViewModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// SetFocus sets whether this pane has keyboard focus.
func (d *DetailsViewModel) SetFocus(focused bool) {
	d.focused = focused
}

// SetState updates current commit, diff stats, and theme.
func (d *DetailsViewModel) SetState(c *git.Commit, diff *git.DiffData, theme Theme) {
	d.commit = c
	d.diffData = diff
	d.theme = theme
}

// View renders the details card.
func (d DetailsViewModel) View() string {
	w := d.width
	h := d.height
	if w <= 0 || h <= 0 {
		return ""
	}

	titleText := " Commit Inspector "
	if d.focused {
		titleText = " ● Commit Inspector "
	}

	if d.workingMode != "" {
		summary := "No tracked file changes."
		if d.diffData != nil {
			summary = fmt.Sprintf("%d files • +%d -%d", len(d.diffData.Stats), d.diffData.TotalAdditions, d.diffData.TotalDeletions)
		}
		return renderPane(d.theme, d.focused, w, h, " Commit Inspector\n"+d.workingMode+"\n"+summary+"\nF: choose file • w: switch view • r: refresh")
	}

	if d.commit == nil {
		return renderPane(d.theme, d.focused, w, h,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(d.theme.Muted)).
				Render("Select a commit to view details."),
		)
	}

	c := d.commit

	// GPG signature badge
	var gpgBadge string
	if c.IsSigned() {
		gpgBadge = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color(d.theme.Success)).
			Foreground(lipgloss.Color(d.theme.Background)).
			Padding(0, 1).
			Render("✓ SIGNED")
	} else {
		gpgBadge = lipgloss.NewStyle().
			Background(lipgloss.Color(d.theme.Border)).
			Foreground(lipgloss.Color(d.theme.Muted)).
			Padding(0, 1).
			Render("UNSIGNED")
	}

	// Ref badges
	var refBadges []string
	for _, r := range c.Refs {
		if r.Type == git.RefTag {
			refBadges = append(refBadges, lipgloss.NewStyle().
				Foreground(lipgloss.Color(d.theme.Warning)).
				Render("["+r.Name+"]"))
		} else if r.Type == git.RefHead {
			refBadges = append(refBadges, lipgloss.NewStyle().
				Foreground(lipgloss.Color(d.theme.Danger)).
				Bold(true).
				Render("[HEAD]"))
		} else if r.Type == git.RefBranch {
			refBadges = append(refBadges, lipgloss.NewStyle().
				Foreground(lipgloss.Color(d.theme.Success)).
				Render("["+r.Name+"]"))
		}
	}

	hashStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(d.theme.Accent)).
		Render(c.Hash)

	authorStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(d.theme.Foreground)).
		Render(fmt.Sprintf("%s <%s>", c.AuthorName, c.AuthorEmail))

	dateStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color(d.theme.Muted)).
		Render(fmt.Sprintf("%s (%s)", c.AuthorDate.Format("2006-01-02 15:04:05"), relativeTime(c.AuthorDate)))

	// Stat summary
	statStr := ""
	if d.diffData != nil {
		addPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.theme.DiffAddFg)).
			Render(fmt.Sprintf("+%d", d.diffData.TotalAdditions))
		delPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.theme.DiffDelFg)).
			Render(fmt.Sprintf("-%d", d.diffData.TotalDeletions))
		filesPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.theme.Foreground)).
			Render(fmt.Sprintf("across %d files", len(d.diffData.Stats)))
		statStr = fmt.Sprintf("Changes: %s %s %s", addPart, delPart, filesPart)
	}

	// Header line with commit hash & GPG badge
	row1 := fmt.Sprintf("Commit: %s  %s", hashStr, gpgBadge)
	if len(refBadges) > 0 {
		row1 += "  " + strings.Join(refBadges, " ")
	}
	row2 := fmt.Sprintf("Author: %s", authorStr)
	row3 := fmt.Sprintf("Date:   %s", dateStr)

	lines := []string{row1, row2, row3}
	if statStr != "" {
		lines = append(lines, statStr)
	}

	// Subject & Body
	lines = append(lines, "")
	subjStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(d.theme.Cursor))
	// Wrap the message within the pane instead of silently cutting off words.
	messageLines := strings.Split(ansi.Wrap(subjStyle.Render(c.Subject), max(1, w-2), ""), "\n")
	body := strings.TrimSpace(c.Body)
	if body != "" && body != c.Subject {
		body = strings.TrimSpace(strings.TrimPrefix(body, c.Subject))
		if body != "" {
			messageLines = append(messageLines, "")
			messageLines = append(messageLines, strings.Split(ansi.Wrap(body, max(1, w-2), ""), "\n")...)
		}
	}
	available := max(0, h-3-len(lines)) // borders and title
	if len(messageLines) > available && available > 0 {
		messageLines = messageLines[:available]
		messageLines[available-1] = ansi.Truncate(messageLines[available-1], max(0, w-3), "") + "…"
	}
	if available > 0 {
		lines = append(lines, messageLines...)
	}

	header := lipgloss.NewStyle().
		Bold(d.focused).
		Foreground(lipgloss.Color(d.theme.BorderFocus)).
		Render(titleText)

	return renderPane(d.theme, d.focused, w, h, header+"\n"+strings.Join(lines, "\n"))
}
