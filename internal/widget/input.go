package widget

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("input", newInput) }

// inputWidget is a single-line editable field, the first widget built
// directly on a bubbles component (bubbles/textinput) rather than the shared
// scrollBox (scroll.go) — there is nothing to scroll, and textinput already
// owns cursor movement, editing and its own blink animation. Content is the
// initial value, Placeholder shows while empty. There is no scripting/
// command layer yet for a widget to act on what's typed (see DESIGN.md's
// "total modularity" goal) — this is the building block a later
// command-templating or form-submission feature would read from.
type inputWidget struct {
	ti textinput.Model
}

func newInput(p *config.Pane) (Widget, error) {
	ti := textinput.New()
	ti.SetValue(p.Content)
	ti.Placeholder = p.Placeholder
	ti.Focus()
	return &inputWidget{ti: ti}, nil
}

func (w *inputWidget) Init() tea.Cmd { return textinput.Blink }

// Update forwards every message, not just key/mouse — the cursor's blink
// tick (kicked off by Init's textinput.Blink) is a bespoke message type that
// only textinput.Update recognizes, and the engine's broadcast path
// (engine.go's Update default case) delivers it here alongside every other
// widget's messages. textinput.Update is a no-op on anything it doesn't
// recognize (and unconditionally so while unfocused), so forwarding
// everything unconditionally is safe.
func (w *inputWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	var cmd tea.Cmd
	w.ti, cmd = w.ti.Update(msg)
	return w, cmd
}

func (w *inputWidget) View(width, height int) string {
	if pw := width - lipgloss.Width(w.ti.Prompt); pw > 0 {
		w.ti.Width = pw // textinput.Width is the visible viewport, excluding the prompt drawn alongside it
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(w.ti.View())
}
