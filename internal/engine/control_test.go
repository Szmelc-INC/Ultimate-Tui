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

// F3 opens the global settings panel, but only from NAV — while a pane is
// locked it must fall through to the focused widget untouched (a locked
// vim/shell keeps its own F3), the same guarantee F1/F2 do NOT make.
func TestF3OpensGlobalPanelOnlyInNav(t *testing.T) {
	m := newModel(t, gridSrc, 90, 26)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF3})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "global settings — F3") {
		t.Fatalf("F3 in NAV did not open the global panel:\n%s", m.View())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // lock the focused pane
	m = u.(*engine.Model)
	base := m.View()
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyF3})
	m = u.(*engine.Model)
	if m.View() != base {
		t.Fatalf("F3 while locked opened the global panel instead of falling through:\n%s", m.View())
	}
}
