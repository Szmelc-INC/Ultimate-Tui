package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/engine"
)

// openEditor: right-click / mod+w on the focused pane, jump to the menu's
// last row ("End" — Edit is always last and always enabled, unlike Close
// which greys out on a single-pane layout) and Enter.
func openEditor(t *testing.T, m *engine.Model) *engine.Model {
	t.Helper()
	_ = m.View()
	u, _ := m.Update(modKey('w'))
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return u.(*engine.Model)
}

func typeStr(m *engine.Model, s string) *engine.Model {
	for _, r := range s {
		u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = u.(*engine.Model)
	}
	return m
}

func TestEditorOpensWithTabsAndFields(t *testing.T) {
	m := newModel(t, ctxSrc, 100, 30)
	m = openEditor(t, m)
	v := m.View()
	for _, want := range []string{"edit · Note", "General", "Layout", "Widget", "type", "title", "border"} {
		if !strings.Contains(v, want) {
			t.Fatalf("editor panel missing %q:\n%s", want, v)
		}
	}
}

func TestEditorTitleEditAppliesToLivePane(t *testing.T) {
	m := newModel(t, ctxSrc, 100, 30)
	m = openEditor(t, m)

	// General tab, move to the "title" row (row 1) and append text.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = u.(*engine.Model)
	m = typeStr(m, " EDITED")

	// Apply, then close.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}, Alt: true})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)

	if m.Focused() != "Note EDITED" {
		t.Fatalf("focused pane title = %q, want %q", m.Focused(), "Note EDITED")
	}
	if !strings.Contains(m.View(), "Note EDITED") {
		t.Fatal("edited title not visible in the frame")
	}
}

func TestEditorRejectsUnbuildablePane(t *testing.T) {
	m := newModel(t, ctxSrc, 100, 30)
	m = openEditor(t, m)

	// "type" row (row 0). Cycle it off "text"; command/shell/markdown all need a
	// field the text pane hasn't got, so apply must surface an error, not crash.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}, Alt: true})
	m = u.(*engine.Model)

	if !strings.Contains(m.View(), "!") {
		t.Fatalf("expected an error marker in the editor panel:\n%s", m.View())
	}
	// Bail out — the pane must be untouched.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if m.Focused() != "Note" {
		t.Fatalf("rejected edit still changed the pane (focused = %q)", m.Focused())
	}
}

func TestConfigEditorFromF2(t *testing.T) {
	m := newModel(t, ctxSrc, 100, 30)
	_ = m.View()

	// F2 -> "Edit config" (second row) -> Enter.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF2})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)

	v := m.View()
	for _, want := range []string{"edit config · ctx", "Config", "General", "Widget", "name", "pane"} {
		if !strings.Contains(v, want) {
			t.Fatalf("config editor missing %q:\n%s", want, v)
		}
	}

	// Config tab, "pane" row: cycle to the next pane.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // row 0 (name) -> row 1 (pane)
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "Build") {
		t.Fatalf("pane selector did not advance to the second pane:\n%s", m.View())
	}

	// Esc backs out to the control panel, then closes it.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "controls — F2") {
		t.Fatal("Esc from the config editor did not return to F2")
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if strings.Contains(m.View(), "controls — F2") {
		t.Fatal("second Esc did not close the control panel")
	}
}

// assertFrame checks the rendered frame is exactly w x h — no line wider or
// narrower, no extra rows. The editor panel is the overlay most exposed to this
// because field values are arbitrary user text.
func assertFrame(t *testing.T, m *engine.Model, w, h int) {
	t.Helper()
	lines := strings.Split(m.View(), "\n")
	if len(lines) != h {
		t.Fatalf("frame height = %d, want %d", len(lines), h)
	}
	for i, ln := range lines {
		if x := lipgloss.Width(ln); x != w {
			t.Fatalf("frame line %d width = %d, want %d", i, x, w)
		}
	}
}

func TestEditorPanelGeometry(t *testing.T) {
	long := `{"name":"g","layout":{"pane":{"type":"text","title":"Big",
	  "content":"line one is quite long indeed\nline two\nline three also reasonably long here\nfour"}}}`

	for _, wh := range [][2]int{{100, 30}, {44, 14}} {
		m := newModel(t, long, wh[0], wh[1])
		m = openEditor(t, m)
		assertFrame(t, m, wh[0], wh[1])

		// Walk to the content field (Widget tab) and start typing — the active
		// text control must not leak the value's newlines into the panel.
		u, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = u.(*engine.Model)
		u, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = u.(*engine.Model)
		m = typeStr(m, "xyz")
		assertFrame(t, m, wh[0], wh[1])
	}

	// Error state: an unbuildable type change puts widget.New's message in the panel.
	m := newModel(t, ctxSrc, 80, 20)
	m = openEditor(t, m)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight}) // type off "text"
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}, Alt: true})
	m = u.(*engine.Model)
	assertFrame(t, m, 80, 20)
}

func TestEditorDirectionControlPresent(t *testing.T) {
	m := newModel(t, ctxSrc, 100, 30) // ctxSrc is a row split
	m = openEditor(t, m)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> Layout tab
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "direction") {
		t.Fatalf("Layout tab has no direction control:\n%s", m.View())
	}
}

func TestEditorTabNavigation(t *testing.T) {
	m := newModel(t, ctxSrc, 100, 30)
	m = openEditor(t, m)

	// Tab to "Widget": a text pane exposes a "content" field there.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = u.(*engine.Model)
	if !strings.Contains(m.View(), "content") {
		t.Fatalf("Widget tab did not show the content field:\n%s", m.View())
	}
}
