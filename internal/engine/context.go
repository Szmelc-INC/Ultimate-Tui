package engine

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/widget"
)

// The pane context menu (right mouse button, or mod+w) is a small keyboard-
// driven overlay. Mouse messages are dropped while any overlay is up (see
// Update), so the click opens it and the keyboard drives it from there.
//
//	Stop    — kill the pane's process (via the shared kill-confirm dialog)
//	Restart — tear the process down and spawn it again
//	Refresh — force a repaint without a restart
//	Split   — open the split prompt (direction + %), dividing this pane in two
//	Close   — remove this pane (via a confirm dialog); disabled on the last pane
//	Edit    — open the pane editor
//
// Right-clicking dead space (there is exactly one such spot: the status bar
// row — the pane tree tiles the rest of the body exactly) opens a one-item
// menu instead: New pane, which wraps the whole layout in a fresh split. Its
// target is ctxLeaf == -1, the one place besides a valid leaf index that
// openContext accepts.
//
// Stop/Restart/Refresh are greyed out for a widget that does not implement the
// matching optional interface; Split and Edit are always available.

type ctxAction int

const (
	ctxStop ctxAction = iota
	ctxRestart
	ctxRefresh
	ctxSplit
	ctxClose
	ctxEdit
	ctxNew
)

type ctxItem struct {
	label   string
	act     ctxAction
	enabled bool
}

func (m *Model) contextItems(idx int) []ctxItem {
	if idx < 0 {
		return []ctxItem{{"New pane", ctxNew, true}}
	}
	var w widget.Widget
	if idx < len(m.leaves) {
		w = m.leaves[idx].w
	}
	_, canStop := w.(widget.Interrupter)
	_, canRestart := w.(widget.Restarter)
	_, canRefresh := w.(widget.Refresher)
	return []ctxItem{
		{"Stop", ctxStop, canStop},
		{"Restart", ctxRestart, canRestart},
		{"Refresh", ctxRefresh, canRefresh},
		{"Split", ctxSplit, true},
		{"Close", ctxClose, len(m.leaves) > 1},
		{"Edit", ctxEdit, true},
	}
}

// openContext opens the menu for leaf idx, or the "New pane" menu for dead
// space when idx is -1 (see contextItems).
func (m *Model) openContext(idx int) {
	if idx < -1 || idx >= len(m.leaves) {
		return
	}
	m.overlay = ovContext
	m.ctxLeaf = idx
	m.ctxRow = ctxFirstEnabled(m.contextItems(idx))
}

func ctxFirstEnabled(items []ctxItem) int {
	for i, it := range items {
		if it.enabled {
			return i
		}
	}
	return 0
}

func ctxLastEnabled(items []ctxItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].enabled {
			return i
		}
	}
	return 0
}

// ctxStepRow moves the cursor by dir, skipping disabled rows, wrapping once.
func ctxStepRow(items []ctxItem, from, dir int) int {
	n := len(items)
	if n == 0 {
		return 0
	}
	for i := 1; i <= n; i++ {
		j := ((from+dir*i)%n + n) % n
		if items[j].enabled {
			return j
		}
	}
	return from
}

func (m *Model) handleContextKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.contextItems(m.ctxLeaf)
	switch msg.String() {
	case "esc", "q", m.mod + "+w":
		m.overlay = ovNone
	case "up", "k":
		m.ctxRow = ctxStepRow(items, m.ctxRow, -1)
	case "down", "j":
		m.ctxRow = ctxStepRow(items, m.ctxRow, +1)
	case "home", "g":
		m.ctxRow = ctxFirstEnabled(items)
	case "end", "G":
		m.ctxRow = ctxLastEnabled(items)
	case "enter", " ", "l", "right":
		if m.ctxRow >= 0 && m.ctxRow < len(items) && items[m.ctxRow].enabled {
			return m.runContextAction(items[m.ctxRow].act)
		}
	}
	return m, nil
}

func (m *Model) runContextAction(a ctxAction) (tea.Model, tea.Cmd) {
	if a == ctxNew {
		m.overlayReturn = ovNone
		m.openSplitPrompt(-1)
		return m, nil
	}

	idx := m.ctxLeaf
	if idx < 0 || idx >= len(m.leaves) {
		m.overlay = ovNone
		return m, nil
	}
	lf := m.leaves[idx]
	switch a {
	case ctxStop:
		// Reuse the kill-confirm dialog; it captures its own target index.
		return m.openConfirmKill(idx, m.leafName(idx))
	case ctxRestart:
		m.overlay = ovNone
		if r, ok := lf.w.(widget.Restarter); ok {
			return m, r.Restart()
		}
	case ctxRefresh:
		m.overlay = ovNone
		if r, ok := lf.w.(widget.Refresher); ok {
			return m, r.Refresh()
		}
	case ctxSplit:
		m.overlayReturn = ovNone
		m.openSplitPrompt(idx)
	case ctxClose:
		if len(m.leaves) > 1 {
			return m.openConfirmClose(idx, m.leafName(idx))
		}
	case ctxEdit:
		m.overlayReturn = ovNone // Esc from the editor returns to the bare frame
		m.openPaneEditor(idx)
	}
	return m, nil
}

// contextMouseClick maps a click's content-relative row to a menu item (see
// contextPanel: title, blank, then one row per item) and runs it right away
// — a menu row is a button, not a focus target that needs a second Enter.
func (m *Model) contextMouseClick(row int) (tea.Model, tea.Cmd) {
	items := m.contextItems(m.ctxLeaf)
	i := row - 2
	if i < 0 || i >= len(items) || !items[i].enabled {
		return m, nil
	}
	m.ctxRow = i
	return m.runContextAction(items[i].act)
}

func (m *Model) renderContext(frame string) string {
	return m.splice(frame, m.contextPanel())
}

func (m *Model) contextPanel() string {
	th := m.theme()
	items := m.contextItems(m.ctxLeaf)

	title := "pane · " + m.leafName(m.ctxLeaf)
	if m.ctxLeaf < 0 {
		title = "empty space"
	}
	var b strings.Builder
	b.WriteString(th.helpTitle.Render(title))
	b.WriteString("\n\n")
	for i, it := range items {
		switch {
		case !it.enabled:
			b.WriteString(th.helpDim.Render("   " + it.label))
		case i == m.ctxRow:
			b.WriteString(th.menuSel.Render(" " + padRight(it.label, 10) + " "))
		default:
			b.WriteString(th.menuItem.Render("   " + it.label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(th.helpDim.Render("↑↓ pick · Enter run · Esc close"))
	return th.helpBox.Render(b.String())
}
