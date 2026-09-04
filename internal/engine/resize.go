package engine

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// mod+arrows resize the focused pane; mod+shift+arrows swap it with its
// neighbor in that direction; mod+s flips its enclosing split's orientation.
// This replaces the old zoom-as-pane-size behaviour (item 10 in the request
// that drove this file) — mod+=/-/0 is now a per-pane *content* zoom
// (widget.Zoomer, see text.go/bigfont.go), fully independent of geometry.

// resizeStep is the percent one lalt+arrow nudge moves between the focused
// pane and its neighbor. resizeFloor is the minimum share a nudge will leave
// the neighbor with — resize refuses rather than push a sibling to zero.
const (
	resizeStep  = 5
	resizeFloor = 5
)

// axisFor maps an arrow direction to the split Direction it acts along:
// left/right move a boundary in a "row" split (children side by side),
// up/down in a "column" split (children stacked) — config's default.
func axisFor(d direction) string {
	if d == dirLeft || d == dirRight {
		return "row"
	}
	return "column"
}

// neighborOffset is which sibling a direction acts on: left/up take from (or
// swap with) the previous child, right/down the next one.
func neighborOffset(d direction) int {
	if d == dirLeft || d == dirUp {
		return -1
	}
	return 1
}

// splitDir normalizes a config.Node's Direction ("" == "column", same as
// build()'s default).
func splitDir(n *config.Node) string {
	if n.Direction == "row" {
		return "row"
	}
	return "column"
}

func specsOf(n *config.Node) []sizeSpec {
	out := make([]sizeSpec, len(n.Children))
	for i := range n.Children {
		out[i] = parseSize(n.Children[i].Size)
	}
	return out
}

// findResizeLink walks a leaf's ancestor chain from its immediate parent
// upward and returns the nearest split whose axis matches d and that has a
// neighbor in d's direction — the split/child-index pair a resize or swap
// acts on. A split whose axis matches but has no neighbor there (the focused
// pane is already at that edge of it) is skipped in favor of an ancestor
// further up. ok is false when nothing in the whole chain qualifies (the
// pane is at that edge of the entire layout).
func findResizeLink(lf *leaf, d direction) (ci, neighbor int, ok bool) {
	axis := axisFor(d)
	off := neighborOffset(d)
	for i := len(lf.chain) - 1; i >= 0; i-- {
		l := lf.chain[i]
		if splitDir(l.split) != axis {
			continue
		}
		n := l.idx + off
		if n < 0 || n >= len(l.split.Children) {
			continue
		}
		return i, n, true
	}
	return 0, 0, false
}

// splitAxisLen replays distribute() along chain[0:upto] (the ancestors above
// chain[upto]) to find the pixel length of chain[upto].split's own
// distribution axis (width if it's a row split, height if column) as of the
// model's last render. bw, bh is that last rendered body size.
func splitAxisLen(chain []ancestorLink, upto, bw, bh int) int {
	w, h := bw, bh
	for i := 0; i < upto; i++ {
		l := chain[i]
		row := splitDir(l.split) == "row"
		axis := h
		if row {
			axis = w
		}
		sizes := distribute(specsOf(l.split), axis)
		if row {
			w = sizes[l.idx]
		} else {
			h = sizes[l.idx]
		}
	}
	if splitDir(chain[upto].split) == "row" {
		return w
	}
	return h
}

// resizePercents converts chain[upto].split's children's currently-computed
// sizes into whole percents summing to exactly 100 — the snapshot a resize
// nudges from, so a split that has never been touched (fixed/flex/mixed
// sizing) gets pinned to percent on its first mod+arrow rather than drifting.
func resizePercents(chain []ancestorLink, upto, bw, bh int) []int {
	total := splitAxisLen(chain, upto, bw, bh)
	if total <= 0 {
		total = 100 // never rendered yet — a reasonable stand-in
	}
	sizes := distribute(specsOf(chain[upto].split), total)
	pct := make([]int, len(sizes))
	sum := 0
	for i, s := range sizes {
		pct[i] = s * 100 / total
		sum += pct[i]
	}
	if d := 100 - sum; d != 0 && len(pct) > 0 {
		big := 0
		for i := 1; i < len(pct); i++ {
			if pct[i] > pct[big] {
				big = i
			}
		}
		pct[big] += d
	}
	return pct
}

