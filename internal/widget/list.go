package widget

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("list", newList) }

// listItem is a plain label — the list widget's config is just a slice of
// strings (Pane.Items), so it needs no richer Item type.
type listItem string

func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return string(i) }

// listWidget is a browsable, filterable list built directly on
// bubbles/list — it already owns scrolling, pagination and "/"-filtering, so
// there is nothing for the shared scrollBox (scroll.go) to add.
type listWidget struct {
	l list.Model
}

func newList(p *config.Pane) (Widget, error) {
	items := make([]list.Item, len(p.Items))
	for i, s := range p.Items {
		items[i] = listItem(s)
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false // one line per item; listItem has no description
	l := list.New(items, delegate, 0, 0)
	l.Title = p.Title
	l.SetShowTitle(false)     // set above for correctness, but suppressed: the pane's own bordered title (chrome.go) already shows it
	l.SetShowStatusBar(false) // redundant with the pane's title/border chrome
	// bubbles/list binds its own Quit/ForceQuit keys (default "q"/"ctrl+c")
	// straight to tea.Quit — left enabled, a keypress inside this pane would
	// kill the whole program instead of just this widget. Never re-enable
	// without also intercepting KeyMap.Quit/ForceQuit some other way.
	l.DisableQuitKeybindings()
	return &listWidget{l: l}, nil
}

func (w *listWidget) Init() tea.Cmd { return nil }

// Update forwards everything unconditionally — see inputWidget.Update for
// why (the filter box's own blink tick needs the same treatment).
func (w *listWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	var cmd tea.Cmd
	w.l, cmd = w.l.Update(msg)
	return w, cmd
}

func (w *listWidget) View(width, height int) string {
	w.l.SetSize(width, height)
	return w.l.View()
}
