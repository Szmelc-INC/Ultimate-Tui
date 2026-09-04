package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

// openF3 now reaches the theme/style picker the new way: F2 opens the control
// panel, Enter on its first row ("Theme & style") opens the picker.
func openF3(t *testing.T, m *engine.Model) *engine.Model {
	t.Helper()
	_ = m.View()
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF2})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return u.(*engine.Model)
}

func TestF3MenuTogglesAndKeepsFrameGeometry(t *testing.T) {
	m := openF3(t, newModel(t, gridSrc, 90, 26))
	v := m.View()
	if !strings.Contains(v, "settings — theme & style") {
		t.Fatalf("F2->Enter did not open the settings menu:\n%s", v)
	}
	lines := strings.Split(v, "\n")
	if len(lines) != 26 {
		t.Fatalf("settings frame height = %d, want 26", len(lines))
	}
	for i, ln := range lines {
		if lipgloss.Width(ln) != 90 {
			t.Fatalf("settings frame line %d width = %d, want 90", i, lipgloss.Width(ln))
		}
	}
	// Esc backs out to the control panel (one level), Esc again closes it.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "controls — F2") {
		t.Fatal("Esc from the picker did not return to the control panel")
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "controls — F2") || strings.Contains(m.View(), "settings — theme & style") {
		t.Fatal("Esc did not close the control panel")
	}
}

func TestF3MenuNarrowSizeDoesNotPanic(t *testing.T) {
	m := openF3(t, newModel(t, gridSrc, 38, 11))
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 11 {
		t.Fatalf("narrow settings frame height = %d, want 11", len(lines))
	}
	for i, ln := range lines {
		if lipgloss.Width(ln) != 38 {
			t.Fatalf("narrow settings frame line %d width = %d", i, lipgloss.Width(ln))
		}
	}
}

func TestF3MenuLivePreviewAndApply(t *testing.T) {
	m := newModel(t, gridSrc, 90, 26) // default style = rounded
	if !strings.Contains(m.View(), "╭") {
		t.Fatal("expected rounded corners before any preview")
	}
	m = openF3(t, m)

	// Move to the style column, then down to a non-default style until the
	// frame stops drawing rounded corners (live preview repaints under panel).
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	changed := false
	for i := 0; i < len(engine.StyleNames()); i++ {
		u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = u.(*engine.Model)
		if !strings.Contains(m.View(), "╭") {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("cycling the style column never changed the rendered border")
	}

	// Enter commits: the picker closes (back to the control panel) and the
	// previewed style stays. A second Esc drops the control panel entirely.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "settings — theme & style") {
		t.Fatal("Enter did not close the settings picker")
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "╭") {
		t.Fatal("committed style reverted to rounded after closing the menu")
	}
}

func TestF3MenuEscRestoresPreOpenTheme(t *testing.T) {
	m := newModel(t, gridSrc, 90, 26)
	before := m.View()
	m = openF3(t, m)

	// Preview every style, then cancel.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	for i := 0; i < len(engine.StyleNames()); i++ {
		u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = u.(*engine.Model)
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // picker -> control (restores theme)
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // control -> closed
	m = u.(*engine.Model)

	if strings.Contains(m.View(), "settings — theme & style") {
		t.Fatal("Esc did not close the settings menu")
	}
	if m.View() != before {
		t.Fatalf("Esc did not restore the pre-open appearance:\n--- before ---\n%s\n--- after ---\n%s", before, m.View())
	}
}

func TestF3MenuApplyCallsSaveCallback(t *testing.T) {
	cfg, err := config.Parse([]byte(gridSrc))
	if err != nil {
		t.Fatal(err)
	}
	var gotTheme, gotStyle string
	calls := 0
	opts := engine.DefaultOptions()
	opts.SaveTheme = func(themeName, styleName string) error {
		calls++
		gotTheme, gotStyle = themeName, styleName
		return nil
	}
	m, err := engine.NewWithOptions(cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 26})
	m = u.(*engine.Model)
	m = openF3(t, m)

	// theme column: step down once (auto -> first named theme), then apply.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)

	if calls != 1 {
		t.Fatalf("SaveTheme called %d times, want 1", calls)
	}
	if gotTheme != engine.ThemeNames()[1] || gotStyle != engine.StyleNames()[0] {
		t.Fatalf("SaveTheme got (%q,%q), want (%q,%q)",
			gotTheme, gotStyle, engine.ThemeNames()[1], engine.StyleNames()[0])
	}

	// Cancelling must NOT persist.
	m = openF3(t, m)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if calls != 1 {
		t.Fatalf("Esc-cancel persisted a choice (calls=%d)", calls)
	}
}
