package engine

import (
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/widget"
)

// The editor overlay: a slim tab navbar over a list of typed field controls for
// one pane. Edits are buffered in editBuf and applied on demand (⌥s / Enter) —
// applying mutates the live *config.Pane / *config.Child in place and, when a
// widget-affecting field changed, rebuilds the pane's widget (closing the old
// one first). Disk persistence and the confirm-on-exit flow come in a later
// commit; today Esc just closes.

type fieldKind int

const (
	fSelect fieldKind = iota // pick from choices
	fText                    // free-typed string
	fBool                    // on/off
	fSlider                  // integer in [min,max] by step
)

type editScope int

const (
	edScopePane   editScope = iota // one pane, opened from the context menu
	edScopeConfig                  // the whole config, opened from F2
)

type edField struct {
	key            string
	label          string
	kind           fieldKind
	choices        []string
	min, max, step int
}

// editBuf is the scratch copy of a pane's editable values.
type editBuf struct {
	typ, title             string
	content, file, mdStyle string
	command, shell         string
	border                 bool
	sizeMode               string // auto | percent | fixed | flex
	sizeVal                int
	hasSize                bool // false for the root pane (no enclosing split)
}

var mdStyleChoices = []string{"", "dark", "light", "dracula", "tokyo-night", "pink", "ascii", "notty"}
var sizeModeChoices = []string{"auto", "percent", "fixed", "flex"}

func (m *Model) editorTabs() []string {
	if m.edScope == edScopeConfig {
		return []string{"Config", "General", "Layout", "Widget"}
	}
	return []string{"General", "Layout", "Widget"}
}

// edLogicalTab maps the visible tab index to a scope-independent one:
// -1 = the config tab (only present in whole-config scope), 0/1/2 =
// General/Layout/Widget.
func (m *Model) edLogicalTab() int {
	if m.edScope == edScopeConfig {
		return m.edTab - 1
	}
	return m.edTab
}

func (m *Model) leafNames() []string {
	out := make([]string, len(m.leaves))
	for i := range m.leaves {
		out[i] = m.leafName(i)
	}
	return out
}

// editorFields is the field list for the current tab, given the buffered type.
func (m *Model) editorFields() []edField {
	if m.edLogicalTab() == -1 { // Config tab
		return []edField{
			{key: "cfg.name", label: "name", kind: fText},
			{key: "cfg.pane", label: "pane", kind: fSelect, choices: m.leafNames()},
		}
	}
	switch m.edLogicalTab() {
	case 0: // General
		return []edField{
			{key: "type", label: "type", kind: fSelect, choices: widget.Kinds()},
			{key: "title", label: "title", kind: fText},
			{key: "border", label: "border", kind: fBool},
		}
	case 1: // Layout
		if !m.eb.hasSize {
			return nil
		}
		fs := []edField{{key: "size.mode", label: "size mode", kind: fSelect, choices: sizeModeChoices}}
		switch m.eb.sizeMode {
		case "percent":
			fs = append(fs, edField{key: "size.value", label: "percent", kind: fSlider, min: 0, max: 100, step: 5})
		case "fixed":
			fs = append(fs, edField{key: "size.value", label: "cells", kind: fSlider, min: 0, max: 200, step: 1})
		case "flex":
			fs = append(fs, edField{key: "size.value", label: "weight", kind: fSlider, min: 1, max: 9, step: 1})
		}
		return fs
	default: // Widget
		switch m.eb.typ {
		case "text":
			return []edField{{key: "content", label: "content", kind: fText}}
		case "markdown":
			return []edField{
				{key: "content", label: "content", kind: fText},
				{key: "file", label: "file", kind: fText},
				{key: "markdownStyle", label: "md style", kind: fSelect, choices: mdStyleChoices},
			}
		case "command", "shell":
			return []edField{
				{key: "command", label: "command", kind: fText},
				{key: "shell", label: "shell", kind: fText},
			}
		}
		return nil
	}
}

