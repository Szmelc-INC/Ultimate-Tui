package engine

import (
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
)

// TestRebuildCarriesWidgetsByPaneIdentity: a rebuild that leaves the config
// tree's pane pointers untouched must not respawn any widget.
func TestRebuildCarriesWidgetsByPaneIdentity(t *testing.T) {
	m := mkModel(t, splitSrc)
	orig0, orig1 := m.leaves[0].w, m.leaves[1].w

	m.rebuild()

	if len(m.leaves) != 2 {
		t.Fatalf("leaves = %d, want 2", len(m.leaves))
	}
	if m.leaves[0].w != orig0 || m.leaves[1].w != orig1 {
		t.Fatal("rebuild respawned a widget whose pane pointer did not change")
	}
}

// TestRebuildClosesOrphanedWidgetAndSpawnsNew simulates a structural edit
// (splitting the first pane) done the way item-3/4/5 code will: mutate cfg in
// place, then call rebuild(). The surviving original pane's widget must
// carry over untouched; the newly introduced pane must get a fresh widget
// whose Init command comes back non-nil-safe (nil Init is fine, just must
// not panic); nothing is orphaned in this case (no pane was removed).
func TestRebuildClosesOrphanedWidgetAndSpawnsNew(t *testing.T) {
	m := mkModel(t, oneText)
	origWidget := m.leaves[0].w
	origPane := m.leaves[0].pane

	// Turn the single leaf into a row-split of itself + a brand-new pane —
	// exactly the shape Split will produce.
	newPane := &config.Pane{Type: "text", Title: "New", Content: ""}
	m.cfg.Layout = config.Node{
		Direction: "row",
		Children: []config.Child{
			{Size: "50%", Node: config.Node{Pane: origPane}},
			{Size: "50%", Node: config.Node{Pane: newPane}},
		},
	}
	m.rebuild()

	if len(m.leaves) != 2 {
		t.Fatalf("leaves = %d, want 2", len(m.leaves))
	}
	if i := m.indexOfPane(origPane); i < 0 || m.leaves[i].w != origWidget {
		t.Fatal("split did not carry the original pane's widget over")
	}
	if i := m.indexOfPane(newPane); i < 0 {
		t.Fatal("new pane not found after rebuild")
	}
}

// TestRebuildClosesRemovedPaneWidget: a pane dropped from cfg.Layout during a
// rebuild must have its widget Close()d — this is what Close-pane relies on
// to not leak a pty.
func TestRebuildClosesRemovedPaneWidget(t *testing.T) {
	m := mkModel(t, splitSrc)
	removedWidget := &recWidget{}
	m.leaves[1].w = removedWidget

	survivingPane := m.leaves[0].pane
	m.cfg.Layout = config.Node{Pane: survivingPane}
	m.rebuild()

	if len(m.leaves) != 1 {
		t.Fatalf("leaves = %d, want 1", len(m.leaves))
	}
	if removedWidget.closed != 1 {
		t.Fatalf("removed pane's widget Close() called %d times, want 1", removedWidget.closed)
	}
}

func TestIndexOfPaneMissingIsNegativeOne(t *testing.T) {
	m := mkModel(t, oneText)
	if i := m.indexOfPane(&config.Pane{}); i != -1 {
		t.Fatalf("indexOfPane(unknown) = %d, want -1", i)
	}
	if i := m.paneOf(-1); i != nil {
		t.Fatal("paneOf(out of range) should be nil")
	}
}
