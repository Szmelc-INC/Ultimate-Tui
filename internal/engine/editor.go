package engine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	fAction                  // a button-like row; Enter runs it, no value to hold
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
	dir                    string // enclosing split's direction: column | row
	hasSize                bool   // false for the root pane (no enclosing split)
	hasDir                 bool   // false for the root pane

	// keybinds is the scratch copy of cfg.Keybindings (Keybinds tab,
	// edScopeConfig only), sorted by key for a stable display order. Loaded
	// once when the config editor opens and preserved across loadPane calls
	// (switching the Config tab's pane selector must not discard it) — see
	// openConfigEditor / loadPane.
	keybinds []kbEntry
}

// kbEntry is one row on the Keybinds tab: a key chord and the action name it
// runs. An empty key is dropped on apply (see applyKeybinds); the action is
// not validated against known action names — an unrecognized one just never
// matches, same as a hand-edited config file today.
type kbEntry struct{ key, action string }

var mdStyleChoices = []string{"", "dark", "light", "dracula", "tokyo-night", "pink", "ascii", "notty"}
var sizeModeChoices = []string{"auto", "percent", "fixed", "flex"}

func (m *Model) editorTabs() []string {
	if m.edScope == edScopeConfig {
		return []string{"Config", "Keybinds", "General", "Layout", "Widget"}
	}
	return []string{"General", "Layout", "Widget"}
}

// edLogicalTab maps the visible tab index to a scope-independent one: -1 =
// the config tab, -2 = the keybinds tab (both only present in whole-config
// scope), 0/1/2 = General/Layout/Widget.
func (m *Model) edLogicalTab() int {
	if m.edScope != edScopeConfig {
		return m.edTab
	}
	switch m.edTab {
	case 0:
		return -1
	case 1:
		return -2
	default:
		return m.edTab - 2
	}
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
			{key: "cfg.version", label: "version", kind: fText},
			{key: "cfg.pane", label: "pane", kind: fSelect, choices: m.leafNames()},
		}
	}
	if m.edLogicalTab() == -2 { // Keybinds tab
		var fs []edField
		for i := range m.eb.keybinds {
			fs = append(fs,
				edField{key: kbFieldKey(i, "key"), label: fmt.Sprintf("#%d key", i+1), kind: fText},
				edField{key: kbFieldKey(i, "action"), label: "→ runs", kind: fText},
			)
		}
		fs = append(fs, edField{key: "kb.new", label: "", kind: fAction})
		return fs
	}
	switch m.edLogicalTab() {
	case 0: // General
		return []edField{
			{key: "type", label: "type", kind: fSelect, choices: widget.Kinds()},
			{key: "title", label: "title", kind: fText},
			{key: "border", label: "border", kind: fBool},
		}
	case 1: // Layout
		if !m.eb.hasSize && !m.eb.hasDir {
			return nil
		}
		var fs []edField
		if m.eb.hasDir {
			fs = append(fs, edField{key: "direction", label: "direction", kind: fSelect, choices: []string{"column", "row"}})
		}
		if !m.eb.hasSize {
			return fs
		}
		fs = append(fs, edField{key: "size.mode", label: "size mode", kind: fSelect, choices: sizeModeChoices})
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
	dir := "column"
	if lf.split != nil && lf.split.Direction == "row" {
		dir = "row"
	}
	keybinds := m.eb.keybinds // Keybinds tab state is independent of which pane is loaded
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
		dir:      dir,
		hasSize:  lf.parent != nil,
		hasDir:   lf.split != nil,
		keybinds: keybinds,
	}
}

// loadKeybinds refills the Keybinds tab's scratch list from cfg.Keybindings,
// sorted by key for a stable display order.
func (m *Model) loadKeybinds() {
	m.eb.keybinds = m.eb.keybinds[:0]
	keys := make([]string, 0, len(m.cfg.Keybindings))
	for k := range m.cfg.Keybindings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		m.eb.keybinds = append(m.eb.keybinds, kbEntry{key: k, action: m.cfg.Keybindings[k]})
	}
}

