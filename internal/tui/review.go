package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BraveRam/git-timemachine/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

type historyLoadedMsg struct {
	generation uint64
	graph      *git.CommitGraph
	hash, path string
	line       int
	err        error
}

type blameLoadedMsg struct {
	generation uint64
	rows       []git.BlameLine
	err        error
}

type editorDoneMsg struct{ err error }

func (m AppModel) currentRequest() git.DiffRequest {
	r := git.DiffRequest{Mode: m.reviewMode, Path: m.selectedPath}
	if c := m.currentCommit(); c != nil {
		r.Commit = c.Hash
	}
	if m.reviewMode == "compare" {
		r.Mode = ""
		r.Base = m.compareBase
	}
	if r.Path == "*" {
		r.Path = ""
	} else if r.Path == "" && m.graph.FilePath != "" && m.reviewMode == "" {
		r.Path = m.graph.FilePaths[r.Commit]
		if r.Path == "" {
			r.Path = m.graph.FilePath
		}
	}
	return r
}

func (m AppModel) diffKey() string { return m.currentRequest().Key() + fmt.Sprint(m.refreshSerial) }

func (m *AppModel) pauseReview() {
	m.isPlaying = false
	m.playbackGeneration++
	m.selection = paneSelection{}
	m.updateChildStates()
}

func (m *AppModel) refreshReview() tea.Cmd {
	m.currentDiff = nil
	m.refreshSerial++
	m.updateChildStates()
	return m.loadCurrentDiffCmd()
}

func (m *AppModel) loadHistory(file, query, hash, path string, line int) tea.Cmd {
	m.pauseReview()
	m.reviewGeneration++
	generation := m.reviewGeneration
	repo := m.graph.RepoPath
	m.loadingReview = true
	m.copyStatus = "Loading history… [Esc] Cancel"
	return func() tea.Msg {
		graph, err := git.ReadFilteredHistory(repo, file, query)
		return historyLoadedMsg{generation: generation, graph: graph, hash: hash, path: path, line: line, err: err}
	}
}

func (m *AppModel) openFiles() tea.Cmd {
	m.pauseReview()
	rows := []reviewChoice{{label: "All changed files", path: "*"}}
	for _, file := range m.fileStats {
		rows = append(rows, reviewChoice{label: fmt.Sprintf("[%s] %s  +%d -%d", file.Status, file.Path, file.Additions, file.Deletions), path: file.Path})
	}
	m.reviewModal = newReviewModal("files", "Changed files — Enter to inspect one file", rows)
	return m.reviewModal.input.Focus()
}

func (m *AppModel) chooseFile(path string) tea.Cmd {
	m.selectedPath = path
	m.focus = FocusDiff
	m.pauseReview()
	m.copyStatus = ""
	return m.refreshReview()
}

func (m *AppModel) handleReviewChoice(choice reviewChoice) tea.Cmd {
	switch m.reviewModal.kind {
	case "files":
		return m.chooseFile(choice.path)
	case "query":
		return m.loadHistory(m.graph.FilePath, choice.label, "", "", 0)
	case "blame":
		return m.loadHistory("", "", choice.hash, choice.path, choice.line)
	}
	return nil
}

