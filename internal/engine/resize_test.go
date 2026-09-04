package engine

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

const resizeGridSrc = `{
  "name": "grid",
  "layout": { "direction": "column", "children": [
    { "size": "50%", "direction": "row", "children": [
      { "size": "50%", "pane": { "type": "text", "title": "TL", "content": "a" } },
      { "size": "50%", "pane": { "type": "text", "title": "TR", "content": "b" } }
    ]},
    { "size": "50%", "direction": "row", "children": [
      { "size": "50%", "pane": { "type": "text", "title": "BL", "content": "c" } },
      { "size": "50%", "pane": { "type": "text", "title": "BR", "content": "d" } }
    ]}
  ]}
}`

func mkModelSized(t *testing.T, src string, w, h int) *Model {
	t.Helper()
	cfg, err := config.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return u.(*Model)
}

func pressAlt(m *Model, s string) *Model {
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Alt: true})
	return u.(*Model)
}

// pressAltType simulates alt+<k> the way bubbletea actually encodes it: a
// plain arrow key with Alt set, or (for the mod+shift+arrow swap bindings) a
// KeyShiftXxx type with Alt set — there's no separate "Shift" field on
// tea.KeyMsg, shift lives in the KeyType itself.
func pressAltType(m *Model, k tea.KeyType) *Model {
	u, _ := m.Update(tea.KeyMsg{Type: k, Alt: true})
	return u.(*Model)
}

// mod+right grows the focused pane at its right neighbor's expense, writing
// real percents into Child.Size — no rebuild, no pane-count change.
func TestResizeRightGrowsFocusedTakesFromRightNeighbor(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View() // stamp rects at 80x24

	if m.Focused() != "TL" {
		t.Fatalf("start focus = %q", m.Focused())
	}
	m = pressAltType(m, tea.KeyRight)

	row := m.root.children[0].node // top row split
	if got := row.children[0].cfg.Size; got != "55%" {
		t.Fatalf("TL size = %q, want 55%%", got)
	}
	if got := row.children[1].cfg.Size; got != "45%" {
		t.Fatalf("TR size = %q, want 45%%", got)
	}
	if len(m.leaves) != 4 {
		t.Fatalf("resize changed pane count: %d", len(m.leaves))
	}
}

// mod+left grows the focused pane leftward, taking from its left neighbor —
// symmetric with mod+right but the other neighbor.
func TestResizeLeftTakesFromLeftNeighbor(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	m.focus = m.indexOfPane(m.leaves[1].pane) // focus TR

	m = pressAltType(m, tea.KeyLeft)

	row := m.root.children[0].node
	if got := row.children[1].cfg.Size; got != "55%" {
		t.Fatalf("TR size = %q, want 55%%", got)
	}
	if got := row.children[0].cfg.Size; got != "45%" {
		t.Fatalf("TL size = %q, want 45%%", got)
	}
}

// mod+down on a pane inside a row split has no column-axis match at that
// level — its enclosing column split (the root) does (up/down -> column),
// with a neighbor below it (the bottom row). This pins that the chain walk
// finds the right axis, skipping the row split the pane is directly inside.
// Uses a body height evenly divisible by the 50/50 split so distribute()'s
// own rounding can't skew the assertion (that's resizeGridSrc's row-axis
// width of 80, mirrored here as an even column-axis height of 24).
func TestResizeDownActsOnOuterColumnSplit(t *testing.T) {
	m := mkModelSized(t, resizeGridSrc, 80, 25) // statusbar -> body height 24
	_ = m.View()

	m = pressAltType(m, tea.KeyDown)

	root := m.root // column split
	if got := root.children[0].cfg.Size; got != "55%" {
		t.Fatalf("top row size = %q, want 55%%", got)
	}
	if got := root.children[1].cfg.Size; got != "45%" {
		t.Fatalf("bottom row size = %q, want 45%%", got)
	}
	// The inner row split TL lives in must be untouched.
	row := root.children[0].node
	if got := row.children[0].cfg.Size; got != "50%" {
		t.Fatalf("inner TL size changed: %q, want unchanged 50%%", got)
	}
}

