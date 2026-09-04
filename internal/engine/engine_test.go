package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

const twoPane = `{
  "name": "test",
  "layout": {
    "direction": "row",
    "children": [
      { "size": "40%", "pane": { "type": "text", "title": "Left",  "content": "LEFTBODY" } },
      { "size": "*",    "pane": { "type": "text", "title": "Right", "content": "RIGHTBODY" } }
    ]
  }
}`

func newModel(t *testing.T, src string, w, h int) *engine.Model {
	t.Helper()
	cfg, err := config.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(*engine.Model)
}

func TestRendersToExactSize(t *testing.T) {
	m := newModel(t, twoPane, 60, 20)
	view := m.View()

	lines := strings.Split(view, "\n")
	if len(lines) != 20 {
		t.Fatalf("height = %d lines, want 20\n%s", len(lines), view)
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != 60 {
			t.Fatalf("line %d width = %d, want 60", i, w)
		}
	}
	if !strings.Contains(view, "LEFTBODY") || !strings.Contains(view, "RIGHTBODY") {
		t.Fatalf("missing pane bodies:\n%s", view)
	}
	if !strings.Contains(view, "Left") || !strings.Contains(view, "Right") {
		t.Fatalf("missing pane titles:\n%s", view)
	}
}

func TestFocusTraversalWraps(t *testing.T) {
	m := newModel(t, twoPane, 60, 20)
	if got := m.Focused(); got != "Left" {
		t.Fatalf("initial focus = %q, want Left", got)
	}
	step := func() { u, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab}); m = u.(*engine.Model) }
	step()
	if got := m.Focused(); got != "Right" {
		t.Fatalf("after tab focus = %q, want Right", got)
	}
	step()
	if got := m.Focused(); got != "Left" {
		t.Fatalf("after 2x tab focus = %q, want Left (wrap)", got)
	}
}

func TestBeforeWindowSize(t *testing.T) {
	cfg, _ := config.Parse([]byte(twoPane))
	m, _ := engine.New(cfg)
	if m.View() != "loading…" {
		t.Fatalf("pre-size view = %q", m.View())
	}
}

func TestUnknownWidgetRejected(t *testing.T) {
	_, err := config.Parse([]byte(`{"name":"x","layout":{"pane":{"type":"nope"}}}`))
	if err != nil {
		return // rejected at parse is fine too
	}
	cfg, _ := config.Parse([]byte(`{"name":"x","layout":{"pane":{"type":"text"}}}`))
	cfg.Layout.Pane.Type = "nope"
	if _, err := engine.New(cfg); err == nil {
		t.Fatal("expected error for unknown widget type")
	}
}

func TestNestedColumnSplit(t *testing.T) {
	src := `{
      "name": "nested",
      "layout": {
        "direction": "row",
        "children": [
          { "size": "20", "pane": { "type": "text", "content": "SIDE" } },
          { "size": "*", "direction": "column", "children": [
            { "size": "50%", "pane": { "type": "text", "content": "TOP" } },
            { "size": "*",   "pane": { "type": "text", "content": "BOTTOM" } }
          ]}
        ]
      }
    }`
	m := newModel(t, src, 80, 24)
	view := m.View()
	for _, want := range []string{"SIDE", "TOP", "BOTTOM"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in:\n%s", want, view)
		}
	}
	if n := len(strings.Split(view, "\n")); n != 24 {
		t.Fatalf("height = %d, want 24", n)
	}
}
