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
// deliberate rather than twitchy. 2 (1 resizeStep == 2.5% per cell) proved
// too sensitive in practice — a routine drag across a terminal pane blew
// through the whole layout in a handful of cells; 5 (1% per cell) leaves
// room for fine control over a normal-width pane without needing a
// deliberately slow drag.
const dragCellsPerStep = 5

// dragMaxStepsPerEvent caps how many resize steps a single motion event can
// apply. Terminals coalesce mouse motion during a fast drag into a few
// events with large deltas rather than many events with small ones —
// dragCellsPerStep alone doesn't help there, since one event can still carry
// enough cells for a double-digit percent swing. Unconsumed movement is not
// discarded (dragLastX/Y only advances by what was actually applied), so a
// fast drag still tracks the cursor smoothly across several events instead
// of jumping once and then lagging.
const dragMaxStepsPerEvent = 2

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
// consumed position into whole resize steps (fractional movement carries
// over to the next event) — a drag is just many small directional resizes,
// one per axis per dragCellsPerStep of movement.
func (m *Model) dragResizeStep(x, y int) {
	dx, dy := x-m.dragLastX, y-m.dragLastY
	sx, sy := dx/dragCellsPerStep, dy/dragCellsPerStep
	sx = clampInt(sx, -dragMaxStepsPerEvent, dragMaxStepsPerEvent)
	sy = clampInt(sy, -dragMaxStepsPerEvent, dragMaxStepsPerEvent)
	for n := sx; n > 0; n-- {
		m.dragResizeAxis(dirRight, dirLeft)
	}
	for n := sx; n < 0; n++ {
		m.dragResizeAxis(dirLeft, dirRight)
	}
	for n := sy; n > 0; n-- {
		m.dragResizeAxis(dirDown, dirUp)
	}
	for n := sy; n < 0; n++ {
		m.dragResizeAxis(dirUp, dirDown)
	}
	m.dragLastX += sx * dragCellsPerStep
	m.dragLastY += sy * dragCellsPerStep
}

// dragResizeAxis applies one drag-resize step toward growDir — the raw
// screen direction the cursor just moved. Unlike mod+arrow (always grows,
// see resize.go), a drag is grabbed at an arbitrary point on the pane, not
// necessarily its growDir edge: if the focused pane has no neighbor to grow
// into on growDir's side (flush against that side of the whole layout), that
// is not the end of what "drag this way" can mean — shrinking the pane from
// its *opposite* edge moves that boundary the same raw direction, just as
// seen from the other side, so it's tried as a fallback. Only a genuine dead
// end — no neighbor anywhere on that axis — is a no-op. This is what makes
// grabbing a pane anywhere and dragging any direction always resize
// *something* when the layout has room to give, rather than only working
// when the grab happened to land on the "correct" half of the pane.
func (m *Model) dragResizeAxis(growDir, shrinkFallbackDir direction) {
	if m.dragLinked(growDir) {
		m.resizeFocusedBy(growDir, 1)
		return
	}
	if m.dragLinked(shrinkFallbackDir) {
		m.resizeFocusedBy(shrinkFallbackDir, -1)
	}
}

// dragLinked reports whether direction d has a neighbor for the focused pane
// to resize against at all (ignoring the resizeFloor — that's a separate,
// legitimate refusal resizeFocusedBy already makes).
func (m *Model) dragLinked(d direction) bool {
	_, _, ok := findResizeLink(m.leaves[m.focus], d)
	return ok
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
