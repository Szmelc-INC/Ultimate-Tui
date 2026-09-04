package widget

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
)

// pumps the shell widget's read loop synchronously until cond holds or it times
// out. Mirrors what engine.Snapshot's drain does.
func pump(t *testing.T, w Widget, cond func(Widget) bool) Widget {
	t.Helper()
	cmd := w.Init()
	deadline := time.Now().Add(3 * time.Second)
	for cmd != nil && time.Now().Before(deadline) {
		if cond(w) {
			return w
		}
		msg := cmd()
		w, cmd = w.Update(msg)
	}
	return w
}

func TestShellWidgetRunsProcessAndRendersOutput(t *testing.T) {
	w, err := newShell(&config.Pane{
		Type:    "shell",
		Command: "printf 'PTY-MARKER-42'; sleep 0.2",
		Shell:   "sh",
	})
	if err != nil {
		t.Fatalf("newShell: %v", err)
	}
	defer w.(Closer).Close()

	w = pump(t, w, func(w Widget) bool {
		return strings.Contains(stripANSI(w.View(40, 6)), "PTY-MARKER-42")
	})

	view := w.View(40, 6)
	if !strings.Contains(stripANSI(view), "PTY-MARKER-42") {
		t.Fatalf("process output not in view:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got != 6 {
		t.Fatalf("view height = %d, want 6", got)
	}
	for i, ln := range strings.Split(view, "\n") {
		if lw := lipgloss.Width(ln); lw != 40 {
			t.Fatalf("line %d width = %d, want 40", i, lw)
		}
	}
}

func TestShellWidgetForwardsKeystrokes(t *testing.T) {
	// `cat` echoes stdin back through the pty; typing must reach it.
	w, err := newShell(&config.Pane{Type: "shell", Command: "cat", Shell: "sh"})
	if err != nil {
		t.Fatal(err)
	}
	defer w.(Closer).Close()

	cmd := w.Init() // starts cat, returns first read
	w, _ = w.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("xyz")})
	w, _ = w.Update(tea.KeyMsg{Type: tea.KeyEnter})

	deadline := time.Now().Add(3 * time.Second)
	for cmd != nil && time.Now().Before(deadline) {
		if strings.Contains(stripANSI(w.View(40, 5)), "xyz") {
			break
		}
		msg := cmd()
		w, cmd = w.Update(msg)
	}
	if !strings.Contains(stripANSI(w.View(40, 5)), "xyz") {
		t.Fatalf("keystrokes not echoed by cat:\n%s", w.View(40, 5))
	}
}

func TestShellWidgetConsumesRawKeysUntilExit(t *testing.T) {
	w, _ := newShell(&config.Pane{Type: "shell", Command: "true", Shell: "sh"})
	defer w.(Closer).Close()

	if rk, ok := w.(RawKeyConsumer); !ok || !rk.ConsumesRawKeys() {
		t.Fatal("shell widget should consume raw keys while running")
	}
	w = pump(t, w, func(w Widget) bool {
		rk, _ := w.(RawKeyConsumer)
		return !rk.ConsumesRawKeys()
	})
	if rk, _ := w.(RawKeyConsumer); rk.ConsumesRawKeys() {
		t.Fatal("shell widget should release raw keys after the process exits")
	}
	if !strings.Contains(w.View(40, 3), "exited") {
		t.Fatalf("expected an exit notice:\n%s", w.View(40, 3))
	}
}

func TestShellWidgetRestartRespawns(t *testing.T) {
	w, err := newShell(&config.Pane{Type: "shell", Command: "printf 'GEN-A'; sleep 0.4", Shell: "sh"})
	if err != nil {
		t.Fatal(err)
	}
	s := w.(*shellWidget)
	defer s.Close()

	drive := func(cmd tea.Cmd, want string) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for cmd != nil && time.Now().Before(deadline) {
			if strings.Contains(stripANSI(s.View(40, 6)), want) {
				return
			}
			nw, next := s.Update(cmd())
			s, cmd = nw.(*shellWidget), next
		}
		if !strings.Contains(stripANSI(s.View(40, 6)), want) {
			t.Fatalf("never saw %q:\n%s", want, s.View(40, 6))
		}
	}

	drive(s.Init(), "GEN-A")

	oldID := s.id
	s.command = "printf 'GEN-B'; sleep 0.4"
	cmd := s.Restart()
	if s.id == oldID {
		t.Fatal("Restart did not bump the widget id")
	}
	// A stale read from the previous pty must be ignored by the id check.
	s.Update(shellOutMsg{id: oldID, data: []byte("STALE-XYZ")})
	if strings.Contains(stripANSI(s.View(40, 6)), "STALE-XYZ") {
		t.Fatal("stale shellOutMsg leaked into the emulator")
	}
	drive(cmd, "GEN-B")
}

func TestNormalizeGridExactShape(t *testing.T) {
	got := normalizeGrid("ab\n\n", 5, 3)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(lines))
	}
	for i, ln := range lines {
		if lipgloss.Width(ln) != 5 {
			t.Fatalf("line %d width = %d, want 5", i, lipgloss.Width(ln))
		}
	}
}

// stripANSI removes SGR/CSI sequences for content assertions.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' && s[i] != 'H' && s[i] != 'K' && s[i] != 'J' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
