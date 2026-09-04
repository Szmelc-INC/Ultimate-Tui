package engine

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// Snapshot builds the model, applies a fixed terminal size, runs every widget's
// Init command to completion once, and returns a single rendered frame. It never
// touches a TTY — this is the headless render path for tests, CI and docs.
func Snapshot(cfg *config.Config, width, height int) (string, error) {
	m, err := New(cfg)
	if err != nil {
		return "", err
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: height})

	// Drain Init commands synchronously. tea.Batch wraps sub-commands in a
	// BatchMsg; execute each and feed results back once (enough for the
	// run-once widgets that exist today).
	drain(&model, m.Init())

	return model.View(), nil
}

func drain(model *tea.Model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			drain(model, c)
		}
	case nil:
		// no-op
	default:
		next, nextCmd := (*model).Update(msg)
		*model = next
		drain(model, nextCmd)
	}
}
