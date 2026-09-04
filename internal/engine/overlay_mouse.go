package engine

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleOverlayMouse is Update's mouse path while an overlay is active
// (splice(), see chrome.go, stamps m.ovX/Y/W/H with its bounding box every
// render). A left/right press outside that box dismisses the overlay exactly
// like Esc would — routed through handleKey so each overlay's own Esc
// semantics (cancel, discard-confirm, return-to-parent-overlay, …) apply
// unchanged. A press inside is handed to that overlay's own hit-test, in
// content-relative coordinates (past the shared border+padding every panel
// has — see panelContentOffsetX/Y).
func (m *Model) handleOverlayMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	inside := msg.X >= m.ovX && msg.X < m.ovX+m.ovW && msg.Y >= m.ovY && msg.Y < m.ovY+m.ovH
	if !inside {
		if msg.Action == tea.MouseActionPress &&
			(msg.Button == tea.MouseButtonLeft || msg.Button == tea.MouseButtonRight) {
			return m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		}
		return m, nil
	}

	cx := msg.X - m.ovX - panelContentOffsetX
	cy := msg.Y - m.ovY - panelContentOffsetY
	press := msg.Action == tea.MouseActionPress
	wheelUp := msg.Button == tea.MouseButtonWheelUp
	wheelDown := msg.Button == tea.MouseButtonWheelDown

	switch m.overlay {
	case ovHelp:
		if press {
			return m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		}
	case ovConfirm:
		if press && msg.Button == tea.MouseButtonLeft {
			return m.confirmMouseClick(cy)
		}
	case ovContext:
		if press && msg.Button == tea.MouseButtonLeft {
			return m.contextMouseClick(cy)
		}
	case ovControl:
		if press && msg.Button == tea.MouseButtonLeft {
			return m.controlMouseClick(cy)
		}
	case ovSettings:
		switch {
		case press && msg.Button == tea.MouseButtonLeft:
			m.settingsMouseClick(msg.X, cy)
		case wheelUp:
			m.settingsMove(-1)
		case wheelDown:
			m.settingsMove(1)
		}
	case ovSplit:
		switch {
		case press && msg.Button == tea.MouseButtonLeft:
			m.splitMouseClick(cy)
		case wheelUp:
			m.splitAdjust(1)
		case wheelDown:
			m.splitAdjust(-1)
		}
	case ovEditor:
		if press && msg.Button == tea.MouseButtonLeft {
			return m.editorMouseClick(cx, cy)
		}
	}
	return m, nil
}
