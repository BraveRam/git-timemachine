package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BraveRam/git-timemachine/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

func workflowApp() (AppModel, *git.DiffData) {
	m := regressionApp()
	m, _ = updateApp(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	d := &git.DiffData{}
	for i := 0; i < 8; i++ {
		d.AllStats = append(d.AllStats, git.FileStat{Path: fmt.Sprintf("file %d.txt", i), Status: "M"})
	}
	d.Stats = d.AllStats
	m, _ = updateApp(m, diffLoadedMsg{hash: m.currentCommit().Hash, data: d})
	return m, d
}

func TestFilePickerAndStaleDiffResponses(t *testing.T) {
	m, data := workflowApp()
	oldKey := m.diffKey()
	m, _ = updateApp(m, resizeKey("F"))
	m, _ = updateApp(m, resizeKey("file 7"))
	if len(m.reviewModal.matches()) != 1 {
		t.Fatal("picker did not find file beyond first four")
	}
	var cmd tea.Cmd
	m, cmd = updateApp(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.reviewModal.active || m.currentRequest().Path != "file 7.txt" || cmd == nil || m.currentDiff != nil {
		t.Fatal("file picker did not request selected file")
	}
	m, _ = updateApp(m, diffLoadedMsg{key: oldKey, data: data})
	if m.currentDiff != nil {
		t.Fatal("late response replaced selected file")
	}
	m, _ = updateApp(m, diffLoadedMsg{key: m.diffKey(), data: data})
	m, _ = updateApp(m, resizeKey("n"))
	if m.currentRequest().Path != "file 0.txt" {
		t.Fatal("next file did not wrap")
	}
	m, _ = updateApp(m, diffLoadedMsg{key: m.diffKey(), data: data})
	m, _ = updateApp(m, resizeKey("p"))
	if m.currentRequest().Path != "file 7.txt" {
		t.Fatal("previous file did not wrap")
	}
}

func TestDiffSearchModalAndResultNavigation(t *testing.T) {
	m, _ := workflowApp()
	m, _ = updateApp(m, resizeKey("d"))
	m, _ = updateApp(m, resizeKey("needle"))
	m, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || m.reviewModal.active {
		t.Fatal("query was not submitted")
	}
	graph := &git.CommitGraph{Commits: m.graph.Commits, Query: "needle"}
	m, _ = updateApp(m, historyLoadedMsg{generation: m.reviewGeneration, graph: graph})
	d := &git.DiffData{}
	for i := 0; i < 80; i++ {
		text := "context"
		if i == 30 || i == 60 {
			text = "+needle"
		}
		d.HighlightedLines = append(d.HighlightedLines, git.DiffLine{Content: text})
	}
	m, _ = updateApp(m, diffLoadedMsg{key: m.diffKey(), data: d})
	if len(m.diffView.matchLines) != 2 {
		t.Fatal("search matches were not highlighted/indexed")
	}
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.diffView.viewport.YOffset == 0 {
		t.Fatal("next match did not scroll")
	}
	m, _ = updateApp(m, resizeKey("u"))
	if m.reviewGeneration < 2 {
		t.Fatal("clear filter did not request all history")
	}
}

func TestComparisonWorkingTreeAndRefreshKeys(t *testing.T) {
	m, data := workflowApp()
	base := m.currentCommit().Hash
	m, _ = updateApp(m, resizeKey("a"))
	m.stepIndex(-1)
	m, _ = updateApp(m, resizeKey("c"))
	r := m.currentRequest()
	if r.Base != base || r.Commit != m.currentCommit().Hash || m.reviewMode != "compare" {
		t.Fatal("comparison endpoints are wrong")
	}
	for _, want := range []string{"unstaged", "staged", ""} {
		m, _ = updateApp(m, resizeKey("w"))
		if m.reviewMode != want {
			t.Fatalf("wanted working-tree mode %q got %q", want, m.reviewMode)
		}
	}
	key := m.diffKey()
	m, _ = updateApp(m, resizeKey("r"))
	m, _ = updateApp(m, diffLoadedMsg{key: key, data: data})
	if m.currentDiff != nil {
		t.Fatal("pre-refresh diff replaced current data")
	}
}

func TestBlameSelectionJumpsToOriginAndLine(t *testing.T) {
	m, _ := workflowApp()
	m, _ = updateApp(m, blameLoadedMsg{generation: m.reviewGeneration, rows: []git.BlameLine{{Commit: "hash0", Path: "old.txt", OriginalLine: 40, Text: "stable"}}})
	m, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("blame selection did not request originating history")
	}
	m, _ = updateApp(m, historyLoadedMsg{generation: m.reviewGeneration, graph: m.graph, hash: "hash0", path: "old.txt", line: 40})
	if m.currentIndex != 0 || m.currentRequest().Path != "old.txt" {
		t.Fatal("blame did not select origin/path")
	}
	d := &git.DiffData{}
	for i := 1; i <= 80; i++ {
		d.HighlightedLines = append(d.HighlightedLines, git.DiffLine{Content: "line", NewLine: i})
	}
	m, _ = updateApp(m, diffLoadedMsg{key: m.diffKey(), data: d})
	if m.diffView.viewport.YOffset == 0 || m.jumpLine != 0 {
		t.Fatal("origin line was not revealed")
	}
}

func TestEditorArgumentsDoNotExecuteFileNames(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "args.txt")
	editor := filepath.Join(dir, "editor")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \""+output+"\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	path := "file with spaces; $(touch SHOULD_NOT_EXIST).txt"
	cmd := editorCommand(editor+" --wait", "ignored", path, 23)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("editor command: %v %s", err, out)
	}
	args, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(args) != "--wait\n+23\n"+path+"\n" {
		t.Fatalf("editor lost quoting: %q", args)
	}
	if _, err := os.Stat(filepath.Join(dir, "SHOULD_NOT_EXIST")); !os.IsNotExist(err) {
		t.Fatal("file name executed shell code")
	}
	code := editorCommand("code --wait", "", path, 23)
	if !strings.Contains(strings.Join(code.Args, "|"), "--goto|"+path+":23") {
		t.Fatal("VS Code line argument missing")
	}
}

func TestReviewModalsAndHelpFitTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {38, 24}} {
		m, _ := workflowApp()
		m, _ = updateApp(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m, _ = updateApp(m, resizeKey("F"))
		assertFits(t, "file picker", m.View(), size[0], size[1])
		m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyEsc})
		m, _ = updateApp(m, resizeKey("?"))
		assertFits(t, "help", m.View(), size[0], size[1])
		for i := 0; i < 50; i++ {
			m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyDown})
		}
		if !strings.Contains(m.View(), "Quit") {
			t.Fatal("help cannot scroll to the last shortcut")
		}
	}
}

func TestCancelledHistoryCannotReplaceCurrentView(t *testing.T) {
	m, _ := workflowApp()
	original := m.graph
	m, _ = updateApp(m, resizeKey("u"))
	generation := m.reviewGeneration
	m, _ = updateApp(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = updateApp(m, historyLoadedMsg{generation: generation, graph: &git.CommitGraph{Query: "stale"}})
	if m.graph != original || m.loadingReview {
		t.Fatal("cancelled history request changed current view")
	}
}

func TestEditorMissingWorkingFileIsActionable(t *testing.T) {
	m, _ := workflowApp()
	m.graph.RepoPath = t.TempDir()
	m.selectedPath = "deleted.txt"
	if cmd := m.openEditor(); cmd != nil || !strings.Contains(m.copyStatus, "absent from the working tree") {
		t.Fatal("missing working file did not produce an actionable message")
	}
}
