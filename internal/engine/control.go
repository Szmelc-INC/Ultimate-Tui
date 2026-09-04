package engine

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The F2 overlay is the global control panel: a short menu of runner-wide
// settings and sub-panels, plus a read-only summary of what is in effect.
// The theme/style picker (was F3) is now reached from here — F3 no longer
// opens anything, so a locked shell/editor gets its F3 key back.

// controlItem is one selectable row in the F2 menu.
type controlItem struct {
	label string
	open  func(m *Model) // what Enter on this row does
}

func (m *Model) controlItems() []controlItem {
	return []controlItem{
		{"Theme & style", func(m *Model) {
			m.overlayReturn = ovControl
			m.openSettings()
		}},
		{"Edit config", func(m *Model) {
			m.overlayReturn = ovControl
			m.openConfigEditor()
		}},
	}
}

func (m *Model) openControl() {
	m.overlay = ovControl
	m.overlayReturn = ovNone
	if m.ctrlRow >= len(m.controlItems()) {
		m.ctrlRow = 0
	}
}

func (m *Model) handleControlKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.controlItems()
	switch msg.String() {
	case "esc", "f2", "q":
		m.overlay = ovNone
	case "up", "k":
		m.ctrlRow = clampInt(m.ctrlRow-1, 0, len(items)-1)
	case "down", "j":
		m.ctrlRow = clampInt(m.ctrlRow+1, 0, len(items)-1)
	case "home", "g":
		m.ctrlRow = 0
	case "end", "G":
		m.ctrlRow = len(items) - 1
	case "enter", " ", "l", "right":
		if m.ctrlRow >= 0 && m.ctrlRow < len(items) {
			items[m.ctrlRow].open(m)
		}
	}
	return m, nil
}

func (m *Model) renderControl(frame string) string {
	return overlayCenter(frame, m.controlPanel(), m.w, m.h)
}

func (m *Model) controlPanel() string {
	th := m.theme()
	items := m.controlItems()

	var b strings.Builder
	b.WriteString(th.helpTitle.Render("controls — F2"))
	b.WriteString("\n\n")
	for i, it := range items {
		if i == m.ctrlRow {
			b.WriteString(th.menuSel.Render(" " + padRight(it.label+"  ", 20) + "› "))
		} else {
			b.WriteString(th.menuItem.Render("  " + it.label))
		}
		b.WriteString("\n")
	}

	stat := func(k, v string) string {
		return th.helpKey.Render(padRight(k, 12)) + th.helpDesc.Render(v)
	}
	b.WriteString("\n")
	b.WriteString(th.helpSection.Render("in effect"))
	b.WriteString("\n")
	b.WriteString(stat("modifier", m.mod) + "\n")
	b.WriteString(stat("status bar", boolWord(m.opts.StatusBar)) + "\n")
	b.WriteString(stat("theme", m.themeName) + "\n")
	b.WriteString(stat("style", m.styleName) + "\n")
	b.WriteString(stat("panes", fmt.Sprintf("%d", len(m.leaves))) + "\n")

	b.WriteString("\n")
	b.WriteString(th.helpDim.Render("↑↓ pick · Enter open · Esc close"))
	return th.helpBox.Render(b.String())
}

func boolWord(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
