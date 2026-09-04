package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/engine"
)

// openEditor: right-click / mod+w on the focused pane, then Enter on the menu
// (the cursor sits on "Edit" because the process rows are greyed for a text pane).
func openEditor(t *testing.T, m *engine.Model) *engine.Model {
	t.Helper()
	_ = m.View()
	u, _ := m.Update(modKey('w'))
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
