package tui

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestSelectionTextExcludesBordersAndStyles(t *testing.T) {
	s := paneSelection{
		pane: paneInspector, width: 12, height: 4,
		snapshot: "┌──────────┐\n│\x1b[31mhello 界\x1b[0m  │\n│next line │\n└──────────┘",
		anchor:   cellPoint{1, 1}, end: cellPoint{9, 2},
	}
	want := "hello 界\nnext line"
	if got := s.text(); got != want || !utf8.ValidString(got) {
		t.Fatalf("got %q, want %q", got, want)
	}
	s.anchor, s.end = s.end, s.anchor
	if got := s.text(); got != want {
		t.Fatalf("backward selection got %q", got)
	}
	s.anchor, s.end = cellPoint{2, 1}, cellPoint{4, 1}
	if got := s.text(); got != "ell" {
		t.Fatalf("partial line got %q", got)
	}
}

func mouseAt(x, y int, action tea.MouseAction) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: action}
}

func TestMouseSelectionStaysInsideStartingPane(t *testing.T) {
	for _, pane := range []paneID{paneTree, paneInspector, paneDiff} {
		g, d := layoutFixture()
		m := NewAppModel(g, ThemeNord)
		m, _ = updateApp(m, tea.WindowSizeMsg{Width: 160, Height: 50})
		m, _ = updateApp(m, diffLoadedMsg{hash: g.Commits[0].Hash, data: d})
		x, y := 1, 6
		if pane != paneTree {
			x += m.treeView.width
		}
		if pane == paneDiff {
			y += m.detailsView.height
		}
		m.isPlaying = true
		m, _ = updateApp(m, mouseAt(x, y, tea.MouseActionPress))
		if m.selection.pane != pane || m.isPlaying {
			t.Fatalf("pane %d: selection not started or playback not paused", pane)
		}
		original := m.selection.snapshot
		m, _ = updateApp(m, mouseAt(1000, 1000, tea.MouseActionMotion))
		m, _ = updateApp(m, mouseAt(1000, 1000, tea.MouseActionRelease))
		if m.selection.pane != pane || m.selection.dragging {
			t.Fatalf("pane %d: crossed into another pane or did not release", pane)
		}
		if strings.ContainsAny(m.selection.text(), "│┌┐└┘") {
			t.Fatal("copied selection includes pane border")
		}
		if m.selection.end.x != m.selection.width-2 || m.selection.end.y != m.selection.height-2 {
			t.Fatal("drag outside pane was not clamped to interior")
		}
		if got := ansi.Strip(m.selection.view(pane, "wrong pane")); got != ansi.Strip(original) {
			t.Fatal("highlight changed the visible pane text")
		}
		assertFits(t, "selection", m.View(), 160, 50)
		_, cmd := updateApp(m, resizeKey("y"))
		if cmd == nil {
			t.Fatal("copy shortcut did not produce clipboard command")
		}
		m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.selection.pane != paneNone {
			t.Fatal("Esc did not clear selection")
		}
	}
}

func TestSelectionRespectsFullscreenAndInvalidation(t *testing.T) {
	g, d := layoutFixture()
	m := NewAppModel(g, ThemeNord)
	m, _ = updateApp(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = updateApp(m, diffLoadedMsg{hash: g.Commits[0].Hash, data: d})
	m, _ = updateApp(m, resizeKey("f"))
	m, _ = updateApp(m, mouseAt(1, 6, tea.MouseActionPress))
	if m.selection.pane != paneDiff || m.selection.width != 120 {
		t.Fatal("fullscreen selection did not target diff")
	}
	m, _ = updateApp(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.selection.pane != paneNone {
		t.Fatal("terminal resize retained stale coordinates")
	}
	m, _ = updateApp(m, mouseAt(1, 6, tea.MouseActionPress))
	m, _ = updateApp(m, resizeKey("f"))
	if m.selection.pane != paneNone {
		t.Fatal("layout key retained stale selection")
	}
	m, _ = updateApp(m, resizeKey("/"))
	m, _ = updateApp(m, mouseAt(1, 6, tea.MouseActionPress))
	if m.selection.pane != paneNone {
		t.Fatal("mouse selected hidden pane behind search")
	}
}

func TestClipboardCommandReportsSuccessAndFailure(t *testing.T) {
	want := "selected text only"
	msg := copySelectionCmd(want, func(got string) error {
		if got != want {
			t.Fatalf("clipboard received %q", got)
		}
		return nil
	})().(copiedMsg)
	m := regressionApp()
	m, _ = updateApp(m, msg)
	if m.copyStatus != "Copied selection" {
		t.Fatal("success not reported")
	}
	msg = copySelectionCmd(want, func(string) error { return errors.New("clipboard unavailable") })().(copiedMsg)
	m, _ = updateApp(m, msg)
	if !strings.Contains(m.copyStatus, "Copy failed") {
		t.Fatal("clipboard failure was not reported")
	}
}
