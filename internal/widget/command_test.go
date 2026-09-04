package widget

import (
	"strings"
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestCommandWidgetCapturesOutput(t *testing.T) {
	w, err := newCommand(&config.Pane{Type: "command", Command: "printf 'MARKER-123'", Shell: "sh"})
	if err != nil {
		t.Fatalf("newCommand: %v", err)
	}

	// Before the command finishes, the pane shows a running notice.
	if got := w.View(40, 3); !strings.Contains(got, "running") {
		t.Fatalf("pre-run view = %q", got)
	}

	// Run the tea.Cmd synchronously and feed its message back.
	msg := w.Init()()
	w, _ = w.Update(msg)

	if got := w.View(40, 3); !strings.Contains(got, "MARKER-123") {
		t.Fatalf("post-run view missing output: %q", got)
	}
}

func TestCommandWidgetReportsExit(t *testing.T) {
	w, _ := newCommand(&config.Pane{Type: "command", Command: "exit 3", Shell: "sh"})
	w, _ = w.Update(w.Init()())
	if got := w.View(40, 3); !strings.Contains(got, "exit") {
		t.Fatalf("expected exit note, got %q", got)
	}
}

func TestCommandWidgetRequiresCommand(t *testing.T) {
	if _, err := newCommand(&config.Pane{Type: "command"}); err == nil {
		t.Fatal("expected error when command is empty")
	}
}

func TestCommandWidgetRestartReRuns(t *testing.T) {
	w, _ := newCommand(&config.Pane{Type: "command", Command: "printf 'FIRST'", Shell: "sh"})
	c := w.(*commandWidget)
	w, _ = w.Update(w.Init()())
	if got := w.View(40, 3); !strings.Contains(got, "FIRST") {
		t.Fatalf("first run view = %q", got)
	}

	// Swap the command and Restart: a fresh run, new id, stale msg ignored.
	oldID := c.id
	c.command = "printf 'SECOND'"
	cmd := c.Restart()
	if c.id == oldID {
		t.Fatal("Restart did not bump the widget id")
	}
	// A late message from the previous run must not overwrite the buffer.
	w, _ = w.Update(cmdDoneMsg{id: oldID, output: "STALE"})
	if got := w.View(40, 3); strings.Contains(got, "STALE") {
		t.Fatalf("stale cmdDoneMsg leaked into the view: %q", got)
	}
	w, _ = w.Update(cmd())
	if got := w.View(40, 3); !strings.Contains(got, "SECOND") {
		t.Fatalf("post-restart view = %q", got)
	}
}