// addKeybindRow appends a blank entry and moves the cursor to its key field,
// ready to type — the "+ add keybind" row's Enter action.
func (m *Model) addKeybindRow() {
	m.eb.keybinds = append(m.eb.keybinds, kbEntry{})
	m.edRow = (len(m.eb.keybinds) - 1) * 2
	m.edDirty, m.edConfigDirty = true, true
}

// removeKeybindRow deletes entry i (ctrl+d on either of its two rows) and
// keeps the cursor in range.
func (m *Model) removeKeybindRow(i int) {
	if i < 0 || i >= len(m.eb.keybinds) {
		return
	}
	m.eb.keybinds = append(m.eb.keybinds[:i], m.eb.keybinds[i+1:]...)
	m.edRow = clampInt(m.edRow, 0, 2*len(m.eb.keybinds))
	m.edDirty, m.edConfigDirty = true, true
}

// applyKeybinds commits the scratch keybind list to cfg.Keybindings (empty
// keys dropped) and, if it actually changed, rebuilds m.keys so an edited or
// newly added binding works immediately — no restart needed.
func (m *Model) applyKeybinds() {
	next := map[string]string{}
	for _, kb := range m.eb.keybinds {
		if k := strings.TrimSpace(kb.key); k != "" {
			next[k] = kb.action
		}
	}
	if !mapsEqual(m.cfg.Keybindings, next) {
		m.cfg.Keybindings = next
		m.edConfigDirty = true
		m.rebuildKeyMap()
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// kbFieldKey/parseKbFieldKey encode/decode a Keybinds-tab row's edField.key
// ("kb.<index>.key" or "kb.<index>.action") — the index lets editGet/
// editSetText address one entry in the dynamic-length m.eb.keybinds list
// with the same fText plumbing every other text field uses.
func kbFieldKey(i int, sub string) string { return fmt.Sprintf("kb.%d.%s", i, sub) }

func parseKbFieldKey(k string) (idx int, sub string, ok bool) {
	if !strings.HasPrefix(k, "kb.") {
		return 0, "", false
	}
	rest := strings.SplitN(strings.TrimPrefix(k, "kb."), ".", 2)
	if len(rest) != 2 {
		return 0, "", false
	}
	i, err := strconv.Atoi(rest[0])
	if err != nil {
		return 0, "", false
	}
	return i, rest[1], true
}

// openPaneEditor edits a single pane, reached from the context menu's Edit row.
func (m *Model) openPaneEditor(idx int) {
	if idx < 0 || idx >= len(m.leaves) {
		return
	}
	m.overlay = ovEditor
	m.edScope = edScopePane
	m.edTab, m.edRow = 0, 0
	m.edDirty, m.edConfigDirty, m.edNote = false, false, ""
	m.loadPane(idx)
}

// openConfigEditor edits the whole config, reached from the F2 control panel.
// Same controls; a leading "Config" tab carries the top-level name and a pane
// selector that walks every leaf, and a "Keybinds" tab lists/edits/adds/
// removes cfg.Keybindings entries.
func (m *Model) openConfigEditor() {
	m.overlay = ovEditor
	m.edScope = edScopeConfig
	m.edTab, m.edRow = 0, 0
	m.edDirty, m.edConfigDirty, m.edNote = false, false, ""
	m.loadKeybinds()
	m.loadPane(0)
}

// closeEditor drops the overlay (back to whatever opened it) and clears all
// editor state. Called on a clean Esc and on "discard" from the confirm dialog.
func (m *Model) closeEditor() {
	m.overlay = m.overlayReturn
	m.overlayReturn = ovNone
	m.edDirty, m.edConfigDirty = false, false
	m.edErr, m.edNote = "", ""
}

// openConfirmDiscard is the "you have unsaved edits" gate on leaving the editor.
func (m *Model) openConfirmDiscard() (tea.Model, tea.Cmd) {
	m.overlay = ovConfirm
	m.confirmKind = cfDiscard
	m.confirmPrompt = "Leave the editor? Unsaved edits are lost."
	return m, nil
}

// editorSave applies the buffer and, if that succeeded, writes the whole config
// to disk via Options.SaveConfig.
func (m *Model) editorSave() tea.Cmd {
	cmd := m.applyEdit()
	if m.edErr != "" {
		return cmd
	}
	if m.opts.SaveConfig == nil {
		m.edErr = "no save target wired"
		return cmd
	}
	if err := m.opts.SaveConfig(m.cfg); err != nil {
		m.edErr = "save failed: " + err.Error()
		return cmd
	}
	m.edConfigDirty, m.edNote = false, "saved to disk"
	return cmd
}

func (m *Model) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.edNote = "" // any keystroke clears the transient "saved" line
	switch key {
	case "esc":
		if m.edDirty || m.edConfigDirty {
			return m.openConfirmDiscard()
		}
		m.closeEditor()
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
		return m, m.editorSave()
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
		if f.kind == fAction && f.key == "kb.new" {
			m.addKeybindRow()
			return m, nil
		}
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
	case "ctrl+d":
		if i, _, ok := parseKbFieldKey(f.key); ok {
			m.removeKeybindRow(i)
			return m, nil
		}
	}
	if f.kind == fText {
		m.editText(f, msg)
	}
	return m, nil
}

// editGet returns a field's current buffered value as a string.
func (m *Model) editGet(f edField) string {
	if i, sub, ok := parseKbFieldKey(f.key); ok && i < len(m.eb.keybinds) {
		if sub == "action" {
			return m.eb.keybinds[i].action
		}
		return m.eb.keybinds[i].key
	}
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
	case "direction":
		return m.eb.dir
	case "cfg.name":
		return m.cfg.Name
	case "cfg.version":
		return m.cfg.Version
	case "cfg.pane":
		return m.leafName(m.ctxLeaf)
	}
	return ""
}

