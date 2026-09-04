package engine

import tea "github.com/charmbracelet/bubbletea"

// The F3 settings overlay is built out in a later change. For now it is a
// placeholder that only knows how to open and close itself, so the overlay
// plumbing has a real third state to route to.
func (m *Model) openSettings() { m.overlay = ovSettings }

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f3", "esc", "q", "enter":
		m.overlay = ovNone
	}
	return m, nil
}

func (m *Model) renderSettings(frame string) string { return frame }
