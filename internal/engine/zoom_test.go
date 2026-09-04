package engine

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/widget"
)

// recZoomWidget records every SetZoom call — used to pin that mod+=/-/0
// reaches the widget through widget.Zoomer, not that it resizes the pane
// (that's resize_test.go now; see item 10 in the request that split this).
type recZoomWidget struct{ zoom int }

func (r *recZoomWidget) Init() tea.Cmd                           { return nil }
func (r *recZoomWidget) Update(tea.Msg) (widget.Widget, tea.Cmd) { return r, nil }
func (r *recZoomWidget) View(w, h int) string                    { return "" }
func (r *recZoomWidget) SetZoom(level int)                       { r.zoom = level }

const zoomGridSrc = `{
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

// The whole content-zoom feature rests on bubbletea encoding Alt+= as
// "alt+=" so the "mod+=" baseBind resolves. Drive the real key path.
func TestContentZoomBindingUpdatesLeafAndWidget(t *testing.T) {
	cfg, err := config.Parse([]byte(zoomGridSrc))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewWithOptions(cfg, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = u.(*Model)

	fake := &recZoomWidget{}
	m.leaves[m.focus].w = fake

	press := func(r rune) {
		u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true})
		m = u.(*Model)
	}

	press('=')
	press('=')
	if z := m.leaves[m.focus].contentZoom; z != 2 {
		t.Fatalf("alt+= x2: contentZoom = %d, want 2", z)
	}
	if fake.zoom != 2 {
		t.Fatalf("widget did not receive SetZoom(2): got %d", fake.zoom)
	}
	press('-')
	if z := m.leaves[m.focus].contentZoom; z != 1 {
		t.Fatalf("alt+-: contentZoom = %d, want 1", z)
	}
	press('0')
	if z := m.leaves[m.focus].contentZoom; z != 0 {
		t.Fatalf("alt+0: contentZoom = %d, want 0", z)
	}
	if fake.zoom != 0 {
		t.Fatalf("widget did not receive SetZoom(0): got %d", fake.zoom)
	}
}

// Content zoom must never touch pane geometry — the frame stays exactly
// 80x24 regardless of how far a pane is zoomed, since it's the same rectangle
// rendering bigger (or unscaled) glyphs, not a bigger rectangle.
func TestContentZoomNeverResizesTheFrame(t *testing.T) {
	cfg, err := config.Parse([]byte(zoomGridSrc))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewWithOptions(cfg, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = u.(*Model)

	press := func(r rune) {
		u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true})
		m = u.(*Model)
	}
	press('=')
	press('=')
	press('=')
	v := m.View()
	if lines := splitLines(v); len(lines) != 24 {
		t.Fatalf("zoomed frame height = %d, want 24", len(lines))
	}
}

// The text widget is the one built-in Zoomer: SetZoom must actually change
// what it renders (big block glyphs), not just record a number.
func TestTextWidgetZoomChangesRenderedContent(t *testing.T) {
	w, err := widget.New(&config.Pane{Type: "text", Content: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	z, ok := w.(widget.Zoomer)
	if !ok {
		t.Fatal("text widget does not implement widget.Zoomer")
	}
	before := w.View(40, 10)
	z.SetZoom(2)
	after := w.View(40, 10)
	if before == after {
		t.Fatal("SetZoom(2) did not change the rendered content")
	}
	z.SetZoom(0)
	reset := w.View(40, 10)
	if reset != before {
		t.Fatal("SetZoom(0) did not restore the original rendering")
	}
}

func splitLines(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
