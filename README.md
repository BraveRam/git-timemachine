<div align="center">

```
       _ _          _   _                            _     _            
  __ _(_) |_       | |_(_)_ __ ___   ___ _ __   __ _|  ___| |__  _ __   
 / _` | | __|______| __| | '_ ` _ \ / _ \ '_ \ / _` | / __| '_ \| '_ \  
| (_| | | ||_______| |_| | | | | | |  __/ | | | (_| || (__| | | | | | | 
 \__, |_|\__|       \__|_|_| |_| |_|\___|_| |_|\__,_(_)___|_| |_|_| |_| 
 |___/                                                                  
```

# `git-timemachine`

**The Interactive Git History Scrubber & Visual Timelapse TUI**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Built with Charm](https://img.shields.io/badge/Built%20with-Charmbracelet-F38BA8?style=flat)](https://charm.sh)
[![License: MIT](https://img.shields.io/badge/License-MIT-A6E3A1.svg?style=flat)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-89B4FA.svg?style=flat)](https://github.com/BraveRam/git-timemachine/pulls)

*Scrub backward and forward through your Git history with a timeline slider, watch branches fork and merge with silky terminal animations, inspect file diffs in real-time, and hit `Space` to run a timelapse movie of your repo's evolution.*

---

![git-timemachine demo](demo.gif)

</div>

## ✨ Highlights

- ⏱ **Interactive Timeline Scrubber**: A fluid horizontal needle (`[◀ ──●───────── ▶]`) indicating exact chronological position, Git tags (`◆ v1.0.0`), and branch tips (`⚑ main`).
- 🎬 **Timelapse "Movie" Auto-Play**: Hit `Space` and watch commits advance forward like a film, animating the branch DAG and code evolution at `0.5x`, `1x`, `2x`, or `5x` speeds.
- 🌳 **Unicode Branch & Merge Graph**: Clean terminal tree rendering (`* │ ╭─╯ ╰─╮`) with automatic lane tracking and dynamic lane coloring.
- 🔍 **Split-Pane Inspector & Live Diff**:
  - **Left Pane**: Branch tree and commit graph.
  - **Right Top Pane**: Active commit card (Author, hash, date, full commit message, GPG verification badge `[✓ SIGNED]`).
  - **Right Bottom Pane**: Changed files summary (`+45 -12`) and scrollable syntax-highlighted unified diff.
  - **Fullscreen Diff**: Toggle full-screen code inspection with `f` or `Enter`.
- ⚡ **Zero-Lag Engine**: Piped zero-allocation Git log streaming and LRU diff caching capable of handling repositories with 10,000+ commits at 60fps.
- 🎨 **Aesthetic Themes**: Cycle on-the-fly between **Catppuccin Mocha**, **Tokyo Night**, **Nord**, and **Monokai Pro** with `t`.
- 🔎 **Instant Search (`/`)**: Real-time fuzzy/substring commit filtering across subjects, authors, and hashes.

---

## 🚀 Installation

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

## 🕹 Usage

Run `gtm` in any Git repository:

```bash
# In current repo
gtm

# Specify path to a repository
gtm /path/to/repo
gtm -p ../my-project

# Launch with custom theme (catppuccin, tokyonight, nord, monokai)
gtm -t tokyonight
```

---

## ⌨️ Keyboard Shortcuts

| Keybinding | Action |
|:---|:---|
| `←` / `h` | Scrub 1 commit backward in time |
| `→` / `l` | Scrub 1 commit forward in time |
| `Shift + ←` / `J` | Jump 10 commits backward |
| `Shift + →` / `K` | Jump 10 commits forward |
| `Home` / `g` | Jump to repository inception (Commit #1) |
| `End` / `G` | Jump to latest commit (`HEAD`) |
| `Space` | **Play / Pause timelapse movie** |
| `[` / `-` | Decrease playback speed (`0.5x`, `1x`, `2x`, `5x`) |
| `]` / `+` | Increase playback speed |
| `Tab` | Switch active pane focus (Tree ↔ Diff View) |
| `f` / `Enter` | Toggle **fullscreen diff** viewer |
| `↑` / `k` / `↓` / `j` | Scroll active tree or diff viewport |
| `/` | **Search commits** by message, author, or hash |
| `t` | **Cycle color theme** (Catppuccin, Tokyo Night, Nord, Monokai) |
| `?` | Toggle Help cheatsheet overlay |
| `Esc` | Close search / Exit fullscreen / Dismiss help |
| `q` / `Ctrl+C` | Exit cleanly |

---

## 🏗 Architecture

```text
git-timemachine/
├── cmd/
│   ├── gtm/
│   │   └── main.go              # Entrypoint producing `gtm` binary
│   └── git-timemachine/
│       └── main.go              # Entrypoint producing `git-timemachine` binary
├── internal/
│   ├── cli/
│   │   └── cli.go               # Shared CLI flags, repo path resolution, Bubble Tea runner
│   ├── git/
│   │   ├── models.go            # Commit, Branch, Tag, FileDiff, CommitGraph structs
│   │   ├── reader.go            # High-throughput streaming git log parser (<50ms for 10k+ commits)
│   │   ├── graph.go             # Unicode branch lane routing algorithm (* │ ╭─╯ ╰─╮)
│   │   ├── diff.go              # LRU-cached diff extractor and syntax chunk classifier
│   │   └── git_test.go          # Unit tests for git engine & parsing
│   └── tui/
│       ├── app.go               # Root BubbleTea model, key router, tick manager, resize handler
│       ├── scrubber.go          # Horizontal timeline slider component [◀ ──●── ▶]
│       ├── tree_view.go         # Unicode branch graph renderer with dynamic lane coloring
│       ├── details_view.go      # Active commit card (author, hash, date, GPG badge, stats)
│       ├── diff_view.go         # Scrollable syntax diff viewport and changed files list
│       ├── search.go            # Live commit search input modal
│       ├── theme.go             # Catppuccin Mocha, Tokyo Night, Nord, Monokai Pro palettes
│       ├── keys.go              # Keymap declarations and help bindings
│       └── tui_test.go          # Unit tests for TUI models and view rendering
├── demo.tape                    # Charm VHS script to record demo GIF
├── go.mod
├── go.sum
└── README.md
```

---

## 🎨 Themes Included

| Theme | Preview Style |
|:---|:---|
| **Catppuccin Mocha** | Soothing pastel palette with mauve cursor and soft rosewater accents |
| **Tokyo Night** | Vibrant cyberpunk hues inspired by the Tokyo dusk |
| **Nord** | Arctic-inspired, ice-clean blue and frost hues |
| **Monokai Pro** | High-contrast energetic yellows, cyans, and magentas |

---

## 📄 License

MIT © [BraveRam](https://github.com/BraveRam)
