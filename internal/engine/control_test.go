package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/engine"
)

func TestF2ControlPanelOpensAndCloses(t *testing.T) {
	m := newModel(t, gridSrc, 90, 26)
	if strings.Contains(m.View(), "controls — F2") {
		t.Fatal("control panel visible before F2")
	}
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF2})
	m = u.(*engine.Model)
	v := m.View()
	if !strings.Contains(v, "controls — F2") {
		t.Fatalf("F2 did not open the control panel:\n%s", v)
	}
	lines := strings.Split(v, "\n")
	if len(lines) != 26 {
		t.Fatalf("control frame height = %d, want 26", len(lines))
	}
	for i, ln := range lines {
		if lipgloss.Width(ln) != 90 {
			t.Fatalf("control frame line %d width = %d, want 90", i, lipgloss.Width(ln))
		}
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "controls — F2") {
		t.Fatal("Esc did not close the control panel")
	}
}

// F3 used to open the theme picker. It must now be inert at the engine level
// and fall through to the focused widget (regression: a locked vim keeps F3).
func TestF3DoesNotOpenAnOverlay(t *testing.T) {
	m := newModel(t, gridSrc, 90, 26)
	base := m.View()
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF3})
	m = u.(*engine.Model)
	if m.View() != base {
		t.Fatalf("F3 opened an overlay / changed the frame:\n%s", m.View())
	}
}
