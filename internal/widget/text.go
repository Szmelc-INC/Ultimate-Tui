package widget

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("text", newText) }

// textWidget renders a fixed string, scrollable if it overflows the pane.
// SetZoom (widget.Zoomer) re-renders it in big block glyphs (bigfont.go)
// instead of resizing the pane — a per-pane content zoom independent of how
// much screen space the pane itself occupies.
type textWidget struct {
	box  scrollBox
	raw  string
	zoom int
}

func newText(p *config.Pane) (Widget, error) {
	t := &textWidget{raw: p.Content}
	t.box.setText(p.Content)
	return t, nil
}

func (t *textWidget) Init() tea.Cmd { return nil }

func (t *textWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		t.box.handleKey(m)
	case tea.MouseMsg:
		t.box.handleMouse(m)
	}
	return t, nil
}

func (t *textWidget) View(width, height int) string {
	return t.box.view(width, height)
}

func (t *textWidget) ScrollFraction() float64 { return t.box.scrollFraction() }

// SetZoom implements widget.Zoomer: 0 is the plain, unscaled content; a
// higher level re-renders it as big block glyphs (bigfont.go) at that scale.
func (t *textWidget) SetZoom(level int) {
	if level == t.zoom {
		return
	}
	t.zoom = level
	if level <= 0 {
		t.box.setText(t.raw)
		return
	}
	t.box.setText(renderBig(t.raw, level))
}
