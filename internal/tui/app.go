package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type FocusArea int

const (
	FocusTree FocusArea = iota
	FocusDiff
)

type PlaybackSpeed struct {
	Label    string
	Interval time.Duration
}

var Speeds = []PlaybackSpeed{
	{Label: "0.5x", Interval: 500 * time.Millisecond},
	{Label: "1x", Interval: 200 * time.Millisecond},
	{Label: "2x", Interval: 100 * time.Millisecond},
	{Label: "5x", Interval: 40 * time.Millisecond},
}

type diffLoadedMsg struct {
	key  string
	hash string
	data *git.DiffData
	err  error
}

type tickMsg struct {
	generation uint64
}

// AppModel is the top-level Bubble Tea state manager.
type AppModel struct {
	graph              *git.CommitGraph
	diffReader         *git.DiffReader
	currentDiff        *git.DiffData
	currentIndex       int
	focus              FocusArea
	isFullScreenDiff   bool
	isPlaying          bool
	playbackGeneration uint64
	treePercent        int
	detailsPercent     int
	speedIndex         int
	themeIndex         int
	showHelp           bool
	helpOffset         int
	selection          paneSelection
	copyStatus         string
	reviewModal        reviewModal
	reviewMode         string
	compareBase        string
	selectedPath       string
	fileStats          []git.FileStat
	loadedDiffKey      string
	refreshSerial      uint64
	reviewGeneration   uint64
	loadingReview      bool
	jumpLine           int
	editorLine         int

	scrubber    ScrubberModel
	treeView    TreeViewModel
	detailsView DetailsViewModel
	diffView    DiffViewModel
	search      SearchModel
	keys        KeyMap

	width  int
	height int
}

// NewAppModel initializes the top-level application.
func NewAppModel(graph *git.CommitGraph, initialTheme Theme) AppModel {
	theme := initialTheme
	themeIdx := 0
	for i, t := range AllThemes {
		if t.Name == initialTheme.Name {
			themeIdx = i
			break
		}
	}

	diffReader := git.NewDiffReader(graph.RepoPath)
	totalCommits := len(graph.Commits)
	initialIdx := 0
	if totalCommits > 0 {
		initialIdx = totalCommits - 1 // Default to latest commit (HEAD)
	}

	app := AppModel{
		graph:          graph,
		diffReader:     diffReader,
		currentIndex:   initialIdx,
		focus:          FocusTree,
		treePercent:    45,
		detailsPercent: 35,
		speedIndex:     1, // 1x
		themeIndex:     themeIdx,
		scrubber:       NewScrubberModel(theme),
		treeView:       NewTreeViewModel(theme),
		detailsView:    NewDetailsViewModel(theme),
		diffView:       NewDiffViewModel(theme),
		search:         NewSearchModel(theme),
		keys:           DefaultKeyMap(),
	}

	if len(graph.Commits) == 0 {
		app.currentDiff = &git.DiffData{}
	}
	return app
}

func (m AppModel) currentTheme() Theme {
	return AllThemes[m.themeIndex]
}

func (m AppModel) currentCommit() *git.Commit {
	if len(m.graph.Commits) == 0 || m.currentIndex < 0 || m.currentIndex >= len(m.graph.Commits) {
		return nil
	}
	return m.graph.Commits[m.currentIndex]
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.loadCurrentDiffCmd(),
	)
}

func (m AppModel) loadCurrentDiffCmd() tea.Cmd {
	request := m.currentRequest()
	if request.Commit == "" && request.Mode == "" {
		return nil
	}
	key := m.diffKey()
	reader := m.diffReader
	return func() tea.Msg {
		data, err := reader.GetDiff(request)
		return diffLoadedMsg{hash: request.Commit, key: key, data: data, err: err}
	}

}