// loadPane refills the scratch buffer from leaf idx and points the editor at
// it. Shared by both entry points and by the whole-config pane selector.
func (m *Model) loadPane(idx int) {
	if idx < 0 || idx >= len(m.leaves) {
		return
	}
	lf := m.leaves[idx]
	m.ctxLeaf = idx
	m.edErr = ""

	p := lf.pane
	mode, val := parseSizeStr("")
	if lf.parent != nil {
		mode, val = parseSizeStr(lf.parent.Size)
	}
	m.eb = editBuf{
		typ:      p.Type,
		title:    p.Title,
		border:   p.Border == nil || *p.Border,
		content:  p.Content,
		file:     p.File,
		mdStyle:  p.MarkdownStyle,
		command:  p.Command,
		shell:    p.Shell,
		sizeMode: mode,
		sizeVal:  val,
		hasSize:  lf.parent != nil,
	}
}

// openPaneEditor edits a single pane, reached from the context menu's Edit row.
func (m *Model) openPaneEditor(idx int) {
	if idx < 0 || idx >= len(m.leaves) {
		return
	}
	m.overlay = ovEditor
	m.edScope = edScopePane
	m.edTab, m.edRow, m.edDirty = 0, 0, false
	m.loadPane(idx)
}

// openConfigEditor edits the whole config, reached from the F2 control panel.
// Same controls; a leading "Config" tab carries the top-level name and a pane
// selector that walks every leaf.
func (m *Model) openConfigEditor() {
	m.overlay = ovEditor
	m.edScope = edScopeConfig
	m.edTab, m.edRow, m.edDirty = 0, 0, false
	m.loadPane(0)
}

func (m *Model) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.overlay = m.overlayReturn
		m.overlayReturn = ovNone
		return m, nil
	case "tab":
		m.edTab = (m.edTab + 1) % len(m.editorTabs())
		m.edRow = 0
		return m, nil
	case "shift+tab":
		m.edTab = (m.edTab - 1 + len(m.editorTabs())) % len(m.editorTabs())
		m.edRow = 0
		return m, nil
	case m.mod + "+s", "ctrl+s":
		return m, m.applyEdit()
	}

	fields := m.editorFields()
	if len(fields) == 0 {
		return m, nil
	}
	m.edRow = clampInt(m.edRow, 0, len(fields)-1)

	switch key {
	case "up", "shift+up":
		m.edRow = (m.edRow - 1 + len(fields)) % len(fields)
		return m, nil
	case "down", "shift+down":
		m.edRow = (m.edRow + 1) % len(fields)
		return m, nil
	}

	f := fields[m.edRow]
	switch key {
	case "left":
		if f.kind != fText {
			return m, m.editCycle(f, -1)
		}
		return m, nil
	case "right":
		if f.kind != fText {
			return m, m.editCycle(f, +1)
		}
		return m, nil
	case "enter":
		if f.kind == fText {
			return m, nil // no multiline editing here
		}
		if f.kind == fSelect && f.key == "cfg.pane" {
			return m, m.editCycle(f, +1)
		}
		return m, m.applyEdit()
	case " ":
		if f.kind == fBool {
			return m, m.editCycle(f, 1)
		}
	}
	if f.kind == fText {
		m.editText(f, msg)
	}
	return m, nil
}

// editGet returns a field's current buffered value as a string.
func (m *Model) editGet(f edField) string {
	switch f.key {
	case "type":
		return m.eb.typ
	case "title":
		return m.eb.title
	case "border":
		return boolWord(m.eb.border)
	case "content":
		return m.eb.content
	case "file":
		return m.eb.file
	case "markdownStyle":
		return m.eb.mdStyle
	case "command":
		return m.eb.command
	case "shell":
		return m.eb.shell
	case "size.mode":
		return m.eb.sizeMode
	case "size.value":
		return strconv.Itoa(m.eb.sizeVal)
	case "cfg.name":
		return m.cfg.Name
	case "cfg.pane":
		return m.leafName(m.ctxLeaf)
	}
	return ""
}

func (m *Model) editSetText(key, s string) {
	switch key {
	case "title":
		m.eb.title = s
	case "content":
		m.eb.content = s
	case "file":
		m.eb.file = s
	case "command":
		m.eb.command = s
	case "shell":
		m.eb.shell = s
	case "cfg.name":
		m.cfg.Name = s // top-level; committed to the live config immediately
	}
	m.edDirty = true
	m.edConfigDirty = true
}

