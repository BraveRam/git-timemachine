package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/BraveRam/git-timemachine/internal/git"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func layoutFixture() (*git.CommitGraph, *git.DiffData) {
	c := &git.Commit{Hash: strings.Repeat("a", 40), ShortHash: "aaaaaaa", AuthorName: "Alice Smith", AuthorEmail: "alice@example.com", AuthorDate: time.Now().Add(-10 * time.Minute), Subject: strings.Repeat("界 commit ", 15), Body: strings.Repeat("Long body line\n", 30), Node: git.GraphNode{Prefix: "● "}}
	d := &git.DiffData{Stats: []git.FileStat{{Path: strings.Repeat("long/path/", 8), Status: "M", Additions: 4}}, HighlightedLines: []git.DiffLine{{Content: strings.Repeat("界", 200)}}}
	return &git.CommitGraph{Commits: []*git.Commit{c}}, d
}

func TestInspectorWrapsLongSubject(t *testing.T) {
	d := NewDetailsViewModel(ThemeNord)
	d.SetSize(60, 14)
	subject := "style(ui): remove background color and borders from landing page cards and footer"
	d.SetState(&git.Commit{Subject: subject}, &git.DiffData{}, ThemeNord)
	view := ansi.Strip(d.View())
	if !strings.Contains(view, "footer") {
		t.Fatalf("subject ending was lost instead of wrapped:\n%s", view)
	}
	assertFits(t, "inspector", view, 60, 14)
	d.SetSize(60, 10)
	d.SetState(&git.Commit{Subject: strings.Repeat(subject, 10)}, &git.DiffData{}, ThemeNord)
	if !strings.Contains(d.View(), "…") {
		t.Fatal("message hidden by pane height needs a visible truncation indicator")
	}
}

func assertFits(t *testing.T, name, view string, width, height int) {
	t.Helper()
	if view == "" {
		return
	}
	if got := lipgloss.Width(view); got > width {
		t.Errorf("%s width %d exceeds %d", name, got, width)
	}
	if got := lipgloss.Height(view); got > height {
		t.Errorf("%s height %d exceeds %d", name, got, height)
	}
	if !utf8.ValidString(view) {
		t.Errorf("%s contains invalid UTF-8", name)
	}
}

func TestLayoutFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {160, 50}, {38, 24}, {12, 12}, {20, 12}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			g, d := layoutFixture()
			a := NewAppModel(g, ThemeCatppuccinMocha)
			model, _ := a.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			a = model.(AppModel)
			assertFits(t, "loading app", a.View(), size[0], size[1])
			empty := NewAppModel(&git.CommitGraph{}, ThemeCatppuccinMocha)
			emptyModel, _ := empty.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			assertFits(t, "empty app", emptyModel.(AppModel).View(), size[0], size[1])
			model, _ = a.Update(diffLoadedMsg{hash: g.Commits[0].Hash, data: d})
			a = model.(AppModel)
			assertFits(t, "app", a.View(), size[0], size[1])
			assertFits(t, "scrubber", a.scrubber.View(), size[0], 5)
			assertFits(t, "tree", a.treeView.View(), a.treeView.width, a.treeView.height)
			assertFits(t, "details", a.detailsView.View(), a.detailsView.width, a.detailsView.height)
			assertFits(t, "diff", a.diffView.View(), a.diffView.width, a.diffView.height)
			model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
			a = model.(AppModel)
			assertFits(t, "fullscreen", a.View(), size[0], size[1])
		})
	}
}

func TestScrubberNarrowWidths(t *testing.T) {
	g, _ := layoutFixture()
	for width := 1; width <= 80; width++ {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			s := NewScrubberModel(ThemeCatppuccinMocha)
			s.SetWidth(width)
			s.SetState(0, 1, g.Commits[0], false, "1x", ThemeCatppuccinMocha)
			assertFits(t, "scrubber", s.View(), width, 5)
		})
	}
}
