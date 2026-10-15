package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// DiffCache provides thread-safe in-memory caching of parsed diff data.
type DiffCache struct {
	mu    sync.RWMutex
	cache map[string]*DiffData
	order []string
	limit int
}

// NewDiffCache initializes an LRU cache with the specified capacity.
func NewDiffCache(limit int) *DiffCache {
	if limit <= 0 {
		limit = 256
	}
	return &DiffCache{
		cache: make(map[string]*DiffData),
		order: make([]string, 0, limit),
		limit: limit,
	}
}

func (c *DiffCache) Get(hash string) (*DiffData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, exists := c.cache[hash]
	return data, exists
}

func (c *DiffCache) Put(hash string, data *DiffData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[hash]; exists {
		c.cache[hash] = data
		return
	}

	if len(c.order) >= c.limit {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.cache, oldest)
	}

	c.order = append(c.order, hash)
	c.cache[hash] = data
}

// DiffReader manages fetching and caching commit diffs.
type DiffReader struct {
	repoPath string
	cache    *DiffCache
}

// NewDiffReader creates a new DiffReader for a repository.
func NewDiffReader(repoPath string) *DiffReader {
	return &DiffReader{
		repoPath: repoPath,
		cache:    NewDiffCache(256),
	}
}

// GetCommitDiff retrieves parsed file stats and syntax-categorized diff lines.
func (dr *DiffReader) GetCommitDiff(hash string) (*DiffData, error) {
	if data, ok := dr.cache.Get(hash); ok {
		return data, nil
	}

	stats, totalAdd, totalDel, err := dr.fetchStats(hash)
	if err != nil {
		return nil, err
	}

	rawDiff, lines, err := dr.fetchDiffLines(hash)
	if err != nil {
		return nil, err
	}

	diffData := &DiffData{
		Stats:            stats,
		TotalAdditions:   totalAdd,
		TotalDeletions:   totalDel,
		RawDiff:          rawDiff,
		HighlightedLines: lines,
	}

	dr.cache.Put(hash, diffData)
	return diffData, nil
}

func (dr *DiffReader) fetchStats(hash string) ([]FileStat, int, int, error) {
	// Status map (A, M, D, R)
	statusCmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-status", "-r", "-m", "--first-parent", hash)
	statusCmd.Dir = dr.repoPath
	statusOutput, _ := statusCmd.Output()

	statuses := make(map[string]string)
	sScanner := bufio.NewScanner(bytes.NewReader(statusOutput))
	for sScanner.Scan() {
		line := strings.TrimSpace(sScanner.Text())
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			statusLetter := string(fields[0][0])
			path := fields[len(fields)-1]
			statuses[path] = statusLetter
		}
	}

	// Numstat map
	cmd := exec.Command("git", "show", "--numstat", "--format=", hash)
	cmd.Dir = dr.repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("git show numstat failed: %w", err)
	}

	var stats []FileStat
	totalAdd := 0
	totalDel := 0

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		addStr := parts[0]
		delStr := parts[1]
		path := parts[2]

		add := 0
		del := 0
		if addStr != "-" {
			add, _ = strconv.Atoi(addStr)
		}
		if delStr != "-" {
			del, _ = strconv.Atoi(delStr)
		}

		status := statuses[path]
		if status == "" {
			if add > 0 && del == 0 {
				status = "A"
			} else if add == 0 && del > 0 {
				status = "D"
			} else {
				status = "M"
			}
		}

		stats = append(stats, FileStat{
			Path:      path,
			Additions: add,
			Deletions: del,
			Status:    status,
		})

		totalAdd += add
		totalDel += del
	}

	return stats, totalAdd, totalDel, nil
}

func (dr *DiffReader) fetchDiffLines(hash string) (string, []DiffLine, error) {
	cmd := exec.Command("git", "show", "--color=never", "--patch", "--format=", hash)
	cmd.Dir = dr.repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", nil, fmt.Errorf("git show patch failed: %w", err)
	}

	raw := string(out)
	var lines []DiffLine
	scanner := bufio.NewScanner(strings.NewReader(raw))

	for scanner.Scan() {
		text := scanner.Text()
		lineType := DiffLineContext

		if strings.HasPrefix(text, "diff --git") ||
			strings.HasPrefix(text, "index ") ||
			strings.HasPrefix(text, "--- ") ||
			strings.HasPrefix(text, "+++ ") ||
			strings.HasPrefix(text, "new file mode") ||
			strings.HasPrefix(text, "deleted file mode") {
			lineType = DiffLineHeader
		} else if strings.HasPrefix(text, "@@ ") {
			lineType = DiffLineChunkHeader
		} else if strings.HasPrefix(text, "+") {
			lineType = DiffLineAddition
		} else if strings.HasPrefix(text, "-") {
			lineType = DiffLineDeletion
		}

		lines = append(lines, DiffLine{
			Type:    lineType,
			Content: text,
		})
	}

	return raw, lines, nil
}
