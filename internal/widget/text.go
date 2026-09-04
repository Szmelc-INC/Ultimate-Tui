package widget

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("text", newText) }

// textWidget renders a fixed string, scrollable if it overflows the pane.
type textWidget struct {
	box scrollBox
}

func newText(p *config.Pane) (Widget, error) {
	t := &textWidget{}
	t.box.setText(p.Content)
	return t, nil
}

func (t *textWidget) Init() tea.Cmd { return nil }

func (t *textWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		t.box.handleKey(key)
	}
	return t, nil
}

func (t *textWidget) View(width, height int) string {
	return t.box.view(width, height)
}
