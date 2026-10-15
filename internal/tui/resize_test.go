package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func resizeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestPaneResizingAndReset(t *testing.T) {
	g, d := layoutFixture()
	m := NewAppModel(g, ThemeNord)
	m, _ = updateApp(m, tea.WindowSizeMsg{Width: 160, Height: 50})
	m, _ = updateApp(m, diffLoadedMsg{hash: g.Commits[0].Hash, data: d})
	initialWidth, initialHeight := m.treeView.width, m.detailsView.height
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyRight, Alt: true})
	if m.treeView.width <= initialWidth {
		t.Fatal("Alt+right did not widen tree")
	}
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyDown, Alt: true})
	if m.detailsView.height <= initialHeight {
		t.Fatal("Alt+down did not grow inspector")
	}
	m, _ = updateApp(m, resizeKey("<"))
	m, _ = updateApp(m, resizeKey("{"))
	if m.treeView.width != initialWidth || m.detailsView.height != initialHeight {
		t.Fatal("fallback keys did not shrink panes back")
	}
	m, _ = updateApp(m, resizeKey(">"))
	m, _ = updateApp(m, resizeKey("}"))
	m, _ = updateApp(m, tea.WindowSizeMsg{Width: 200, Height: 60})
	if m.treePercent != 50 || m.detailsPercent != 40 || m.treeView.width != 100 {
		t.Fatal("proportions were lost on terminal resize")
	}
	m, _ = updateApp(m, resizeKey("f"))
	m, _ = updateApp(m, resizeKey("f"))
	if m.treeView.width != 100 {
		t.Fatal("fullscreen lost the chosen split")
	}
	m, _ = updateApp(m, resizeKey("0"))
	if m.treePercent != 45 || m.detailsPercent != 35 || m.treeView.width != 90 {
		t.Fatal("reset did not restore defaults")
	}
}

func TestResizeLimitsFitTerminal(t *testing.T) {
	for _, size := range [][2]int{{12, 12}, {38, 24}, {80, 24}, {160, 50}} {
		g, d := layoutFixture()
		m := NewAppModel(g, ThemeNord)
		m, _ = updateApp(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m, _ = updateApp(m, diffLoadedMsg{hash: g.Commits[0].Hash, data: d})
		for _, key := range []string{"<", "{", ">", "}"} {
			for i := 0; i < 25; i++ {
				m, _ = updateApp(m, resizeKey(key))
				assertFits(t, "resized app", m.View(), size[0], size[1])
			}
		}
		if m.treePercent != 80 || m.detailsPercent != 80 {
			t.Fatal("resize limits not enforced")
		}
	}
}
