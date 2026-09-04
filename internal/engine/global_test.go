package engine

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestGlobalPanelCyclesModAndTogglesStatusBar(t *testing.T) {
	m := mkModel(t, oneText)
	m.openGlobalPanel()
	if m.glMod != "alt" {
		t.Fatalf("openGlobalPanel: glMod = %q, want alt (m.mod)", m.glMod)
	}

	m.globalCycle(1)
	if m.glMod != "ctrl" {
		t.Fatalf("cycling modifier: glMod = %q, want ctrl", m.glMod)
	}
	if !m.glDirty {
		t.Fatal("cycling modifier should mark the panel dirty")
	}

	m.glRow = glRowStatusBar
	before := m.glStatusBar
	m.globalCycle(1)
	if m.glStatusBar == before {
		t.Fatal("cycling status bar did not toggle it")
	}
}

func TestApplyGlobalPreservesOptsKeysAndCallsSaveGlobal(t *testing.T) {
	m := mkModel(t, oneText)
	m.opts.Keys = map[string]string{"alt+n": "focus-next"}
	m.openGlobalPanel()
	m.glMod = "ctrl"
	m.glStatusBar = false

	var saved *config.GlobalConfig
	m.opts.SaveGlobal = func(gc *config.GlobalConfig) error { saved = gc; return nil }

	cmd := m.applyGlobal()
	_ = cmd
	if saved == nil {
		t.Fatal("SaveGlobal was not called")
	}
	if saved.Mod != "ctrl" {
		t.Fatalf("saved.Mod = %q, want ctrl", saved.Mod)
	}
	if saved.StatusBar == nil || *saved.StatusBar != false {
		t.Fatal("saved.StatusBar did not carry the buffered false")
	}
	if saved.Keybindings["alt+n"] != "focus-next" {
		t.Fatalf("saved.Keybindings dropped the existing global binding: %+v", saved.Keybindings)
	}
	if m.glDirty {
		t.Fatal("apply should clear glDirty")
	}
	if m.glNote == "" {
		t.Fatal("apply should leave an on-screen confirmation")
	}
}

func TestApplyGlobalWithoutCallbackErrors(t *testing.T) {
	m := mkModel(t, oneText) // DefaultOptions -> SaveGlobal nil
	m.openGlobalPanel()
	m.applyGlobal()
	if m.glNote == "" {
		t.Fatal("expected a note when no SaveGlobal is wired")
	}
}

func TestGlobalPanelEscCloses(t *testing.T) {
	m := mkModel(t, oneText)
	m.openGlobalPanel()
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	if m.overlay != ovNone {
		t.Fatalf("Esc did not close the global panel (overlay=%d)", m.overlay)
	}
}

func TestGlobalPanelThemeRowOpensSettingsAndReturns(t *testing.T) {
	m := mkModel(t, oneText)
	m.openGlobalPanel()
	m.glRow = glRowTheme
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = u.(*Model)
	if m.overlay != ovSettings {
		t.Fatalf("Enter on theme row did not open settings (overlay=%d)", m.overlay)
	}
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	if m.overlay != ovGlobal {
		t.Fatalf("Esc from settings should return to the global panel (overlay=%d)", m.overlay)
	}
}

func TestGlobalMouseClickAndOutsideDismiss(t *testing.T) {
	m := mkModel(t, oneText)
	m.openGlobalPanel()
	_ = m.View()

	u, _ := clickAt(m, colX(m, 1), rowY(m, 4)) // status bar row
	m = u.(*Model)
	if m.glRow != glRowStatusBar {
		t.Fatalf("clicking the status bar row left glRow = %d, want %d", m.glRow, glRowStatusBar)
	}
	if !m.glDirty {
		t.Fatal("clicking the status bar row should toggle it")
	}

	u, _ = clickAt(m, 0, 0) // corner, outside the panel
	m = u.(*Model)
	if m.overlay != ovNone {
		t.Fatalf("outside click did not dismiss the global panel (overlay=%d)", m.overlay)
	}
}
