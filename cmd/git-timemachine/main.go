package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BraveRam/git-timemachine/internal/git"
	"github.com/BraveRam/git-timemachine/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "1.0.0"
)

func main() {
	var (
		repoPathFlag = flag.String("path", ".", "Path to the git repository (default: current directory)")
		themeFlag    = flag.String("theme", "catppuccin", "Initial theme: catppuccin, tokyonight, nord, monokai")
		versionFlag  = flag.Bool("version", false, "Print git-timemachine version")
	)
	flag.StringVar(repoPathFlag, "p", ".", "Path to the git repository (shorthand)")
	flag.StringVar(themeFlag, "t", "catppuccin", "Initial theme (shorthand)")
	flag.BoolVar(versionFlag, "v", false, "Print version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "⏱ git-timemachine - Interactive Git History Scrubber TUI (v%s)\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n  git-timemachine [flags] [path]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("git-timemachine version %s\n", version)
		os.Exit(0)
	}

	// Positional argument takes priority if provided
	targetPath := *repoPathFlag
	if flag.NArg() > 0 {
		targetPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to resolve path '%s': %v\n", targetPath, err)
		os.Exit(1)
	}

	// Verify target is a valid Git repository
	checkCmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	checkCmd.Dir = absPath
	if err := checkCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: '%s' is not a valid Git repository (or git is not installed).\n", absPath)
		os.Exit(1)
	}

	// Read repository history
	fmt.Printf("Loading Git history from %s...\n", absPath)
	graph, err := git.ReadHistory(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading git history: %v\n", err)
		os.Exit(1)
	}

	// Select initial theme
	initialTheme := tui.ThemeCatppuccinMocha
	switch strings.ToLower(*themeFlag) {
	case "tokyonight", "tokyo", "tokyo-night":
		initialTheme = tui.ThemeTokyoNight
	case "nord":
		initialTheme = tui.ThemeNord
	case "monokai", "monokai-pro":
		initialTheme = tui.ThemeMonokaiPro
	case "catppuccin", "mocha":
		initialTheme = tui.ThemeCatppuccinMocha
	}

	// Initialize Bubble Tea application
	app := tui.NewAppModel(graph, initialTheme)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running git-timemachine: %v\n", err)
		os.Exit(1)
	}
}
