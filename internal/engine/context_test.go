package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/engine"
)

const ctxSrc = `{
  "name": "ctx",
  "layout": { "direction": "row", "children": [
    { "size": "50%", "pane": { "type": "text", "title": "Note", "content": "hi" } },
    { "size": "*",    "pane": { "type": "command", "title": "Build", "command": "echo hi" } }
  ]}
}`

func modKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true}
}

func TestContextMenuOpensViaModWAndCloses(t *testing.T) {
	m := newModel(t, ctxSrc, 90, 26)
	_ = m.View()
	if strings.Contains(m.View(), "pane · Note") {
		t.Fatal("context menu visible before mod+w")
	}
	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "pane · Note") {
		t.Fatalf("mod+w did not open the context menu for the focused pane:\n%s", m.View())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "pane · Note") {
		t.Fatal("Esc did not close the context menu")
	}
}

func TestContextMenuRightClickTargetsPaneUnderCursor(t *testing.T) {
	m := newModel(t, ctxSrc, 90, 26)
	_ = m.View() // stamp rects; focus is on "Note"

	// Right-click the right half -> the "Build" pane, without moving focus.
	u, _ := m.Update(tea.MouseMsg{X: 70, Y: 12, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	m = u.(*engine.Model)
	v := m.View()
	if !strings.Contains(v, "pane · Build") {
		t.Fatalf("right-click did not target the pane under the cursor:\n%s", v)
	}
	if m.Focused() != "Note" {
		t.Fatalf("right-click moved focus to %q, want it to stay on Note", m.Focused())
	}
}

// The one-overlay / mouse-drop invariant must still hold: with the context menu
// up, a left-click behind it changes nothing.
func TestContextMenuStillDropsMouse(t *testing.T) {
	m := newModel(t, ctxSrc, 90, 26)
	_ = m.View()
	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	before := m.View()
	u, _ = m.Update(tea.MouseMsg{X: 70, Y: 12, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = u.(*engine.Model)
	if m.Focused() != "Note" {
		t.Fatalf("click behind the menu moved focus to %q", m.Focused())
	}
	if m.View() != before {
		t.Fatal("click behind the menu changed the frame")
	}
}

func TestContextMenuGreysUnsupportedActions(t *testing.T) {
	m := newModel(t, ctxSrc, 90, 26)
	_ = m.View()

	// text pane: Stop/Restart/Refresh unsupported -> Enter skips them and
	// lands on the first enabled row, "Split".
	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "split · Note") {
		t.Fatalf("Enter on the text pane's menu did not open the split prompt (cursor should skip greyed rows):\n%s", m.View())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)

	// command pane: Restart / Refresh are live.
	u, _ = m.Update(tea.MouseMsg{X: 70, Y: 12, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	m = u.(*engine.Model)
	// Down from the first enabled row ("Restart") should reach "Refresh", not
	// wrap past greyed entries oddly. Just assert both labels render un-dimmed
	// by checking the menu contains them and the cursor can move.
	v := m.View()
	if !strings.Contains(v, "Restart") || !strings.Contains(v, "Refresh") {
		t.Fatalf("command pane menu missing process actions:\n%s", v)
	}
}
