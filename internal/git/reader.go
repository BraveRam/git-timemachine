package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	fieldSepByte  = '\x1f' // ASCII Unit Separator
	recordSepByte = '\x1e' // ASCII Record Separator
)

// ReadHistory streams commits and refs from the git repository at repoPath.
func ReadHistory(repoPath string) (*CommitGraph, error) {
	// Use git's internal %x1f and %x1e escapes so the command line argument contains no null bytes.
	format := "--pretty=format:%H%x1f%h%x1f%P%x1f%an%x1f%ae%x1f%at%x1f%s%x1f%B%x1f%G?%x1f%D%x1e"

	cmd := exec.Command("git", "log", "--all", "--topo-order", format)
	cmd.Dir = repoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open git log pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start git log: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	// Custom split function for 0x1e record separator
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, recordSepByte); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	// Allocate buffer for scanner to handle large commit bodies
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	var topologicalCommits []*Commit
	commitMap := make(map[string]*Commit)
	var allTags []Ref
	var allBranches []Ref

	for scanner.Scan() {
		raw := scanner.Text()
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		fields := strings.Split(raw, string(fieldSepByte))
		if len(fields) < 10 {
			continue
		}

		hash := fields[0]
		shortHash := fields[1]
		parentsStr := fields[2]
		authorName := fields[3]
		authorEmail := fields[4]
		epochStr := fields[5]
		subject := fields[6]
		body := fields[7]
		gpgStatus := fields[8]
		refsStr := fields[9]

		var parents []string
		if parentsStr != "" {
			parents = strings.Fields(parentsStr)
		}

		epoch, _ := strconv.ParseInt(epochStr, 10, 64)
		authorDate := time.Unix(epoch, 0)

		refs := parseRefs(refsStr)
		for _, r := range refs {
			if r.Type == RefTag {
				allTags = append(allTags, r)
			} else if r.Type == RefBranch || r.Type == RefHead {
				allBranches = append(allBranches, r)
			}
		}

		commit := &Commit{
			Hash:        hash,
			ShortHash:   shortHash,
			Parents:     parents,
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
			AuthorDate:  authorDate,
			Subject:     subject,
			Body:        body,
			GPGStatus:   gpgStatus,
			Refs:        refs,
		}

		topologicalCommits = append(topologicalCommits, commit)
		commitMap[hash] = commit
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return nil, fmt.Errorf("git log scanner error: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		// If repository has no commits yet
		if len(topologicalCommits) == 0 {
			return &CommitGraph{
				RepoPath:  repoPath,
				Commits:   []*Commit{},
				CommitMap: commitMap,
				Tags:      []Ref{},
				Branches:  []Ref{},
				MaxLanes:  1,
			}, nil
		}
	}

	// Compute Unicode branch lanes based on topological order
	maxLanes := BuildGraphRoutes(topologicalCommits)

	// Build chronological list: index 0 is oldest commit (initial commit), index len-1 is latest commit (HEAD)
	n := len(topologicalCommits)
	chronologicalCommits := make([]*Commit, n)
	for i := 0; i < n; i++ {
		c := topologicalCommits[n-1-i]
		c.Index = i
		chronologicalCommits[i] = c
	}

	return &CommitGraph{
		RepoPath:  repoPath,
		Commits:   chronologicalCommits,
		CommitMap: commitMap,
		Tags:      allTags,
		Branches:  allBranches,
		MaxLanes:  maxLanes,
	}, nil
}

// parseRefs parses decorations like "HEAD -> master, origin/master, tag: v1.0.0"
func parseRefs(refsStr string) []Ref {
	if refsStr == "" {
		return nil
	}

	var results []Ref
	parts := strings.Split(refsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		if strings.HasPrefix(p, "HEAD -> ") {
			branchName := strings.TrimPrefix(p, "HEAD -> ")
			results = append(results, Ref{Name: "HEAD", Type: RefHead})
			results = append(results, Ref{Name: branchName, Type: RefBranch})
		} else if strings.HasPrefix(p, "tag: ") {
			tagName := strings.TrimPrefix(p, "tag: ")
			results = append(results, Ref{Name: tagName, Type: RefTag})
		} else if strings.HasPrefix(p, "origin/") || strings.Contains(p, "/") {
			results = append(results, Ref{Name: p, Type: RefRemote})
		} else {
			results = append(results, Ref{Name: p, Type: RefBranch})
		}
	}

	return results
}
