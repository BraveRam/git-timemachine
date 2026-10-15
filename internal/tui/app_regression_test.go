package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BraveRam/git-timemachine/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

func regressionApp() AppModel {
	commits := make([]*git.Commit, 8)
	for i := range commits {
		commits[i] = &git.Commit{Hash: fmt.Sprintf("hash%d", i), ShortHash: fmt.Sprintf("hash%d", i), Subject: fmt.Sprintf("match subject %d", i)}
	}
	return NewAppModel(&git.CommitGraph{Commits: commits}, ThemeNord)
}

func updateApp(m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(AppModel), cmd
}

func TestPlaybackIgnoresTicksFromPreviousSession(t *testing.T) {
	m := regressionApp()
	m.currentIndex = 0
	space := tea.KeyMsg{Type: tea.KeySpace}
	m, _ = updateApp(m, space)
	// Capture the real scheduled timer's message before pausing/resuming.
	oldTick := m.scheduleTickCmd()()
	m, _ = updateApp(m, space)
	m, _ = updateApp(m, space)
	var cmd tea.Cmd
	m, cmd = updateApp(m, oldTick)
	if m.currentIndex != 0 || cmd != nil {
		t.Fatalf("stale timer advanced playback or scheduled work: index=%d cmd=%v", m.currentIndex, cmd != nil)
	}
	m, cmd = updateApp(m, m.scheduleTickCmd()())
	if m.currentIndex != 1 || cmd == nil {
		t.Fatalf("current timer did not advance and continue playback: index=%d", m.currentIndex)
	}
}

func TestFullscreenArrowsScrollDiffWithoutChangingCommit(t *testing.T) {
	m := regressionApp()
	m, _ = updateApp(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	lines := make([]git.DiffLine, 100)
	for i := range lines {
		lines[i] = git.DiffLine{Content: fmt.Sprintf("line %d", i)}
	}
	m.currentDiff = &git.DiffData{HighlightedLines: lines}
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	index := m.currentIndex
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.currentIndex != index || m.diffView.viewport.YOffset == 0 {
		t.Fatal("fullscreen down did not scroll diff independently of commit selection")
	}
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.currentIndex != index || m.diffView.viewport.YOffset != 0 {
		t.Fatal("fullscreen up did not scroll back independently of commit selection")
	}
}

func TestSearchKeepsSelectedResultVisible(t *testing.T) {
	m := regressionApp()
	s := NewSearchModel(ThemeNord)
	s.Activate(m.graph.Commits)
	s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("match")})
	for i := 0; i < 7; i++ {
		s.Update(tea.KeyMsg{Type: tea.KeyDown})
		if !strings.Contains(s.View(100), fmt.Sprintf("▶ [hash%d]", i+1)) {
			t.Fatalf("selected result %d is invisible", i+1)
		}
	}
	for i := 6; i >= 0; i-- {
		s.Update(tea.KeyMsg{Type: tea.KeyUp})
		if !strings.Contains(s.View(100), fmt.Sprintf("▶ [hash%d]", i)) {
			t.Fatalf("selected result %d is invisible while moving up", i)
		}
	}
	selected, _ := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if selected == nil || *selected != 0 {
		t.Fatal("enter did not select the visible result")
	}
}
