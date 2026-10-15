package tui

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func luminance(t *testing.T, color string) float64 {
	t.Helper()
	var r, g, b uint64
	if n, err := fmt.Sscanf(color, "#%02x%02x%02x", &r, &g, &b); err != nil || n != 3 || len(color) != 7 {
		t.Fatalf("invalid theme color %q", color)
	}
	linear := func(v uint64) float64 {
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return .2126*linear(r) + .7152*linear(g) + .0722*linear(b)
}

func TestNewThemeContrastAndCLISelection(t *testing.T) {
	light := 0
	ids := map[string]bool{}
	for _, theme := range AllThemes {
		if ids[theme.ID] {
			t.Fatalf("duplicate theme ID: %s", theme.ID)
		}
		ids[theme.ID] = true
		for _, name := range []string{theme.ID, theme.Name} {
			found, ok := ThemeByName(name)
			if !ok || found.Name != theme.Name {
				t.Fatalf("theme %q cannot be selected", name)
			}
		}
		if theme.Light {
			light++
			if luminance(t, theme.Background) < .9 {
				t.Fatalf("light theme %s does not have a white background", theme.Name)
			}
		}
		// Check new palettes' normal text and semantic colors on their surface.
		if theme.ID == "catppuccin" || theme.ID == "tokyonight" || theme.ID == "nord" || theme.ID == "monokai" {
			continue
		}
		bg := luminance(t, theme.Background)
		for _, color := range append([]string{theme.Foreground, theme.Muted, theme.DiffAddFg, theme.DiffDelFg}, theme.Lanes...) {
			fg := luminance(t, color)
			contrast := (max(fg, bg) + .05) / (min(fg, bg) + .05)
			if contrast < 4.5 {
				t.Errorf("%s color %s has contrast %.2f", theme.Name, color, contrast)
			}
		}
	}
	if light != 3 || len(AllThemes)-light != 11 {
		t.Fatalf("wanted 3 light + 11 dark total, got %d light", light)
	}
	for _, alias := range []string{"mocha", "tokyo", "tokyo-night", "monokai-pro", " PAPER WHITE "} {
		if _, ok := ThemeByName(alias); !ok {
			t.Errorf("alias %q no longer works", alias)
		}
	}
	if _, ok := ThemeByName("unknown"); ok {
		t.Fatal("unknown theme unexpectedly accepted")
	}
}

func TestThemeCycleUpdatesScreenAndSearch(t *testing.T) {
	g, d := layoutFixture()
	m := NewAppModel(g, AllThemes[0])
	m.width, m.height = 120, 40
	m.currentDiff = d
	m.updateLayoutDimensions()
	for _, want := range AllThemes {
		if m.currentTheme().ID != want.ID || m.search.theme.ID != want.ID {
			t.Fatalf("theme cycle/search failed for %s", want.ID)
		}
		assertFits(t, want.ID, m.View(), 120, 40)
		m, _ = updateApp(m, resizeKey("t"))
	}
	if m.currentTheme().ID != AllThemes[0].ID {
		t.Fatal("theme cycle did not wrap")
	}
}

func TestLightThemePaintsBackground(t *testing.T) {
	// Force true color so this test inspects the actual emitted color codes.
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
	m := NewAppModel(regressionApp().graph, ThemePaperWhite)
	m.width, m.height = 120, 40
	m.updateLayoutDimensions()
	view := m.View()
	if view == ansi.Strip(view) {
		t.Fatal("render did not emit theme colors")
	}
	if !strings.Contains(view, "48;2;255;255;255") {
		t.Fatal("white theme does not paint the terminal background")
	}
}

func TestScreenRestoresBackgroundAfterStyledText(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
	m := NewAppModel(regressionApp().graph, ThemePaperWhite)
	m.width, m.height = 30, 1
	view := m.themedScreen("\x1b[31mred\x1b[0m plain")
	afterReset := strings.SplitN(view, "\x1b[0m", 2)
	if len(afterReset) < 2 || !strings.HasPrefix(afterReset[1], "\x1b[") {
		t.Fatal("styled text reset did not restore canvas colors")
	}
	beforePlain := strings.SplitN(afterReset[1], " plain", 2)[0]
	if !strings.Contains(beforePlain, "48;2;255;255;255") {
		t.Fatal("text after nested style fell back to terminal background")
	}
}
