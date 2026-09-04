package widget

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
)

const mdDoc = "# Big Heading\n\nA paragraph with **bold** and `code`.\n\n" +
	"- one\n- two\n- three\n\n## Second Heading\n\n" +
	"```go\nfunc main() {}\n```\n\n> a quote\n\nlast line\n"

func TestMarkdownWidgetRendersAndShapes(t *testing.T) {
	w, err := newMarkdown(&config.Pane{Type: "markdown", Content: mdDoc, MarkdownStyle: "notty"})
	if err != nil {
		t.Fatalf("newMarkdown: %v", err)
	}
	view := w.View(50, 12)

	lines := strings.Split(view, "\n")
	if len(lines) != 12 {
		t.Fatalf("height = %d, want 12", len(lines))
	}
	for i, ln := range lines {
		if lw := lipgloss.Width(ln); lw != 50 {
			t.Fatalf("line %d width = %d, want 50", i, lw)
		}
	}
	if !strings.Contains(stripANSI(view), "Big Heading") {
		t.Fatalf("heading missing from render:\n%s", view)
	}
}

func TestMarkdownWidgetScrolls(t *testing.T) {
	w, err := newMarkdown(&config.Pane{Type: "markdown", Content: mdDoc, MarkdownStyle: "notty"})
	if err != nil {
		t.Fatal(err)
	}
	top := w.View(50, 6)
	for i := 0; i < 8; i++ {
		w, _ = w.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if w.View(50, 6) == top {
		t.Fatal("markdown view did not scroll after 8x down")
	}
}

func TestMarkdownWidgetNeedsSource(t *testing.T) {
	if _, err := newMarkdown(&config.Pane{Type: "markdown"}); err == nil {
		t.Fatal("expected an error when neither content nor file is set")
	}
}
