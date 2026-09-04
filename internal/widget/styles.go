package widget

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Styles is the resolved color set for one theme, pushed into every widget
// that implements Styler once per frame — the same idiom this package
// already uses for size (list.SetSize, form.WithWidth/Height, textinput.Width
// are all set in View, not from a one-time message) applied to color instead.
// Defined here rather than in engine because widget cannot import engine
// (see CLAUDE.md's shell.go exit-bar exception, the one place that already
// hits this wall); engine's theme.go builds one of these from the resolved
// palette and renderLeaf pushes it every frame, so a live theme-picker
// preview (F2/F3, invalidateTheme) recolors every widget with no rebuild.
type Styles struct {
	Accent  lipgloss.TerminalColor // primary accent — focus, selection
	Accent2 lipgloss.TerminalColor // secondary accent — filter/cursor affordances
	Text    lipgloss.TerminalColor // normal foreground
	Subtle  lipgloss.TerminalColor // dimmed/inactive foreground
	Warn    lipgloss.TerminalColor // errors, destructive affordances
	Success lipgloss.TerminalColor // positive/confirm affordances
	ModeFg  lipgloss.TerminalColor // foreground for text drawn on an Accent-coloured chip
}

// key is a comparable fingerprint of s, used by widgets that memoize the
// last-pushed Styles to skip rebuilding derived lipgloss styles on a frame
// where the theme hasn't changed (SetStyles fires every frame regardless).
// Deliberately NOT a plain `==` on the struct: every field is a
// lipgloss.TerminalColor interface, and `==` on an interface panics at
// runtime if both sides hold the same non-comparable dynamic type — true
// today (every palette in theme.go uses the comparable Color/AdaptiveColor),
// but not a guarantee a future TerminalColor implementation should be able
// to break by accident. fmt.Sprint never panics.
func (s Styles) key() string {
	return fmt.Sprint(s.Accent, s.Accent2, s.Text, s.Subtle, s.Warn, s.Success, s.ModeFg)
}

// Styler is an optional interface for a widget that wants the resolved
// theme colors pushed into it every frame. A widget that doesn't implement
// it just keeps whatever default styling its underlying component ships
// with (or, for text/command/markdown, renders user content unstyled on
// purpose — colorizing that is where "pretty" becomes "wrong").
type Styler interface {
	SetStyles(Styles)
}

// Compile-time reminder that every themed widget actually implements Styler
// — a rename of SetStyles on any of these would otherwise fail silently
// (interface satisfaction, not a call), reverting that widget to its
// underlying component's stock colors with a still-green test suite.
var (
	_ Styler = (*listWidget)(nil)
	_ Styler = (*inputWidget)(nil)
	_ Styler = (*formWidget)(nil)
	_ Styler = (*shellWidget)(nil)
)
