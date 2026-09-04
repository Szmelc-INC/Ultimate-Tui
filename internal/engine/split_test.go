package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/engine"
)

// Full key-driven Split flow: mod+w, step to "Split", Enter opens the
// prompt, Enter again confirms the 50/50 default and the pane count grows.
func TestSplitFlowViaContextMenu(t *testing.T) {
	m := newModel(t, ctxSrc, 90, 26) // ctxSrc: 2 panes, "Note" focused
	_ = m.View()

	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "pane · Note") {
		t.Fatalf("mod+w did not open the context menu:\n%s", m.View())
	}
	// The cursor starts on the first enabled row for a text pane: Split.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "split · Note") {
		t.Fatalf("Enter on the first enabled row (Split) did not open the split prompt:\n%s", m.View())
	}

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm 50/50 row split
	m = u.(*engine.Model)

	if m.PaneCount() != 3 {
		t.Fatalf("pane count after split = %d, want 3", m.PaneCount())
	}
	if strings.Contains(m.View(), "split ·") {
		t.Fatal("split prompt still open after confirming")
	}
}

// Full key-driven Close flow: mod+w, step to "Close", Enter opens the
// confirm dialog, "y" confirms and the pane count shrinks.
func TestCloseFlowViaContextMenu(t *testing.T) {
	m := newModel(t, ctxSrc, 90, 26)
	_ = m.View()

	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Split -> Close
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "pane · Note") {
		t.Fatalf("cursor should be on the context menu list:\n%s", m.View())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "Close Note?") {
		t.Fatalf("Enter on Close did not open the confirm dialog:\n%s", m.View())
	}

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = u.(*engine.Model)

	if m.PaneCount() != 1 {
		t.Fatalf("pane count after close = %d, want 1", m.PaneCount())
	}
}

// Close is greyed out on a single-pane layout — Enter on it must do nothing.
func TestCloseGreyedOutOnLastPane(t *testing.T) {
	m := newModel(t, oneTextSrc, 80, 24)
	_ = m.View()

	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd}) // jump to the last row (Edit; Close is disabled)
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "edit ·") {
		t.Fatalf("End should have skipped disabled Close and landed on Edit:\n%s", m.View())
	}
}

// Right-click on dead space (the status bar row) opens the "New pane" menu
// instead of a per-pane one; confirming grows the pane count and wraps root.
func TestNewPaneFromDeadSpace(t *testing.T) {
	m := newModel(t, oneTextSrc, 90, 26) // statusbar on -> body rows 0..24, row 25 is dead space
	_ = m.View()

	u, _ := m.Update(tea.MouseMsg{X: 10, Y: 25, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "empty space") {
		t.Fatalf("right-click on the status bar did not open the New-pane menu:\n%s", m.View())
	}

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "new pane") {
		t.Fatalf("Enter on New did not open the split prompt:\n%s", m.View())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)

	if m.PaneCount() != 2 {
		t.Fatalf("pane count after New = %d, want 2", m.PaneCount())
	}
}

const oneTextSrc = `{"name":"x","layout":{"pane":{"type":"text","title":"Solo","content":"hi"}}}`
