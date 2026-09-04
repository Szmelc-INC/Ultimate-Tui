package engine

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// clickAt sends a left-button press at absolute screen coordinates.
func clickAt(m *Model, x, y int) (tea.Model, tea.Cmd) {
	return m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
}

// rowY/colX convert a panel-content-relative row/col (the same coordinates
// each xxxMouseClick handler computes internally, e.g. contextMouseClick's
// "row - 2 == item index") into absolute screen coordinates using the
// bounding box splice() stamped on the last render — the inverse of
// handleOverlayMouse's cx/cy math.
func rowY(m *Model, row int) int { return m.ovY + panelContentOffsetY + row }
func colX(m *Model, col int) int { return m.ovX + panelContentOffsetX + col }

func TestContextMenuMouseClickRunsAction(t *testing.T) {
	m := mkModel(t, oneText) // single pane -> Close disabled; items: Stop,Restart,Refresh,Split,Close,Edit
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}, Alt: true})
	m = u.(*Model)
	_ = m.View() // stamp m.ovX/Y/W/H for the now-open context menu

	// contextMouseClick: item i is at content row i+2. Edit is item 5.
	u, _ = clickAt(m, colX(m, 1), rowY(m, 5+2))
	m = u.(*Model)

	if m.overlay != ovEditor {
		t.Fatalf("clicking Edit did not open the editor (overlay=%d)", m.overlay)
	}
}

func TestContextMenuMouseClickIgnoresDisabledRow(t *testing.T) {
	m := mkModel(t, oneText)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}, Alt: true})
	m = u.(*Model)
	_ = m.View()

	// Close is item 4 (disabled: single pane).
	u, _ = clickAt(m, colX(m, 1), rowY(m, 4+2))
	m = u.(*Model)

	if m.overlay != ovContext {
		t.Fatalf("clicking a disabled row should not run anything (overlay=%d)", m.overlay)
	}
	if len(m.leaves) != 1 {
		t.Fatal("disabled Close ran anyway")
	}
}

func TestControlMouseClickOpensRow(t *testing.T) {
	m := mkModel(t, oneText)
	m.openControl()
	_ = m.View()

	// controlMouseClick: item i is at content row i+2. "Theme & style" is item 0.
	u, _ := clickAt(m, colX(m, 1), rowY(m, 0+2))
	m = u.(*Model)

	if m.overlay != ovSettings {
		t.Fatalf("clicking the first control row did not open settings (overlay=%d)", m.overlay)
	}
}

func TestEditorMouseClickTogglesBoolField(t *testing.T) {
	m := mkModel(t, oneText)
	m.openPaneEditor(0)
	_ = m.View()
	before := m.eb.border

	// General tab fields: type(0), title(1), border(2); field i is at row
	// edFieldsStart + i*edFieldStride.
	u, _ := clickAt(m, colX(m, 1), rowY(m, edFieldsStart+2*edFieldStride))
	m = u.(*Model)

	if m.edRow != 2 {
		t.Fatalf("edRow = %d, want 2 (border)", m.edRow)
	}
	if m.eb.border == before {
		t.Fatal("clicking the border row did not toggle it")
	}
}

func TestEditorMouseClickSwitchesTab(t *testing.T) {
	m := mkModel(t, oneText)
	m.openPaneEditor(0)
	_ = m.View()
	if m.edTab != 0 {
		t.Fatalf("precondition: edTab = %d, want 0", m.edTab)
	}

	// Tabs render on content row edTabRow as "  General  ","  Layout  ", etc.
	tabs := m.editorTabs()
	tab0W := edTabCellWidth(tabs[0])
	u, _ := clickAt(m, colX(m, tab0W+1), rowY(m, edTabRow)) // 1 cell into the Layout tab
	m = u.(*Model)

	if m.edTab != 1 {
		t.Fatalf("clicking the Layout tab left edTab = %d, want 1", m.edTab)
	}
}

