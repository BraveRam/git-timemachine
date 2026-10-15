package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	hash string
	data *git.DiffData
	err  error
}

type tickMsg time.Time

// AppModel is the top-level Bubble Tea state manager.
type AppModel struct {
	graph            *git.CommitGraph
	diffReader       *git.DiffReader
	currentDiff      *git.DiffData
	currentIndex     int
	focus            FocusArea
	isFullScreenDiff bool
	isPlaying        bool
	speedIndex       int
	themeIndex       int
	showHelp         bool

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
		graph:        graph,
		diffReader:   diffReader,
		currentIndex: initialIdx,
		focus:        FocusTree,
		speedIndex:   1, // 1x
		themeIndex:   themeIdx,
		scrubber:     NewScrubberModel(theme),
		treeView:     NewTreeViewModel(theme),
		detailsView:  NewDetailsViewModel(theme),
		diffView:     NewDiffViewModel(theme),
		search:       NewSearchModel(theme),
		keys:         DefaultKeyMap(),
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
	commit := m.currentCommit()
	if commit == nil {
		return nil
	}
	hash := commit.Hash
	reader := m.diffReader

	return func() tea.Msg {
		data, err := reader.GetCommitDiff(hash)
		return diffLoadedMsg{hash: hash, data: data, err: err}
	}
}

func (m AppModel) scheduleTickCmd() tea.Cmd {
	if !m.isPlaying {
		return nil
	}
	interval := Speeds[m.speedIndex].Interval
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayoutDimensions()
		return m, nil

	case diffLoadedMsg:
		commit := m.currentCommit()
		if commit != nil && commit.Hash == msg.hash {
			m.currentDiff = msg.data
			m.diffView.SetState(msg.data, m.currentTheme())
			m.detailsView.SetState(commit, msg.data, m.currentTheme())
		}
		return m, nil

	case tickMsg:
		if m.isPlaying {
			total := len(m.graph.Commits)
			if total > 0 {
				if m.currentIndex < total-1 {
					m.currentIndex++
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
		// 1. Search modal takes priority if active
		if m.search.IsActive() {
			selectedCommit, searchCmd := m.search.Update(msg)
			if searchCmd != nil {
				cmds = append(cmds, searchCmd)
			}
			if selectedCommit != nil {
				m.currentIndex = *selectedCommit
				m.updateChildStates()
				cmds = append(cmds, m.loadCurrentDiffCmd())
			}
			return m, tea.Batch(cmds...)
		}

		// 2. Help modal dismissal
		if m.showHelp {
			if key.Matches(msg, m.keys.Help, m.keys.Esc, m.keys.Quit) {
				m.showHelp = false
			}
			return m, nil
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
			m.treeView.SetFocus(m.focus == FocusTree)
			m.diffView.SetFocus(m.focus == FocusDiff)
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
			m.currentIndex = 0
			m.updateChildStates()
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.JumpLast):
			if len(m.graph.Commits) > 0 {
				m.currentIndex = len(m.graph.Commits) - 1
			}
			m.updateChildStates()
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Up):
			if m.focus == FocusDiff {
				var diffCmd tea.Cmd
				m.diffView, diffCmd = m.diffView.Update(msg)
				return m, diffCmd
			}
			// When tree is focused, Up steps 1 commit backward
			m.stepIndex(-1)
			cmds = append(cmds, m.loadCurrentDiffCmd())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Down):
			if m.focus == FocusDiff {
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
	m.currentIndex = newIdx
	m.updateChildStates()
}

func (m *AppModel) updateChildStates() {
	c := m.currentCommit()
	theme := m.currentTheme()
	speed := Speeds[m.speedIndex].Label
	total := len(m.graph.Commits)

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
	if mainHeight < 5 {
		mainHeight = 5
	}

	m.scrubber.SetWidth(w)

	if m.isFullScreenDiff {
		m.diffView.SetSize(w, mainHeight)
	} else {
		// Split pane: Left 45%, Right 55%
		leftWidth := w * 45 / 100
		rightWidth := w - leftWidth
		if leftWidth < 25 {
			leftWidth = 25
			rightWidth = w - leftWidth
		}

		m.treeView.SetSize(leftWidth, mainHeight)

		// Right side split: Details (top 35%), Diff (bottom 65%)
		detailsHeight := mainHeight * 35 / 100
		if detailsHeight < 7 {
			detailsHeight = 7
		}
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

	// 1. Scrubber Header
	scrubberView := m.scrubber.View()

	// 2. Main Middle Area
	var mainView string
	if m.isFullScreenDiff {
		mainView = m.diffView.View()
	} else {
		left := m.treeView.View()
		rightTop := m.detailsView.View()
		rightBottom := m.diffView.View()
		rightCol := lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBottom)
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, left, rightCol)
	}

	// 3. Footer Bar
	footerText := " [Space] Play • [←/→] Scrub • [J/K] Jump 10 • [Tab] Focus • [f] Fullscreen • [/] Search • [t] Theme • [?] Help • [q] Quit"
	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color(theme.Muted)).
		Render(footerText)

	baseLayout := lipgloss.JoinVertical(lipgloss.Left, scrubberView, mainView, footer)

	// Overlay Modals
	if m.search.IsActive() {
		searchModal := m.search.View(m.width)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, searchModal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color(theme.Background)))
	}

	if m.showHelp {
		helpModal := m.renderHelpModal()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpModal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color(theme.Background)))
	}

	return baseLayout
}

func (m AppModel) renderHelpModal() string {
	theme := m.currentTheme()

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.Cursor)).
		Render("git-timemachine — Keyboard Shortcuts")

	shortcuts := [][]string{
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
		{"/", "Search & filter commits by message, author, hash"},
		{"t", "Cycle color theme (Catppuccin, Tokyo Night, Nord, Monokai)"},
		{"?", "Toggle this help modal"},
		{"q / Ctrl+C", "Quit cleanly"},
	}

	var tableLines []string
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Accent)).Width(16)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Foreground))

	for _, s := range shortcuts {
		line := fmt.Sprintf("%s %s", keyStyle.Render(s[0]), descStyle.Render(s[1]))
		tableLines = append(tableLines, line)
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted)).
		Render("\nPress ? or Esc to close help.")

	content := fmt.Sprintf("%s\n\n%s\n%s", title, strings.Join(tableLines, "\n"), footer)

	return lipgloss.NewStyle().
		Width(60).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderFocus)).
		Background(lipgloss.Color(theme.Background)).
		Padding(1, 2).
		Render(content)
}