func (m *Model) editSetText(key, s string) {
	if i, sub, ok := parseKbFieldKey(key); ok && i < len(m.eb.keybinds) {
		if sub == "action" {
			m.eb.keybinds[i].action = s
		} else {
			m.eb.keybinds[i].key = s
		}
		m.edDirty, m.edConfigDirty = true, true
		return
	}
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
	case "cfg.version":
		m.cfg.Version = s
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
	if f.kind == fAction {
		return nil // a button row has nothing to cycle; Enter runs it instead
	}
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
		case "direction":
			m.eb.dir = v
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
	if m.edScope == edScopeConfig {
		m.applyKeybinds()
	}
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
	oldDir := "column"
	if lf.split != nil && lf.split.Direction == "row" {
		oldDir = "row"
	}
	sizeChanged := lf.parent != nil && lf.parent.Size != composeSize(eb.sizeMode, eb.sizeVal)
	dirChanged := lf.split != nil && eb.hasDir && oldDir != eb.dir
	chromeChanged := lf.pane.Title != cand.Title || oldBorder != eb.border
	if rebuild || sizeChanged || dirChanged || chromeChanged {
		m.edConfigDirty = true
	}

	*lf.pane = cand
	lf.title = cand.Title
	lf.border = eb.border
	if lf.parent != nil {
		lf.parent.Size = composeSize(eb.sizeMode, eb.sizeVal)
	}
	if lf.split != nil && eb.hasDir {
		if eb.dir == "row" {
			lf.split.Direction = "row"
		} else {
			lf.split.Direction = "" // "" == column; keeps the saved file terse
		}
	}

	var cmd tea.Cmd
	if newW != nil {
		if c, ok := lf.w.(widget.Closer); ok {
			_ = c.Close()
		}
		lf.w = newW
		if lf.contentZoom != 0 {
			if z, ok := newW.(widget.Zoomer); ok {
				z.SetZoom(lf.contentZoom)
			}
		}
		cmd = newW.Init()
	}
	m.edErr = ""
	m.edDirty = false
	return cmd
}

// editorPanel's row layout, shared by editorPanel and editorMouseClick so the
// two can never drift apart: title, tabs, a separator rule, a blank line,
// then one row per field with a blank spacer row after it (the extra
// breathing room item 1 asked for — a dense one-row-per-field list read as
// cramped). edMinWidth pads every line out to a floor so a panel with just a
// couple of short fields (e.g. the Widget tab on a "text" pane) doesn't look
// squeezed next to General/Layout.
const (
	edTabRow      = 1
	edFieldsStart = 4
	edFieldStride = 2
	edMinWidth    = 48
	edLabelWidth  = 14
)

