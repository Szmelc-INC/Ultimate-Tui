package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

func styledModel(t *testing.T, src, themeName, styleName string, w, h int) *engine.Model {
	t.Helper()
	cfg, err := config.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	opts := engine.DefaultOptions()
	opts.Theme, opts.Style = themeName, styleName
	m, err := engine.NewWithOptions(cfg, opts)
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return u.(*engine.Model)
}

func TestStylesetChangesBorderRunes(t *testing.T) {
	// Border runes are plain glyphs (no ANSI), so they survive the headless
	// Ascii render profile and are safe to assert on.
	dbl := styledModel(t, gridSrc, "auto", "double", 80, 24).View()
	if !strings.Contains(dbl, "╔") || !strings.Contains(dbl, "╚") {
		t.Fatalf("double styleset did not draw double-line corners:\n%s", dbl)
	}

	asc := styledModel(t, gridSrc, "auto", "ascii", 80, 24).View()
	if strings.Contains(asc, "╭") || strings.Contains(asc, "╔") {
		t.Fatal("ascii styleset still drew unicode box corners")
	}
	if !strings.Contains(asc, "+") || !strings.Contains(asc, "|") {
		t.Fatalf("ascii styleset did not draw +/| borders:\n%s", asc)
	}
	if strings.Contains(asc, "\U000f0238") {
		t.Fatal("ascii styleset leaked the Nerd Font nav glyph into the status bar")
	}
}

func TestStatusBarWidthHoldsAcrossStylesets(t *testing.T) {
	// The bar's exact-width guarantee rests on lipgloss.Width matching for the
	// mode glyph of every preset; check each one at a few widths.
	for _, style := range engine.StyleNames() {
		for _, w := range []int{40, 80, 133} {
			m := styledModel(t, gridSrc, "auto", style, w, 20)
			lines := strings.Split(m.View(), "\n")
			last := lines[len(lines)-1]
			if lipgloss.Width(last) != w {
				t.Fatalf("style %q at w=%d: status bar width = %d", style, w, lipgloss.Width(last))
			}
			// Lock mode uses a different glyph — check that width too.
			u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = u.(*engine.Model)
			lines = strings.Split(m.View(), "\n")
			last = lines[len(lines)-1]
			if lipgloss.Width(last) != w {
				t.Fatalf("style %q at w=%d (locked): status bar width = %d", style, w, lipgloss.Width(last))
			}
		}
	}
}

func TestNamedThemeRendersToExactSize(t *testing.T) {
	for _, name := range engine.ThemeNames() {
		m := styledModel(t, gridSrc, name, "rounded", 100, 30)
		v := m.View()
		if n := len(strings.Split(v, "\n")); n != 30 {
			t.Fatalf("theme %q: height = %d, want 30", name, n)
		}
		for i, ln := range strings.Split(v, "\n") {
			if lipgloss.Width(ln) != 100 {
				t.Fatalf("theme %q: line %d width = %d, want 100", name, i, lipgloss.Width(ln))
			}
		}
	}
}
