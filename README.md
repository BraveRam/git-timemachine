<div align="center">

# `git-timemachine`

**The Interactive Git History Scrubber & Visual Timelapse TUI**

[![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Built with Charm](https://img.shields.io/badge/Built%20with-Charmbracelet-F38BA8?style=flat)](https://charm.sh)
[![License: MIT](https://img.shields.io/badge/License-MIT-A6E3A1.svg?style=flat)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-89B4FA.svg?style=flat)](https://github.com/BraveRam/git-timemachine/pulls)

*Scrub backward and forward through your Git history with a timeline slider, watch branches fork and merge with silky terminal animations, inspect file diffs in real-time, and hit `Space` to run a timelapse movie of your repo's evolution.*

---

</div>

## Highlights

- **Interactive Timeline Scrubber**: A horizontal needle indicating exact chronological position, Git tags, and branch tips.
- **Timelapse Movie Auto-Play**: Press Space and watch commits advance forward, animating the branch DAG and code evolution at 0.5x, 1x, 2x, or 5x speeds.
- **Unicode Branch and Merge Graph**: Clean terminal tree rendering showing branch histories and merge commits with automatic lane tracking and dynamic coloring.
- **Split-Pane Inspector and Live Diff**:
  - **Left Pane**: Branch tree and commit graph.
  - **Right Top Pane**: Active commit card (Author, hash, date, full commit message, GPG verification badge).
  - **Right Bottom Pane**: Changed files summary (+45 -12) and scrollable syntax-highlighted unified diff.
  - **Fullscreen Diff**: Toggle full-screen code inspection with `f` or `Enter`.
- **Zero-Lag Engine**: Piped streaming Git log reader and LRU diff caching capable of handling repositories with 10,000+ commits at 60fps.
- **Aesthetic Themes**: Cycle through 14 palettes (3 light and 11 dark) with `t`.
- **File Investigation**: Choose a changed file with `F`, cycle files with `n` / `p`, and follow rename-aware file history with `H`.
- **Changed-Text Search**: Press `d` to find commits that added or removed literal text, with highlighted diff matches.
- **Compare and Review**: Mark two commits with `a` / `c`, inspect staged and unstaged changes with `w`, and refresh with `r`.
- **Blame and Editor**: Trace a line to its originating commit with `b`, or open the working file in your editor with `e`.
- **Commit Search**: Real-time substring commit filtering across subjects, authors, and hashes with `/`.

---

## Installation

### Using Go (Installs as `gtm`)

```bash
# Recommended: installs `gtm` binary to $GOPATH/bin
go install github.com/BraveRam/git-timemachine/cmd/gtm@latest

# Or install with the full name `git-timemachine`:
go install github.com/BraveRam/git-timemachine/cmd/git-timemachine@latest
```

### From Source

```bash
git clone https://github.com/BraveRam/git-timemachine.git
cd git-timemachine
go build -o gtm ./cmd/gtm
sudo mv gtm /usr/local/bin/
```

---

## Usage

Run `gtm` in any Git repository:

```bash
# In current repository
gtm

# Specify path to a repository
gtm /path/to/repo
gtm -p ../my-project

# Launch with a light or dark theme
gtm -t paper
gtm -t midnight
```

---

## Investigate a Change

```bash
# Follow a file from HEAD through renames (relative to the repository path)
gtm --file src/auth.ts

# Find commits that added or removed this literal, case-sensitive text
gtm --search 'createSession'

# Combine file history and changed-text search
gtm --file src/auth.ts --search 'createSession'

# Compare two commit-ish revisions; use real refs from your repository
gtm --compare main..feature

# Inspect tracked unstaged changes or staged changes
gtm --working
gtm --staged
```

Inside the app, press `F`, type part of a filename, and press Enter. The diff
shows only that file. `n` / `p` cycle through all changed files, including files
beyond the first four. Pick “All changed files” in `F` to restore the full patch.

Press `H` to follow that file from the current branch's HEAD, including rename
history. Press `d`, type text, and Enter to keep commits that added or removed
that text. Matching lines are highlighted; `Ctrl+n` / `Ctrl+p` jump between hits.
This searches changed lines, not commit messages or unchanged file contents.
`/` still searches commit messages/authors/hashes. Press `u` to clear filters.

For comparisons, select a commit and press `a`, select the other commit, then
press `c`. The diff shows the net change **from the marked commit to the selected
commit**. `Esc` returns to ordinary commit diffs (exit fullscreen first if active).

Press `w` to cycle through unstaged, staged, and history views. `r` refreshes
changes made outside the app. These views do not stage, discard, or commit
anything. Untracked files are excluded; staged new files are included.

Choose a file and press `b` to browse blame at the selected commit. Select a line
and Enter to jump to the originating commit and historical filename. This works
for committed files; deleted files may not exist at the selected revision.

Press `e` to open the selected file in the **current working tree**, near the
first visible diff line (or the selected diff text). This uses `$VISUAL`, then
`$EDITOR`, then `vi`. For VS Code, set `EDITOR='code --wait'`; for terminal editors,
use e.g. `EDITOR=nvim`. Historical line numbers may differ from today's file, and
files missing from the working tree cannot be opened. No checkout is performed.

---

## Keyboard Shortcuts

| Keybinding | Action |
|:---|:---|
| `F` | Search the complete changed-file list and select a file |
| `n` / `p` | Inspect next / previous changed file |
| `H` | Follow selected file through history, including renames |
| `d` | Find literal text added or removed across commit history |
| `Ctrl + n` / `Ctrl + p` | Jump to next / previous highlighted diff match |
| `u` | Clear file/text filters and return to all history |
| `a` / `c` | Mark comparison start / compare it to selected commit |
| `w` | Cycle unstaged → staged → commit view |
| `r` | Refresh the current diff |
| `b` | Show selected file's blame; Enter jumps to a line's origin |
| `e` | Open selected working file in `$VISUAL` / `$EDITOR` (default: `vi`) |
| `←` / `h` | Scrub 1 commit backward in time |
| `→` / `l` | Scrub 1 commit forward in time |
| `Shift + ←` / `J` | Jump 10 commits backward |
| `Shift + →` / `K` | Jump 10 commits forward |
| `Home` / `g` | Jump to repository inception (Commit #1) |
| `End` / `G` | Jump to latest commit (`HEAD`) |
| `Space` | Play / Pause timelapse movie |
| `[` / `-` | Decrease playback speed (0.5x, 1x, 2x, 5x) |
| `]` / `+` | Increase playback speed |
| `Tab` | Switch active pane focus (Tree <-> Diff View) |
| `Alt + ←/→` or `<` / `>` | Shrink / grow tree pane |
| `Alt + ↑/↓` or `{` / `}` | Shrink / grow commit inspector; diff gets remaining space |
| `0` | Reset pane sizes |
| Mouse drag | Select text within one pane |
| `y` / `Ctrl + y` | Copy selected pane text to the system clipboard |
| `Esc` | Clear text selection |
| `f` / `Enter` | Toggle fullscreen diff viewer |
| `↑` / `k` / `↓` / `j` | Scroll active tree or diff viewport |
| `/` | Search commits by message, author, or hash |
| `t` | Cycle through all 14 color themes |
| `?` | Toggle Help cheatsheet overlay |
| `Esc` | Close search / Exit fullscreen / Dismiss help |
| `q` / `Ctrl+C` | Exit cleanly |

---


Text selection stays inside the pane where the drag begins, even when the pointer
crosses another pane. Press `y` to copy. On Linux, clipboard copying uses
`wl-copy`/`wl-paste` (Wayland) or `xclip`/`xsel` (X11). Most terminals let you
hold Shift while dragging to use their native selection instead.


## Themes Included

| Theme | CLI name | Appearance |
|:---|:---|:---|
| **Catppuccin Mocha** | `catppuccin` | Dark, soft pastels and mauve |
| **Tokyo Night** | `tokyonight` | Dark, blue and purple |
| **Nord** | `nord` | Dark, arctic blue and frost |
| **Monokai Pro** | `monokai` | Dark, yellow and cyan |
| **Paper White** | `paper` | White, ink text and violet accents |
| **Arctic Light** | `arctic` | Cool white, navy and teal |
| **Rose Quartz** | `rose` | Warm white, berry and plum |
| **Midnight Blue** | `midnight` | Dark navy, blue and lavender |
| **Ember** | `ember` | Dark brown, copper and gold |
| **Forest** | `forest` | Dark evergreen, sage and mint |
| **Cyberpunk** | `cyberpunk` | Dark violet, neon pink and cyan |
| **Deep Ocean** | `ocean` | Dark ocean blue, turquoise and ice |
| **Amethyst** | `amethyst` | Dark plum, lilac and periwinkle |
| **Slate** | `slate` | Dark charcoal, silver and cool pastels |

Use `gtm -t <name>` to start with a theme, or press `t` to cycle.

---

## License

MIT © [BraveRam](https://github.com/BraveRam)
