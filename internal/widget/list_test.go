package widget

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestListWidgetShowsItems(t *testing.T) {
	w, err := newList(&config.Pane{Type: "list", Items: []string{"alpha", "beta", "gamma"}})
	if err != nil {
		t.Fatalf("newList: %v", err)
	}
	got := w.View(40, 10)
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(got, want) {
			t.Fatalf("view missing item %q:\n%s", want, got)
		}
	}
}

func TestListWidgetDownMovesSelection(t *testing.T) {
	w, _ := newList(&config.Pane{Type: "list", Items: []string{"alpha", "beta", "gamma"}})
	_ = w.View(40, 10) // SetSize before Update, matching how the engine drives it
	lw := w.(*listWidget)
	if lw.l.Index() != 0 {
		t.Fatalf("start index = %d, want 0", lw.l.Index())
	}
	w, _ = w.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := w.(*listWidget).l.Index(); got != 1 {
		t.Fatalf("index after down = %d, want 1", got)
	}
}

// bubbles/list binds "q" and "ctrl+c" straight to tea.Quit by default — that
// must be disabled (see newList) or a keypress inside this pane would kill
// the whole program, not just the widget.
func TestListWidgetQuitKeybindingsDisabled(t *testing.T) {
	w, _ := newList(&config.Pane{Type: "list", Items: []string{"alpha"}})
	_ = w.View(40, 10)
	_, cmd := w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		return
	}
	if _, isQuit := cmd().(tea.QuitMsg); isQuit {
		t.Fatal("'q' inside the list widget produced tea.Quit")
	}
}
