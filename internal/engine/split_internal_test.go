package engine

import "testing"

// confirmSplit on an existing pane (Split): the original pane's widget must
// survive untouched and the new half is a blank text pane.
func TestConfirmSplitTargetKeepsOriginalWidget(t *testing.T) {
	m := mkModel(t, oneText)
	orig := m.leaves[0].w
	origPane := m.leaves[0].pane

	m.openSplitPrompt(0)
	_, cmd := m.confirmSplit()
	_ = cmd

	if len(m.leaves) != 2 {
		t.Fatalf("leaves = %d, want 2", len(m.leaves))
	}
	i := m.indexOfPane(origPane)
	if i < 0 || m.leaves[i].w != orig {
		t.Fatal("split respawned the original pane's widget")
	}
	other := 1 - i
	if m.leaves[other].pane.Type != "text" {
		t.Fatalf("new pane type = %q, want text", m.leaves[other].pane.Type)
	}
	if m.focus != i {
		t.Fatal("split should leave focus on the original pane")
	}
}

// confirmSplit with target -1 (the "New pane" row, opened on dead space)
// wraps the whole root and focuses the new pane.
func TestConfirmSplitWrapsRootForNewTarget(t *testing.T) {
	m := mkModel(t, oneText)
	origPane := m.leaves[0].pane

	m.openSplitPrompt(-1)
	m.spDir = "column"
	_, _ = m.confirmSplit()

	if len(m.leaves) != 2 {
		t.Fatalf("leaves = %d, want 2", len(m.leaves))
	}
	if m.root.direction != "column" {
		t.Fatalf("root direction = %q, want column", m.root.direction)
	}
	if m.indexOfPane(origPane) < 0 {
		t.Fatal("original pane lost after wrapping the root")
	}
	if m.focus == m.indexOfPane(origPane) {
		t.Fatal("New should focus the freshly created pane, not the original")
	}
}

func TestConfirmSplitRespectsChosenPercent(t *testing.T) {
	m := mkModel(t, oneText)
	m.openSplitPrompt(0)
	m.spPct = 70
	_, _ = m.confirmSplit()

	if got := m.leaves[0].parent.Size; got != "70%" {
		t.Fatalf("first slot size = %q, want 70%%", got)
	}
	if got := m.leaves[1].parent.Size; got != "30%" {
		t.Fatalf("second slot size = %q, want 30%%", got)
	}
}

// A two-child split collapses into its surviving sibling rather than leaving
// a degenerate one-child node; the surviving pane's widget is untouched.
func TestCloseLeafCollapsesTwoChildSplit(t *testing.T) {
	m := mkModel(t, splitSrc) // L, R
	survivorWidget := m.leaves[1].w
	survivorPane := m.leaves[1].pane

	m.closeLeaf(0)

	if len(m.leaves) != 1 {
		t.Fatalf("leaves = %d, want 1", len(m.leaves))
	}
	if m.leaves[0].pane != survivorPane || m.leaves[0].w != survivorWidget {
		t.Fatal("close did not leave the surviving pane's identity untouched")
	}
	if m.leaves[0].parent != nil || m.leaves[0].split != nil {
		t.Fatal("surviving leaf should now be the root leaf (no enclosing split)")
	}
}

// Closing the removed pane's widget is rebuild()'s job — this pins that
// closeLeaf actually triggers it end to end.
func TestCloseLeafClosesRemovedWidget(t *testing.T) {
	m := mkModel(t, splitSrc)
	removed := &recWidget{}
	m.leaves[0].w = removed

	m.closeLeaf(0)

	if removed.closed != 1 {
		t.Fatalf("removed pane's widget Close() called %d times, want 1", removed.closed)
	}
}

// The last pane refuses to close.
func TestCloseLeafRefusesLastPane(t *testing.T) {
	m := mkModel(t, oneText)
	m.closeLeaf(0)
	if len(m.leaves) != 1 {
		t.Fatalf("leaves = %d, want 1 (close of the last pane must refuse)", len(m.leaves))
	}
}

// Closing a middle child of a 3-way percent split renormalizes the survivors
// back to summing to 100.
func TestCloseLeafRenormalizesThreeWaySplit(t *testing.T) {
	const three = `{"name":"x","layout":{"direction":"row","children":[
	  {"size":"33%","pane":{"type":"text","title":"A","content":"a"}},
	  {"size":"33%","pane":{"type":"text","title":"B","content":"b"}},
	  {"size":"34%","pane":{"type":"text","title":"C","content":"c"}}
	]}}`
	m := mkModel(t, three)
	bPane := m.leaves[1].pane
	m.closeLeaf(1) // close B

	if len(m.leaves) != 2 {
		t.Fatalf("leaves = %d, want 2", len(m.leaves))
	}
	if m.indexOfPane(bPane) >= 0 {
		t.Fatal("B should be gone")
	}
	sum := 0
	for _, c := range m.root.children {
		v := parseSize(c.cfg.Size)
		if v.kind != percent {
			t.Fatalf("child size %q is not percent after renormalize", c.cfg.Size)
		}
		sum += int(v.value)
	}
	if sum != 100 {
		t.Fatalf("renormalized percents sum to %d, want 100", sum)
	}
}

// Closing the currently-focused pane must not crash and must land focus on a
// still-live pane.
func TestCloseLeafMovesFocusOffClosedPane(t *testing.T) {
	m := mkModel(t, splitSrc)
	m.focus = 0
	m.closeLeaf(0)
	if m.focus < 0 || m.focus >= len(m.leaves) {
		t.Fatalf("focus = %d out of range after closing it, len=%d", m.focus, len(m.leaves))
	}
}
