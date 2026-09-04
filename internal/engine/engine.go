// Package engine turns a parsed config into a running bubbletea program: it
// builds the layout tree, owns focus, the focus-lock state machine and global
// keybindings, and renders each pane's widget into its computed rectangle.
package engine

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/widget"
)

// Actions a keybinding can name.
const (
	actQuit      = "quit"
	actFocusNext = "focus-next"
	actFocusPrev = "focus-prev"
	actHelp      = "help"
	actLock      = "lock"
	actUnlock    = "unlock"
)

// baseBinds are merged under the global options and the TUI config (config
// wins). "mod+" is expanded to the configured modifier before use.
var baseBinds = map[string]string{
	"ctrl+c":    actQuit,
	"mod+q":     actQuit,
	"tab":       actFocusNext,
	"shift+tab": actFocusPrev,
	"f1":        actHelp,
}

type direction int

const (
	dirUp direction = iota
	dirDown
	dirLeft
	dirRight
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

	// absolute rectangle from the last render, for mouse hit-testing and
	// directional focus movement.
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

	opts    Options
	mod     string            // resolved modifier token ("alt")
	tuiKeys map[string]string // the config's own bindings, for the help view

	locked     bool // focus is locked into the focused pane (keys go to it)
	pendingEsc bool // one Esc seen inside a locked raw pane; a second = literal
	helpOpen   bool

	rectW, rectH int // body size the leaf rectangles were computed at
}

// New builds a Model from config with the default runner options.
func New(cfg *config.Config) (*Model, error) {
	return NewWithOptions(cfg, DefaultOptions())
}

// NewWithOptions builds a Model, applying runner-wide Options. Binding
// precedence, lowest to highest: baseBinds < opts.Keys < cfg.Keybindings.
func NewWithOptions(cfg *config.Config, opts Options) (*Model, error) {
	if opts.Mod == "" {
		opts.Mod = "alt"
	}
	m := &Model{keys: map[string]string{}, opts: opts, mod: opts.Mod}

	put := func(src map[string]string) {
		for k, v := range src {
			m.keys[expandMod(k, m.mod)] = v
		}
	}
	put(baseBinds)
	put(opts.Keys)
	put(cfg.Keybindings)
	m.tuiKeys = cfg.Keybindings

	root, err := m.build(&cfg.Layout, "layout")
	if err != nil {
		return nil, err
	}
	m.root = root
	if len(m.leaves) == 0 {
		return nil, fmt.Errorf("engine: layout has no panes")
	}
	return m, nil
}

// expandMod rewrites a leading "mod+" token to the configured modifier.
func expandMod(key, mod string) string {
	if strings.HasPrefix(key, "mod+") {
		return mod + "+" + strings.TrimPrefix(key, "mod+")
	}
	return key
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

// Update routes messages. Keys flow through handleKey (global binds, then the
// focus-lock state machine). Mouse clicks focus / release the lock; wheel
// scrolls the pane under the cursor. Everything else is broadcast to all
// widgets.
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

// handleKey implements the whole keyboard contract:
//
//   - The help overlay, when open, swallows every key (Esc/F1/q/Ctrl+C close it).
//   - Global bindings (the "mod+…" set, Ctrl+C, F1) always win — even inside a
//     locked interactive pane. That is what makes "mod" a reliable escape hatch.
//   - Unlocked: arrows / Tab move focus; Enter locks into the focused pane.
//     Unbound keys still fall through to the focused widget (text-pane scroll).
//   - Locked: every key goes to the widget. Esc releases the lock; for a raw
//     pane the first Esc is buffered and a second Esc sends a literal Esc to the
//     child (Esc-Esc), so vim stays usable.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.helpOpen {
		switch key {
		case "ctrl+c":
			m.Close()
			return m, tea.Quit
		case "f1", "esc", "q", "enter", " ":
			m.helpOpen = false
		}
		return m, nil
	}

	// Global bindings win unconditionally.
	if m.isGlobalKey(key) {
		if action, ok := m.keys[key]; ok {
			return m.runAction(action)
		}
	}

	focused := m.leaves[m.focus].w
	rk, isRaw := focused.(widget.RawKeyConsumer)
	rawWanted := isRaw && rk.ConsumesRawKeys()

	if m.locked {
		if key == "esc" {
			if m.pendingEsc {
				m.pendingEsc = false
				return m.forwardKey(msg) // Esc-Esc → literal Esc to child
			}
			if rawWanted {
				m.pendingEsc = true // buffer; wait to see if a second Esc follows
				return m, nil
			}
			m.locked = false
			return m, nil
		}
		if m.pendingEsc {
			// A non-Esc key after a buffered Esc: treat the lone Esc as "release"
			// and re-handle this key unlocked.
			m.pendingEsc = false
			m.locked = false
			return m.handleKey(msg)
		}
		return m.forwardKey(msg)
	}

	// Unlocked / navigation mode.
	switch key {
	case "up":
		m.focusDir(dirUp)
		return m, nil
	case "down":
		m.focusDir(dirDown)
		return m, nil
	case "left":
		m.focusDir(dirLeft)
		return m, nil
	case "right":
		m.focusDir(dirRight)
		return m, nil
	case "enter":
		m.locked = true
		return m, nil
	}
	if action, ok := m.keys[key]; ok {
		return m.runAction(action)
	}
	return m.forwardKey(msg)
}

