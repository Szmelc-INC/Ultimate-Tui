package engine

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/widget"
)

// overlayKind is the modal layer drawn on top of the pane tree. At most one is
// active; while one is, handleKey routes every key to it and the pane tree sees
// nothing.
type overlayKind int

const (
	ovNone overlayKind = iota
	ovHelp
	ovConfirm
	ovSettings
	ovControl
	ovContext
	ovEditor
)

// confirmKind is what a confirmed (y / Enter) confirm dialog does.
type confirmKind int

const (
	cfQuit    confirmKind = iota // end the program
	cfKill                       // stop the focused pane's process
	cfDiscard                    // leave the editor with unsaved edits
)

func (m *Model) overlayActive() bool { return m.overlay != ovNone }

// openConfirmQuit raises the "Quit ultui?" dialog. This is the only way out of
// the program now — Ctrl+C in NAV mode lands here instead of quitting outright.
func (m *Model) openConfirmQuit() (tea.Model, tea.Cmd) {
	m.overlay = ovConfirm
	m.confirmKind = cfQuit
	m.confirmPrompt = "Quit ultui?"
	return m, nil
}

// openConfirmKill raises the "kill this pane's process?" dialog. Reached by
// Ctrl+C while a pane that hosts a killable process is locked. The target leaf
// is captured now, not re-read at confirm time — a click behind the dialog
// must not be able to redirect the kill.
func (m *Model) openConfirmKill(idx int, name string) (tea.Model, tea.Cmd) {
	m.overlay = ovConfirm
	m.confirmKind = cfKill
	m.confirmLeaf = idx
	m.confirmPrompt = "Kill the process in " + name + "?"
	return m, nil
}

// handleConfirmKey runs while the confirm dialog is up. y / Y / Enter (and a
// second Ctrl+C, for muscle memory) confirm; n / N / Esc cancel; anything else
// is ignored and the dialog stays.
func (m *Model) handleConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y", "enter", "ctrl+c":
		return m.confirmYes()
	case "n", "N", "esc":
		if m.confirmKind == cfDiscard {
			m.overlay = ovEditor // cancel → back to the editor, edits intact
		} else {
			m.overlay = ovNone
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) confirmYes() (tea.Model, tea.Cmd) {
	m.overlay = ovNone
	switch m.confirmKind {
	case cfQuit:
		m.Close()
		return m, tea.Quit
	case cfKill:
		if m.confirmLeaf >= 0 && m.confirmLeaf < len(m.leaves) {
			if it, ok := m.leaves[m.confirmLeaf].w.(widget.Interrupter); ok {
				_ = it.Interrupt()
			}
		}
		m.locked, m.pendingEsc = false, false
		return m, nil
	case cfDiscard:
		m.closeEditor()
		return m, nil
	}
	return m, nil
}

// lockedInterrupt is Ctrl+C inside a locked pane: if the pane hosts a killable
// process, confirm-then-kill just that process; otherwise forward the key to
// the widget (a shell child gets its own ^C byte).
func (m *Model) lockedInterrupt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	lf := m.leaves[m.focus]
	if _, ok := lf.w.(widget.Interrupter); ok {
		return m.openConfirmKill(m.focus, m.Focused())
	}
	return m.forwardKey(msg)
}

// confirmPanel is the centered dialog body.
func (m *Model) confirmPanel() string {
	th := m.theme()
	b := th.helpTitle.Render(m.confirmPrompt) + "\n\n" +
		th.helpKey.Render("y") + th.helpDesc.Render(" / ") +
		th.helpKey.Render("Enter") + th.helpDesc.Render("   confirm") + "\n" +
		th.helpKey.Render("n") + th.helpDesc.Render(" / ") +
		th.helpKey.Render("Esc") + th.helpDesc.Render("     cancel")
	return th.helpBox.Render(b)
}

func (m *Model) renderConfirm(frame string) string {
	return overlayCenter(frame, m.confirmPanel(), m.w, m.h)
}
