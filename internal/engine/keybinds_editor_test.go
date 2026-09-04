package engine

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestKeybindsTabLoadsSortedEntries(t *testing.T) {
	m := mkModel(t, oneText)
	m.cfg.Keybindings = map[string]string{"b": "help", "a": "quit"}
	m.openConfigEditor()

	if len(m.eb.keybinds) != 2 {
		t.Fatalf("keybinds = %d, want 2", len(m.eb.keybinds))
	}
	if m.eb.keybinds[0].key != "a" || m.eb.keybinds[1].key != "b" {
		t.Fatalf("keybinds not sorted: %+v", m.eb.keybinds)
	}
}

func TestKeybindsTabSurvivesPaneSelectorSwitch(t *testing.T) {
	m := mkModel(t, splitSrc) // 2 panes
	m.cfg.Keybindings = map[string]string{"x": "help"}
	m.openConfigEditor()

	// Config tab, "pane" row: cycle to the next pane — must not clear the
	// Keybinds scratch list loaded before loadPane ran.
	m.edTab, m.edRow = 0, 1
	m.editCycle(m.editorFields()[1], 1)

	if len(m.eb.keybinds) != 1 || m.eb.keybinds[0].key != "x" {
		t.Fatalf("keybinds lost across pane selector switch: %+v", m.eb.keybinds)
	}
}

func TestApplyKeybindsDropsEmptyKeyAndRebuildsKeyMap(t *testing.T) {
	m := mkModel(t, oneText)
	m.openConfigEditor()
	m.eb.keybinds = []kbEntry{
		{key: "y", action: "settings"},
		{key: "", action: "ignored"}, // empty key -> dropped
	}

	m.applyKeybinds()

	if len(m.cfg.Keybindings) != 1 || m.cfg.Keybindings["y"] != "settings" {
		t.Fatalf("cfg.Keybindings = %+v, want only {y: settings}", m.cfg.Keybindings)
	}
	if m.keys["y"] != "settings" {
		t.Fatalf("m.keys not rebuilt with the new binding: %+v", m.keys)
	}
}

func TestAddAndRemoveKeybindRow(t *testing.T) {
	m := mkModel(t, oneText)
	m.openConfigEditor()
	m.eb.keybinds = []kbEntry{{key: "x", action: "help"}}

	m.addKeybindRow()
	if len(m.eb.keybinds) != 2 {
		t.Fatalf("addKeybindRow: len = %d, want 2", len(m.eb.keybinds))
	}
	if m.edRow != 2 { // (len-1)*2 -> new entry's key row
		t.Fatalf("addKeybindRow did not focus the new entry's key field: edRow=%d", m.edRow)
	}

	m.removeKeybindRow(0)
	if len(m.eb.keybinds) != 1 || m.eb.keybinds[0].key != "" {
		t.Fatalf("removeKeybindRow(0) left %+v, want the blank second entry alone", m.eb.keybinds)
	}
}

func TestRemoveKeybindRowOutOfRangeIsNoop(t *testing.T) {
	m := mkModel(t, oneText)
	m.openConfigEditor()
	m.eb.keybinds = []kbEntry{{key: "x", action: "help"}}
	m.removeKeybindRow(5)
	if len(m.eb.keybinds) != 1 {
		t.Fatal("out-of-range removeKeybindRow should be a no-op")
	}
}

// End-to-end: open the config editor, add a keybind by typing through the
// real key path, save it, and confirm the new binding fires immediately
// (no restart) once the editor closes.
func TestKeybindsEndToEndAddAndFires(t *testing.T) {
	cfg, err := config.Parse([]byte(oneText))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	var saved *config.Config
	opts := DefaultOptions()
	opts.SaveConfig = func(c *config.Config) error { calls++; saved = c; return nil }
	m, err := NewWithOptions(cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = u.(*Model)

	press := func(msg tea.KeyMsg) { u, _ := m.Update(msg); m = u.(*Model) }
	typeRunes := func(s string) {
		for _, r := range s {
			press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}

	m.openConfigEditor()
	m.edTab, m.edRow = 1, 0 // Keybinds tab, "+ add keybind" row (no entries yet)
	press(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.eb.keybinds) != 1 {
		t.Fatalf("Enter on + add keybind did not add a row: %+v", m.eb.keybinds)
	}

	typeRunes("y") // key field (cursor already there per addKeybindRow)
	press(tea.KeyMsg{Type: tea.KeyDown})
	typeRunes("settings") // action field

	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}, Alt: true}) // mod+s: apply + save

	if calls != 1 {
		t.Fatalf("SaveConfig called %d times, want 1", calls)
	}
	if saved.Keybindings["y"] != "settings" {
		t.Fatalf("saved config missing the new binding: %+v", saved.Keybindings)
	}

	// Close the editor and press the new key — it must act as "settings"
	// (opens the theme/style picker) with no restart.
	press(tea.KeyMsg{Type: tea.KeyEsc})
	if m.overlay != ovNone {
		t.Fatalf("editor did not close cleanly (overlay=%d)", m.overlay)
	}
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.overlay != ovSettings {
		t.Fatalf("newly bound key 'y' did not fire the settings action (overlay=%d)", m.overlay)
	}
}
