package engine

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// Run builds and runs a program for cfg, blocking until the user quits. Both
// the `ultui run` command and go:embed-compiled wrappers call this — it is the
// single execution path for every build mode.
func Run(cfg *config.Config) error {
	m, err := New(cfg)
	if err != nil {
		return err
	}
	// Mouse reporting: click focuses a pane, wheel scrolls it. This does trade
	// away the terminal's native text selection while ultui runs.
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	m.Close() // kill any child processes / release ptys
	return err
}
