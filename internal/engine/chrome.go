package engine

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// statusBar renders the one-line bottom bar: a mode chip, the focused pane's
// title, and a right-aligned hint strip. Always exactly w cells, one line.
func (m *Model) statusBar(w int) string {
	th := m.theme()
	mode := th.barMode.Render(" " + m.Mode() + " ")

	name := m.Focused()
	left := mode + th.bar.Render("  "+name)

	hint := strings.Join([]string{
		th.barKey.Render("↑↓←→") + th.bar.Render(" focus"),
		th.barKey.Render("enter") + th.bar.Render(" lock"),
		th.barKey.Render("esc") + th.bar.Render(" release"),
		th.barKey.Render("F1") + th.bar.Render(" help"),
		th.barKey.Render(m.mod+"+q") + th.bar.Render(" quit"),
	}, th.bar.Render("  ·  "))

	gap := w - lipgloss.Width(left) - lipgloss.Width(hint)
	if gap < 2 {
		// Too narrow for the full hint strip — keep the mode + name only.
		return th.bar.Width(w).MaxWidth(w).Render(ansi.Truncate(mode+th.bar.Render("  "+name), w, "…"))
	}
	return left + th.bar.Render(strings.Repeat(" ", gap)) + hint
}

// renderHelp splices a centered help panel over the rendered frame. The panel
// content is fully self-describing: engine navigation plus this TUI's own
// keybindings.
func (m *Model) renderHelp(frame string) string {
	panel := m.helpPanel()
	return overlayCenter(frame, panel, m.w, m.h)
}

func (m *Model) helpPanel() string {
	th := m.theme()

	row := func(key, desc string) string {
		return th.helpKey.Render(padRight(key, 16)) + th.helpDesc.Render(desc)
	}

	var b strings.Builder
	b.WriteString(th.helpTitle.Render("ULTIMATE TUI — keys"))
	b.WriteString("\n\n")
	b.WriteString(th.helpSection.Render("navigation"))
	b.WriteString("\n")
	for _, ln := range []string{
		row("↑ ↓ ← →", "move focus to the pane in that direction"),
		row("Tab / Shift+Tab", "cycle focus through every pane"),
		row("Enter", "lock focus into the pane — keys now go to it"),
		row("Esc", "release the focus lock"),
		row("Esc Esc", "send a literal Esc to a locked shell/editor"),
		row("F1", "toggle this help"),
		row(m.mod+"+q", "quit  ("+m.mod+" = the modifier key)"),
		row("Ctrl+C", "force quit"),
		row("mouse", "click a pane to focus · wheel scrolls under cursor"),
	} {
		b.WriteString(ln)
		b.WriteString("\n")
	}

	if len(m.tuiKeys) > 0 {
		b.WriteString("\n")
		b.WriteString(th.helpSection.Render("this TUI"))
		b.WriteString("\n")
		keys := mapKeys(m.tuiKeys)
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(row(k, m.tuiKeys[k]))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(th.helpDim.Render("F1 or Esc to close"))

	return th.helpBox.Render(b.String())
}

// overlayCenter places overlay centered on top of base, both measured in display
// cells. base is assumed to be w x h. Lines of overlay replace the middle span
// of the corresponding base lines, ANSI-safely.
func overlayCenter(base, overlay string, w, h int) string {
	baseLines := strings.Split(base, "\n")
	ovLines := strings.Split(overlay, "\n")
	ovH := len(ovLines)
	ovW := 0
	for _, ln := range ovLines {
		if x := lipgloss.Width(ln); x > ovW {
			ovW = x
		}
	}

	top := (h - ovH) / 2
	if top < 0 {
		top = 0
	}
	left := (w - ovW) / 2
	if left < 0 {
		left = 0
	}

	for i, ov := range ovLines {
		y := top + i
		if y < 0 || y >= len(baseLines) {
			continue
		}
		bl := baseLines[y]
		leftPart := ansi.Truncate(bl, left, "")
		leftPart += strings.Repeat(" ", max(0, left-lipgloss.Width(leftPart)))
		ov += strings.Repeat(" ", max(0, ovW-lipgloss.Width(ov)))
		rightStart := left + ovW
		rightPart := ansi.TruncateLeft(bl, rightStart, "")
		line := leftPart + "\x1b[0m" + ov + "\x1b[0m" + rightPart
		// Keep the composed line exactly w wide.
		if lipgloss.Width(line) > w {
			line = ansi.Truncate(line, w, "")
		} else {
			line += strings.Repeat(" ", w-lipgloss.Width(line))
		}
		baseLines[y] = line
	}
	return strings.Join(baseLines, "\n")
}

func padRight(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
