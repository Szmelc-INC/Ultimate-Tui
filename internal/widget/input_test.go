package widget

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestInputWidgetShowsInitialContentAndPlaceholder(t *testing.T) {
	w, err := newInput(&config.Pane{Type: "input", Content: "hello"})
	if err != nil {
		t.Fatalf("newInput: %v", err)
	}
	if got := w.View(40, 3); !strings.Contains(got, "hello") {
		t.Fatalf("view missing initial content: %q", got)
	}

	w2, _ := newInput(&config.Pane{Type: "input", Placeholder: "type here"})
	if got := w2.View(40, 3); !strings.Contains(got, "type here") {
		t.Fatalf("view missing placeholder: %q", got)
	}
}

func TestInputWidgetTypingEditsValue(t *testing.T) {
	w, _ := newInput(&config.Pane{Type: "input"})
	for _, r := range "hi" {
		w, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if got := w.(*inputWidget).ti.Value(); got != "hi" {
		t.Fatalf("typed value = %q, want %q", got, "hi")
	}
	if got := w.View(40, 3); !strings.Contains(got, "hi") {
		t.Fatalf("view does not show typed text: %q", got)
	}
}

func TestInputWidgetBackspaceEditsValue(t *testing.T) {
	w, _ := newInput(&config.Pane{Type: "input", Content: "abc"})
	w, _ = w.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := w.(*inputWidget).ti.Value(); got != "ab" {
		t.Fatalf("value after backspace = %q, want %q", got, "ab")
	}
}