// Resize refuses once the neighbor would drop below resizeFloor.
func TestResizeRefusesPastFloor(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	row := m.root.children[0].node
	row.children[1].cfg.Size = "5%" // TR already at the floor
	row.children[0].cfg.Size = "95%"

	m = pressAltType(m, tea.KeyRight)
	if got := row.children[1].cfg.Size; got != "5%" {
		t.Fatalf("resize pushed neighbor past the floor: %q", got)
	}
}

// resizeFocusedBy(d, -1) shrinks the focused pane from d's edge, giving the
// freed space to the same neighbor resizeFocused(d) would take from — the
// primitive mouse-drag's fallback-shrink relies on (see drag_test.go).
func TestResizeFocusedByNegativeShrinksFocusedGrowsNeighbor(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	if m.Focused() != "TL" {
		t.Fatalf("start focus = %q", m.Focused())
	}

	m.resizeFocusedBy(dirRight, -1)

	row := m.root.children[0].node
	if got := row.children[0].cfg.Size; got != "45%" {
		t.Fatalf("TL size = %q, want 45%% (shrunk)", got)
	}
	if got := row.children[1].cfg.Size; got != "55%" {
		t.Fatalf("TR size = %q, want 55%% (grew)", got)
	}
}

// Shrinking refuses once the focused pane itself would drop below
// resizeFloor — the mirror of TestResizeRefusesPastFloor for the other side.
func TestResizeFocusedByNegativeRefusesPastFloor(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	row := m.root.children[0].node
	row.children[0].cfg.Size = "5%" // TL already at the floor
	row.children[1].cfg.Size = "95%"

	m.resizeFocusedBy(dirRight, -1)
	if got := row.children[0].cfg.Size; got != "5%" {
		t.Fatalf("shrink pushed the focused pane past the floor: %q", got)
	}
}

// mod+shift+right swaps the focused pane with its right neighbor; content
// (titles) trade places, pane count is unchanged, and focus follows the
// swapped pane's identity.
func TestSwapRightExchangesPanes(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	if m.Focused() != "TL" {
		t.Fatalf("start focus = %q", m.Focused())
	}

	m = pressAltType(m, tea.KeyShiftRight)

	if len(m.leaves) != 4 {
		t.Fatalf("swap changed pane count: %d", len(m.leaves))
	}
	if m.Focused() != "TL" {
		t.Fatalf("focus should follow the swapped pane (still titled TL): got %q", m.Focused())
	}
	row := m.root.children[0].node
	if row.children[0].node.leaf.title != "TR" {
		t.Fatalf("left slot after swap = %q, want TR", row.children[0].node.leaf.title)
	}
	if row.children[1].node.leaf.title != "TL" {
		t.Fatalf("right slot after swap = %q, want TL", row.children[1].node.leaf.title)
	}
}

// mod+s flips the focused pane's enclosing split's direction.
func TestFlipSplitTogglesDirection(t *testing.T) {
	m := mkModel(t, resizeGridSrc)
	_ = m.View()
	if m.leaves[m.focus].split.Direction != "row" {
		t.Fatalf("precondition: TL's split should be row")
	}

	m = pressAlt(m, "s")
	if got := m.leaves[m.focus].split.Direction; got != "" {
		t.Fatalf("flip-split: Direction = %q, want \"\" (column)", got)
	}
	m = pressAlt(m, "s")
	if got := m.leaves[m.focus].split.Direction; got != "row" {
		t.Fatalf("flip-split back: Direction = %q, want row", got)
	}
}

// The root leaf (no enclosing split) is a no-op for both resize and flip.
func TestResizeAndFlipNoopOnRootLeaf(t *testing.T) {
	m := mkModel(t, oneText)
	_ = m.View()

	before := m.View()
	m = pressAltType(m, tea.KeyRight)
	m = pressAlt(m, "s")
	if m.View() != before {
		t.Fatal("resize/flip on the single root pane changed the frame")
	}
}
