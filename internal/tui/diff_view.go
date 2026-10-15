package tui

import (
	"fmt"
	"strings"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffViewModel manages rendering changed files and the scrollable syntax diff.
type DiffViewModel struct {
	width        int
	height       int
	viewport     viewport.Model
	diffData     *git.DiffData
	focused      bool
	isFullScreen bool
	theme        Theme
}

// NewDiffViewModel creates a new DiffViewModel.
func NewDiffViewModel(theme Theme) DiffViewModel {
	vp := viewport.New(40, 10)
	return DiffViewModel{
		viewport: vp,
		theme:    theme,
	}
}

// SetSize updates pane dimensions.
func (dv *DiffViewModel) SetSize(w, h int) {
	dv.width = w
	dv.height = h

	vpWidth := w - 4
	vpHeight := h - 7 // reserve space for header, files summary, and border
	if vpWidth < 10 {
		vpWidth = 10
	}
	if vpHeight < 3 {
		vpHeight = 3
	}
	dv.viewport.Width = vpWidth
	dv.viewport.Height = vpHeight
}

// SetFocus sets whether this pane has keyboard focus.
func (dv *DiffViewModel) SetFocus(focused bool) {
	dv.focused = focused
}

// SetFullScreen toggles fullscreen mode.
func (dv *DiffViewModel) SetFullScreen(fs bool) {
	dv.isFullScreen = fs
}

// SetState updates diff data and regenerates styled diff content.
func (dv *DiffViewModel) SetState(diff *git.DiffData, theme Theme) {
	dv.diffData = diff
	dv.theme = theme
	dv.updateContent()
}

// Update passes messages to the internal viewport.
func (dv *DiffViewModel) Update(msg tea.Msg) (DiffViewModel, tea.Cmd) {
	var cmd tea.Cmd
	dv.viewport, cmd = dv.viewport.Update(msg)
	return *dv, cmd
}

func (dv *DiffViewModel) updateContent() {
	if dv.diffData == nil || len(dv.diffData.HighlightedLines) == 0 {
		dv.viewport.SetContent(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.Muted)).
				Render("No diff content for this commit."),
		)
		return
	}

	var styledLines []string
	for _, l := range dv.diffData.HighlightedLines {
		switch l.Type {
		case git.DiffLineAddition:
			styled := lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.DiffAddFg)).
				Render(l.Content)
			styledLines = append(styledLines, styled)
		case git.DiffLineDeletion:
			styled := lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.DiffDelFg)).
				Render(l.Content)
			styledLines = append(styledLines, styled)
		case git.DiffLineChunkHeader:
			styled := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(dv.theme.DiffChunkFg)).
				Render(l.Content)
			styledLines = append(styledLines, styled)
		case git.DiffLineHeader:
			styled := lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.DiffHeaderFg)).
				Render(l.Content)
			styledLines = append(styledLines, styled)
		default:
			styled := lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.Foreground)).
				Render(l.Content)
			styledLines = append(styledLines, styled)
		}
	}

	dv.viewport.SetContent(strings.Join(styledLines, "\n"))
}

// View renders the diff inspector.
func (dv DiffViewModel) View() string {
	w := dv.width
	h := dv.height
	if w <= 0 {
		w = 40
	}
	if h <= 0 {
		h = 10
	}

	titleText := " Diff Viewer "
	if dv.isFullScreen {
		titleText = " ● Fullscreen Diff (Press 'f' or 'Esc' to exit) "
	} else if dv.focused {
		titleText = " ● Diff Viewer [Focus: Tab / f for Fullscreen] "
	}

	borderStyle := dv.theme.BorderStyle(dv.focused || dv.isFullScreen).
		Width(w - 2).
		Height(h - 2)

	if dv.diffData == nil {
		return borderStyle.Render(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.Muted)).
				Render("Loading diff..."),
		)
	}

	// Files bar
	var filePills []string
	for i, f := range dv.diffData.Stats {
		if i >= 4 {
			filePills = append(filePills, lipgloss.NewStyle().Foreground(lipgloss.Color(dv.theme.Muted)).Render(fmt.Sprintf("+%d more files", len(dv.diffData.Stats)-4)))
			break
		}

		statusColor := dv.theme.Foreground
		if f.Status == "A" {
			statusColor = dv.theme.Success
		} else if f.Status == "D" {
			statusColor = dv.theme.Danger
		} else if f.Status == "M" {
			statusColor = dv.theme.Warning
		}

		statusBadge := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(statusColor)).
			Render(fmt.Sprintf("[%s]", f.Status))

		p := f.Path
		if len(p) > 20 {
			p = "..." + p[len(p)-17:]
		}

		addBadge := lipgloss.NewStyle().Foreground(lipgloss.Color(dv.theme.DiffAddFg)).Render(fmt.Sprintf("+%d", f.Additions))
		delBadge := lipgloss.NewStyle().Foreground(lipgloss.Color(dv.theme.DiffDelFg)).Render(fmt.Sprintf("-%d", f.Deletions))

		pill := fmt.Sprintf("%s %s (%s %s)", statusBadge, p, addBadge, delBadge)
		filePills = append(filePills, pill)
	}

	filesBar := strings.Join(filePills, "  ")
	if filesBar == "" {
		filesBar = lipgloss.NewStyle().Foreground(lipgloss.Color(dv.theme.Muted)).Render("Empty commit (no file changes).")
	}

	scrollPct := fmt.Sprintf("%3.f%%", dv.viewport.ScrollPercent()*100)
	scrollBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(dv.theme.Muted)).
		Render(scrollPct)

	header := lipgloss.NewStyle().
		Bold(dv.focused).
		Foreground(lipgloss.Color(dv.theme.BorderFocus)).
		Render(titleText)

	subHeader := fmt.Sprintf("%s  [%s]", filesBar, scrollBadge)

	content := fmt.Sprintf("%s\n%s\n%s", header, subHeader, dv.viewport.View())
	return borderStyle.Render(content)
}
