package widget

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("shell", newShell) }

// shellWidget hosts a real interactive process (htop, vim, cmatrix, a shell)
// on a pty, emulates its output into a screen grid, and forwards keystrokes to
// it. This is the multiplexing pane — distinct from "command", which runs once
// and only captures text.
type shellWidget struct {
	id      int64
	shell   string
	command string

	emu  *vt.SafeEmulator
	ptmx *os.File
	cmd  *exec.Cmd

	w, h   int
	exited bool
	note   string // shown before first paint / after exit
}

func newShell(p *config.Pane) (Widget, error) {
	if p.Command == "" {
		return nil, fmt.Errorf("shell widget: \"command\" is required")
	}
	sh := p.Shell
	if sh == "" {
		sh = defaultShell()
	}
	return &shellWidget{
		id:      cmdSeq.Add(1),
		shell:   sh,
		command: p.Command,
		w:       80,
		h:       24,
		note:    "starting: " + p.Command,
	}, nil
}

// shellOutMsg carries one chunk of pty output (or the read error that ends the
// stream). It implements StreamMsg so Snapshot knows not to chase it forever.
type shellOutMsg struct {
	id   int64
	data []byte
	err  error
}

func (shellOutMsg) stream() {}

// Init starts the child synchronously (on the bubbletea goroutine, before the
// event loop) so no field is written concurrently, then returns the blocking
// read as a command.
func (s *shellWidget) Init() tea.Cmd {
	s.emu = vt.NewSafeEmulator(s.w, s.h)
	c := exec.Command(s.shell, "-c", s.command)
	c.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(c, &pty.Winsize{Rows: uint16(s.h), Cols: uint16(s.w)})
	if err != nil {
		s.exited = true
		s.note = "failed to start: " + err.Error()
		return nil
	}
	s.ptmx = ptmx
	s.cmd = c

	// Note: the emulator can answer terminal queries (device attributes,
	// cursor-position reports) on its Read side, but we do not pump those back
	// to the child. x/vt's Read and Close share an unsynchronised flag, so a
	// background copy races Close. Full-screen apps (htop, vim, less, cmatrix,
	// btop) render fine without it; an app that *blocks* waiting for a query
	// answer may misbehave. Tracked in the roadmap.
	return s.readCmd()
}

func (s *shellWidget) readCmd() tea.Cmd {
	ptmx, id := s.ptmx, s.id
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		n, err := ptmx.Read(buf)
		return shellOutMsg{id: id, data: append([]byte(nil), buf[:n]...), err: err}
	}
}

func (s *shellWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	switch m := msg.(type) {
	case shellOutMsg:
		if m.id != s.id || s.emu == nil {
			return s, nil
		}
		if len(m.data) > 0 {
			_, _ = s.emu.Write(m.data)
		}
		if m.err != nil {
			s.finish()
			return s, nil
		}
		return s, s.readCmd()

	case tea.KeyMsg:
		if s.exited || s.ptmx == nil {
			return s, nil
		}
		if b := keyToBytes(m); len(b) > 0 {
			_, _ = s.ptmx.Write(b)
		}
		return s, nil
	}
	return s, nil
}

func (s *shellWidget) finish() {
	if s.exited {
		return
	}
	s.exited = true
	s.note = "process exited"
	_ = s.Close()
}

// Close kills the child, reaps it (so it does not linger as a zombie), and
// releases the pty. The emulator is left intact so View can keep drawing the
// final frame. Safe to call repeatedly — the child is detached after the first
// call.
func (s *shellWidget) Close() error {
	if s.cmd != nil {
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		cmd := s.cmd
		s.cmd = nil
		go func() { _ = cmd.Wait() }() // reap off the UI goroutine
	}
	if s.ptmx != nil {
		err := s.ptmx.Close()
		s.ptmx = nil
		return err
	}
	return nil
}

// Interrupt stops the child now, as if it had exited on its own: the pane drops
// out of raw-key mode and View shows the exit overlay. Fired by a confirmed
// Ctrl+C on the locked pane — the program keeps running.
func (s *shellWidget) Interrupt() error {
	if s.exited {
		return nil
	}
	s.exited = true
	s.note = "process killed"
	return s.Close()
}

// Restart kills the current child + pty and spawns the command again on a fresh
// emulator. The previous read goroutine, if still blocked, unblocks on the
// closed pty and its shellOutMsg is dropped by the id check in Update. Called
// from the pane context menu. All state mutation stays on the bubbletea
// goroutine (Init is synchronous), matching the x/vt threading rule.
func (s *shellWidget) Restart() tea.Cmd {
	_ = s.Close()
	s.exited = false
	s.emu = nil
	s.id = cmdSeq.Add(1)
	s.note = "restarting: " + s.command
	return s.Init()
}

// Refresh asks the running child to repaint by sending it Ctrl+L (form feed) —
// the near-universal "redraw" key. A no-op once the process has exited.
func (s *shellWidget) Refresh() tea.Cmd {
	if s.exited || s.ptmx == nil {
		return nil
	}
	_, _ = s.ptmx.Write([]byte{0x0c})
	return nil
}

// ConsumesRawKeys is true while the process runs, so htop gets its own 'q'/F-keys
// and the engine only reserves the prefix.
func (s *shellWidget) ConsumesRawKeys() bool { return !s.exited }

func (s *shellWidget) View(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	if s.emu != nil && (width != s.w || height != s.h) {
		s.w, s.h = width, height
		s.emu.Resize(width, height)
		if s.ptmx != nil {
			_ = pty.Setsize(s.ptmx, &pty.Winsize{Rows: uint16(height), Cols: uint16(width)})
		}
	}
	if s.emu == nil {
		return fitBox(s.note, width, height)
	}

	screen := normalizeGrid(s.emu.Render(), width, height)
	if s.exited {
		return overlayLastLine(screen, "  "+s.note+"  —  press Esc to release, then an arrow key  ", width)
	}
	return screen
}

// normalizeGrid forces the emulator's Render() output to exactly h lines of w
// display cells: Render() emits "" for blank rows and may add a trailing
// newline.
func normalizeGrid(s string, w, h int) string {
	s = strings.TrimSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, h)
	for i := 0; i < h; i++ {
		ln := ""
		if i < len(lines) {
			ln = lines[i]
		}
		switch cw := lipgloss.Width(ln); {
		case cw < w:
			ln += strings.Repeat(" ", w-cw)
		case cw > w:
			ln = ansi.Truncate(ln, w, "")
		}
		out[i] = ln
	}
	return strings.Join(out, "\n")
}

func fitBox(text string, w, h int) string {
	return lipgloss.NewStyle().Width(w).Height(h).MaxWidth(w).MaxHeight(h).Render(text)
}

func overlayLastLine(grid, msg string, w int) string {
	lines := strings.Split(grid, "\n")
	if len(lines) == 0 {
		return grid
	}
	bar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).
		Width(w).MaxWidth(w).Render(ansi.Truncate(msg, w, ""))
	lines[len(lines)-1] = bar
	return strings.Join(lines, "\n")
}
