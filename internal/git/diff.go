package git

import (
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
	statusCmd := exec.Command("git", "diff-tree", "--root", "--no-commit-id", "--name-status", "-z", "-M", "-r", "-m", "--first-parent", hash)
	statusCmd.Dir = dr.repoPath
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("git diff-tree name-status failed: %w", err)
	}

	statuses := make(map[string]string)
	statusFields := strings.Split(string(statusOutput), "\x00")
	for i := 0; i+1 < len(statusFields); {
		status := statusFields[i]
		path := statusFields[i+1]
		i += 2
		if status == "" {
			continue
		}
		if status[0] == 'R' || status[0] == 'C' {
			if i >= len(statusFields) {
				return nil, 0, 0, fmt.Errorf("incomplete git name-status rename record")
			}
			path = statusFields[i]
			i++
		}
		statuses[path] = status[:1]
	}

	// NUL delimiters preserve whitespace, tabs, newlines, and unquoted paths.
	cmd := exec.Command("git", "show", "--numstat", "-z", "-M", "--format=", hash)
	cmd.Dir = dr.repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("git show numstat failed: %w", err)
	}

	var stats []FileStat
	totalAdd := 0
	totalDel := 0

	statFields := strings.Split(string(out), "\x00")
	for i := 0; i < len(statFields); i++ {
		parts := strings.SplitN(statFields[i], "\t", 3)
		if len(parts) < 3 {
			continue
		}

		addStr := parts[0]
		delStr := parts[1]
		path := parts[2]
		if path == "" {
			// Rename/copy records contain separate old and new path fields.
			if i+2 >= len(statFields) {
				return nil, 0, 0, fmt.Errorf("incomplete git numstat rename record")
			}
			path = statFields[i+2]
			i += 2
		}

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
	if raw == "" {
		return raw, lines, nil
	}
	// The patch is already buffered; iterate without Scanner's token-size limit.
	for text := range strings.SplitSeq(strings.TrimSuffix(raw, "\n"), "\n") {
		text = strings.TrimSuffix(text, "\r")
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
