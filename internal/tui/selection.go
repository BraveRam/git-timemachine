package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type paneID int

const (
	paneNone paneID = iota
	paneTree
	paneInspector
	paneDiff
)

type cellPoint struct{ x, y int }

type paneSelection struct {
	pane          paneID
	origin        cellPoint
	width, height int
	anchor, end   cellPoint
	dragging      bool
	snapshot      string
}

type copiedMsg struct{ err error }

func copySelectionCmd(text string, write func(string) error) tea.Cmd {
	return func() tea.Msg { return copiedMsg{err: write(text)} }
}

func (s paneSelection) bounds(row int) (int, int, bool) {
	a, b := s.anchor, s.end
	if a.y > b.y || (a.y == b.y && a.x > b.x) {
		a, b = b, a
	}
	if row < a.y || row > b.y {
		return 0, 0, false
	}
	left, right := 1, s.width-1 // exclude the border
	if row == a.y {
		left = a.x
	}
	if row == b.y {
		right = b.x + 1
	}
	return left, right, true
}

func (s *paneSelection) move(x, y int) {
	s.end = cellPoint{
		x: max(1, min(x-s.origin.x, s.width-2)),
		y: max(1, min(y-s.origin.y, s.height-2)),
	}
}

func (s paneSelection) text() string {
	if s.pane == paneNone {
		return ""
	}
	var selected []string
	for row, line := range strings.Split(ansi.Strip(s.snapshot), "\n") {
		if left, right, ok := s.bounds(row); ok {
			selected = append(selected, strings.TrimRight(ansi.Cut(line, left, right), " "))
		}
	}
	return strings.Join(selected, "\n")
}

func (s paneSelection) view(pane paneID, fallback string) string {
	if s.pane != pane {
		return fallback
	}
	lines := strings.Split(s.snapshot, "\n")
	style := lipgloss.NewStyle().Reverse(true)
	for row, line := range lines {
		if left, right, ok := s.bounds(row); ok {
			lines[row] = ansi.Cut(line, 0, left) +
				style.Render(ansi.Strip(ansi.Cut(line, left, right))) +
				ansi.Cut(line, right, s.width)
		}
	}
	return strings.Join(lines, "\n")
}

func (m AppModel) paneAt(x, y int) paneSelection {
	if m.width < 12 || m.height < 12 {
		return paneSelection{}
	}
	panes := []paneSelection{{pane: paneDiff, origin: cellPoint{0, 5}, width: m.diffView.width, height: m.diffView.height, snapshot: m.diffView.View()}}
	if !m.isFullScreenDiff {
		panes = []paneSelection{
			{pane: paneTree, origin: cellPoint{0, 5}, width: m.treeView.width, height: m.treeView.height, snapshot: m.treeView.View()},
			{pane: paneInspector, origin: cellPoint{m.treeView.width, 5}, width: m.detailsView.width, height: m.detailsView.height, snapshot: m.detailsView.View()},
			{pane: paneDiff, origin: cellPoint{m.treeView.width, 5 + m.detailsView.height}, width: m.diffView.width, height: m.diffView.height, snapshot: m.diffView.View()},
		}
	}
	for _, pane := range panes {
		if x > pane.origin.x && x < pane.origin.x+pane.width-1 && y > pane.origin.y && y < pane.origin.y+pane.height-1 {
			return pane
		}
	}
	return paneSelection{}
}

func (m AppModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.search.IsActive() || m.showHelp || m.reviewModal.active || m.loadingReview {
		return m, nil
	}
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		m.selection = m.paneAt(msg.X, msg.Y)
		m.copyStatus = ""
		if m.selection.pane != paneNone {
			m.selection.move(msg.X, msg.Y)
			m.selection.anchor = m.selection.end
			m.selection.dragging = true
			// Freeze playback so the text under the pointer stays still.
			m.isPlaying = false
			m.playbackGeneration++
			m.updateChildStates()
		}
		return m, nil
	}
	if m.selection.dragging && (msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionRelease) {
		m.selection.move(msg.X, msg.Y)
		if msg.Action == tea.MouseActionRelease {
			m.selection.dragging = false
		}
		return m, nil
	}
	if tea.MouseEvent(msg).IsWheel() {
		pane := m.paneAt(msg.X, msg.Y)
		if pane.pane == paneDiff {
			m.selection = paneSelection{}
			var cmd tea.Cmd
			m.diffView, cmd = m.diffView.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m AppModel) copySelection() tea.Cmd {
	if text := m.selection.text(); text != "" {
		return copySelectionCmd(text, clipboard.WriteAll)
	}
	return nil
}
