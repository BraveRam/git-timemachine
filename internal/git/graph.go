package git

import (
	"strings"
)

// GraphLaneManager tracks active lanes during topological graph construction.
type GraphLaneManager struct {
	lanes []string // commit hashes currently occupying each lane
}

// NewGraphLaneManager initializes an empty lane manager.
func NewGraphLaneManager() *GraphLaneManager {
	return &GraphLaneManager{
		lanes: make([]string, 0),
	}
}

// BuildGraphRoutes assigns lanes and generates Unicode graph prefixes for commits.
// Commits are assumed to be sorted in reverse-chronological (topological) order:
// newest at index 0, oldest at index len-1.
func BuildGraphRoutes(commits []*Commit) int {
	if len(commits) == 0 {
		return 0
	}

	maxLanes := 1
	activeLanes := make([]string, 0) // slice of commit hashes expected in each column

	for _, c := range commits {
		// 1. Find which lane this commit belongs to
		commitLane := -1
		for l, expectedHash := range activeLanes {
			if expectedHash == c.Hash {
				commitLane = l
				break
			}
		}

		// If this commit wasn't expected in any existing lane, assign a new or free slot
		if commitLane == -1 {
			for l, expectedHash := range activeLanes {
				if expectedHash == "" {
					commitLane = l
					activeLanes[l] = c.Hash
					break
				}
			}
			if commitLane == -1 {
				commitLane = len(activeLanes)
				activeLanes = append(activeLanes, c.Hash)
			}
		}

		c.Node.Lane = commitLane
		c.Node.LaneColor = commitLane % 8
		if len(activeLanes) > maxLanes {
			maxLanes = len(activeLanes)
		}

		// 2. Build the visual line prefix for this commit row
		// e.g., "│  *  │  "
		var sb strings.Builder
		for l := 0; l < len(activeLanes); l++ {
			if l == commitLane {
				if len(c.Parents) > 1 {
					sb.WriteString("⬢ ") // Merge commit marker
				} else {
					sb.WriteString("● ") // Standard commit marker
				}
			} else if activeLanes[l] != "" {
				sb.WriteString("│ ")
			} else {
				sb.WriteString("  ")
			}
		}
		c.Node.Prefix = sb.String()

		// 3. Update active lanes for parents
		if len(c.Parents) == 0 {
			// Root commit (no parents), free up this lane
			activeLanes[commitLane] = ""
		} else {
			// First parent inherits this lane
			activeLanes[commitLane] = c.Parents[0]

			// Additional parents (merges) look for existing lanes or allocate new ones
			for i := 1; i < len(c.Parents); i++ {
				parentHash := c.Parents[i]
				alreadyTracked := false
				for _, exp := range activeLanes {
					if exp == parentHash {
						alreadyTracked = true
						break
					}
				}
				if !alreadyTracked {
					// Allocate free slot or append
					allocated := false
					for l, exp := range activeLanes {
						if exp == "" {
							activeLanes[l] = parentHash
							allocated = true
							break
						}
					}
					if !allocated {
						activeLanes = append(activeLanes, parentHash)
					}
				}
			}
		}

		// Trim trailing empty lanes
		for len(activeLanes) > 0 && activeLanes[len(activeLanes)-1] == "" {
			activeLanes = activeLanes[:len(activeLanes)-1]
		}
	}

	return maxLanes
}
