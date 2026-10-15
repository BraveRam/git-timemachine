package cli

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

var Version = "1.0.0"

// Run executes the CLI application with the specified binary name (e.g. "gtm" or "git-timemachine").
func Run(binName string) {
	var (
		repoPathFlag = flag.String("path", ".", "Path to the git repository (default: current directory)")
		themeFlag    = flag.String("theme", "catppuccin", "Initial theme: "+strings.Join(tui.ThemeIDs(), ", "))
		versionFlag  = flag.Bool("version", false, "Print version")
	)
	flag.StringVar(repoPathFlag, "p", ".", "Path to the git repository (shorthand)")
	flag.StringVar(themeFlag, "t", "catppuccin", "Initial theme (shorthand)")
	flag.BoolVar(versionFlag, "v", false, "Print version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - Interactive Git History Scrubber TUI (v%s)\n\n", binName, Version)
		fmt.Fprintf(os.Stderr, "Usage:\n  %s [flags] [path]\n\n", binName)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s version %s\n", binName, Version)
		os.Exit(0)
	}

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

	initialTheme, ok := tui.ThemeByName(*themeFlag)
	if !ok {
		initialTheme = tui.ThemeCatppuccinMocha
	}

	app := tui.NewAppModel(graph, initialTheme)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s: %v\n", binName, err)
		os.Exit(1)
	}
}
