package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DiffRequest describes a commit, a comparison, or tracked working-tree changes.
// Mode is empty for history, "unstaged", or "staged". Base compares Base..Commit.
type DiffRequest struct {
	Commit, Base, Mode, Path string
}

func (r DiffRequest) Key() string {
	return strings.Join([]string{r.Commit, r.Base, r.Mode, r.Path}, "\x00")
}

func gitOutput(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(e.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func (dr *DiffReader) GetDiff(r DiffRequest) (*DiffData, error) {
	if r.Mode == "" {
		if data, ok := dr.cache.Get(r.Key()); ok {
			return data, nil
		}
	}
	args := []string{"show", "--format=", "--first-parent", "-M"}
	switch {
	case r.Mode == "staged":
		args = []string{"diff", "--cached", "-M"}
	case r.Mode == "unstaged":
		args = []string{"diff", "-M"}
	case r.Base != "":
		args = []string{"diff", "-M", r.Base, r.Commit}
	default:
		args = append(args, r.Commit)
	}
	args = append(args, "--no-ext-diff", "--no-textconv", "--color=never")
	status, err := gitOutput(dr.repoPath, append(append([]string{}, args...), "--name-status", "-z", "--")...)
	if err != nil {
		return nil, err
	}
	numstat, err := gitOutput(dr.repoPath, append(append([]string{}, args...), "--numstat", "-z", "--")...)
	if err != nil {
		return nil, err
	}
	stats, err := parseStats(status, numstat)
	if err != nil {
		return nil, err
	}
	data := &DiffData{AllStats: stats}
	paths := []string{}
	for _, stat := range stats {
		if r.Path == "" || stat.Path == r.Path {
			data.Stats = append(data.Stats, stat)
			data.TotalAdditions += stat.Additions
			data.TotalDeletions += stat.Deletions
			if r.Path != "" {
				paths = append(paths, ":(literal)"+stat.Path)
				if stat.OldPath != "" {
					paths = append(paths, ":(literal)"+stat.OldPath)
				}
			}
		}
	}
	if r.Path != "" && len(paths) == 0 {
		paths = []string{":(literal)" + r.Path}
	}
	patchArgs := append(append([]string{}, args...), "--patch", "--")
	patchArgs = append(patchArgs, paths...)
	patch, err := gitOutput(dr.repoPath, patchArgs...)
	if err != nil {
		return nil, err
	}
	data.RawDiff = string(patch)
	data.HighlightedLines = parsePatch(data.RawDiff)
	if r.Mode == "" {
		dr.cache.Put(r.Key(), data)
	}
	return data, nil
}

func parseStats(status, numstat []byte) ([]FileStat, error) {
	statuses := map[string]FileStat{}
	fields := strings.Split(string(status), "\x00")
	for i := 0; i+1 < len(fields); {
		kind, path := fields[i], fields[i+1]
		i += 2
		if kind == "" {
			continue
		}
		stat := FileStat{Path: path, Status: kind[:1]}
		if kind[0] == 'R' || kind[0] == 'C' {
			if i >= len(fields) {
				return nil, fmt.Errorf("incomplete rename status")
			}
			stat.OldPath, stat.Path = path, fields[i]
			i++
		}
		statuses[stat.Path] = stat
	}
	var stats []FileStat
	fields = strings.Split(string(numstat), "\x00")
	for i := 0; i < len(fields); i++ {
		parts := strings.SplitN(fields[i], "\t", 3)
		if len(parts) != 3 {
			continue
		}
		path := parts[2]
		if path == "" {
			if i+2 >= len(fields) {
				return nil, fmt.Errorf("incomplete rename numstat")
			}
			path = fields[i+2]
			i += 2
		}
		stat := statuses[path]
		stat.Path = path
		if stat.Status == "" {
			stat.Status = "M"
		}
		stat.Additions, _ = strconv.Atoi(parts[0])
		stat.Deletions, _ = strconv.Atoi(parts[1])
		stats = append(stats, stat)
	}
	return stats, nil
}

func parsePatch(raw string) []DiffLine {
	if raw == "" {
		return nil
	}
	var lines []DiffLine
	old, new := 0, 0
	for text := range strings.SplitSeq(strings.TrimSuffix(raw, "\n"), "\n") {
		text = strings.TrimSuffix(text, "\r")
		line := DiffLine{Content: text}
		switch {
		case strings.HasPrefix(text, "diff "), strings.HasPrefix(text, "index "), strings.HasPrefix(text, "--- "), strings.HasPrefix(text, "+++ "), strings.HasPrefix(text, "new file mode"), strings.HasPrefix(text, "deleted file mode"), strings.HasPrefix(text, "rename "), strings.HasPrefix(text, "similarity index"):
			line.Type = DiffLineHeader
		case strings.HasPrefix(text, "@@ "):
			line.Type = DiffLineChunkHeader
			fields := strings.Fields(text)
			if len(fields) > 2 {
				old, _ = strconv.Atoi(strings.Split(strings.TrimPrefix(fields[1], "-"), ",")[0])
				new, _ = strconv.Atoi(strings.Split(strings.TrimPrefix(fields[2], "+"), ",")[0])
			}
		case strings.HasPrefix(text, "+"):
			line.Type, line.NewLine = DiffLineAddition, new
			new++
		case strings.HasPrefix(text, "-"):
			line.Type, line.OldLine = DiffLineDeletion, old
			old++
		case strings.HasPrefix(text, " "):
			line.OldLine, line.NewLine = old, new
			old++
			new++
		}
		lines = append(lines, line)
	}
	return lines
}

func historicalPaths(repo, file string) (map[string]string, error) {
	out, err := gitOutput(repo, "log", "--follow", "-M", "--format=%H", "-z", "--name-status", "HEAD", "--", ":(literal)"+file)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(out), "\x00")
	paths := map[string]string{}
	path := file
	for i := 0; i+2 < len(fields); {
		hash := strings.TrimSpace(fields[i])
		i++
		kind := strings.TrimLeft(fields[i], "\n")
		i++
		paths[hash] = path
		if strings.HasPrefix(kind, "R") || strings.HasPrefix(kind, "C") {
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("incomplete file-history rename")
			}
			paths[hash], path = fields[i+1], fields[i]
			i += 2
		} else {
			i++
		}
	}
	return paths, nil
}

type BlameLine struct {
	Commit, Path, Author, Text string
	Line, OriginalLine         int
}

func ReadBlame(repo, commit, path string) ([]BlameLine, error) {
	out, err := gitOutput(repo, "blame", "--line-porcelain", commit, "--", path)
	if err != nil {
		return nil, err
	}
	var rows []BlameLine
	var row BlameLine
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		switch {
		case !strings.HasPrefix(line, "\t") && len(fields) >= 3 && (len(fields[0]) == 40 || len(fields[0]) == 64):
			row = BlameLine{Commit: fields[0], Path: path}
			row.OriginalLine, _ = strconv.Atoi(fields[1])
			row.Line, _ = strconv.Atoi(fields[2])
		case strings.HasPrefix(line, "author "):
			row.Author = strings.TrimPrefix(line, "author ")
		case strings.HasPrefix(line, "filename "):
			row.Path = strings.TrimPrefix(line, "filename ")
			if strings.HasPrefix(row.Path, "\"") {
				if p, e := strconv.Unquote(row.Path); e == nil {
					row.Path = p
				}
			}
		case strings.HasPrefix(line, "\t"):
			row.Text = strings.TrimPrefix(line, "\t")
			rows = append(rows, row)
		}
	}
	return rows, nil
}