// edTabCellWidth is a tab's rendered width (" " + label + " ", doubled) —
// used by both editorPanel (to build the navbar) and editorMouseClick (to
// hit-test it), so they can't disagree on where one tab ends and the next
// begins.
func edTabCellWidth(label string) int { return lipgloss.Width(label) + 4 }

// editorMouseClick maps a click's content-relative (x, y) to editorPanel's
// two interactive regions: the tab navbar (edTabRow) and the field list
// (edFieldsStart onward, one row per field, edFieldStride apart). A tab
// click switches tabs; a field click selects it, and immediately toggles an
// fBool field — a checkbox reacts to one click, not click-then-arrow.
func (m *Model) editorMouseClick(cx, cy int) (tea.Model, tea.Cmd) {
	if cy == edTabRow {
		x := 0
		for i, t := range m.editorTabs() {
			w := edTabCellWidth(t)
			if cx >= x && cx < x+w {
				m.edTab, m.edRow = i, 0
				return m, nil
			}
			x += w
		}
		return m, nil
	}

	fields := m.editorFields()
	rel := cy - edFieldsStart
	if rel < 0 || rel%edFieldStride != 0 {
		return m, nil
	}
	i := rel / edFieldStride
	if i >= len(fields) {
		return m, nil
	}
	m.edRow = i
	switch fields[i].kind {
	case fBool:
		return m, m.editCycle(fields[i], 1)
	case fAction:
		if fields[i].key == "kb.new" {
			m.addKeybindRow()
		}
	}
	return m, nil
}

func (m *Model) renderEditor(frame string) string {
	return m.splice(frame, m.editorPanel())
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

	// tab navbar, then a separator rule marking it off from the fields below.
	var tabs []string
	for i, t := range m.editorTabs() {
		cell := "  " + t + "  "
		if i == m.edTab {
			tabs = append(tabs, th.menuSel.Render(cell))
		} else {
			tabs = append(tabs, th.menuItem.Render(cell))
		}
	}
	tabsLine := strings.Join(tabs, "")
	b.WriteString(tabsLine)
	b.WriteString("\n")
	sepW := lipgloss.Width(tabsLine)
	if sepW < edMinWidth {
		sepW = edMinWidth
	}
	b.WriteString(th.helpDim.Render(strings.Repeat("─", sepW)))
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
		b.WriteString(mark + th.helpDesc.Render(padRight(f.label, edLabelWidth)) + "  " + m.editControl(f, i == m.edRow))
		b.WriteString("\n\n") // edFieldStride: one blank spacer row per field
	}

	switch {
	case m.edErr != "":
		b.WriteString(th.helpKey.Render("! "+m.edErr) + "\n")
	case m.edNote != "":
		b.WriteString(th.helpKey.Render(m.edNote) + "\n")
	case m.edConfigDirty:
		b.WriteString(th.helpDim.Render("● unsaved — "+m.mod+"+s to write to disk") + "\n")
	case m.edDirty:
		b.WriteString(th.helpDim.Render("● unapplied edits") + "\n")
	}
	hint := "↑↓ field · ←→ change · type to edit text"
	if m.edLogicalTab() == -2 {
		hint = "↑↓ field · type to edit · ctrl+d remove · Enter on + adds"
	}
	b.WriteString(th.helpDim.Render(hint) + "\n")
	b.WriteString(th.helpDim.Render("Enter apply · " + m.mod + "+s save · Esc exit"))
	return th.helpBox.Render(padBlockWidth(b.String(), edMinWidth))
}

// padBlockWidth right-pads every line of s to at least minW cells (never
// narrower than the block's own widest line) — the "bigger, roomier" panel
// floor: a tab with only a couple of short fields still fills a decent box
// instead of shrink-wrapping to its narrowest content.
func padBlockWidth(s string, minW int) string {
	lines := strings.Split(s, "\n")
	w := minW
	for _, ln := range lines {
		if x := lipgloss.Width(ln); x > w {
			w = x
		}
	}
	for i, ln := range lines {
		lines[i] = padRight(ln, w)
	}
	return strings.Join(lines, "\n")
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
			return oneLine(v) + "▏" // one row always — newlines/overlong text can't grow the box
		}
		if v == "" {
			return th.helpDim.Render("—")
		}
		return oneLine(v)
	case fAction:
		return th.helpKey.Render("+ add keybind")
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
