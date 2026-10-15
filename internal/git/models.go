package git

import "time"

// RefType indicates whether a ref is a branch, tag, HEAD, or remote.
type RefType int

const (
	RefBranch RefType = iota
	RefTag
	RefHead
	RefRemote
)

// Ref represents a Git reference (tag, local branch, remote branch, HEAD).
type Ref struct {
	Name string
	Type RefType
}

// FileStat contains insertion/deletion stats for a single file in a commit.
type FileStat struct {
	Path      string
	Additions int
	Deletions int
	Status    string // "A" Added, "M" Modified, "D" Deleted, "R" Renamed
}

// DiffLineType defines the semantic nature of a line in a unified diff.
type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAddition
	DiffLineDeletion
	DiffLineHeader
	DiffLineChunkHeader
)

// DiffLine holds an individual line of a diff with its syntax classification.
type DiffLine struct {
	Type    DiffLineType
	Content string
}

// DiffData encapsulates stats and syntax-categorized lines for a commit diff.
type DiffData struct {
	Stats            []FileStat
	TotalAdditions   int
	TotalDeletions   int
	RawDiff          string
	HighlightedLines []DiffLine
}

// GraphNode holds the Unicode graph visual elements for a commit line.
type GraphNode struct {
	Lane      int    // 0-indexed column lane
	Symbol    string // e.g. "*", "●"
	Prefix    string // Unicode connectors e.g. "│  ╭─╯"
	LaneColor int    // Palette index for dynamic lane coloring
}

// Commit holds parsed Git commit information and its graph representation.
type Commit struct {
	Hash        string
	ShortHash   string
	Parents     []string
	AuthorName  string
	AuthorEmail string
	AuthorDate  time.Time
	Subject     string
	Body        string
	GPGStatus   string // "G" = Good/Signed, "B" = Bad, "N" = None, etc.
	Refs        []Ref
	Index       int       // 0 = oldest commit, N-1 = newest commit
	Node        GraphNode // Unicode tree node information
}

// IsSigned returns true if the commit has a verified GPG/SSH signature.
func (c *Commit) IsSigned() bool {
	return c.GPGStatus == "G"
}

// CommitGraph represents the entire loaded commit history.
type CommitGraph struct {
	RepoPath  string
	Commits   []*Commit          // Chronological order: 0 = oldest commit, len-1 = newest
	CommitMap map[string]*Commit // Hash -> Commit
	Tags      []Ref
	Branches  []Ref
	MaxLanes  int
}
