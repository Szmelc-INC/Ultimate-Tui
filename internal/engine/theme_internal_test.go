package engine

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestResolveThemeAndStyleNames(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", "auto"}, {"auto", "auto"}, {"dark", "auto"}, {"light", "auto"},
		{"gruvbox", "gruvbox"}, {"tokyo-night", "tokyo-night"}, {"midnight", "midnight"},
		{"nonsense", "auto"},
	} {
		if got := resolveThemeName(c.in); got != c.want {
			t.Errorf("resolveThemeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	for _, c := range []struct{ in, want string }{
		{"", "rounded"}, {"normal", "rounded"}, {"thick", "heavy"}, {"bold", "heavy"},
		{"ascii", "ascii"}, {"double", "double"}, {"bogus", "rounded"},
	} {
		if got := resolveStyleName(c.in); got != c.want {
			t.Errorf("resolveStyleName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildThemeResolvesPaletteAndStyleset(t *testing.T) {
	// A named palette must be concrete, never adaptive — tokyo-night is dark.
	tk := buildTheme("tokyo-night", "rounded")
	if c, ok := tk.borderFocus.(lipgloss.Color); !ok || string(c) != "#7aa2f7" {
		t.Fatalf("tokyo-night borderFocus = %#v, want lipgloss.Color #7aa2f7", tk.borderFocus)
	}
	// The "auto" palette stays adaptive.
	au := buildTheme("auto", "rounded")
	if _, ok := au.borderFocus.(lipgloss.AdaptiveColor); !ok {
		t.Fatalf("auto borderFocus = %#v, want AdaptiveColor", au.borderFocus)
	}
	// The styleset drives the border rune set, independent of palette.
	if got := buildTheme("auto", "double").border.TopLeft; got != "╔" {
		t.Fatalf("double border TopLeft = %q, want ╔", got)
	}
	if got := buildTheme("gruvbox", "ascii").border.TopLeft; got != "+" {
		t.Fatalf("ascii border TopLeft = %q, want +", got)
	}
	// The ascii styleset must not smuggle in Nerd Font glyphs.
	as := buildTheme("auto", "ascii")
	for _, g := range []string{as.glyphNav, as.glyphLock, as.bullet, as.sep, as.ellipsis} {
		for _, r := range g {
			if r > 0x7f {
				t.Fatalf("ascii styleset glyph %q contains non-ASCII rune %U", g, r)
			}
		}
	}
}

// TestBuildThemeFillsWidgetStyles pins buildTheme's half of the theme ->
// widget.Styler pipeline: every named palette must actually populate
// widgetStyles (not the zero value), and two different palettes must
// resolve to different colors. The other half — that renderLeaf actually
// pushes it into a widget every frame — is styler_probe_test.go, since that
// needs a live widget instance, not just buildTheme's return value.
func TestBuildThemeFillsWidgetStyles(t *testing.T) {
	gv := buildTheme("gruvbox", "rounded")
	if c, ok := gv.widgetStyles.Accent.(lipgloss.Color); !ok || string(c) != "#fabd2f" {
		t.Fatalf("gruvbox widgetStyles.Accent = %#v, want lipgloss.Color(#fabd2f)", gv.widgetStyles.Accent)
	}
	if c, ok := gv.widgetStyles.Success.(lipgloss.Color); !ok || string(c) != "#b8bb26" {
		t.Fatalf("gruvbox widgetStyles.Success = %#v, want lipgloss.Color(#b8bb26)", gv.widgetStyles.Success)
	}

	sl := buildTheme("slate", "rounded")
	if gv.widgetStyles.Accent == sl.widgetStyles.Accent {
		t.Fatal("widgetStyles.Accent identical for gruvbox and slate")
	}

	// auto stays adaptive here too, same rule as borderFocus above.
	au := buildTheme("auto", "rounded")
	if _, ok := au.widgetStyles.Accent.(lipgloss.AdaptiveColor); !ok {
		t.Fatalf("auto widgetStyles.Accent = %#v, want AdaptiveColor", au.widgetStyles.Accent)
	}
}