func (m AppModel) scheduleTickCmd() tea.Cmd {
	if !m.isPlaying {
		return nil
	}
	interval := Speeds[m.speedIndex].Interval
	generation := m.playbackGeneration
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg{generation: generation}
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		return m.handleMouse(msg)

	case copiedMsg:
		if msg.err != nil {
			m.copyStatus = "Copy failed: " + msg.err.Error()
		} else {
			m.copyStatus = "Copied selection"
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.selection = paneSelection{}
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayoutDimensions()
		return m, nil

	case historyLoadedMsg:
		if msg.generation != m.reviewGeneration {
			return m, nil
		}
		m.loadingReview = false
		if msg.err != nil {
			m.copyStatus = msg.err.Error()
			return m, nil
		}
		m.graph = msg.graph
		m.currentIndex = max(0, len(m.graph.Commits)-1)
		m.reviewMode = ""
		m.selectedPath = msg.path
		m.currentDiff = nil
		m.fileStats = nil
		m.loadedDiffKey = ""
		m.jumpLine = msg.line
		if msg.line > 0 {
			m.focus = FocusDiff
		}
		if msg.hash != "" {
			for i, c := range m.graph.Commits {
				if c.Hash == msg.hash {
					m.currentIndex = i
					break
				}
			}
		}
		m.copyStatus = ""
		if len(m.graph.Commits) == 0 {
			m.currentDiff = &git.DiffData{}
			m.copyStatus = "No commits matched. Press u for all history"
		}
		m.updateChildStates()
		return m, m.loadCurrentDiffCmd()

	case blameLoadedMsg:
		if msg.generation != m.reviewGeneration {
			return m, nil
		}
		m.loadingReview = false
		if msg.err != nil {
			m.copyStatus = msg.err.Error()
			return m, nil
		}
		rows := make([]reviewChoice, 0, len(msg.rows))
		for _, row := range msg.rows {
			rows = append(rows, reviewChoice{label: fmt.Sprintf("%5d %s %s • %s", row.Line, row.Commit[:min(7, len(row.Commit))], row.Text, row.Author), path: row.Path, hash: row.Commit, line: row.OriginalLine})
		}
		m.copyStatus = ""
		m.reviewModal = newReviewModal("blame", "Blame — select a line to jump to its originating commit", rows)
		return m, m.reviewModal.input.Focus()

	case editorDoneMsg:
		if msg.err != nil {
			m.copyStatus = "Editor: " + msg.err.Error()
		} else {
			m.copyStatus = "Returned from editor"
		}
		return m, nil

	case diffLoadedMsg:
		commit := m.currentCommit()
		matches := msg.key != "" && msg.key == m.diffKey()
		if msg.key == "" && commit != nil {
			matches = commit.Hash == msg.hash
		}
		if matches {
			if msg.err != nil {
				m.copyStatus = "Diff error: " + msg.err.Error()
				return m, nil
			}
			m.currentDiff = msg.data
			m.loadedDiffKey = m.diffKey()
			m.fileStats = nil
			if msg.data != nil {
				m.fileStats = msg.data.AllStats
				if m.fileStats == nil {
					m.fileStats = msg.data.Stats
				}
			}
			m.updateChildStates()
			if m.jumpLine > 0 {
				m.diffView.gotoLine(m.jumpLine)
				m.jumpLine = 0
			}
		}
		return m, nil

	case tickMsg:
		if m.isPlaying && msg.generation == m.playbackGeneration {
			total := len(m.graph.Commits)
			if total > 0 {
				if m.currentIndex < total-1 {
					m.currentIndex++
					m.selectedPath = ""
					m.updateChildStates()
					cmds = append(cmds, m.loadCurrentDiffCmd())
					cmds = append(cmds, m.scheduleTickCmd())
				} else {
					// Reached latest commit, stop playing
					m.isPlaying = false
					m.updateChildStates()
				}
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.loadingReview {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc":
				m.reviewGeneration++
				m.loadingReview = false
				m.copyStatus = "Cancelled"
			}
			return m, nil
		}
		if m.reviewModal.active {
			choice, cmd := m.reviewModal.update(msg)
			if choice != nil {
				cmd = m.handleReviewChoice(*choice)
			}
			return m, cmd
		}
		// 1. Search modal takes priority if active
		if m.search.IsActive() {
			selectedCommit, searchCmd := m.search.Update(msg)
			if searchCmd != nil {
				cmds = append(cmds, searchCmd)
			}
			if selectedCommit != nil {
				m.currentIndex = *selectedCommit
				m.selectedPath = ""
				m.updateChildStates()
				cmds = append(cmds, m.loadCurrentDiffCmd())
			}
			return m, tea.Batch(cmds...)
		}

		// 2. Help modal dismissal
		if m.showHelp {
			switch msg.String() {
			case "up", "k":
				m.helpOffset = max(0, m.helpOffset-1)
			case "down", "j":
				m.helpOffset++
			case "pgdown":
				m.helpOffset += 10
			case "pgup":
				m.helpOffset = max(0, m.helpOffset-10)
			}
			if key.Matches(msg, m.keys.Help, m.keys.Esc, m.keys.Quit) {
				m.showHelp = false
			}
			return m, nil
		}

		// Copy only the selected pane text; Esc clears the highlight.
		if m.selection.pane != paneNone {
			if msg.String() == "y" || msg.String() == "ctrl+y" {
				return m, m.copySelection()
			}
			if msg.String() == "e" && m.selection.pane == paneDiff && m.currentDiff != nil {
				start := m.diffView.viewport.YOffset + max(0, m.selection.anchor.y-3)
				for i := start; i < len(m.currentDiff.HighlightedLines); i++ {
					if line := m.currentDiff.HighlightedLines[i].NewLine; line > 0 {
						m.editorLine = line
						break
					}
				}
			}
			m.selection = paneSelection{}
			m.copyStatus = ""
			if key.Matches(msg, m.keys.Esc) {
				return m, nil
			}
		}

		if handled, cmd := m.workflowKey(msg); handled {
			return m, cmd
		}

		// 3. Global shortcuts
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Theme):
			m.themeIndex = (m.themeIndex + 1) % len(AllThemes)
			m.updateChildStates()
			return m, nil

		case key.Matches(msg, m.keys.Search):
			cmd := m.search.Activate(m.graph.Commits)
			return m, cmd

		case key.Matches(msg, m.keys.TogglePlay):
			m.playbackGeneration++ // Invalidate timers from the previous playback session.
			m.isPlaying = !m.isPlaying
			if m.isPlaying {
				// If at the end, restart from beginning for a full movie experience
				if m.currentIndex >= len(m.graph.Commits)-1 {
					m.currentIndex = 0
					m.updateChildStates()
					cmds = append(cmds, m.loadCurrentDiffCmd())
				}
				cmds = append(cmds, m.scheduleTickCmd())
			}
			m.updateChildStates()
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.SpeedDown):
			if m.speedIndex > 0 {
				m.speedIndex--
				m.updateChildStates()
			}
			return m, nil

		case key.Matches(msg, m.keys.SpeedUp):
			if m.speedIndex < len(Speeds)-1 {
				m.speedIndex++
				m.updateChildStates()
			}
			return m, nil

		case key.Matches(msg, m.keys.TreeNarrower):
			m.treePercent = max(20, m.treePercent-5)
			m.updateLayoutDimensions()
			return m, nil
		case key.Matches(msg, m.keys.TreeWider):
			m.treePercent = min(80, m.treePercent+5)
			m.updateLayoutDimensions()
			return m, nil
		case key.Matches(msg, m.keys.InspectorShorter):
			m.detailsPercent = max(20, m.detailsPercent-5)
			m.updateLayoutDimensions()
			return m, nil
		case key.Matches(msg, m.keys.InspectorTaller):
			m.detailsPercent = min(80, m.detailsPercent+5)
			m.updateLayoutDimensions()
			return m, nil
		case key.Matches(msg, m.keys.ResetLayout):
			m.treePercent, m.detailsPercent = 45, 35
			m.updateLayoutDimensions()
			return m, nil

		case key.Matches(msg, m.keys.ToggleDiffFull):
			m.isFullScreenDiff = !m.isFullScreenDiff
			m.diffView.SetFullScreen(m.isFullScreenDiff)
			m.updateLayoutDimensions()
			return m, nil

		case key.Matches(msg, m.keys.Esc):
			if m.isFullScreenDiff {
				m.isFullScreenDiff = false
				m.diffView.SetFullScreen(false)
				m.updateLayoutDimensions()
				return m, nil
			}

		case key.Matches(msg, m.keys.Tab):
			if m.focus == FocusTree {
				m.focus = FocusDiff
			} else {
				m.focus = FocusTree
			}
			m.treeView.SetFocus(m.focus == FocusTree && !m.isFullScreenDiff)
			m.diffView.SetFocus(m.focus == FocusDiff || m.isFullScreenDiff)
			return m, nil

		// Scrubbing navigation
		case key.Matches(msg, m.keys.ScrubLeft):
			m.stepIndex(-1)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.ScrubRight):
			m.stepIndex(1)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.JumpBack10):
			m.stepIndex(-10)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.JumpForward10):
			m.stepIndex(10)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.JumpFirst):
			m.selectedPath = ""
			m.currentIndex = 0
			m.updateChildStates()
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.JumpLast):
			m.selectedPath = ""
			if len(m.graph.Commits) > 0 {
				m.currentIndex = len(m.graph.Commits) - 1
			}
			m.updateChildStates()
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Up):
			if m.focus == FocusDiff || m.isFullScreenDiff {
				var diffCmd tea.Cmd
				m.diffView, diffCmd = m.diffView.Update(msg)
				return m, diffCmd
			}
			// When tree is focused, Up steps 1 commit backward
			m.stepIndex(-1)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Down):
			if m.focus == FocusDiff || m.isFullScreenDiff {
				var diffCmd tea.Cmd
				m.diffView, diffCmd = m.diffView.Update(msg)
				return m, diffCmd
			}
			// When tree is focused, Down steps 1 commit forward
			m.stepIndex(1)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)
		}
	}

	// Forward other messages to diff viewport
	if m.focus == FocusDiff || m.isFullScreenDiff {
		var diffCmd tea.Cmd
		m.diffView, diffCmd = m.diffView.Update(msg)
		cmds = append(cmds, diffCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) stepIndex(delta int) {
	total := len(m.graph.Commits)
	if total == 0 {
		return
	}
	newIdx := m.currentIndex + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= total {
		newIdx = total - 1
	}
	if m.currentIndex != newIdx {
		m.selectedPath = ""
	}
	m.currentIndex = newIdx
	m.updateChildStates()
}

func (m *AppModel) updateChildStates() {
	if m.loadedDiffKey != "" && m.loadedDiffKey != m.diffKey() {
		m.currentDiff = nil
		m.fileStats = nil
	}
	m.diffView.label = m.viewLabel()
	m.detailsView.workingMode = ""
	if m.reviewMode == "staged" || m.reviewMode == "unstaged" {
		m.detailsView.workingMode = m.viewLabel()
	}
	if path := m.currentRequest().Path; path != "" {
		m.diffView.label += " • " + path
	}
	query := m.graph.Query
	if m.reviewMode != "" {
		query = ""
	}
	m.diffView.SetQuery(query)
	c := m.currentCommit()
	theme := m.currentTheme()
	speed := Speeds[m.speedIndex].Label
	total := len(m.graph.Commits)

	m.search.SetTheme(theme)
	m.scrubber.SetState(m.currentIndex, total, c, m.isPlaying, speed, theme)
	m.treeView.SetState(m.graph.Commits, m.currentIndex, theme)
	m.detailsView.SetState(c, m.currentDiff, theme)
	m.diffView.SetState(m.currentDiff, theme)
	m.treeView.SetFocus(m.focus == FocusTree && !m.isFullScreenDiff)
	m.diffView.SetFocus(m.focus == FocusDiff || m.isFullScreenDiff)
}

func (m *AppModel) updateLayoutDimensions() {
	w := m.width
	h := m.height
	if w <= 0 || h <= 0 {
		return
	}

	scrubberHeight := 5
	footerHeight := 1
	mainHeight := h - scrubberHeight - footerHeight
	mainHeight = max(0, mainHeight)

	m.scrubber.SetWidth(w)

	if m.isFullScreenDiff {
		m.diffView.SetSize(w, mainHeight)
	} else {
		// Keep the chosen proportions across terminal resizes, with usable minima.
		minPaneWidth := min(25, w/2)
		leftWidth := max(minPaneWidth, min(w*m.treePercent/100, w-minPaneWidth))
		rightWidth := w - leftWidth
		m.treeView.SetSize(leftWidth, mainHeight)

		minDetailsHeight := min(7, mainHeight/2)
		minDiffHeight := min(5, mainHeight/2)
		detailsHeight := max(minDetailsHeight, min(mainHeight*m.detailsPercent/100, mainHeight-minDiffHeight))
		diffHeight := mainHeight - detailsHeight

		m.detailsView.SetSize(rightWidth, detailsHeight)
		m.diffView.SetSize(rightWidth, diffHeight)
	}

	m.updateChildStates()
}

func (m AppModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing git-timemachine..."
	}

	theme := m.currentTheme()
	if m.width < 12 || m.height < 12 {
		return ansi.Truncate("Terminal too small; resize to continue.", m.width, "")
	}

	// 1. Scrubber Header
	scrubberView := m.scrubber.View()

	// 2. Main Middle Area
	var mainView string
	if m.isFullScreenDiff {
		mainView = m.selection.view(paneDiff, m.diffView.View())
	} else {
		left := m.selection.view(paneTree, m.treeView.View())
		rightTop := m.selection.view(paneInspector, m.detailsView.View())
		rightBottom := m.selection.view(paneDiff, m.diffView.View())
		rightCol := lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBottom)
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, left, rightCol)
	}

	// 3. Footer Bar
	footerText := " [F] Files • [n/p] File • [H] History • [d] Diff search • [a/c] Compare • [w] Working tree • [?] Help • [q] Quit"
	if m.selection.pane != paneNone {
		footerText = " Drag within pane • [y] Copy selection • [Esc] Clear • [Alt+arrows] Resize"
	}
	if m.copyStatus != "" {
		footerText = " " + m.copyStatus + " • [?] Help"
	}
	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color(theme.Muted)).
		Render(ansi.Truncate(footerText, m.width, ""))

	baseLayout := lipgloss.JoinVertical(lipgloss.Left, scrubberView, mainView, footer)

	// Overlay Modals
	if m.reviewModal.active {
		return m.themedScreen(m.reviewModal.view(m.width, m.height, theme))
	}
	if m.search.IsActive() {
		searchModal := m.search.View(m.width)
		return m.themedScreen(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, searchModal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color(theme.Background))))
	}

	if m.showHelp {
		helpModal := m.renderHelpModal()
		return m.themedScreen(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpModal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color(theme.Background))))
	}

	return m.themedScreen(baseLayout)
}

