package widget

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/silverx/ultimate-tui/internal/config"
)

// Proves a real full-screen ncurses app (htop) paints through the pty +
// emulator pipeline. Skipped if htop is not installed.
func TestShellWidgetRendersHtop(t *testing.T) {
	if _, err := exec.LookPath("htop"); err != nil {
		t.Skip("htop not installed")
	}
	w, err := newShell(&config.Pane{Type: "shell", Command: "htop", Shell: "sh"})
	if err != nil {
		t.Fatal(err)
	}
	defer w.(Closer).Close()

	cmd := w.Init()
	deadline := time.Now().Add(5 * time.Second)
	for cmd != nil && time.Now().Before(deadline) {
		v := stripANSI(w.View(120, 30))
		if strings.Contains(v, "CPU") || strings.Contains(v, "Load average") || strings.Contains(v, "PID") {
			return // painted
		}
		msg := cmd()
		w, cmd = w.Update(msg)
	}
	t.Fatalf("htop never painted a recognisable frame:\n%s", stripANSI(w.View(120, 30)))
}
