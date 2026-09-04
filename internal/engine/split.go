package engine

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// The split prompt (ovSplit) serves two context-menu rows: Split (divide one
// existing pane in two) and New (right-click on dead space — wrap the whole
// layout in a fresh split). Both ask the same two questions — direction and
// the percentage the original content keeps — and both create exactly one
// new blank text pane; only what gets wrapped differs (spTarget: a leaf
// index, or -1 for the whole root). Confirming performs a structural edit
// (the Children slice grows) so it always ends with rebuild().

// openSplitPrompt opens the prompt for leaf idx, or for the whole layout
// when idx is -1 (the "New pane" context-menu row).
func (m *Model) openSplitPrompt(idx int) {
	m.overlay = ovSplit
	m.spTarget = idx
	m.spDir = "row"
	m.spPct = 50
	m.spField = 0
}

func (m *Model) handleSplitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.overlay = m.overlayReturn
		m.overlayReturn = ovNone
	case "tab", "down", "j":
		m.spField = (m.spField + 1) % 2
	case "shift+tab", "up", "k":
		m.spField = (m.spField - 1 + 2) % 2
	case "left", "h":
		m.splitAdjust(-1)
	case "right", "l":
		m.splitAdjust(1)
	case "enter":
		return m.confirmSplit()
	}
	return m, nil
}

func (m *Model) splitAdjust(dir int) {
	switch m.spField {
	case 0:
		if m.spDir == "row" {
			m.spDir = "column"
		} else {
			m.spDir = "row"
		}
	case 1:
		m.spPct = clampInt(m.spPct+dir*5, 5, 95)
	}
}

// confirmSplit performs the tree edit Enter commits: wrap either one leaf's
// Pane or the whole cfg.Layout in a new split with a fresh blank pane as the
// other half, then rebuild(). Focus follows the new pane (New) or stays on
// the original content (Split) — re-resolved by pane identity since rebuild
// invalidates every index.
func (m *Model) confirmSplit() (tea.Model, tea.Cmd) {
	dir, pct := m.spDir, m.spPct
	newPane := &config.Pane{Type: "text", Title: "New pane"}
	sizeA := strconv.Itoa(pct) + "%"
	sizeB := strconv.Itoa(100-pct) + "%"

	var focusPane *config.Pane
	if m.spTarget < 0 {
		old := m.cfg.Layout
		m.cfg.Layout = config.Node{
			Direction: dir,
			Children: []config.Child{
				{Size: sizeA, Node: old},
				{Size: sizeB, Node: config.Node{Pane: newPane}},
			},
		}
		focusPane = newPane
	} else {
		if m.spTarget >= len(m.leaves) {
			m.overlay, m.overlayReturn = m.overlayReturn, ovNone
			return m, nil
		}
		lf := m.leaves[m.spTarget]
		target := &m.cfg.Layout
		if lf.parent != nil {
			target = &lf.parent.Node
		}
		*target = config.Node{
			Direction: dir,
			Children: []config.Child{
				{Size: sizeA, Node: config.Node{Pane: lf.pane}},
				{Size: sizeB, Node: config.Node{Pane: newPane}},
			},
		}
		focusPane = lf.pane
	}

	cmd := m.rebuild()
	if i := m.indexOfPane(focusPane); i >= 0 {
		m.focus = i
	}
	m.overlay = m.overlayReturn
	m.overlayReturn = ovNone
	return m, cmd
}

