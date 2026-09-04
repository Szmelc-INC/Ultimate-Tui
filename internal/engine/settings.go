package engine

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The F3 overlay is a live theme/style picker: two columns (colour palette,
// styleset), arrow keys preview the highlighted choice by repainting the whole
// UI under the panel, Enter commits (and persists via Options.SaveTheme when
// set), Esc restores what was active on open.

func (m *Model) openSettings() {
	m.overlay = ovSettings
	m.setSavedTheme, m.setSavedStyle = m.themeName, m.styleName
	m.setCol = 0
	m.setRow[0] = indexOf(ThemeNames(), m.themeName)
	m.setRow[1] = indexOf(StyleNames(), m.styleName)
}

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "f3", "q":
		m.themeName, m.styleName = m.setSavedTheme, m.setSavedStyle
		m.invalidateTheme()
		m.overlay = ovNone
	case "enter":
		m.overlay = ovNone
		if m.opts.SaveTheme != nil {
			_ = m.opts.SaveTheme(m.themeName, m.styleName)
		}
	case "left", "h", "shift+tab":
		m.setCol = 0
	case "right", "l", "tab":
		m.setCol = 1
	case "up", "k":
		m.settingsMove(-1)
	case "down", "j":
		m.settingsMove(1)
	case "home", "g":
		m.settingsMove(-1 << 30)
	case "end", "G":
		m.settingsMove(1 << 30)
	}
	return m, nil
}

// settingsMove shifts the cursor in the active column by d (saturating), applies
// the newly highlighted value as a live preview, and drops the theme memo so the
// next View repaints in it.
func (m *Model) settingsMove(d int) {
	if m.setCol == 0 {
		names := ThemeNames()
		m.setRow[0] = clampInt(m.setRow[0]+d, 0, len(names)-1)
		m.themeName = names[m.setRow[0]]
	} else {
		names := StyleNames()
		m.setRow[1] = clampInt(m.setRow[1]+d, 0, len(names)-1)
		m.styleName = names[m.setRow[1]]
	}
	m.invalidateTheme()
}

func (m *Model) renderSettings(frame string) string {
	return overlayCenter(frame, m.settingsPanel(), m.w, m.h)
}

func (m *Model) settingsPanel() string {
	th := m.theme()

	column := func(heading string, items []string, sel int, active bool) string {
		mark := "  "
		if active {
			mark = "▸ "
		}
		lines := []string{th.menuColHdr.Render(mark + heading)}
		for i, it := range items {
			switch {
			case i == sel && active:
				lines = append(lines, th.menuSel.Render(" "+padRight(it, 13)+" "))
			case i == sel:
				lines = append(lines, th.helpKey.Render(" "+padRight(it, 13)+" "))
			default:
				lines = append(lines, th.menuItem.Render("  "+it))
			}
		}
		return strings.Join(lines, "\n")
	}

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		column("theme", ThemeNames(), m.setRow[0], m.setCol == 0),
		"    ",
		column("style", StyleNames(), m.setRow[1], m.setCol == 1),
	)

	var b strings.Builder
	b.WriteString(th.helpTitle.Render("settings — theme & style"))
	b.WriteString("\n\n")
	b.WriteString(cols)
	b.WriteString("\n\n")
	b.WriteString(th.helpDim.Render("↑↓ pick · ←→ column · Enter apply · Esc cancel"))
	return th.helpBox.Render(b.String())
}

func indexOf(ss []string, v string) int {
	for i, s := range ss {
		if s == v {
			return i
		}
	}
	return 0
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return max(lo, min(v, hi))
}
