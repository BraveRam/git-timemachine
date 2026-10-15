package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGitFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiffRetainsLongLinesAndLaterFiles(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	longLine := strings.Repeat("x", 70000)
	writeGitFixtureFile(t, dir, "a.js", longLine+"\n")
	writeGitFixtureFile(t, dir, "z.txt", "later file content\n")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "Add long line and later file")
	diff, err := NewDiffReader(dir).GetCommitDiff("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	foundLong, foundLater := false, false
	var reconstructed strings.Builder
	for _, line := range diff.HighlightedLines {
		reconstructed.WriteString(line.Content)
		reconstructed.WriteByte('\n')
		if line.Content == "+"+longLine {
			foundLong = line.Type == DiffLineAddition
		}
		if line.Content == "+later file content" {
			foundLater = line.Type == DiffLineAddition
		}
	}
	if !foundLong || !foundLater {
		t.Fatalf("missing patch additions: long=%v later=%v", foundLong, foundLater)
	}
	if reconstructed.String() != diff.RawDiff {
		t.Fatal("parsed diff does not preserve the complete raw patch")
	}
}

func TestDiffStatusesPreserveSpecialPaths(t *testing.T) {
	for _, name := range []string{"my file.txt", "tab\tfile.txt", "line\nfile.txt", "非ASCII.txt"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			runGitCmd(t, dir, "init")
			writeGitFixtureFile(t, dir, name, "one\n")
			runGitCmd(t, dir, "add", ".")
			runGitCmd(t, dir, "commit", "-m", "Base")
			writeGitFixtureFile(t, dir, name, "one\ntwo\n")
			runGitCmd(t, dir, "commit", "-am", "Append")
			diff, err := NewDiffReader(dir).GetCommitDiff("HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if len(diff.Stats) != 1 {
				t.Fatalf("expected one file, got %+v", diff.Stats)
			}
			stat := diff.Stats[0]
			if stat.Path != name || stat.Status != "M" || stat.Additions != 1 || stat.Deletions != 0 {
				t.Fatalf("incorrect modified file stat: %+v", stat)
			}
		})
	}
}

func TestDiffRenamePreservesDestinationPath(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	writeGitFixtureFile(t, dir, "old file.txt", "unchanged content\n")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "Base")
	runGitCmd(t, dir, "mv", "old file.txt", "new\tfile.txt")
	runGitCmd(t, dir, "commit", "-m", "Rename")
	diff, err := NewDiffReader(dir).GetCommitDiff("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Stats) != 1 || diff.Stats[0].Path != "new\tfile.txt" || diff.Stats[0].Status != "R" {
		t.Fatalf("incorrect rename stats: %+v", diff.Stats)
	}
}
