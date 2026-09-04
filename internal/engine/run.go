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
	// No WithMouseCellMotion yet: enabling mouse reporting breaks terminal
	// text selection, and nothing consumes tea.MouseMsg until click-to-focus
	// lands. Turn it on together with that feature.
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
