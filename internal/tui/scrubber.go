package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ScrubberModel handles rendering of the horizontal timeline bar and metadata.
type ScrubberModel struct {
	width         int
	currentIndex  int
	totalCommits  int
	currentCommit *git.Commit
	isPlaying     bool
	speedText     string
	theme         Theme
}

// NewScrubberModel creates a new scrubber component.
func NewScrubberModel(theme Theme) ScrubberModel {
	return ScrubberModel{
		theme:     theme,
		speedText: "1x",
	}
}

// SetWidth sets the available rendering width.
func (s *ScrubberModel) SetWidth(w int) {
	s.width = w
}

// SetState updates scrubber state.
func (s *ScrubberModel) SetState(idx, total int, commit *git.Commit, isPlaying bool, speed string, theme Theme) {
	s.currentIndex = idx
	s.totalCommits = total
	s.currentCommit = commit
	s.isPlaying = isPlaying
	s.speedText = speed
	s.theme = theme
}

// View renders the horizontal timeline scrubber.
func (s ScrubberModel) View() string {
	if s.totalCommits == 0 {
		return renderPane(s.theme, true, s.width, 5, "No commits found in repository.")
	}

	w := s.width
	if w <= 0 {
		w = 80
	}

	// 1. Top status line
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(s.theme.Cursor)).
		Render("git-timemachine")

	var playBadge string
	if s.isPlaying {
		playBadge = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color(s.theme.Success)).
			Foreground(lipgloss.Color(s.theme.Background)).
			Padding(0, 1).
			Render(fmt.Sprintf("PLAYING %s", s.speedText))
	} else {
		playBadge = lipgloss.NewStyle().
			Background(lipgloss.Color(s.theme.Muted)).
			Foreground(lipgloss.Color(s.theme.Foreground)).
			Padding(0, 1).
			Render(fmt.Sprintf("PAUSED %s", s.speedText))
	}

	themeBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Accent)).
		Render(fmt.Sprintf("Theme: %s", s.theme.Name))

	progressPercent := 0
	if s.totalCommits > 1 {
		progressPercent = int(float64(s.currentIndex) / float64(s.totalCommits-1) * 100)
	}

	counterStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(s.theme.Foreground)).
		Render(fmt.Sprintf("[ %d / %d ] %d%%", s.currentIndex+1, s.totalCommits, progressPercent))

	topRowLeft := fmt.Sprintf("%s  %s", titleStyle, counterStyle)
	topRowRight := fmt.Sprintf("%s  %s", playBadge, themeBadge)
	topSpace := w - 4 - lipgloss.Width(topRowLeft) - lipgloss.Width(topRowRight)
	if topSpace < 1 {
		topSpace = 1
	}
	topRow := topRowLeft + strings.Repeat(" ", topSpace) + topRowRight

	// 2. Timeline slider track
	// [◀ ───●─────────────── ▶]
	sliderInnerWidth := w - 10 // account for margins and buttons
	if sliderInnerWidth < 1 {
		sliderInnerWidth = 1
	}

	needlePos := 0
	if s.totalCommits > 1 {
		needlePos = int(float64(s.currentIndex) / float64(s.totalCommits-1) * float64(sliderInnerWidth-1))
	}

	trackChars := make([]string, sliderInnerWidth)
	for i := 0; i < sliderInnerWidth; i++ {
		trackChars[i] = "─"
	}

	// Add needle
	if needlePos >= 0 && needlePos < sliderInnerWidth {
		needleChar := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(s.theme.Cursor)).
			Render("●")
		trackChars[needlePos] = needleChar
	}

	trackStr := strings.Join(trackChars, "")
	btnLeft := lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Accent)).Render("◀")
	btnRight := lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Accent)).Render("▶")
	bracketLeft := lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Muted)).Render("[")
	bracketRight := lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Muted)).Render("]")

	sliderBar := fmt.Sprintf("%s%s %s %s%s", bracketLeft, btnLeft, trackStr, btnRight, bracketRight)

	// 3. Current commit info row
	var infoRow string
	if s.currentCommit != nil {
		hashStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(s.theme.Accent)).
			Render(fmt.Sprintf("[%s]", s.currentCommit.ShortHash))

		authorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.theme.Foreground)).
			Render(s.currentCommit.AuthorName)

		relTime := relativeTime(s.currentCommit.AuthorDate)
		timeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.theme.Muted)).
			Render(relTime)

		subject := s.currentCommit.Subject
		maxSubLen := w - 4 - lipgloss.Width(hashStyle) - lipgloss.Width(authorStyle) - lipgloss.Width(timeStyle) - 7
		subject = ansi.Truncate(subject, max(0, maxSubLen), "…")
		subjStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.theme.Foreground)).
			Render(subject)

		infoRow = fmt.Sprintf("%s %s • %s • %s", hashStyle, subjStyle, authorStyle, timeStyle)
	}

	content := fmt.Sprintf("%s\n%s\n%s", topRow, sliderBar, infoRow)
	// Two border cells and two padding cells belong to the assigned width.
	if w < 5 {
		return renderPane(s.theme, true, w, 5, content)
	}
	content = fitContent(content, w-4, 3)
	return s.theme.BorderStyle(true).Width(w-2).Height(3).Padding(0, 1).Render(content)
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return fmt.Sprintf("%dd ago", days)
	}
	if days < 365 {
		return fmt.Sprintf("%dmo ago", days/30)
	}
	return fmt.Sprintf("%dy ago", days/365)
}
