package engine

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// mod+drag rearranges and resizes panes with the mouse: mod+left-press grabs
// the pane under the cursor to move (drop it on another pane to swap places),
// mod+right-press grabs it to resize (drag toward an edge to grow that way).
// Both preempt the plain right-click context menu (see handleMouse) — that is
// what "holding mod disables the normal context-menu function" means. Motion
// events only arrive while a button is held (xterm's cell-motion mode), so
// there is no separate "hover" state to track outside a drag.
type dragMode int

const (
	dragNone dragMode = iota
	dragMove
	dragResize
)

// dragCellsPerStep is how many cells of mouse movement one resizeStep (see
// resize.go) costs while dragging — coarser than 1:1 so a drag resize feels
// deliberate rather than twitchy.
const dragCellsPerStep = 2

func (m *Model) handleDragMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.dragMode == dragNone && msg.Alt && msg.Action == tea.MouseActionPress:
		i := m.leafAt(msg.X, msg.Y)
		if i < 0 {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonLeft:
			m.dragMode = dragMove
		case tea.MouseButtonRight:
			m.dragMode = dragResize
		default:
			return m, nil
		}
		m.dragLeaf = i
		m.focus = i
		m.dragLastX, m.dragLastY = msg.X, msg.Y

	case m.dragMode == dragResize && msg.Action == tea.MouseActionMotion:
		m.dragResizeStep(msg.X, msg.Y)

	case m.dragMode != dragNone && msg.Action == tea.MouseActionRelease:
		return m.dragRelease(msg)
	}
	return m, nil
}

// dragResizeStep converts however much the cursor has moved since the last
// consumed position into whole resizeFocused() calls (fractional movement
// carries over to the next event), reusing mod+arrow's exact grow-toward-
// that-edge, take-from-the-neighbor logic — a drag is just many small
// directional resizes.
func (m *Model) dragResizeStep(x, y int) {
	dx, dy := x-m.dragLastX, y-m.dragLastY
	sx, sy := dx/dragCellsPerStep, dy/dragCellsPerStep
	for n := sx; n > 0; n-- {
		m.resizeFocused(dirRight)
	}
	for n := sx; n < 0; n++ {
		m.resizeFocused(dirLeft)
	}
	for n := sy; n > 0; n-- {
		m.resizeFocused(dirDown)
	}
	for n := sy; n < 0; n++ {
		m.resizeFocused(dirUp)
	}
	m.dragLastX += sx * dragCellsPerStep
	m.dragLastY += sy * dragCellsPerStep
}

// dragRelease ends the drag. A resize just stops (every step already
// happened live in dragResizeStep). A move swaps the grabbed pane's content
// with whatever pane is under the cursor now — "grabs the pane, releasing it
// anywhere drops it back in a new place" — leaving Size with the slot, same
// as swapFocused. Dropping back on itself, or outside any pane, is a no-op.
func (m *Model) dragRelease(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	mode, from := m.dragMode, m.dragLeaf
	m.dragMode = dragNone
	if mode != dragMove || from < 0 || from >= len(m.leaves) {
		return m, nil
	}
	to := m.leafAt(msg.X, msg.Y)
	if to < 0 || to == from {
		return m, nil
	}

	nodeA, nodeB := m.nodeOfLeaf(from), m.nodeOfLeaf(to)
	focusPane := m.leaves[from].pane
	nodeA.Pane, nodeB.Pane = nodeB.Pane, nodeA.Pane

	cmd := m.rebuild()
	if i := m.indexOfPane(focusPane); i >= 0 {
		m.focus = i
	}
	return m, cmd
}

// nodeOfLeaf is the config.Node that directly holds leaf idx's Pane — its
// enclosing Child's Node if it has one, or cfg.Layout itself for the root
// leaf (the single-pane case).
func (m *Model) nodeOfLeaf(idx int) *config.Node {
	lf := m.leaves[idx]
	if lf.parent != nil {
		return &lf.parent.Node
	}
	return &m.cfg.Layout
}
