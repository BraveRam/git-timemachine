# Build Mission: `git-timemachine` (Interactive Git History Scrubber TUI)

## 🎯 High-Level Vision
Build an ultra-fast, aesthetically stunning Terminal User Interface (TUI) in Go called **`git-timemachine`**. 

`git-timemachine` transforms any Git repository into an interactive, scrubbable visual timeline. Developers can scrub backward and forward through commit history with a timeline slider, watch branches fork and merge with smooth terminal animations, inspect file diffs in real-time, and hit `Space` to run a timelapse "movie" of how the repository evolved from commit #1 to HEAD.

---

## 🛠 Tech Stack Requirements
- **Language**: Go 1.22+
- **TUI Framework**: Charmbracelet ecosystem:
  - `github.com/charmbracelet/bubbletea` (Elm-architecture terminal state manager)
  - `github.com/charmbracelet/lipgloss` (Styling, layout, borders, color palettes)
  - `github.com/charmbracelet/bubbles` (Viewport, progress bar, spinners, help menus)
- **Git Engine**:
  - Primary: `github.com/go-git/go-git/v5` or high-throughput piped `git log` / `git rev-list` calls.
  - Crucial: Lazy loading or chunked caching so repositories with 10,000+ commits load instantly and scrub at 60fps without hitching.

---

## 🚀 Key Features & Specifications

### 1. Interactive Timeline Scrubber
- A horizontal scrubber bar anchored at the top or bottom of the screen.
- Visual scrubber needle: `[◀ ──●───────── ▶]` displaying current commit index, total commits, hash, author, and timestamp.
- Jump controls: jump to first commit (`Home` or `g`), jump to latest (`End` or `G`), or scrub by steps.
- Visual markers along the timeline bar indicating Git tags (`v1.0.0`) and branch heads (`main`, `dev`).

### 2. Unicode Branch & Merge Graph
- Render a clean Unicode tree (`* │ ╭─╯ ╰─╮`) showing branch histories and merge commits.
- Dynamic lane coloring: each branch lane gets a distinct contrasting hue (Tokyo Night / Catppuccin inspired).
- Active commit indicator: current commit is brightly highlighted, while past/future commits dim gracefully.

### 3. Split-Pane Inspector & Live Diff Viewer
- Layout:
  - **Left pane**: Visual commit tree & timeline graph.
  - **Right top pane**: Active commit card (Author, commit hash, date, full commit message, GPG signature badge if signed).
  - **Right bottom pane**: Changed files list with additions/deletions summary (`+45 -12`), plus a syntax-highlighted diff viewer (`bubbles/viewport`).
- Support toggling full-screen diff view with `Enter` or `f`.

### 4. "Timelapse" Auto-Play Mode
- Pressing `Space` starts an automated timeline playback.
- Commits automatically advance forward in time, animating the branch tree and changed files as the codebase grows.
- Playback speed toggles: `[` and `]` to cycle between `0.5x`, `1x`, `2x`, and `5x` speeds.

### 5. Keyboard Navigation & Shortcuts
- `←` / `→` or `h` / `l`: Scrub 1 commit backward / forward.
- `Shift + ←` / `Shift + →` or `J` / `K`: Scrub 10 commits backward / forward.
- `Space`: Play / Pause timelapse auto-scrub.
- `Tab`: Switch active focus between Tree view, File list, and Diff viewport.
- `/`: Filter / fuzzy search commits by message, author, or hash.
- `t`: Cycle color themes (Catppuccin Mocha, Tokyo Night, Nord, Monokai).
- `?`: Toggle help overlay.
- `q` or `Ctrl+C`: Exit cleanly, restoring terminal alternate buffer.

---

## 📁 Repository & Code Architecture
You have full authorization to initialize files, run Go commands, install dependencies, and build the project. Structure the codebase cleanly:

```text
git-timemachine/
├── cmd/
│   └── git-timemachine/
│       └── main.go              # CLI flags, repo path resolution, Bubble Tea runner
├── internal/
│   ├── git/
│   │   ├── reader.go            # Commit history parser, fast graph builder
│   │   ├── diff.go              # Diff extractor and chunk formatter
│   │   └── models.go            # Commit, Branch, FileChange structs
│   └── tui/
│       ├── app.go               # Top-level BubbleTea model & message dispatch
│       ├── scrubber.go          # Horizontal timeline slider component
│       ├── tree_view.go         # Unicode branch graph renderer
│       ├── details_view.go      # Commit info & changed files inspector
│       ├── diff_view.go         # Scrollable syntax diff viewport
│       ├── theme.go             # Lip Gloss color tokens and layout styles
│       └── keys.go              # Keymap declarations and help bindings
├── demo.tape                    # Charm VHS script to record a viral demo GIF
├── go.mod
├── go.sum
└── README.md                    # Showcasing badges, features, and keybindings
```

---

## 🎬 Viral Deliverables & Polish
1. **Charm VHS Script (`demo.tape`)**:
   - Provide a complete `vhs` configuration script in the root directory that simulates launching `git-timemachine`, scrubbing through commits, toggling auto-play, and inspecting diffs to produce a crisp `demo.gif`.
2. **README.md**:
   - Polished GitHub README with title header, installation instructions (`go install github.com/...@latest`, binary release instructions), animated GIF placeholder, and keybindings table.

---

## ⚡ Execution Instructions for the Agent
1. Initialize the Go module and download dependencies (`bubbletea`, `lipgloss`, `bubbles`, `go-git`).
2. Implement the Git reader first and verify it reads commits from any local git directory.
3. Build the core Bubble Tea layout (scrubber + tree + diff).
4. Implement keyboard controls and the auto-play timelapse ticker.
5. Test against a real git repository (or test git repo with branches and merges) to confirm smooth rendering.
6. Generate the `demo.tape` file and README.md.
