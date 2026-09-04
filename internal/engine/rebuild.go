package engine

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/widget"
)

// rebuild re-derives the whole pane tree (m.root, m.leaves) from m.cfg after
// a structural edit — split, close, new pane, swap, or anything else that
// reallocates a config.Node's Children slice or moves Nodes around. Every
// pointer the engine holds into the config tree (leaf.parent, leaf.split) and
// every index into m.leaves (m.focus, m.ctxLeaf, m.confirmLeaf, …) is
// potentially invalidated by such an edit — Go slices reallocate on append,
// silently stranding old pointers — so nothing may mutate cfg.Layout's shape
// without calling this immediately after. Use indexOfPane to re-resolve a
// leaf identity (captured as a *config.Pane before the edit) once this
// returns.
//
// Widgets are carried over by *config.Pane pointer identity: a pane whose
// address is unchanged keeps its live widget untouched (a Split does not
// respawn the pane it split; a resize or swap never touches a widget at
// all). A pane no longer reachable from cfg.Layout has its widget Close()d
// here. A freshly created pane (Split's new half, New's blank pane) gets a
// widget via widget.New, and its Init command is returned batched so the
// caller can hand it back to bubbletea — bubbletea only calls Model.Init once
// at program start, so a pane born at runtime must kick off its own command.
func (m *Model) rebuild() tea.Cmd {
	old := make(map[*config.Pane]*leaf, len(m.leaves))
	for _, lf := range m.leaves {
		old[lf.pane] = lf
	}

	m.leaves = nil
	var fresh []widget.Widget
	root, err := m.build(&m.cfg.Layout, nil, nil, "layout", old, &fresh, nil)
	if err != nil || len(m.leaves) == 0 {
		// A caller must keep the tree valid (Validate() before mutating);
		// this is a defensive bail, not an expected path.
		return nil
	}
	m.root = root

	keep := make(map[*config.Pane]bool, len(m.leaves))
	for _, lf := range m.leaves {
		keep[lf.pane] = true
	}
	for p, lf := range old {
		if !keep[p] {
			if c, ok := lf.w.(widget.Closer); ok {
				_ = c.Close()
			}
		}
	}

	var cmds []tea.Cmd
	for _, w := range fresh {
		if c := w.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}

// indexOfPane finds a leaf by config.Pane pointer identity — the way to
// re-resolve a focus/target index across a rebuild(). -1 if that pane is no
// longer in the tree (e.g. it was just closed).
func (m *Model) indexOfPane(p *config.Pane) int {
	for i, lf := range m.leaves {
		if lf.pane == p {
			return i
		}
	}
	return -1
}

// paneOf is the identity to capture before a structural edit — nil-safe for
// an out-of-range index.
func (m *Model) paneOf(idx int) *config.Pane {
	if idx < 0 || idx >= len(m.leaves) {
		return nil
	}
	return m.leaves[idx].pane
}
