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

// closeSettings dismisses the picker, returning to whatever overlay opened it
// (the F2 control panel) or to the bare frame. overlayReturn is one level deep.
func (m *Model) closeSettings() {
	m.overlay = m.overlayReturn
	m.overlayReturn = ovNone
}

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.themeName, m.styleName = m.setSavedTheme, m.setSavedStyle
		m.invalidateTheme()
		m.closeSettings()
	case "enter":
		m.closeSettings()
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
		m.settingsSetRow(0, m.setRow[0]+d)
	} else {
		m.settingsSetRow(1, m.setRow[1]+d)
	}
}

// settingsSetRow jumps (not shifts) the given column's cursor to row i,
// clamped to the list's bounds, and applies it as a live preview — shared by
// keyboard (settingsMove, relative) and mouse (settingsMouseClick, absolute).
func (m *Model) settingsSetRow(col, i int) {
	if col == 0 {
		names := ThemeNames()
		m.setRow[0] = clampInt(i, 0, len(names)-1)
		m.themeName = names[m.setRow[0]]
	} else {
		names := StyleNames()
		m.setRow[1] = clampInt(i, 0, len(names)-1)
		m.styleName = names[m.setRow[1]]
	}
	m.invalidateTheme()
}

// settingsMouseClick picks a column from absolute screen X against the exact
// boundary settingsPanel stamped in setCol0W (the theme column's own width —
// it carries a color swatch the style column doesn't, so the two columns
// aren't equal width and ovW/2 would misjudge a click near the middle) and a
// row from the content-relative Y — see settingsPanel: title, blank, a
// header line, then one row per item. Unlike settingsMove (keyboard,
// relative — clamping into range is correct there), a click outside the item
// list must be a no-op: settingsSetRow's clamp would otherwise let a click on
// the title/header row (i < 0) apply row 0 as a live theme/style preview.
func (m *Model) settingsMouseClick(absX, row int) {
	// setCol0W is stamped by settingsPanel on every render, and in normal use
	// a render always happens between openSettings() and the first click the
	// user can possibly make on what it drew. But nothing enforces that here
	// the way handleOverlayMouse's m.ready guard does elsewhere — so if this
	// is ever reached with setCol0W still at its zero value, fall back to the
	// coarse half-width split rather than let the boundary collapse to the
	// panel's left edge and misroute every click into the style column.
	boundary := m.ovX + m.ovW/2
	if m.setCol0W > 0 {
		boundary = m.ovX + panelContentOffsetX + m.setCol0W
	}
	col := 0
	if absX >= boundary {
		col = 1
	}
	i := row - 3
	n := len(ThemeNames())
	if col == 1 {
		n = len(StyleNames())
	}
	if i < 0 || i >= n {
		return
	}
	m.setCol = col
	m.settingsSetRow(col, i)
}

func (m *Model) renderSettings(frame string) string {
	return m.splice(frame, m.settingsPanel())
}

func (m *Model) settingsPanel() string {
	th := m.theme()

	// column renders one picker list. swatch, when non-nil, prepends a fixed
	// 2-cell preview block to each item row — OUTSIDE the padded/highlighted
	// label (padRight + menuSel/helpKey operate on `it` alone) so the
	// selection highlight bar stays a uniform width across rows regardless
	// of what color the swatch next to it happens to be.
	column := func(heading string, items []string, sel int, active bool, swatch func(string) string) string {
		mark := "  "
		if active {
			mark = "▸ "
		}
		lines := []string{th.menuColHdr.Render(mark + heading)}
		for i, it := range items {
			var row string
			switch {
			case i == sel && active:
				row = th.menuSel.Render(" " + padRight(it, 13) + " ")
			case i == sel:
				row = th.helpKey.Render(" " + padRight(it, 13) + " ")
			default:
				row = th.menuItem.Render("  " + it)
			}
			if swatch != nil {
				row = swatch(it) + " " + row
			}
			lines = append(lines, row)
		}
		return strings.Join(lines, "\n")
	}

	// A real preview, not decoration: each theme row gets a two-tone block —
	// one cell of that palette's accent, one of its accent2 — sourced
	// straight from the palettes registry. Two colors, not a single
	// Background(pal.accent) block, on purpose: menuSel (the selected row's
	// own highlight) is ALSO Background(pal.accent), so on the currently
	// selected row a single-accent swatch would be the exact same color as
	// the highlight bar sitting right next to it and stop reading as a swatch
	// at all — the accent2 half guarantees the block always has a color the
	// highlight doesn't. Style has no comparable color axis (it's border +
	// glyphs), so only the theme column gets a swatch; a glyph preview was
	// considered and rejected for the style column — it would leak another
	// styleset's own unicode border runes into a frame currently rendered in
	// "ascii", the one styleset that must stay pure 7-bit.
	themeSwatch := func(name string) string {
		pal := palettes[resolveThemeName(name)]
		return lipgloss.NewStyle().Background(pal.accent).Render(" ") +
			lipgloss.NewStyle().Background(pal.accent2).Render(" ")
	}

	col0 := column("theme", ThemeNames(), m.setRow[0], m.setCol == 0, themeSwatch)
	col1 := column("style", StyleNames(), m.setRow[1], m.setCol == 1, nil)

	m.setCol0W = 0
	for _, ln := range strings.Split(col0, "\n") {
		if x := lipgloss.Width(ln); x > m.setCol0W {
			m.setCol0W = x
		}
	}

	cols := lipgloss.JoinHorizontal(lipgloss.Top, col0, "    ", col1)

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
