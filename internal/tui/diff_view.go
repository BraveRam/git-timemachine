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
	label        string
	query        string
	matchLines   []int
	matchIndex   int
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

	dv.viewport.Width = max(0, w-2)
	// Two border rows, the title, and the files summary precede the viewport.
	dv.viewport.Height = max(0, h-4)
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
	if dv.diffData != diff {
		dv.viewport.GotoTop()
		dv.matchIndex = 0
	}
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
		dv.matchLines = nil
		dv.viewport.SetContent(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.Muted)).
				Render("No diff content for this commit."),
		)
		return
	}

	dv.matchLines = nil
	var styledLines []string
	for index, l := range dv.diffData.HighlightedLines {
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
		if dv.query != "" && strings.Contains(l.Content, dv.query) {
			dv.matchLines = append(dv.matchLines, index)
			styledLines[len(styledLines)-1] = lipgloss.NewStyle().Background(lipgloss.Color(dv.theme.Warning)).Foreground(lipgloss.Color(dv.theme.Background)).Render(l.Content)
		}
	}

	dv.viewport.SetContent(strings.Join(styledLines, "\n"))
}

// View renders the diff inspector.
func (dv DiffViewModel) View() string {
	w := dv.width
	h := dv.height
	if w <= 0 || h <= 0 {
		return ""
	}

	titleText := " Diff Viewer "
	if dv.isFullScreen {
		titleText = " ● Fullscreen Diff (Press 'f' or 'Esc' to exit) "
	} else if dv.focused {
		titleText = " ● Diff Viewer [Focus: Tab / f for Fullscreen] "
	}

	if dv.diffData == nil {
		return renderPane(dv.theme, dv.focused || dv.isFullScreen, w, h,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(dv.theme.Muted)).
				Render("Loading diff..."),
		)
	}

	filesBar := dv.label
	if filesBar == "" {
		filesBar = "All changed files"
	}
	if len(dv.diffData.Stats) == 0 {
		filesBar += " • no changes"
	}
	if dv.query != "" {
		filesBar += fmt.Sprintf(" • %d hits", len(dv.matchLines))
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
	return renderPane(dv.theme, dv.focused || dv.isFullScreen, w, h, content)
}

func (dv *DiffViewModel) SetQuery(query string) {
	if dv.query == query {
		return
	}
	dv.query = query
	dv.matchIndex = 0
	dv.updateContent()
}
func (dv *DiffViewModel) nextMatch(delta int) {
	if len(dv.matchLines) == 0 {
		return
	}
	dv.matchIndex = (dv.matchIndex + delta + len(dv.matchLines)) % len(dv.matchLines)
	dv.viewport.SetYOffset(dv.matchLines[dv.matchIndex])
}
func (dv *DiffViewModel) gotoLine(line int) {
	if dv.diffData == nil {
		return
	}
	for i, row := range dv.diffData.HighlightedLines {
		if row.NewLine >= line {
			dv.viewport.SetYOffset(i)
			return
		}
	}
}