func (m *Model) editText(f edField, msg tea.KeyMsg) {
	cur := m.editGet(f)
	switch msg.Type {
	case tea.KeyBackspace, tea.KeyDelete:
		r := []rune(cur)
		if len(r) > 0 {
			m.editSetText(f.key, string(r[:len(r)-1]))
		}
	case tea.KeySpace:
		m.editSetText(f.key, cur+" ")
	case tea.KeyRunes:
		if !msg.Alt {
			m.editSetText(f.key, cur+string(msg.Runes))
		}
	}
}

// editCycle changes a select / bool / slider field by dir (-1 or +1). It
// returns a command only for the whole-config pane selector, which commits the
// pane being left (and can refuse to switch if that pane no longer builds).
func (m *Model) editCycle(f edField, dir int) tea.Cmd {
	if f.key == "cfg.pane" {
		if len(m.leaves) < 2 {
			return nil
		}
		next := (m.ctxLeaf + dir + len(m.leaves)) % len(m.leaves)
		cmd := m.applyEdit()
		if m.edErr != "" {
			return nil // stay put; the current pane won't build
		}
		m.loadPane(next)
		return cmd
	}
	switch f.kind {
	case fSelect:
		if len(f.choices) == 0 {
			return nil
		}
		i := (indexOf(f.choices, m.editGet(f)) + dir + len(f.choices)) % len(f.choices)
		v := f.choices[i]
		switch f.key {
		case "type":
			m.eb.typ = v
		case "markdownStyle":
			m.eb.mdStyle = v
		case "size.mode":
			m.eb.sizeMode = v
			switch v {
			case "percent":
				if m.eb.sizeVal <= 0 || m.eb.sizeVal > 100 {
					m.eb.sizeVal = 50
				}
			case "fixed":
				if m.eb.sizeVal <= 0 {
					m.eb.sizeVal = 20
				}
			case "flex":
				if m.eb.sizeVal < 1 || m.eb.sizeVal > 9 {
					m.eb.sizeVal = 1
				}
			}
		}
	case fBool:
		m.eb.border = !m.eb.border
	case fSlider:
		m.eb.sizeVal = clampInt(m.eb.sizeVal+dir*f.step, f.min, f.max)
	}
	m.edDirty = true
	m.edConfigDirty = true
	return nil
}

// applyEdit commits the buffer to the live config and, if a widget-affecting
// field changed, rebuilds the widget (closing the old one). Returns the new
// widget's Init command, if any.
func (m *Model) applyEdit() tea.Cmd {
	if m.ctxLeaf < 0 || m.ctxLeaf >= len(m.leaves) {
		return nil
	}
	lf := m.leaves[m.ctxLeaf]
	eb := &m.eb

	cand := *lf.pane // copy; commit only if it builds
	cand.Type = eb.typ
	cand.Title = eb.title
	bb := eb.border
	cand.Border = &bb
	cand.Content = eb.content
	cand.File = eb.file
	cand.MarkdownStyle = eb.mdStyle
	cand.Command = eb.command
	cand.Shell = eb.shell

	rebuild := lf.pane.Type != cand.Type ||
		lf.pane.Content != cand.Content ||
		lf.pane.File != cand.File ||
		lf.pane.MarkdownStyle != cand.MarkdownStyle ||
		lf.pane.Command != cand.Command ||
		lf.pane.Shell != cand.Shell

	var newW widget.Widget
	if rebuild {
		w, err := widget.New(&cand)
		if err != nil {
			m.edErr = err.Error()
			return nil
		}
		newW = w
	}

	oldBorder := lf.pane.Border == nil || *lf.pane.Border
	sizeChanged := lf.parent != nil && lf.parent.Size != composeSize(eb.sizeMode, eb.sizeVal)
	chromeChanged := lf.pane.Title != cand.Title || oldBorder != eb.border
	if rebuild || sizeChanged || chromeChanged {
		m.edConfigDirty = true
	}

	*lf.pane = cand
	lf.title = cand.Title
	lf.border = eb.border
	if lf.parent != nil {
		lf.parent.Size = composeSize(eb.sizeMode, eb.sizeVal)
	}

	var cmd tea.Cmd
	if newW != nil {
		if c, ok := lf.w.(widget.Closer); ok {
			_ = c.Close()
		}
		lf.w = newW
		cmd = newW.Init()
	}
	m.edErr = ""
	m.edDirty = false
	return cmd
}