func (m AppModel) renderHelpModal() string {
	theme := m.currentTheme()

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.Cursor)).
		Render("git-timemachine — Keyboard Shortcuts")

	shortcuts := [][]string{
		{"F / n / p", "Choose file / next file / previous file"},
		{"H / u", "Follow selected file / return to all history"},
		{"d / Ctrl+n,p", "Find changed text / next, previous hit"},
		{"a / c", "Mark comparison start / compare to current"},
		{"w / r", "Unstaged → staged → history / refresh"},
		{"b / e", "Blame selected file / open working file in editor"},
		{"← / h", "Scrub backward 1 commit"},
		{"→ / l", "Scrub forward 1 commit"},
		{"Shift+← / J", "Jump backward 10 commits"},
		{"Shift+→ / K", "Jump forward 10 commits"},
		{"Home / g", "Jump to initial commit (#1)"},
		{"End / G", "Jump to latest commit (HEAD)"},
		{"Space", "Play / Pause timelapse auto-scrub"},
		{"[ / -", "Decrease playback speed (0.5x, 1x, 2x, 5x)"},
		{"] / +", "Increase playback speed"},
		{"Tab", "Toggle focus between Tree and Diff view"},
		{"f / Enter", "Toggle fullscreen live diff view"},
		{"Alt+←/→ or </>", "Resize tree / right column"},
		{"Alt+↑/↓ or {/}", "Resize inspector / diff"},
		{"0", "Reset pane sizes"},
		{"Mouse drag", "Select text within one pane"},
		{"y / Ctrl+y", "Copy selection; Esc clears it"},
		{"/", "Search & filter commits by message, author, hash"},
		{"t", "Cycle through 14 color themes"},
		{"?", "Toggle this help modal"},
		{"q / Ctrl+C", "Quit cleanly"},
	}

	w, h := min(92, max(1, m.width-4)), min(len(shortcuts)+4, max(1, m.height-2))
	visible := max(1, h-4)
	start := min(m.helpOffset, max(0, len(shortcuts)-visible))
	lines := []string{title}
	for _, row := range shortcuts[start:min(len(shortcuts), start+visible)] {
		lines = append(lines, fmt.Sprintf("%-18s %s", row[0], row[1]))
	}
	lines = append(lines, "↑/↓ Scroll • ? / Esc Close")
	return renderPane(theme, true, w, h, strings.Join(lines, "\n"))

}

// Paint the entire canvas so light themes work in dark terminals and vice versa.
func (m AppModel) themedScreen(content string) string {
	theme := m.currentTheme()
	base := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Foreground)).Background(lipgloss.Color(theme.Background))
	screen := base.Width(m.width).Height(m.height).Render(content)
	// Child styles reset SGR attributes after colored text. Restore the canvas
	// colors there too, otherwise light text surfaces fall back to a dark terminal.
	colors, _, _ := strings.Cut(base.Render(" "), " ")
	if colors != "" {
		screen = strings.NewReplacer("\x1b[0m", "\x1b[0m"+colors, "\x1b[m", "\x1b[m"+colors).Replace(screen)
		screen += "\x1b[0m"
	}
	return screen
}
