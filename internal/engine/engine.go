// Package engine turns a parsed config into a running bubbletea program: it
// builds the layout tree, owns focus and global keybindings, and renders each
// pane's widget into its computed rectangle.
package engine

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/widget"
)

// defaultKeys are merged under the config's keybindings (config wins).
var defaultKeys = map[string]string{
	"ctrl+c":    "quit",
	"q":         "quit",
	"tab":       "focus-next",
	"shift+tab": "focus-prev",
}

// Actions a keybinding can name.
const (
	actQuit      = "quit"
	actFocusNext = "focus-next"
	actFocusPrev = "focus-prev"
)

type node struct {
	direction string // "" == leaf
	children  []child
	leaf      *leaf
}

type child struct {
	spec sizeSpec
	node node
}

type leaf struct {
	title  string
	border bool
	w      widget.Widget

	// absolute rectangle from the last render, for mouse hit-testing.
	x, y, pw, ph int
}

// Model is the root bubbletea model.
type Model struct {
	root   node
	leaves []*leaf
	focus  int
	keys   map[string]string
	w, h   int
	ready  bool

	prefix        string // leader key for interactive panes
	pendingPrefix bool   // prefix seen, next key is an action
	rectW, rectH  int    // size the leaf rectangles were computed at
}

// New builds a Model from config.
func New(cfg *config.Config) (*Model, error) {
	m := &Model{keys: map[string]string{}}
	for k, v := range defaultKeys {
		m.keys[k] = v
	}
	for k, v := range cfg.Keybindings {
		m.keys[k] = v
	}

	root, err := m.build(&cfg.Layout, "layout")
	if err != nil {
		return nil, err
	}
	m.root = root
	if len(m.leaves) == 0 {
		return nil, fmt.Errorf("engine: layout has no panes")
	}

	m.prefix = cfg.Prefix
	if m.prefix == "" {
		m.prefix = "ctrl+b"
	}
	return m, nil
}

func (m *Model) build(n *config.Node, path string) (node, error) {
	if n.Pane != nil {
		w, err := widget.New(n.Pane)
		if err != nil {
			return node{}, fmt.Errorf("%s: %w", path, err)
		}
		lf := &leaf{
			title:  n.Pane.Title,
			border: n.Pane.Border == nil || *n.Pane.Border,
			w:      w,
		}
		m.leaves = append(m.leaves, lf)
		return node{leaf: lf}, nil
	}

	dir := n.Direction
	if dir == "" {
		dir = "column"
	}
	out := node{direction: dir}
	for i := range n.Children {
		ch := &n.Children[i]
		sub, err := m.build(&ch.Node, fmt.Sprintf("%s.children[%d]", path, i))
		if err != nil {
			return node{}, err
		}
		out.children = append(out.children, child{spec: parseSize(ch.Size), node: sub})
	}
	return out, nil
}

// Init starts every widget's command.
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, lf := range m.leaves {
		if c := lf.w.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}

// Update routes messages. Keys: a focused interactive pane (RawKeyConsumer)
// gets everything except the prefix; otherwise global keybindings win and the
// rest fall through to the focused widget. Mouse: click focuses the pane under
// the cursor, wheel scrolls it. Everything else is broadcast to all widgets.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h, m.ready = msg.Width, msg.Height, true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	default:
		var cmds []tea.Cmd
		for _, lf := range m.leaves {
			nw, cmd := lf.w.Update(msg)
			lf.w = nw
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	focused := m.leaves[m.focus].w
	rk, ok := focused.(widget.RawKeyConsumer)
	rawActive := ok && rk.ConsumesRawKeys()

	// A prefix was pressed last: this key selects an action (or, if it is the
	// prefix again, is sent to the child verbatim).
	if m.pendingPrefix {
		m.pendingPrefix = false
		if rawActive && key == m.prefix {
			return m.forwardKey(msg)
		}
		if action, ok := m.keys[key]; ok {
			return m.runAction(action)
		}
		return m, nil
	}

	if rawActive {
		if key == m.prefix {
			m.pendingPrefix = true
			return m, nil
		}
		return m.forwardKey(msg)
	}

	if action, ok := m.keys[key]; ok {
		return m.runAction(action)
	}
	return m.forwardKey(msg)
}

func (m *Model) forwardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	nw, cmd := m.leaves[m.focus].w.Update(msg)
	m.leaves[m.focus].w = nw
	return m, cmd
}

func (m *Model) runAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case actQuit:
		m.Close()
		return m, tea.Quit
	case actFocusNext:
		m.focus = (m.focus + 1) % len(m.leaves)
	case actFocusPrev:
		m.focus = (m.focus - 1 + len(m.leaves)) % len(m.leaves)
	}
	return m, nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Rectangles are only meaningful for the size they were rendered at.
	if !m.ready || m.rectW != m.w || m.rectH != m.h {
		return m, nil
	}
	i := m.leafAt(msg.X, msg.Y)
	if i < 0 {
		return m, nil
	}
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		m.focus = i
		m.pendingPrefix = false
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown,
		tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		// Scroll the pane under the cursor without stealing focus.
		nw, cmd := m.leaves[i].w.Update(msg)
		m.leaves[i].w = nw
		return m, cmd
	}
	return m, nil
}

