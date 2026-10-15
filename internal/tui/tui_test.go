package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/BraveRam/git-timemachine/internal/git"
)

func TestThemes(t *testing.T) {
	if len(AllThemes) != 4 {
		t.Fatalf("expected 4 themes, got %d", len(AllThemes))
	}

	for _, th := range AllThemes {
		if th.Name == "" {
			t.Errorf("theme missing name")
		}
		if len(th.Lanes) != 8 {
			t.Errorf("theme %s expected 8 lanes, got %d", th.Name, len(th.Lanes))
		}
		col := th.GetLaneColor(0)
		if col == "" {
			t.Errorf("theme %s returned empty lane color", th.Name)
		}
	}
}

func TestScrubberModel(t *testing.T) {
	theme := ThemeCatppuccinMocha
	s := NewScrubberModel(theme)
	s.SetWidth(80)

	// Test empty state
	emptyView := s.View()
	if !strings.Contains(emptyView, "No commits") {
		t.Errorf("expected empty message, got: %s", emptyView)
	}

	// Test populated state
	c := &git.Commit{
		Hash:       "1234567890abcdef1234567890abcdef12345678",
		ShortHash:  "1234567",
		AuthorName: "Alice Smith",
		AuthorDate: time.Now().Add(-10 * time.Minute),
		Subject:    "Initial commit",
	}

	s.SetState(0, 5, c, false, "1x", theme)
	view := s.View()
	if !strings.Contains(view, "git-timemachine") {
		t.Errorf("expected title in view, got: %s", view)
	}
	if !strings.Contains(view, "1234567") {
		t.Errorf("expected short hash in view, got: %s", view)
	}
	if !strings.Contains(view, "Alice Smith") {
		t.Errorf("expected author in view, got: %s", view)
	}
	if !strings.Contains(view, "PAUSED 1x") {
		t.Errorf("expected paused badge in view, got: %s", view)
	}
}

func TestTreeViewModel(t *testing.T) {
	theme := ThemeTokyoNight
	tv := NewTreeViewModel(theme)
	tv.SetSize(80, 20)

	c1 := &git.Commit{
		Hash:      "c111111",
		ShortHash: "c111111",
		Subject:   "Commit 1",
		Node:      git.GraphNode{Lane: 0, Prefix: "● "},
	}
	c2 := &git.Commit{
		Hash:      "c222222",
		ShortHash: "c222222",
		Subject:   "Commit 2",
		Node:      git.GraphNode{Lane: 0, Prefix: "● "},
	}

	tv.SetState([]*git.Commit{c1, c2}, 1, theme)
	view := tv.View()

	if !strings.Contains(view, "Commit Tree") {
		t.Errorf("expected Commit Tree in view, got: %s", view)
	}
	if !strings.Contains(view, "Commit 2") {
		t.Errorf("expected Commit 2 in view, got: %s", view)
	}
}

func TestDetailsViewModel(t *testing.T) {
	theme := ThemeNord
	dv := NewDetailsViewModel(theme)
	dv.SetSize(80, 15)

	c := &git.Commit{
		Hash:        "abcdef1234567890abcdef1234567890abcdef12",
		AuthorName:  "Bob Ross",
		AuthorEmail: "bob@example.com",
		AuthorDate:  time.Now(),
		Subject:     "Happy little commit",
		Body:        "Happy little commit\n\nFull details here.",
		GPGStatus:   "G",
	}
	diff := &git.DiffData{
		TotalAdditions: 42,
		TotalDeletions: 7,
		Stats: []git.FileStat{
			{Path: "canvas.go", Additions: 42, Deletions: 7, Status: "M"},
		},
	}

	dv.SetState(c, diff, theme)
	view := dv.View()

	if !strings.Contains(view, "SIGNED") {
		t.Errorf("expected SIGNED badge, got: %s", view)
	}
	if !strings.Contains(view, "Bob Ross") {
		t.Errorf("expected author name, got: %s", view)
	}
	if !strings.Contains(view, "+42") || !strings.Contains(view, "-7") {
		t.Errorf("expected additions/deletions stats, got: %s", view)
	}
}

func TestSearchModel(t *testing.T) {
	theme := ThemeMonokaiPro
	sm := NewSearchModel(theme)

	c1 := &git.Commit{Subject: "Add login page", AuthorName: "Alice", Hash: "a111"}
	c2 := &git.Commit{Subject: "Fix database bug", AuthorName: "Bob", Hash: "b222"}
	c3 := &git.Commit{Subject: "Update README", AuthorName: "Charlie", Hash: "c333"}
	commits := []*git.Commit{c1, c2, c3}

	sm.Activate(commits)
	if !sm.IsActive() {
		t.Fatalf("expected search to be active")
	}

	// Set value
	sm.input.SetValue("database")
	sm.filterCommits()

	if len(sm.matches) != 1 || sm.matches[0] != 1 {
		t.Errorf("expected match index 1, got %v", sm.matches)
	}

	sm.input.SetValue("Alice")
	sm.filterCommits()
	if len(sm.matches) != 1 || sm.matches[0] != 0 {
		t.Errorf("expected match index 0, got %v", sm.matches)
	}
}
