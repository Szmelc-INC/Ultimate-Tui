package engine_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

const twoCols = `{
  "name": "mouse",
  "layout": {
    "direction": "row",
    "children": [
      { "size": "50%", "pane": { "type": "text", "title": "Left",  "content": "L" } },
      { "size": "*",    "pane": { "type": "text", "title": "Right", "content": "R" } }
    ]
  }
}`

func TestMouseClickFocusesPaneUnderCursor(t *testing.T) {
	m := newModel(t, twoCols, 80, 20)
	_ = m.View() // establishes rectangles

	if m.Focused() != "Left" {
		t.Fatalf("initial focus = %q", m.Focused())
	}

	click := tea.MouseMsg{X: 60, Y: 10, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	u, _ := m.Update(click)
	m = u.(*engine.Model)
	if m.Focused() != "Right" {
		t.Fatalf("after click at x=60 focus = %q, want Right", m.Focused())
	}

	u, _ = m.Update(tea.MouseMsg{X: 5, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = u.(*engine.Model)
	if m.Focused() != "Left" {
		t.Fatalf("after click at x=5 focus = %q, want Left", m.Focused())
	}
}

func TestMouseBeforeFirstRenderIsSafeNoop(t *testing.T) {
	cfg, _ := config.Parse([]byte(twoCols))
	m, _ := engine.New(cfg)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m2.(*engine.Model)
	// No View() yet -> rects not stamped. Must not panic or change focus.
	u, cmd := m.Update(tea.MouseMsg{X: 60, Y: 10, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if cmd != nil {
		t.Fatal("expected no command")
	}
	if u.(*engine.Model).Focused() != "Left" {
		t.Fatal("focus should not move before rects exist")
	}
}

func TestMouseWheelScrollsWithoutChangingFocus(t *testing.T) {
	src := `{
      "name": "wheel",
      "layout": { "direction": "row", "children": [
        { "size": "50%", "pane": { "type": "text", "title": "A",
          "content": "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20" } },
        { "size": "*", "pane": { "type": "text", "title": "B", "content": "b" } }
      ]}
    }`
	m := newModel(t, src, 40, 8)
	first := m.View()

	// Wheel over the right pane; focus must stay on A, but B's content scrolls.
	u, _ := m.Update(tea.MouseMsg{X: 5, Y: 4, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	m = u.(*engine.Model)
	if m.Focused() != "A" {
		t.Fatalf("wheel changed focus to %q", m.Focused())
	}
	for i := 0; i < 6; i++ {
		u, _ = m.Update(tea.MouseMsg{X: 5, Y: 4, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
		m = u.(*engine.Model)
	}
	if m.View() == first {
		t.Fatal("wheel over pane A did not scroll it")
	}
}

func TestFocusLockGuardsInteractivePane(t *testing.T) {
	// Pane 0 is a shell (RawKeyConsumer). Unlocked, arrow keys navigate away
	// from it (the shell does NOT swallow them). Enter locks focus in; while
	// locked, bare keys reach the child but the modifier binding still quits.
	src := `{
      "name": "lock",
      "layout": { "direction": "row", "children": [
        { "size": "50%", "pane": { "type": "shell", "title": "sh", "command": "sleep 30" } },
        { "size": "*", "pane": { "type": "text", "title": "note", "content": "x" } }
      ]}
    }`
	cfg, err := config.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	m, err := engine.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	m.Init() // spawn the shell child
	defer m.Close()
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	m = mm.(*engine.Model)
	_ = m.View() // stamp rectangles for directional nav

	// Unlocked: right arrow moves focus off the shell pane.
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	if m.Focused() != "note" {
		t.Fatalf("right arrow did not leave the shell pane; at %q", m.Focused())
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = u.(*engine.Model)
	if m.Focused() != "sh" {
		t.Fatalf("left arrow did not return to the shell pane; at %q", m.Focused())
	}

	// Bare 'q' while focused-but-unlocked: falls through to the shell, no quit.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if isQuit(cmd) {
		t.Fatal("bare 'q' quit while a shell pane was focused")
	}

	// Enter locks; arrows now reach the child, focus stays put.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*engine.Model)
	if !m.Locked() {
		t.Fatal("Enter did not lock focus")
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	if m.Focused() != "sh" {
		t.Fatalf("locked pane lost focus to an arrow key; at %q", m.Focused())
	}

	// Modifier binding still quits, even while locked.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q"), Alt: true})
	if !isQuit(cmd) {
		t.Fatal("mod+q did not quit from inside a locked pane")
	}

	// Esc on a locked raw pane buffers (Esc-Esc = literal Esc to the child), so
	// a single Esc must NOT release.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if !m.Locked() {
		t.Fatal("first Esc released a locked shell pane; it should buffer")
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	if !m.Locked() {
		t.Fatal("Esc-Esc released the lock; it should forward a literal Esc and stay locked")
	}
	// A non-Esc key after a lone buffered Esc releases and is re-handled.
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*engine.Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = u.(*engine.Model)
	if m.Locked() {
		t.Fatal("Esc then arrow did not release the lock")
	}
	if m.Focused() != "note" {
		t.Fatalf("Esc then Right did not re-handle as nav; at %q", m.Focused())
	}
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestModQuitsButBareQDoesNot(t *testing.T) {
	m := newModel(t, twoCols, 60, 12)
	_ = m.View()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if isQuit(cmd) {
		t.Fatal("bare 'q' should no longer quit")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q"), Alt: true})
	if !isQuit(cmd) {
		t.Fatal("mod+q (alt+q) should quit")
	}
	_ = strings.TrimSpace("")
}
