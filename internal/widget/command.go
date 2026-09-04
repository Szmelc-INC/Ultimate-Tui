package widget

import (
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("command", newCommand) }

var cmdSeq atomic.Int64

// commandWidget runs one non-interactive command through a shell and shows its
// combined output. It does not allocate a pty and cannot host interactive
// programs — that is a deliberately separate, unbuilt feature.
type commandWidget struct {
	id      int64
	command string
	shell   string
	box     scrollBox
	done    bool
}

func newCommand(p *config.Pane) (Widget, error) {
	if p.Command == "" {
		return nil, fmt.Errorf("command widget: \"command\" is required")
	}
	shell := p.Shell
	if shell == "" {
		shell = defaultShell()
	}
	c := &commandWidget{id: cmdSeq.Add(1), command: p.Command, shell: shell}
	c.box.setText("running: " + p.Command)
	return c, nil
}

type cmdDoneMsg struct {
	id     int64
	output string
}

func (c *commandWidget) Init() tea.Cmd {
	id, shell, command := c.id, c.shell, c.command
	return func() tea.Msg {
		out, err := exec.Command(shell, "-c", command).CombinedOutput()
		text := string(out)
		if err != nil {
			if text != "" && text[len(text)-1] != '\n' {
				text += "\n"
			}
			text += "[exit: " + err.Error() + "]"
		}
		return cmdDoneMsg{id: id, output: text}
	}
}

func (c *commandWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	switch m := msg.(type) {
	case cmdDoneMsg:
		if m.id == c.id {
			c.done = true
			c.box.setText(m.output)
		}
	case tea.KeyMsg:
		c.box.handleKey(m)
	case tea.MouseMsg:
		c.box.handleMouse(m)
	}
	return c, nil
}

func (c *commandWidget) View(width, height int) string {
	return c.box.view(width, height)
}

func defaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "sh"
}