func (m *AppModel) workflowKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "F":
		return true, m.openFiles()
	case "n", "p":
		if len(m.fileStats) == 0 {
			m.copyStatus = "No changed files in this view"
			return true, nil
		}
		index := -1
		for i, file := range m.fileStats {
			if file.Path == m.currentRequest().Path {
				index = i
				break
			}
		}
		if msg.String() == "n" {
			index = (index + 1) % len(m.fileStats)
		} else {
			if index < 0 {
				index = 0
			}
			index = (index - 1 + len(m.fileStats)) % len(m.fileStats)
		}
		return true, m.chooseFile(m.fileStats[index].Path)
	case "H":
		path := m.currentRequest().Path
		if path == "" {
			m.copyStatus = "Choose a file with F first"
			return true, nil
		}
		if m.reviewMode != "" {
			m.copyStatus = "Return to history with Esc before following a file"
			return true, nil
		}
		return true, m.loadHistory(path, "", "", "", 0)
	case "d":
		m.pauseReview()
		m.reviewModal = newReviewModal("query", "Search text added or removed across commits (literal, case-sensitive)", nil)
		return true, m.reviewModal.input.Focus()
	case "ctrl+n", "ctrl+p":
		m.selection = paneSelection{}
		delta := 1
		if msg.String() == "ctrl+p" {
			delta = -1
		}
		m.diffView.nextMatch(delta)
		return true, nil
	case "u":
		return true, m.loadHistory("", "", "", "", 0)
	case "a":
		if c := m.currentCommit(); c != nil {
			m.compareBase = c.Hash
			m.copyStatus = "Comparison start marked: " + c.ShortHash + " — move to another commit and press c"
		}
		return true, nil
	case "c":
		if m.compareBase == "" {
			m.copyStatus = "Press a to mark the comparison start first"
			return true, nil
		}
		if m.currentCommit() == nil {
			return true, nil
		}
		m.pauseReview()
		m.reviewMode = "compare"
		m.focus = FocusDiff
		m.selectedPath = ""
		m.copyStatus = "Comparing marked commit → selected commit"
		return true, m.refreshReview()
	case "w":
		m.focus = FocusDiff
		m.pauseReview()
		m.selectedPath = ""
		switch m.reviewMode {
		case "unstaged":
			m.reviewMode = "staged"
		case "staged":
			m.reviewMode = ""
		default:
			m.reviewMode = "unstaged"
		}
		m.copyStatus = ""
		return true, m.refreshReview()
	case "r":
		return true, m.refreshReview()
	case "b":
		path := m.currentRequest().Path
		if path == "" || m.currentCommit() == nil {
			m.copyStatus = "Choose a file with F first"
			return true, nil
		}
		if m.reviewMode == "staged" || m.reviewMode == "unstaged" {
			m.copyStatus = "Blame uses a committed file; return to history with Esc"
			return true, nil
		}
		m.pauseReview()
		m.reviewGeneration++
		generation := m.reviewGeneration
		repo, hash := m.graph.RepoPath, m.currentCommit().Hash
		m.loadingReview = true
		m.copyStatus = "Loading blame… [Esc] Cancel"
		return true, func() tea.Msg {
			rows, err := git.ReadBlame(repo, hash, path)
			return blameLoadedMsg{generation: generation, rows: rows, err: err}
		}
	case "e":
		return true, m.openEditor()
	case "esc":
		if m.reviewMode != "" && !m.isFullScreenDiff {
			m.reviewMode = ""
			m.selectedPath = ""
			m.copyStatus = ""
			return true, m.refreshReview()
		}
	}
	return false, nil
}

func (m AppModel) viewLabel() string {
	label := "History"
	switch m.reviewMode {
	case "unstaged":
		label = "UNSTAGED (tracked files)"
	case "staged":
		label = "STAGED"
	case "compare":
		label = "COMPARE " + m.compareBase[:min(7, len(m.compareBase))] + " → selected"
	}
	if m.graph.FilePath != "" && m.reviewMode == "" {
		label = "File history: " + m.graph.FilePath
	}
	if m.graph.Query != "" {
		label += " • Changes containing: " + m.graph.Query
	}
	return label
}

func (m *AppModel) openEditor() tea.Cmd {
	r := m.currentRequest()
	if r.Path == "" {
		m.copyStatus = "Choose a file with F first"
		return nil
	}
	path := filepath.Join(m.graph.RepoPath, filepath.FromSlash(r.Path))
	if _, err := os.Stat(path); err != nil {
		m.copyStatus = "File is absent from the working tree; cannot open it in editor"
		return nil
	}
	line := 1
	if m.currentDiff != nil {
		start := m.diffView.viewport.YOffset
		for i := start; i < len(m.currentDiff.HighlightedLines); i++ {
			if n := m.currentDiff.HighlightedLines[i].NewLine; n > 0 {
				line = n
				break
			}
		}
	}
	if m.editorLine > 0 {
		line = m.editorLine
		m.editorLine = 0
	}
	command := editorCommand(os.Getenv("VISUAL"), os.Getenv("EDITOR"), path, line)
	m.pauseReview()
	return tea.ExecProcess(command, func(err error) tea.Msg { return editorDoneMsg{err: err} })
}

// The editor is a user-controlled shell command; paths are passed as arguments,
// never interpolated into shell code. Line numbers refer to the working file.
func editorCommand(visual, editor, path string, line int) *exec.Cmd {
	visual, editor = strings.TrimSpace(visual), strings.TrimSpace(editor)
	if visual != "" {
		editor = visual
	}
	if editor == "" {
		editor = "vi"
	}
	args := []string{fmt.Sprintf("+%d", line), path}
	name := filepath.Base(strings.Fields(editor)[0])
	if name == "code" || name == "codium" || name == "code-insiders" {
		args = []string{"--goto", fmt.Sprintf("%s:%d", path, line)}
	}
	shellArgs := append([]string{"-c", editor + " \"$@\"", "gtm-editor"}, args...)
	return exec.Command("sh", shellArgs...)
}

// ConfigureReview sets the initial noninteractive CLI review mode.
func (m *AppModel) ConfigureReview(mode, base, end string) error {
	m.reviewMode = mode
	if mode != "" {
		m.selectedPath = m.graph.FilePath
	}
	m.compareBase = base
	if base != "" {
		m.reviewMode = "compare"
		for i, c := range m.graph.Commits {
			if c.Hash == end {
				m.currentIndex = i
				return nil
			}
		}
		return fmt.Errorf("comparison end is not in loaded history; omit --file/--search filters")
	}
	return nil
}