func (m *Model) leafAt(x, y int) int {
	for i, lf := range m.leaves {
		if x >= lf.x && x < lf.x+lf.pw && y >= lf.y && y < lf.y+lf.ph {
			return i
		}
	}
	return -1
}

// Close releases every widget that holds OS resources (child processes, ptys).
func (m *Model) Close() {
	for _, lf := range m.leaves {
		if c, ok := lf.w.(widget.Closer); ok {
			_ = c.Close()
		}
	}
}

// View renders the whole tree and records each pane's rectangle for mouse
// hit-testing.
func (m *Model) View() string {
	if !m.ready {
		return "loading…"
	}
	out := m.render(m.root, 0, 0, m.w, m.h)
	m.rectW, m.rectH = m.w, m.h
	return out
}

func (m *Model) render(n node, x, y, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	if n.leaf != nil {
		n.leaf.x, n.leaf.y, n.leaf.pw, n.leaf.ph = x, y, w, h
		return m.renderLeaf(n.leaf, w, h)
	}

	specs := make([]sizeSpec, len(n.children))
	for i, c := range n.children {
		specs[i] = c.spec
	}
	row := n.direction == "row"
	axis := h
	if row {
		axis = w
	}
	sizes := distribute(specs, axis)

	parts := make([]string, len(n.children))
	off := 0
	for i, c := range n.children {
		if row {
			parts[i] = m.render(c.node, x+off, y, sizes[i], h)
		} else {
			parts[i] = m.render(c.node, x, y+off, w, sizes[i])
		}
		off += sizes[i]
	}
	if row {
		return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	focusBorderCol  = lipgloss.Color("62")
	idleBorderColor = lipgloss.Color("240")
)

func (m *Model) renderLeaf(lf *leaf, w, h int) string {
	innerW, innerH := w, h
	if lf.border {
		innerW, innerH = w-2, h-2
	}
	if innerW < 1 || innerH < 1 {
		return lipgloss.NewStyle().Width(max(w, 0)).Height(max(h, 0)).Render("")
	}

	focused := m.leaves[m.focus] == lf

	bodyH := innerH
	var head string
	title := lf.title
	if focused && m.pendingPrefix {
		title = strings.TrimSpace(title + "  ⌨ PREFIX — press a nav key")
	}
	if title != "" {
		// One line, always: collapse newlines and let MaxWidth truncate. A
		// Width() here would *wrap* an over-long title into 2+ lines and blow
		// the pane's height.
		flat := strings.ReplaceAll(title, "\n", " ")
		head = titleStyle.MaxWidth(innerW).Inline(true).Render(flat)
		bodyH = innerH - 1
	}

	// scrollBox returns exactly bodyH lines, but JoinVertical(head, "") is two
	// lines — so when the title eats the last row, render the title alone.
	var content string
	switch {
	case head == "":
		content = lf.w.View(innerW, innerH)
	case bodyH >= 1:
		content = lipgloss.JoinVertical(lipgloss.Left, head, lf.w.View(innerW, bodyH))
	default:
		content = head
	}
	// Defensive clamp: content must be exactly innerH lines before we add a
	// border. MaxHeight on the bordered style can't do this — it would trim the
	// border itself.
	content = clampLines(content, innerH)

	// lipgloss Width/Height size the content area; a border adds 2 cells on
	// each axis outside that. So the bordered box uses innerW/innerH (== w-2 /
	// h-2) and lands at exactly w x h. MaxWidth/MaxHeight would clip the final
	// output *including* the border, shearing off the bottom/right edge.
	box := lipgloss.NewStyle().Width(innerW).Height(innerH)
	if lf.border {
		box = box.Border(lipgloss.NormalBorder())
		if focused {
			box = box.BorderForeground(focusBorderCol)
		} else {
			box = box.BorderForeground(idleBorderColor)
		}
	} else {
		box = box.MaxWidth(innerW).MaxHeight(innerH)
	}
	return box.Render(content)
}

// Focused returns the title (or type index) of the focused pane — handy for
// tests and `validate`.
func (m *Model) Focused() string {
	lf := m.leaves[m.focus]
	if lf.title != "" {
		return lf.title
	}
	return fmt.Sprintf("pane#%d", m.focus)
}

// PaneCount reports the number of leaves.
func (m *Model) PaneCount() int { return len(m.leaves) }

// clampLines trims s to at most n lines (never pads — lipgloss Height does that).
func clampLines(s string, n int) string {
	if n < 1 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}
