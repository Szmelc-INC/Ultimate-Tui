package widget

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestFormWidgetRequiresFields(t *testing.T) {
	if _, err := newForm(&config.Pane{Type: "form"}); err == nil {
		t.Fatal("expected error when fields is empty")
	}
}

func TestFormWidgetRequiresFieldKey(t *testing.T) {
	_, err := newForm(&config.Pane{Type: "form", Fields: []config.FormField{{Label: "Name"}}})
	if err == nil {
		t.Fatal("expected error when a field has no key")
	}
}

func TestFormWidgetRejectsDuplicateKeys(t *testing.T) {
	fields := []config.FormField{{Key: "x"}, {Key: "x"}}
	if _, err := newForm(&config.Pane{Type: "form", Fields: fields}); err == nil {
		t.Fatal("expected error on duplicate field key")
	}
}

func TestFormWidgetSelectRequiresOptions(t *testing.T) {
	fields := []config.FormField{{Key: "x", Type: "select"}}
	if _, err := newForm(&config.Pane{Type: "form", Fields: fields}); err == nil {
		t.Fatal("expected error when a select field has no options")
	}
}

func TestFormWidgetRendersFieldLabel(t *testing.T) {
	fields := []config.FormField{{Key: "name", Label: "Your Name"}}
	w, err := newForm(&config.Pane{Type: "form", Fields: fields})
	if err != nil {
		t.Fatalf("newForm: %v", err)
	}
	_ = w.Init()
	if got := w.View(40, 10); !strings.Contains(got, "Your Name") {
		t.Fatalf("view missing field label:\n%s", got)
	}
}

// Typing into a single-field text form and pressing Enter submits it (huh's
// default keymap binds Enter to both "next field" and "submit" — on the
// only/last field that's a submit) — the summary view then shows the typed
// value, since there's no scripting layer yet for the widget to act on it.
func TestFormWidgetSubmitShowsSummary(t *testing.T) {
	fields := []config.FormField{{Key: "name", Label: "Name"}}
	w, err := newForm(&config.Pane{Type: "form", Fields: fields})
	if err != nil {
		t.Fatalf("newForm: %v", err)
	}
	_ = w.View(40, 10) // sizes the form before feeding it keys, like the engine would
	w = drainForm(t, w, w.Init())

	// Discard the cmds from each letter keystroke: they only
	// carry the cursor's cosmetic blink re-arm (a real cmd.Tick, so draining
	// it would burn real wall-clock time for no test value — see drainForm).
	// The value itself is already committed by the direct Update() call.
	for _, r := range "Ada" {
		w, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// The Enter that submits the only field is the one drain that matters:
	// huh moves to "next group" (and, being the last group, completes the
	// form) via its own chained cmd/msg, which a real bubbletea run loop
	// would drain automatically.
	next, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	w = drainForm(t, next, cmd)

	fw := w.(*formWidget)
	if fw.form.State != huh.StateCompleted {
		t.Fatalf("form did not complete after submitting its only field (state=%v)", fw.form.State)
	}
	got := w.View(40, 10)
	if !strings.Contains(got, "name: Ada") {
		t.Fatalf("summary missing submitted value:\n%s", got)
	}
}

// After submission huh.Form has no built-in way back to a blank form — the
// context menu's Restart row (widget.Restarter) is the only way out of the
// summary view, so it must actually produce a fresh, unsubmitted form.
func TestFormWidgetRestartProducesFreshForm(t *testing.T) {
	fields := []config.FormField{{Key: "name", Label: "Name"}}
	w, err := newForm(&config.Pane{Type: "form", Fields: fields})
	if err != nil {
		t.Fatalf("newForm: %v", err)
	}
	_ = w.View(40, 10)
	w = drainForm(t, w, w.Init())
	for _, r := range "Ada" {
		w, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	next, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	w = drainForm(t, next, cmd)

	fw := w.(*formWidget)
	if fw.form.State != huh.StateCompleted {
		t.Fatal("precondition: form should be completed before Restart")
	}

	r, ok := w.(interface{ Restart() tea.Cmd })
	if !ok {
		t.Fatal("formWidget does not implement Restarter")
	}
	initCmd := r.Restart()
	if fw.form.State == huh.StateCompleted {
		t.Fatal("Restart did not produce a fresh, unsubmitted form")
	}
	got := w.View(40, 10)
	if strings.Contains(got, "Ada") || strings.Contains(got, "submitted:") {
		t.Fatalf("restarted form still shows the previous submission:\n%s", got)
	}
	if initCmd == nil {
		t.Fatal("Restart should return the fresh form's Init cmd")
	}
}

// drainForm executes a returned tea.Cmd and feeds its message back into the
// widget, repeating for whatever cmd that produces (huh chains its own
// internal messages — e.g. submitting a field returns a cmd that emits a
// "move to next group" message, which itself may return another cmd — a
// real bubbletea run loop drains this automatically; a direct widget test
// has to do it by hand). tea.BatchMsg cmds are unwrapped and drained
// individually, each with its own small budget below.
//
// The cap is deliberately tight (not the dozens of steps a generous bound
// would allow): the cursor's own blink cmd re-arms itself forever by design,
// and each hop is a real tea.Tick — cmd() really sleeps for the blink
// interval, it's not simulated. An unbounded or generous drain here doesn't
// fail, it just burns real wall-clock time chasing a cosmetic animation that
// never terminates (this is what made an earlier version of this test take
// ~30s). The state transition this test actually cares about (huh's
// "move to next group" chain) resolves in one or two hops, so a small cap
// finds it quickly while still bounding the blink branch's cost.
//
// If TestFormWidgetSubmitShowsSummary starts failing with "form did not
// complete" after a huh upgrade, the real chain likely got one hop longer —
// raise this cap to match, don't delete the bound (that's what caused the
// ~30s version of this test in the first place).
func drainForm(t *testing.T, w Widget, cmd tea.Cmd) Widget {
	t.Helper()
	for i := 0; cmd != nil && i < 3; i++ {
		msg := cmd()
		var next tea.Cmd
		switch m := msg.(type) {
		case tea.BatchMsg:
			for _, c := range m {
				w = drainForm(t, w, c)
			}
			return w
		case nil:
			return w
		default:
			w, next = w.Update(m)
		}
		cmd = next
	}
	return w
}
