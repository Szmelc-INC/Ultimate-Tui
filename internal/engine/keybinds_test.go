package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

// Pins the "mod+" rewrite: the whole modifier contract rests on this encoding.
func TestModKeyEncoding(t *testing.T) {
	if got := (tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q"), Alt: true}).String(); got != "alt+q" {
		t.Fatalf("alt+q encodes as %q", got)
	}
}

const gridSrc = `{
  "name": "grid",
  "layout": { "direction": "column", "children": [
    { "size": "50%", "direction": "row", "children": [
      { "pane": { "type": "text", "title": "TL", "content": "a" } },
      { "pane": { "type": "text", "title": "TR", "content": "b" } }
    ]},
    { "size": "*", "direction": "row", "children": [
      { "pane": { "type": "text", "title": "BL", "content": "c" } },
      { "pane": { "type": "text", "title": "BR", "content": "d" } }
    ]}
  ]}
}`

func TestArrowsMoveFocusDirectionally(t *testing.T) {
	m := newModel(t, gridSrc, 80, 24)
	_ = m.View() // stamp rects

	step := func(k tea.KeyType) { u, _ := m.Update(tea.KeyMsg{Type: k}); m = u.(*engine.Model) }

	if m.Focused() != "TL" {
		t.Fatalf("start focus = %q", m.Focused())
	}
	step(tea.KeyRight)
	if m.Focused() != "TR" {
		t.Fatalf("right -> %q, want TR", m.Focused())
	}
	step(tea.KeyDown)
	if m.Focused() != "BR" {
		t.Fatalf("down -> %q, want BR", m.Focused())
	}
	step(tea.KeyLeft)
	if m.Focused() != "BL" {
		t.Fatalf("left -> %q, want BL", m.Focused())
	}
	step(tea.KeyUp)
	if m.Focused() != "TL" {
		t.Fatalf("up -> %q, want TL", m.Focused())
	}
}

func TestEnterLocksEscReleasesOnTextPane(t *testing.T) {
	m := newModel(t, gridSrc, 80, 24)
	_ = m.View()

	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !m.Locked() || m.Mode() != "LOCKED" {
		t.Fatalf("Enter did not lock (mode=%q)", m.Mode())
	}
	// Arrow now goes to the widget; focus must not move.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	if m.Focused() != "TL" {
		t.Fatalf("locked: arrow moved focus to %q", m.Focused())
	}
	// Esc releases (text pane: no vim to protect, immediate).
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if m.Locked() {
		t.Fatal("Esc did not release the lock")
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	if m.Focused() != "TR" {
		t.Fatalf("after release, arrow -> %q, want TR", m.Focused())
	}
}

func TestF1HelpOverlayToggles(t *testing.T) {
	m := newModel(t, gridSrc, 90, 26)
	base := m.View()
	if strings.Contains(base, "ULTIMATE TUI — keys") {
		t.Fatal("help text visible before F1")
	}

	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m = u.(*engine.Model)
	help := m.View()
	if !strings.Contains(help, "ULTIMATE TUI — keys") {
		t.Fatalf("F1 did not open help:\n%s", help)
	}
	// Frame geometry must survive the overlay splice.
	lines := strings.Split(help, "\n")
	if len(lines) != 26 {
		t.Fatalf("help frame height = %d, want 26", len(lines))
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != 90 {
			t.Fatalf("help frame line %d width = %d, want 90", i, w)
		}
	}

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "ULTIMATE TUI — keys") {
		t.Fatal("Esc did not close help")
	}
}

func TestF1HelpOverlayGeometryAtNarrowSize(t *testing.T) {
	// The help panel is wider than the frame here — overlayCenter must clamp,
	// not panic, and must preserve the exact frame geometry.
	m := newModel(t, gridSrc, 40, 12)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m = u.(*engine.Model)
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != 12 {
		t.Fatalf("narrow help frame height = %d, want 12", len(lines))
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != 40 {
			t.Fatalf("narrow help frame line %d width = %d, want 40", i, w)
		}
	}
}

func TestStatusBarPresentAndSized(t *testing.T) {
	m := newModel(t, gridSrc, 100, 20)
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != 20 {
		t.Fatalf("height = %d, want 20", len(lines))
	}
	last := lines[len(lines)-1]
	if lipgloss.Width(last) != 100 {
		t.Fatalf("status bar width = %d, want 100", lipgloss.Width(last))
	}
	if !strings.Contains(last, "NAV") || !strings.Contains(last, "quit") {
		t.Fatalf("status bar missing mode/hints:\n%q", last)
	}
}

func TestStatusBarDisabledViaOptions(t *testing.T) {
	cfg, err := config.Parse([]byte(gridSrc))
	if err != nil {
		t.Fatal(err)
	}
	opts := engine.DefaultOptions()
	opts.StatusBar = false
	m, err := engine.NewWithOptions(cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m = u.(*engine.Model)
	v := m.View()
	if strings.Contains(v, "NAV") {
		t.Fatalf("status bar rendered despite StatusBar=false:\n%s", v)
	}
	if n := len(strings.Split(v, "\n")); n != 20 {
		t.Fatalf("height = %d, want 20", n)
	}
}

func TestGlobalConfigModOverride(t *testing.T) {
	gc, err := config.ParseGlobal([]byte(
		`{"mod":"ctrl","statusbar":false,"keybindings":{"mod+n":"focus-next"}}`))
	if err != nil {
		t.Fatal(err)
	}
	opts := engine.OptionsFromGlobal(gc)
	if opts.Mod != "ctrl" || opts.StatusBar {
		t.Fatalf("OptionsFromGlobal = %+v", opts)
	}

	cfg, _ := config.Parse([]byte(gridSrc))
	m, err := engine.NewWithOptions(cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = u.(*engine.Model)
	_ = m.View()

	// "mod+n" with mod=ctrl expands to ctrl+n; it must be an intercepted global
	// binding (works even without a stamped rect / while locked) and advance focus.
	if m.Focused() != "TL" {
		t.Fatalf("start focus = %q", m.Focused())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = u.(*engine.Model)
	if m.Focused() == "TL" {
		t.Fatal("ctrl+n did not act as the mod-expanded focus-next binding")
	}
}