// isGlobalKey reports whether a key is one the engine intercepts even inside a
// locked pane: anything carrying the modifier, plus Ctrl+C and F1.
func (m *Model) isGlobalKey(key string) bool {
	return key == "ctrl+c" || key == "f1" || strings.HasPrefix(key, m.mod+"+")
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
		m.locked, m.pendingEsc = false, false
	case actFocusPrev:
		m.focus = (m.focus - 1 + len(m.leaves)) % len(m.leaves)
		m.locked, m.pendingEsc = false, false
	case actHelp:
		m.helpOpen = true
	case actLock:
		m.locked = true
	case actUnlock:
		m.locked, m.pendingEsc = false, false
	}
	return m, nil
}

// focusDir moves focus to the nearest pane in a direction. It uses the leaf
// rectangles from the last render; when those are stale/unstamped it falls back
// to ring order (up/left = previous, down/right = next).
func (m *Model) focusDir(d direction) {
	bw, bh := m.bodyDims()
	if len(m.leaves) < 2 {
		return
	}
	if m.rectW != bw || m.rectH != bh {
		m.ringStep(d)
		return
	}
	cur := m.leaves[m.focus]
	ccx, ccy := cur.x+cur.pw/2, cur.y+cur.ph/2
	best, bestScore := -1, 1<<62
	for i, lf := range m.leaves {
		if i == m.focus {
			continue
		}
		dx := lf.x + lf.pw/2 - ccx
		dy := lf.y + lf.ph/2 - ccy
		switch d {
		case dirUp:
			if dy >= 0 {
				continue
			}
		case dirDown:
			if dy <= 0 {
				continue
			}
		case dirLeft:
			if dx >= 0 {
				continue
			}
		case dirRight:
			if dx <= 0 {
				continue
			}
		}
		// Distance, biased hard against off-axis drift.
		score := dx*dx + dy*dy
		if d == dirUp || d == dirDown {
			score += 3 * dx * dx
		} else {
			score += 3 * dy * dy
		}
		if score < bestScore {
			best, bestScore = i, score
		}
	}
	if best >= 0 {
		m.focus = best
		return
	}
	m.ringStep(d)
}

func (m *Model) ringStep(d direction) {
	if d == dirDown || d == dirRight {
		m.focus = (m.focus + 1) % len(m.leaves)
	} else {
		m.focus = (m.focus - 1 + len(m.leaves)) % len(m.leaves)
	}
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	bw, bh := m.bodyDims()
	if !m.ready || m.rectW != bw || m.rectH != bh {
		return m, nil
	}
	i := m.leafAt(msg.X, msg.Y)
	if i < 0 {
		return m, nil
	}
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		if i != m.focus {
			m.locked, m.pendingEsc = false, false
		}
		m.focus = i
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown,
		tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
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

// bodyDims returns the area available to the layout tree — the full terminal
// minus the status bar row when it is shown.
func (m *Model) bodyDims() (int, int) {
	if m.statusBarVisible() {
		return m.w, m.h - 1
	}
	return m.w, m.h
}

func (m *Model) statusBarVisible() bool { return m.opts.StatusBar && m.h >= 6 }

// View renders the whole tree and records each pane's rectangle for mouse
// hit-testing.
func (m *Model) View() string {
	if !m.ready {
		return "loading…"
	}
	bw, bh := m.bodyDims()
	body := m.render(m.root, 0, 0, bw, bh)
	m.rectW, m.rectH = bw, bh

	out := body
	if m.statusBarVisible() {
		out = lipgloss.JoinVertical(lipgloss.Left, body, m.statusBar(m.w))
	}
	if m.helpOpen {
		out = m.renderHelp(out)
	}
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

func (m *Model) renderLeaf(lf *leaf, w, h int) string {
	th := m.theme()
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
	if focused && m.locked {
		title = strings.TrimSpace(title + "  ● LOCKED")
	}
	if title != "" {
		// One line, always: collapse newlines and let MaxWidth truncate. A
		// Width() here would *wrap* an over-long title into 2+ lines and blow
		// the pane's height.
		flat := strings.ReplaceAll(title, "\n", " ")
		ts := th.title
		if focused {
			ts = th.titleFocus
		}
		head = ts.MaxWidth(innerW).Inline(true).Render(flat)
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
		box = box.Border(lipgloss.RoundedBorder())
		switch {
		case focused && m.locked:
			box = box.BorderForeground(th.borderLock)
		case focused:
			box = box.BorderForeground(th.borderFocus)
		default:
			box = box.BorderForeground(th.borderIdle)
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

// Locked reports whether focus is currently locked into the focused pane.
func (m *Model) Locked() bool { return m.locked }

// Mode is "NAV" or "LOCKED" — the string the status bar shows.
func (m *Model) Mode() string {
	if m.locked {
		return "LOCKED"
	}
	return "NAV"
}

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
