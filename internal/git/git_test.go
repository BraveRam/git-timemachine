package git

import (
	"testing"
	"time"
)

func TestBuildGraphRoutes(t *testing.T) {
	// Setup a 3-commit history:
	// c1 (root) <- c2 <- c3
	c1 := &Commit{Hash: "c1", Parents: []string{}}
	c2 := &Commit{Hash: "c2", Parents: []string{"c1"}}
	c3 := &Commit{Hash: "c3", Parents: []string{"c2"}}

	// In topological order: c3, c2, c1
	commits := []*Commit{c3, c2, c1}
	maxLanes := BuildGraphRoutes(commits)

	if maxLanes != 1 {
		t.Fatalf("expected 1 lane for linear history, got %d", maxLanes)
	}

	if c3.Node.Lane != 0 || c2.Node.Lane != 0 || c1.Node.Lane != 0 {
		t.Errorf("expected all commits in lane 0, got c3=%d, c2=%d, c1=%d",
			c3.Node.Lane, c2.Node.Lane, c1.Node.Lane)
	}
}

func TestBuildGraphRoutesMerge(t *testing.T) {
	// c1 (root)
	// c2 branching from c1
	// c3 branching from c1
	// c4 merge of c2 and c3
	c1 := &Commit{Hash: "c1", Parents: []string{}}
	c2 := &Commit{Hash: "c2", Parents: []string{"c1"}}
	c3 := &Commit{Hash: "c3", Parents: []string{"c1"}}
	c4 := &Commit{Hash: "c4", Parents: []string{"c2", "c3"}}

	commits := []*Commit{c4, c3, c2, c1}
	maxLanes := BuildGraphRoutes(commits)

	if maxLanes < 2 {
		t.Errorf("expected at least 2 lanes for merge history, got %d", maxLanes)
	}

	if c4.Node.Symbol != "⬢ " && c4.Node.Prefix == "" {
		t.Errorf("expected non-empty prefix for merge commit")
	}
}

func TestParseRefs(t *testing.T) {
	refsStr := "HEAD -> master, origin/master, tag: v1.0.0, dev"
	refs := parseRefs(refsStr)

	if len(refs) != 5 {
		t.Fatalf("expected 5 refs, got %d", len(refs))
	}

	hasHead := false
	hasTag := false
	hasDev := false
	for _, r := range refs {
		if r.Type == RefHead && r.Name == "HEAD" {
			hasHead = true
		}
		if r.Type == RefTag && r.Name == "v1.0.0" {
			hasTag = true
		}
		if r.Type == RefBranch && r.Name == "dev" {
			hasDev = true
		}
	}

	if !hasHead || !hasTag || !hasDev {
		t.Errorf("missing parsed refs: hasHead=%v, hasTag=%v, hasDev=%v", hasHead, hasTag, hasDev)
	}
}

func TestDiffCache(t *testing.T) {
	cache := NewDiffCache(2)
	cache.Put("h1", &DiffData{TotalAdditions: 10})
	cache.Put("h2", &DiffData{TotalAdditions: 20})

	d1, ok1 := cache.Get("h1")
	if !ok1 || d1.TotalAdditions != 10 {
		t.Errorf("failed to get h1")
	}

	// Insert h3 -> should evict oldest (h1)
	cache.Put("h3", &DiffData{TotalAdditions: 30})
	_, ok1After := cache.Get("h1")
	if ok1After {
		t.Errorf("expected h1 to be evicted")
	}

	d3, ok3 := cache.Get("h3")
	if !ok3 || d3.TotalAdditions != 30 {
		t.Errorf("failed to get h3")
	}
}

func TestCommitIsSigned(t *testing.T) {
	cGood := &Commit{GPGStatus: "G", AuthorDate: time.Now()}
	cNone := &Commit{GPGStatus: "N", AuthorDate: time.Now()}

	if !cGood.IsSigned() {
		t.Errorf("expected cGood to be signed")
	}
	if cNone.IsSigned() {
		t.Errorf("expected cNone to be unsigned")
	}
}

func TestBuildGraphRoutesConvergesMergedLanes(t *testing.T) {
	for _, reverseSiblings := range []bool{false, true} {
		main := &Commit{Hash: "main", Parents: []string{"base"}}
		feature := &Commit{Hash: "feature", Parents: []string{"base"}}
		commits := []*Commit{
			{Hash: "merge", Parents: []string{"main", "feature"}},
			feature, main,
			{Hash: "base", Parents: []string{"root"}},
			{Hash: "root"},
		}
		if reverseSiblings {
			commits[1], commits[2] = commits[2], commits[1]
		}
		if lanes := BuildGraphRoutes(commits); lanes != 2 {
			t.Fatalf("expected two branch lanes, got %d", lanes)
		}
		for _, c := range commits[3:] {
			if c.Node.Prefix != "● " || c.Node.Lane != 0 {
				t.Errorf("reverseSiblings=%v: %s retained a merged lane: %+v", reverseSiblings, c.Hash, c.Node)
			}
		}
	}
}