// closeLeaf removes leaf idx from the layout: the last pane refuses, a
// two-child split collapses into its surviving sibling rather than leaving a
// degenerate one-child node, and any other split just drops that child and
// renormalizes percent siblings back to summing to 100. rebuild() re-derives
// the tree and Closes the removed pane's widget (it's no longer reachable
// from cfg.Layout, which is exactly rebuild's orphan-cleanup rule).
func (m *Model) closeLeaf(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.leaves) || len(m.leaves) <= 1 {
		return nil
	}
	lf := m.leaves[idx]
	if lf.parent == nil {
		return nil // root leaf, but len>1 means it can't actually be one
	}

	var focusPane *config.Pane
	if idx != m.focus && m.focus < len(m.leaves) {
		focusPane = m.leaves[m.focus].pane
	}

	splitNode := lf.split
	removeIdx := -1
	for i := range splitNode.Children {
		if &splitNode.Children[i] == lf.parent {
			removeIdx = i
			break
		}
	}
	if removeIdx < 0 {
		return nil
	}
	splitNode.Children = append(splitNode.Children[:removeIdx], splitNode.Children[removeIdx+1:]...)

	if len(splitNode.Children) == 1 {
		*splitNode = splitNode.Children[0].Node
	} else {
		renormalizePercents(splitNode)
	}

	cmd := m.rebuild()
	if focusPane != nil {
		if i := m.indexOfPane(focusPane); i >= 0 {
			m.focus = i
		} else {
			m.focus = 0
		}
	} else {
		m.focus = clampInt(idx, 0, len(m.leaves)-1)
	}
	return cmd
}

// renormalizePercents rescales n's children back to whole percents summing to
// exactly 100, but only when every child is already percent-sized — a
// fixed/flex mix is left alone rather than guessed at.
func renormalizePercents(n *config.Node) {
	vals := make([]float64, len(n.Children))
	sum := 0.0
	for i, c := range n.Children {
		s := parseSize(c.Size)
		if s.kind != percent {
			return
		}
		vals[i] = s.value
		sum += s.value
	}
	if sum <= 0 {
		return
	}
	acc := 0
	for i := range n.Children {
		pct := int(vals[i] / sum * 100)
		acc += pct
		n.Children[i].Size = strconv.Itoa(pct) + "%"
	}
	if d := 100 - acc; d != 0 {
		last := len(n.Children) - 1
		cur := int(vals[last] / sum * 100)
		n.Children[last].Size = strconv.Itoa(cur+d) + "%"
	}
}

func (m *Model) openConfirmClose(idx int, name string) (tea.Model, tea.Cmd) {
	m.overlay = ovConfirm
	m.confirmKind = cfClose
	m.confirmLeaf = idx
	m.confirmPrompt = "Close " + name + "? Its process (if any) is stopped."
	return m, nil
}

// splitMouseClick selects the field a click landed on (see splitPanel: title,
// blank, then one row per field — direction, then percent).
func (m *Model) splitMouseClick(row int) {
	switch row - 2 {
	case 0:
		m.spField = 0
	case 1:
		m.spField = 1
	}
}

func (m *Model) renderSplit(frame string) string {
	return m.splice(frame, m.splitPanel())
}

func (m *Model) splitPanel() string {
	th := m.theme()

	title := "split · " + m.leafName(m.spTarget)
	if m.spTarget < 0 {
		title = "new pane"
	}

	dirLabel := "vertical  (side by side)"
	if m.spDir == "column" {
		dirLabel = "horizontal  (stacked)"
	}

	field := func(row int, label, value string) string {
		mark := "  "
		v := th.menuItem.Render(value)
		if row == m.spField {
			mark = th.helpKey.Render("› ")
			v = th.helpKey.Render("‹ ") + value + th.helpKey.Render(" ›")
		}
		return mark + th.helpDesc.Render(padRight(label, 11)) + v
	}

	var b strings.Builder
	b.WriteString(th.helpTitle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(field(0, "direction", dirLabel))
	b.WriteString("\n")
	b.WriteString(field(1, "keeps", strconv.Itoa(m.spPct)+"%  ("+sliderBar(m.spPct, 0, 100)+")  new pane: "+strconv.Itoa(100-m.spPct)+"%"))
	b.WriteString("\n\n")
	b.WriteString(th.helpDim.Render("↑↓ field · ←→ change · Enter split · Esc cancel"))
	return th.helpBox.Render(b.String())
}
