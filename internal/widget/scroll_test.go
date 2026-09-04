package widget

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
)

func longBody() string {
	var b strings.Builder
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&b, "row %02d unique marker text\n", i)
	}
	return b.String()
}

func TestScrollBoxViewportScrollsAndReportsFraction(t *testing.T) {
	w, _ := newText(&config.Pane{Type: "text", Content: longBody()})

	first := w.View(30, 8)
	if lines := strings.Split(first, "\n"); len(lines) != 8 {
		t.Fatalf("view height = %d, want 8", len(lines))
	}
	for i, ln := range strings.Split(first, "\n") {
		if x := lipgloss.Width(ln); x != 30 {
			t.Fatalf("line %d width = %d, want 30", i, x)
		}
	}

	sp, ok := w.(ScrollProgress)
	if !ok {
		t.Fatal("text widget should implement ScrollProgress")
	}
	if f := sp.ScrollFraction(); f != 0 {
		t.Fatalf("fraction at top = %v, want 0", f)
	}

	for i := 0; i < 40; i++ {
		w, _ = w.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	_ = w.View(30, 8)
	if w.View(30, 8) == first {
		t.Fatal("viewport did not scroll after 40x down")
	}
	if f := sp.ScrollFraction(); f <= 0 {
		t.Fatalf("fraction after scrolling down = %v, want > 0", f)
	}

	// 'g' jumps back to the top.
	w, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if w.View(30, 8) != first {
		t.Fatal("'g' did not return to the top")
	}
}
