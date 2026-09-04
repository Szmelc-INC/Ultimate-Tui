package engine

import "testing"

// isGlobalKey decides which keys the engine swallows even inside a locked pane.
// F3 must NOT be here any more — otherwise a locked shell/editor loses its F3.
func TestGlobalKeySet(t *testing.T) {
	m := &Model{mod: "alt"}
	if m.isGlobalKey("f3") {
		t.Error("f3 must not be a global key (it now falls through to the widget)")
	}
	if !m.isGlobalKey("f1") {
		t.Error("f1 must be a global key")
	}
	if !m.isGlobalKey("f2") {
		t.Error("f2 must be a global key")
	}
	if !m.isGlobalKey("alt+x") {
		t.Error("modifier-prefixed keys must be global")
	}
}
