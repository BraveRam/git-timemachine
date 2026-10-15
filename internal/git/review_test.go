package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reviewFixture(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	writeGitFixtureFile(t, dir, "original file.txt", "stable\nneedle(old)\n")
	writeGitFixtureFile(t, dir, "other.txt", "separate\n")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "Initial files")
	first, _ := gitOutput(dir, "rev-parse", "HEAD")
	writeGitFixtureFile(t, dir, "other.txt", "unrelated\n")
	runGitCmd(t, dir, "commit", "-am", "Unrelated change")
	runGitCmd(t, dir, "mv", "original file.txt", "new [name].txt")
	runGitCmd(t, dir, "commit", "-m", "Rename file")
	writeGitFixtureFile(t, dir, "new [name].txt", "stable\nneedle(new)\nextra\n")
	writeGitFixtureFile(t, dir, "other.txt", "another delta\n")
	runGitCmd(t, dir, "commit", "-am", "Change both files")
	last, _ := gitOutput(dir, "rev-parse", "HEAD")
	return dir, strings.TrimSpace(string(first)), strings.TrimSpace(string(last))
}

func TestReviewFileNavigatorAndComparison(t *testing.T) {
	dir, first, last := reviewFixture(t)
	dr := NewDiffReader(dir)
	selected, err := dr.GetDiff(DiffRequest{Commit: last, Path: "new [name].txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.AllStats) != 2 || len(selected.Stats) != 1 || !strings.Contains(selected.RawDiff, "+needle(new)") || strings.Contains(selected.RawDiff, "another delta") {
		t.Fatalf("file selection leaked changes: %+v", selected)
	}
	compared, err := dr.GetDiff(DiffRequest{Base: first, Commit: last, Path: "new [name].txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(compared.AllStats) < 2 || len(compared.Stats) != 1 || !strings.Contains(compared.RawDiff, "needle(new)") {
		t.Fatalf("comparison missing selected change: %+v", compared)
	}
	graph, err := ReadFilteredHistory(dir, "new [name].txt", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Commits) != 3 || graph.FilePaths[first] != "original file.txt" || graph.FilePaths[last] != "new [name].txt" {
		t.Fatalf("rename history incorrect: %+v", graph)
	}
	for _, commit := range graph.Commits {
		data, err := dr.GetDiff(DiffRequest{Commit: commit.Hash, Path: graph.FilePaths[commit.Hash]})
		if err != nil || len(data.Stats) != 1 || data.RawDiff == "" {
			t.Fatalf("historical path %s: %+v %v", commit.Hash, data, err)
		}
	}
}

func TestDiffSearchFindsAddedRemovedLiteralLines(t *testing.T) {
	dir, first, last := reviewFixture(t)
	for _, file := range []string{"", "new [name].txt"} {
		graph, err := ReadFilteredHistory(dir, file, "needle(")
		if err != nil {
			t.Fatal(err)
		}
		if len(graph.Commits) != 2 || graph.Commits[0].Hash != first || graph.Commits[1].Hash != last {
			t.Fatalf("literal added/removed search missed history for %q: %+v", file, graph.Commits)
		}
	}
	graph, err := ReadFilteredHistory(dir, "", "never appears")
	if err != nil || len(graph.Commits) != 0 {
		t.Fatalf("no-match search: %+v %v", graph, err)
	}
}

func TestWorkingTreeModesRefreshWithoutWriting(t *testing.T) {
	dir, _, _ := reviewFixture(t)
	file := "new [name].txt"
	writeGitFixtureFile(t, dir, file, "stable\nneedle(new)\nextra\nstaged\n")
	runGitCmd(t, dir, "add", "--", file)
	writeGitFixtureFile(t, dir, file, "stable\nneedle(new)\nextra\nstaged\nunstaged\n")
	before, _ := gitOutput(dir, "status", "--porcelain=v1", "-z")
	dr := NewDiffReader(dir)
	staged, err := dr.GetDiff(DiffRequest{Mode: "staged", Path: file})
	if err != nil {
		t.Fatal(err)
	}
	unstaged, err := dr.GetDiff(DiffRequest{Mode: "unstaged", Path: file})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(staged.RawDiff, "+staged") || strings.Contains(staged.RawDiff, "+unstaged") || !strings.Contains(unstaged.RawDiff, "+unstaged") {
		t.Fatal("staged/unstaged changes mixed")
	}
	after, _ := gitOutput(dir, "status", "--porcelain=v1", "-z")
	if string(before) != string(after) {
		t.Fatal("review changed index or working tree")
	}
	writeGitFixtureFile(t, dir, file, "stable\nneedle(new)\nextra\nstaged\nfresh\n")
	fresh, err := dr.GetDiff(DiffRequest{Mode: "unstaged", Path: file})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fresh.RawDiff, "+fresh") || strings.Contains(fresh.RawDiff, "+unstaged") {
		t.Fatal("working diff was incorrectly cached")
	}
}

func TestBlameFindsOriginalCommitAcrossRename(t *testing.T) {
	dir, first, last := reviewFixture(t)
	rows, err := ReadBlame(dir, last, "new [name].txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Commit != first || rows[0].Path != "original file.txt" || rows[0].OriginalLine != 1 || rows[1].Commit != last {
		t.Fatalf("incorrect blame: %+v", rows)
	}
	if _, err := ReadBlame(dir, last, "missing"); err == nil {
		t.Fatal("missing file should return a useful error")
	}
	if _, err := os.Stat(filepath.Join(dir, "new [name].txt")); err != nil {
		t.Fatal(err)
	}
}

func TestDiffLineNumbers(t *testing.T) {
	lines := parsePatch("@@ -10,2 +10,2 @@\n context\n-old\n+new\n")
	if lines[1].NewLine != 10 || lines[2].OldLine != 11 || lines[2].NewLine != 0 || lines[3].NewLine != 11 {
		t.Fatalf("bad line mapping: %+v", lines)
	}
}
