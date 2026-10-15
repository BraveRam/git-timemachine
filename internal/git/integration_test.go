package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=TestAuthor",
		"GIT_AUTHOR_EMAIL=author@test.com",
		"GIT_COMMITTER_NAME=TestCommitter",
		"GIT_COMMITTER_EMAIL=committer@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v, output: %s", args, err, string(out))
	}
}

func TestEndToEndGitHistory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-tm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Initialize git repo
	runGitCmd(t, tempDir, "init")
	runGitCmd(t, tempDir, "config", "user.name", "TestUser")
	runGitCmd(t, tempDir, "config", "user.email", "test@test.com")

	// 2. Commit 1: Initial commit
	file1 := filepath.Join(tempDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("Hello World\nLine 2\n"), 0644)
	runGitCmd(t, tempDir, "add", "file1.txt")
	runGitCmd(t, tempDir, "commit", "-m", "Initial commit: add file1")

	// 3. Tag v1.0.0
	runGitCmd(t, tempDir, "tag", "v1.0.0")

	// 4. Branch feature-x
	runGitCmd(t, tempDir, "checkout", "-b", "feature-x")
	file2 := filepath.Join(tempDir, "file2.txt")
	_ = os.WriteFile(file2, []byte("Feature X content\n"), 0644)
	runGitCmd(t, tempDir, "add", "file2.txt")
	runGitCmd(t, tempDir, "commit", "-m", "Feature X: add file2")

	// 5. Checkout master/main, make another commit
	runGitCmd(t, tempDir, "checkout", "-")
	_ = os.WriteFile(file1, []byte("Hello World\nLine 2 Modified\nLine 3\n"), 0644)
	runGitCmd(t, tempDir, "add", "file1.txt")
	runGitCmd(t, tempDir, "commit", "-m", "Main: update file1")

	// 6. Merge feature-x into master
	runGitCmd(t, tempDir, "merge", "--no-ff", "feature-x", "-m", "Merge branch feature-x")

	// 7. Verify ReadHistory
	graph, err := ReadHistory(tempDir)
	if err != nil {
		t.Fatalf("ReadHistory failed: %v", err)
	}

	if len(graph.Commits) != 4 {
		t.Fatalf("expected 4 commits in history, got %d", len(graph.Commits))
	}

	// Oldest commit is at index 0
	if graph.Commits[0].Subject != "Initial commit: add file1" {
		t.Errorf("expected commit 0 to be initial commit, got: %s", graph.Commits[0].Subject)
	}

	// Newest commit is at index 3 (Merge commit)
	latest := graph.Commits[3]
	if latest.Subject != "Merge branch feature-x" {
		t.Errorf("expected commit 3 to be merge commit, got: %s", latest.Subject)
	}
	if len(latest.Parents) != 2 {
		t.Errorf("expected merge commit to have 2 parents, got %d", len(latest.Parents))
	}

	// 8. Verify DiffReader on commit 1
	dr := NewDiffReader(tempDir)
	diff, err := dr.GetCommitDiff(graph.Commits[0].Hash)
	if err != nil {
		t.Fatalf("GetCommitDiff failed: %v", err)
	}

	if len(diff.Stats) != 1 {
		t.Fatalf("expected 1 file stat, got %d", len(diff.Stats))
	}
	if diff.Stats[0].Path != "file1.txt" {
		t.Errorf("expected file1.txt, got %s", diff.Stats[0].Path)
	}
	if diff.Stats[0].Additions != 2 {
		t.Errorf("expected 2 additions, got %d", diff.Stats[0].Additions)
	}
}