func (m *Model) renderEditor(frame string) string {
	return overlayCenter(frame, m.editorPanel(), m.w, m.h)
}

func (m *Model) editorPanel() string {
	th := m.theme()
	var b strings.Builder

	title := "edit · " + m.leafName(m.ctxLeaf)
	if m.edScope == edScopeConfig {
		title = "edit config · " + m.cfg.Name
	}
	b.WriteString(th.helpTitle.Render(title))
	b.WriteString("\n")

	// tab navbar — always visible
	var tabs []string
	for i, t := range m.editorTabs() {
		if i == m.edTab {
			tabs = append(tabs, th.menuSel.Render(" "+t+" "))
		} else {
			tabs = append(tabs, th.menuItem.Render(" "+t+" "))
		}
	}
	b.WriteString(strings.Join(tabs, ""))
	b.WriteString("\n\n")

	fields := m.editorFields()
	if len(fields) == 0 {
		b.WriteString(th.helpDim.Render("  root pane — no enclosing split to size"))
		b.WriteString("\n")
	}
	for i, f := range fields {
		mark := "  "
		if i == m.edRow {
			mark = th.helpKey.Render("› ")
		}
		b.WriteString(mark + th.helpDesc.Render(padRight(f.label, 11)) + m.editControl(f, i == m.edRow))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.edErr != "" {
		b.WriteString(th.helpKey.Render("! "+m.edErr) + "\n")
	} else if m.edDirty {
		b.WriteString(th.helpDim.Render("unsaved changes") + "\n")
	}
	b.WriteString(th.helpDim.Render("↑↓ field · ←→ change · type to edit text") + "\n")
	b.WriteString(th.helpDim.Render(m.mod + "+s / Enter apply · Esc close"))
	return th.helpBox.Render(b.String())
}

func (m *Model) editControl(f edField, active bool) string {
	th := m.theme()
	switch f.kind {
	case fSelect:
		v := m.editGet(f)
		if v == "" {
			v = "(default)"
		}
		if active {
			return th.helpKey.Render("‹ ") + v + th.helpKey.Render(" ›")
		}
		return th.menuItem.Render("  " + v)
	case fBool:
		box := "[ ]"
		if m.eb.border {
			box = "[x]"
		}
		return th.menuItem.Render(box + " " + m.editGet(f))
	case fSlider:
		return sliderBar(m.eb.sizeVal, f.min, f.max) + " " + strconv.Itoa(m.eb.sizeVal)
	case fText:
		v := m.editGet(f)
		if active {
			return v + "▏"
		}
		if v == "" {
			return th.helpDim.Render("—")
		}
		return oneLine(v)
	}
	return ""
}

func sliderBar(v, lo, hi int) string {
	const w = 12
	if hi <= lo {
		return "[" + strings.Repeat("─", w) + "]"
	}
	fill := int(math.Round(float64(v-lo) / float64(hi-lo) * float64(w)))
	fill = clampInt(fill, 0, w)
	return "[" + strings.Repeat("█", fill) + strings.Repeat("─", w-fill) + "]"
}

// parseSizeStr splits a Child.Size string into (mode, value).
func parseSizeStr(s string) (string, int) {
	s = strings.TrimSpace(s)
	switch {
	case s == "" || s == "*":
		return "auto", 1
	case strings.HasSuffix(s, "%"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil || v <= 0 {
			return "percent", 50
		}
		return "percent", int(math.Round(v))
	case strings.HasSuffix(s, "*"):
		v, err := strconv.Atoi(strings.TrimSuffix(s, "*"))
		if err != nil || v < 1 {
			v = 1
		}
		return "flex", v
	default:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 {
			return "auto", 1
		}
		return "fixed", int(math.Round(v))
	}
}

func composeSize(mode string, val int) string {
	switch mode {
	case "percent":
		return strconv.Itoa(val) + "%"
	case "fixed":
		return strconv.Itoa(val)
	case "flex":
		if val <= 1 {
			return "*"
		}
		return strconv.Itoa(val) + "*"
	default:
		return "*"
	}
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", "⏎")
	if len([]rune(s)) > 44 {
		s = string([]rune(s)[:43]) + "…"
	}
	return s
}
