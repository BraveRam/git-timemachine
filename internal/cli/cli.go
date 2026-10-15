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

var Version = "1.1.0"

// Run executes the CLI application with the specified binary name (e.g. "gtm" or "git-timemachine").
func Run(binName string) {
	var (
		repoPathFlag = flag.String("path", ".", "Path to the git repository (default: current directory)")
		themeFlag    = flag.String("theme", "catppuccin", "Initial theme: "+strings.Join(tui.ThemeIDs(), ", "))
		fileFlag     = flag.String("file", "", "Follow one file's history, including renames (relative to repository path)")
		searchFlag   = flag.String("search", "", "Find literal text added or removed across commit diffs")
		compareFlag  = flag.String("compare", "", "Compare two revisions: START..END")
		workingFlag  = flag.Bool("working", false, "Show unstaged tracked working-tree changes")
		stagedFlag   = flag.Bool("staged", false, "Show staged changes")
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

	// Anchor Git commands at the repository root, including launches from subfolders.
	rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootCmd.Dir = absPath
	rootBytes, err := rootCmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error resolving repository root:", err)
		os.Exit(1)
	}
	root := strings.TrimSpace(string(rootBytes))
	if *fileFlag != "" {
		target := *fileFlag
		if !filepath.IsAbs(target) {
			target = filepath.Join(absPath, target)
		}
		relative, err := filepath.Rel(root, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			fmt.Fprintln(os.Stderr, "Error: file must be inside the repository")
			os.Exit(1)
		}
		*fileFlag = filepath.ToSlash(relative)
	}
	absPath = root
	if *workingFlag && *stagedFlag {
		fmt.Fprintln(os.Stderr, "Error: use either --working or --staged")
		os.Exit(1)
	}
	if *compareFlag != "" && (*workingFlag || *stagedFlag) {
		fmt.Fprintln(os.Stderr, "Error: comparison cannot be combined with working-tree modes")
		os.Exit(1)
	}
	// Read repository history
	fmt.Printf("Loading Git history from %s...\n", absPath)
	graph, err := git.ReadFilteredHistory(absPath, *fileFlag, *searchFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading git history: %v\n", err)
		os.Exit(1)
	}

	initialTheme, ok := tui.ThemeByName(*themeFlag)
	if !ok {
		initialTheme = tui.ThemeCatppuccinMocha
	}

	app := tui.NewAppModel(graph, initialTheme)
	mode := ""
	if *workingFlag {
		mode = "unstaged"
	}
	if *stagedFlag {
		mode = "staged"
	}
	base, end := "", ""
	if *compareFlag != "" {
		parts := strings.Split(*compareFlag, "..")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			fmt.Fprintln(os.Stderr, "Error: --compare expects START..END")
			os.Exit(1)
		}
		refs := []string{}
		for _, ref := range parts {
			cmd := exec.Command("git", "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
			cmd.Dir = absPath
			hash, err := cmd.Output()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: invalid comparison revision", ref)
				os.Exit(1)
			}
			refs = append(refs, strings.TrimSpace(string(hash)))
		}
		base, end = refs[0], refs[1]
	}
	if err := app.ConfigureReview(mode, base, end); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s: %v\n", binName, err)
		os.Exit(1)
	}
}