// resizeFocused grows the focused pane toward d's edge by resizeStep percent,
// taking the space from the neighbor found by findResizeLink. A no-op if the
// pane is already at that edge of the whole layout, or if the neighbor is
// already at resizeFloor. mod+arrow always grows (n=1) — see resizeFocusedBy
// for the signed primitive mouse-drag needs.
func (m *Model) resizeFocused(d direction) { m.resizeFocusedBy(d, 1) }

// resizeFocusedBy is resizeFocused generalized to a signed step count: n>0
// grows the focused pane toward d's edge (taking from the d-side neighbor,
// same as resizeFocused); n<0 shrinks the focused pane from d's edge instead,
// giving the freed space to that same neighbor — the same boundary moving
// the same way, just described from the focused pane's other side. This is
// what lets mouse-drag resize (drag.go) do something useful when the pane
// grabbed has no neighbor to grow into on the dragged side but does have one
// on the opposite side: shrinking that opposite edge moves a real boundary
// in the direction the mouse moved, rather than silently refusing. mod+arrow
// never shrinks (always n=1) so keyboard resize keeps its documented
// grow-only semantics unchanged. Mutates Child.Size in place — render()
// re-parses Size every frame, so no rebuild() is needed.
func (m *Model) resizeFocusedBy(d direction, n int) {
	if n == 0 {
		return
	}
	lf := m.leaves[m.focus]
	ci, nb, ok := findResizeLink(lf, d)
	if !ok {
		return
	}
	link := lf.chain[ci]
	bw, bh := m.rectW, m.rectH
	if bw == 0 || bh == 0 {
		bw, bh = m.bodyDims()
	}
	pct := resizePercents(lf.chain, ci, bw, bh)
	step := n * resizeStep
	if step > 0 {
		if pct[nb]-step < resizeFloor {
			return
		}
	} else {
		if pct[link.idx]+step < resizeFloor {
			return
		}
	}
	pct[link.idx] += step
	pct[nb] -= step
	for i := range link.split.Children {
		link.split.Children[i].Size = strconv.Itoa(pct[i]) + "%"
	}
}

// swapFocused exchanges the focused pane with its neighbor in direction d —
// literally swapping their config.Node content (Pane or nested split alike)
// while leaving Size with the slot, tmux-style. Unlike resize this reshapes
// which *config.Pane lives in which leaf, so leaf.parent/leaf.split/leaf.chain
// across the whole subtree are now wrong; rebuild() re-derives them and the
// returned command carries any newly-built widget's Init (there won't be one
// here — swap moves existing panes, it never creates one).
func (m *Model) swapFocused(d direction) tea.Cmd {
	lf := m.leaves[m.focus]
	ci, nb, ok := findResizeLink(lf, d)
	if !ok {
		return nil
	}
	link := lf.chain[ci]
	focusPane := lf.pane
	link.split.Children[link.idx].Node, link.split.Children[nb].Node =
		link.split.Children[nb].Node, link.split.Children[link.idx].Node

	cmd := m.rebuild()
	if i := m.indexOfPane(focusPane); i >= 0 {
		m.focus = i
	}
	return cmd
}

// flipFocusedSplit toggles the focused pane's immediate enclosing split
// between row and column. A no-op on the root leaf (nothing encloses it). No
// rebuild needed — render() re-reads Direction from node.cfg every frame.
func (m *Model) flipFocusedSplit() {
	lf := m.leaves[m.focus]
	if lf.split == nil {
		return
	}
	if lf.split.Direction == "row" {
		lf.split.Direction = ""
	} else {
		lf.split.Direction = "row"
	}
}
