package git

import (
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

// GetCommitDiff retrieves the first-parent patch for a commit.
func (dr *DiffReader) GetCommitDiff(hash string) (*DiffData, error) {
	return dr.GetDiff(DiffRequest{Commit: hash})
}