func TestSplitMouseClickSelectsFieldAndWheelAdjusts(t *testing.T) {
	m := mkModel(t, oneText)
	m.openSplitPrompt(0)
	_ = m.View()
	if m.spField != 0 {
		t.Fatalf("precondition: spField = %d, want 0", m.spField)
	}

	// splitPanel: title(0), blank(1), direction(2), percent(3).
	u, _ := clickAt(m, colX(m, 1), rowY(m, 3))
	m = u.(*Model)
	if m.spField != 1 {
		t.Fatalf("clicking the percent row left spField = %d, want 1", m.spField)
	}

	before := m.spPct
	u, _ = m.Update(tea.MouseMsg{X: colX(m, 1), Y: rowY(m, 3), Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	m = u.(*Model)
	if m.spPct != before+5 {
		t.Fatalf("wheel-up on the percent field: spPct = %d, want %d", m.spPct, before+5)
	}
}

func TestSettingsMouseClickPicksColumnAndRow(t *testing.T) {
	m := mkModel(t, oneText)
	m.openSettings()
	_ = m.View()

	styles := StyleNames()
	if len(styles) < 2 {
		t.Skip("need at least 2 styles to pick a non-default row")
	}
	// Right half of the panel -> the style column. settingsPanel: title(0),
	// blank(1), column header(2), items from row 3.
	x := m.ovX + m.ovW - 2
	u, _ := clickAt(m, x, rowY(m, 3+1))
	m = u.(*Model)

	if m.setCol != 1 {
		t.Fatalf("clicking the right column left setCol = %d, want 1", m.setCol)
	}
	if m.setRow[1] != 1 {
		t.Fatalf("setRow[1] = %d, want 1", m.setRow[1])
	}
	if m.styleName != styles[1] {
		t.Fatalf("styleName = %q, want %q (live preview)", m.styleName, styles[1])
	}
}

// Clicking above the item list (the panel title/header rows, content-relative
// row < 3 per settingsPanel: title(0), blank(1), column header(2)) must be a
// no-op, not clamp into row 0 and silently apply/preview that theme — see
// advisor review: this let a click on the panel's own heading commit a theme
// change on Enter.
func TestSettingsMouseClickAboveListIsNoop(t *testing.T) {
	m := mkModel(t, oneText)
	m.openSettings()
	_ = m.View()

	before := m.themeName
	beforeRow := m.setRow[0]
	u, _ := clickAt(m, colX(m, 1), rowY(m, 0)) // title row
	m = u.(*Model)

	if m.themeName != before {
		t.Fatalf("clicking the settings panel title changed themeName from %q to %q", before, m.themeName)
	}
	if m.setRow[0] != beforeRow {
		t.Fatalf("clicking the settings panel title moved setRow[0] from %d to %d", beforeRow, m.setRow[0])
	}
}

func TestConfirmMouseClickCancelReturnsToEditor(t *testing.T) {
	m := mkModel(t, oneText)
	m.openPaneEditor(0)
	m.eb.title = "changed"
	m.edDirty = true
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	if m.overlay != ovConfirm {
		t.Fatalf("Esc with dirty edits should raise the discard confirm (overlay=%d)", m.overlay)
	}
	_ = m.View()

	// confirmPanel: title(0), blank(1), confirm(2), cancel(3).
	u, _ = clickAt(m, colX(m, 1), rowY(m, 3))
	m = u.(*Model)
	if m.overlay != ovEditor {
		t.Fatalf("clicking cancel should return to the editor (overlay=%d)", m.overlay)
	}
	if m.eb.title != "changed" {
		t.Fatal("cancel should not have discarded the edit")
	}
}

func TestConfirmMouseClickConfirmDiscards(t *testing.T) {
	m := mkModel(t, oneText)
	m.openPaneEditor(0)
	m.eb.title = "changed"
	m.edDirty = true
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	_ = m.View()

	u, _ = clickAt(m, colX(m, 1), rowY(m, 2)) // "confirm" row
	m = u.(*Model)
	if m.overlay != ovNone {
		t.Fatalf("clicking confirm should close everything (overlay=%d)", m.overlay)
	}
}

func TestHelpOverlayClickAnywhereCloses(t *testing.T) {
	m := mkModel(t, oneText)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m = u.(*Model)
	if !strings.Contains(m.View(), "ULTIMATE TUI") {
		t.Fatal("help did not open")
	}
	u, _ = clickAt(m, colX(m, 1), rowY(m, 1))
	m = u.(*Model)
	if strings.Contains(m.View(), "ULTIMATE TUI") {
		t.Fatal("clicking inside the help panel should close it")
	}
}
