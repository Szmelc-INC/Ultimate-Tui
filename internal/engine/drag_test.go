package engine

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func altPress(m *Model, x, y int, right bool) *Model {
	btn := tea.MouseButtonLeft
	if right {
		btn = tea.MouseButtonRight
	}
	u, _ := m.Update(tea.MouseMsg{X: x, Y: y, Alt: true, Button: btn, Action: tea.MouseActionPress})
	return u.(*Model)
}

func motion(m *Model, x, y int) *Model {
	u, _ := m.Update(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion})
	return u.(*Model)
}

func release(m *Model, x, y int) *Model {
	u, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonNone, Action: tea.MouseActionRelease})
	return u.(*Model)
}

// mod+right-drag grows the grabbed pane toward wherever the cursor moves,
// reusing resizeFocused live during the drag.
func TestDragResizeGrowsTowardCursor(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View() // stamp leaf rects at 80x24

	tl := m.leaves[0] // TL, top-left of the 2x2 grid
	startX, startY := tl.x+2, tl.y+2

	m = altPress(m, startX, startY, true)
	if m.dragMode != dragResize {
		t.Fatalf("mod+right-press did not start a resize drag (mode=%d)", m.dragMode)
	}
	if m.focus != 0 {
		t.Fatalf("drag should focus the grabbed pane, focus=%d", m.focus)
	}

	m = motion(m, startX+dragCellsPerStep*3, startY) // 3 resize steps rightward

	row := m.root.children[0].node
	if got := row.children[0].cfg.Size; got != "65%" {
		t.Fatalf("TL size after drag = %q, want 65%% (3 steps of 5%%)", got)
	}

	m = release(m, startX+dragCellsPerStep*3, startY)
	if m.dragMode != dragNone {
		t.Fatal("release did not end the drag")
	}
}

// mod+left-drag grabs a pane and, on drop over another pane, swaps their
// content — dropping back on the same pane (or outside any pane) is a no-op.
func TestDragMoveSwapsOnDrop(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()

	tl, tr := m.leaves[0], m.leaves[1]
	tlPane, trPane := tl.pane, tr.pane

	m = altPress(m, tl.x+2, tl.y+2, false)
	if m.dragMode != dragMove {
		t.Fatalf("mod+left-press did not start a move drag (mode=%d)", m.dragMode)
	}
	m = release(m, tr.x+2, tr.y+2) // drop on TR

	if m.dragMode != dragNone {
		t.Fatal("release did not end the drag")
	}
	iTL := m.indexOfPane(tlPane)
	iTR := m.indexOfPane(trPane)
	if m.leaves[iTL].title != "TL" || m.leaves[iTR].title != "TR" {
		t.Fatal("pane identities got confused")
	}
	// After the swap, TL's pane content now lives where TR's slot used to be
	// (left slot) and vice versa — check via the slot, not the identity.
	row := m.root.children[0].node
	if row.children[0].node.leaf.pane != trPane {
		t.Fatalf("left slot after drop = %q pane, want TR's", m.leafName(m.indexOfPane(row.children[0].node.leaf.pane)))
	}
	if row.children[1].node.leaf.pane != tlPane {
		t.Fatal("right slot after drop should hold TL's pane")
	}
	if m.focus != m.indexOfPane(tlPane) {
		t.Fatal("focus should follow the dragged pane's identity")
	}
}

func TestDragMoveDropOnSelfIsNoop(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	tl := m.leaves[0]
	before := m.View()

	m = altPress(m, tl.x+2, tl.y+2, false)
	m = release(m, tl.x+2, tl.y+2)

	if m.View() != before {
		t.Fatal("dropping a pane on itself should change nothing")
	}
}

// Holding mod during a right-press must NOT open the context menu.
func TestAltRightPressDoesNotOpenContextMenu(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	tl := m.leaves[0]

	m = altPress(m, tl.x+2, tl.y+2, true)
	if m.overlay == ovContext {
		t.Fatal("mod+right-press opened the context menu instead of starting a resize drag")
	}
}
