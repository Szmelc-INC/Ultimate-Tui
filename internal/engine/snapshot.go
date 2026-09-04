package engine

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/widget"
)

// snapshot drain bounds — a shell pane's read loop is infinite by design, so
// the headless renderer must cap how long it chases commands.
const (
	snapMaxSteps   = 400
	snapMaxElapsed = 2 * time.Second
	snapCmdTimeout = 250 * time.Millisecond
)

// Snapshot builds the model, applies a fixed terminal size, runs widget Init
// commands until they settle (or the drain bounds are hit), and returns one
// rendered frame. It never touches a TTY and always releases any child
// processes it spawned.
func Snapshot(cfg *config.Config, width, height int) (string, error) {
	return SnapshotWithOptions(cfg, width, height, DefaultOptions())
}

// SnapshotWithOptions is Snapshot with explicit runner options.
func SnapshotWithOptions(cfg *config.Config, width, height int, opts Options) (string, error) {
	m, err := NewWithOptions(cfg, opts)
	if err != nil {
		return "", err
	}
	defer m.Close()

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: height})
	drain(&model, m.Init())

	return model.View(), nil
}

// drain executes a command tree breadth-first with hard limits. A StreamMsg
// (from a widget's endless read loop) is applied once but its follow-up command
// is dropped, so a shell pane contributes its first frame and no more.
func drain(model *tea.Model, root tea.Cmd) {
	if root == nil {
		return
	}
	queue := []tea.Cmd{root}
	deadline := time.Now().Add(snapMaxElapsed)

	for steps := 0; len(queue) > 0 && steps < snapMaxSteps; steps++ {
		if time.Now().After(deadline) {
			return
		}
		cmd := queue[0]
		queue = queue[1:]
		if cmd == nil {
			continue
		}

		msg, ok := runWithTimeout(cmd, snapCmdTimeout)
		if !ok || msg == nil {
			continue
		}
		if batch, isBatch := msg.(tea.BatchMsg); isBatch {
			queue = append(queue, batch...)
			continue
		}

		next, nextCmd := (*model).Update(msg)
		*model = next
		if _, isStream := msg.(widget.StreamMsg); isStream {
			continue // applied once; do not follow the loop
		}
		if nextCmd != nil {
			queue = append(queue, nextCmd)
		}
	}
}

func runWithTimeout(cmd tea.Cmd, d time.Duration) (tea.Msg, bool) {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg, true
	case <-time.After(d):
		return nil, false // goroutine is abandoned; its process is killed by m.Close
	}
}
