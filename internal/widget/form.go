package widget

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("form", newForm) }

// formWidget hosts a huh.Form — the first widget built on the huh package
// rather than raw bubbles/lipgloss. huh.Form is itself a tea.Model
// (Init/Update/View), so embedding it needs no new engine machinery: it just
// slots into the Widget interface like any other. Fields are declared in
// config (config.FormField); there is no scripting/command layer yet for a
// widget to act on a submitted value, so completion just renders the
// collected values back — the building block a later "run this with these
// values" feature would read from. cfgFields is kept so Restart (the
// context menu's Restart row) can rebuild a fresh form after submission —
// without it, a huh.Form has no "go back to a blank form" of its own, and
// the pane would be permanently stuck showing the summary.
type formWidget struct {
	cfgFields []config.FormField
	form      *huh.Form
	keys      []string // field keys, in declared order, for a stable summary
	values    map[string]*string
	bools     map[string]*bool
}

func newForm(p *config.Pane) (Widget, error) {
	if len(p.Fields) == 0 {
		return nil, fmt.Errorf("form widget: \"fields\" is required")
	}
	w := &formWidget{cfgFields: p.Fields}
	if err := w.build(); err != nil {
		return nil, err
	}
	return w, nil
}

// build (re)constructs form/keys/values/bools from w.cfgFields — the shared
// core of newForm and Restart. huh.Form field bindings are pointers written
// on every keystroke (see Update), so a fresh Form needs fresh pointers too;
// reusing the old ones would let a restarted form's fields start pre-filled
// with the previous submission's values.
func (w *formWidget) build() error {
	w.values = map[string]*string{}
	w.bools = map[string]*bool{}
	w.keys = nil
	seen := map[string]bool{}
	fields := make([]huh.Field, 0, len(w.cfgFields))
	for _, f := range w.cfgFields {
		if f.Key == "" {
			return fmt.Errorf("form widget: field %q missing \"key\"", f.Label)
		}
		if seen[f.Key] {
			return fmt.Errorf("form widget: duplicate field key %q", f.Key)
		}
		seen[f.Key] = true
		label := f.Label
		if label == "" {
			label = f.Key
		}
		switch f.Type {
		case "confirm":
			v := new(bool)
			w.bools[f.Key] = v
			fields = append(fields, huh.NewConfirm().Title(label).Value(v))
		case "select":
			if len(f.Options) == 0 {
				return fmt.Errorf("form widget: select field %q needs \"options\"", f.Key)
			}
			v := new(string)
			w.values[f.Key] = v
			opts := make([]huh.Option[string], len(f.Options))
			for i, o := range f.Options {
				opts[i] = huh.NewOption(o, o)
			}
			fields = append(fields, huh.NewSelect[string]().Title(label).Options(opts...).Value(v))
		default:
			v := new(string)
			w.values[f.Key] = v
			fields = append(fields, huh.NewInput().Title(label).Value(v))
		}
		w.keys = append(w.keys, f.Key)
	}
	w.form = huh.NewForm(huh.NewGroup(fields...)).WithShowHelp(false)
	return nil
}

func (w *formWidget) Init() tea.Cmd { return w.form.Init() }

func (w *formWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	next, cmd := w.form.Update(msg)
	if f, ok := next.(*huh.Form); ok {
		w.form = f
	}
	return w, cmd
}

// View sizes the form to the pane every frame — huh.Form normally learns its
// size from a tea.WindowSizeMsg, but the engine never forwards that message
// to widgets (it's consumed at the top of engine.Model.Update to size the
// whole layout tree instead), so WithWidth/WithHeight stand in for it here,
// the same way listWidget.View calls list.Model.SetSize every render.
func (w *formWidget) View(width, height int) string {
	if width > 0 && height > 0 {
		w.form = w.form.WithWidth(width).WithHeight(height)
	}
	if w.form.State == huh.StateCompleted {
		return w.summary(width, height)
	}
	return w.form.View()
}

func (w *formWidget) summary(width, height int) string {
	var b strings.Builder
	b.WriteString("submitted:\n\n")
	for _, k := range w.keys {
		switch {
		case w.values[k] != nil:
			fmt.Fprintf(&b, "%s: %s\n", k, *w.values[k])
		case w.bools[k] != nil:
			fmt.Fprintf(&b, "%s: %v\n", k, *w.bools[k])
		}
	}
	b.WriteString("\n(context menu -> Restart for a blank form)")
	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

// Restart rebuilds a fresh, blank form from the original config — the only
// way back from the post-submission summary, since huh.Form has no built-in
// "reset" of its own. The config was already validated once in newForm, so
// the error return here is unreachable in practice; it's kept because build
// returns one, not swallowed silently.
func (w *formWidget) Restart() tea.Cmd {
	if err := w.build(); err != nil {
		return nil
	}
	return w.form.Init()
}
