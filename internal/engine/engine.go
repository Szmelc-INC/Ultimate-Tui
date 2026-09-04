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
	actQuit        = "quit"
	actFocusNext   = "focus-next"
	actFocusPrev   = "focus-prev"
	actHelp        = "help"
	actSettings    = "settings"
	actControl     = "control"
	actContext     = "context-menu"
	actLock        = "lock"
	actUnlock      = "unlock"
	actZoomIn      = "zoom-in"
	actZoomOut     = "zoom-out"
	actZoomReset   = "zoom-reset"
	actResizeUp    = "resize-up"
	actResizeDown  = "resize-down"
	actResizeLeft  = "resize-left"
	actResizeRight = "resize-right"
	actSwapUp      = "swap-up"
	actSwapDown    = "swap-down"
	actSwapLeft    = "swap-left"
	actSwapRight   = "swap-right"
	actFlipSplit   = "flip-split"
)

// contentZoomMax bounds mod+= (see actZoomIn) — the low end is always 0,
// "smaller than one cell" has no meaning for text.
const contentZoomMax = 4

// baseBinds are merged under the global options and the TUI config (config
// wins). "mod+" is expanded to the configured modifier before use. Ctrl+C is
// deliberately absent — it is handled directly in handleKey (quit-confirm in
// NAV, per-pane interrupt when locked) and must never be a plain quit binding.
var baseBinds = map[string]string{
	"tab":       actFocusNext,
	"shift+tab": actFocusPrev,
	"f1":        actHelp,
	"f2":        actControl,
	// Per-pane content zoom: scales what a widget draws (a text pane's
	// glyphs), independent of the pane's own rectangle — see widget.Zoomer.
	// This is NOT pane resize (that's mod+arrows below); mixing the two up
	// was the point-10 bug this split fixes. Both "=" and "+" are bound so an
	// unshifted or shifted key both work.
	"mod+=": actZoomIn,
	"mod++": actZoomIn,
	"mod+-": actZoomOut,
	"mod+_": actZoomOut,
	"mod+0": actZoomReset,
	// Pane resize: grows the focused pane toward the arrow's edge, taking the
	// space from whichever sibling sits there (see resize.go). Directional,
	// unlike the old zoom-as-size behaviour this replaces.
	"mod+up":    actResizeUp,
	"mod+down":  actResizeDown,
	"mod+left":  actResizeLeft,
	"mod+right": actResizeRight,
	// Pane swap: exchanges the focused pane with its neighbor in that
	// direction (same split-level neighbor resize would grow into).
	"mod+shift+up":    actSwapUp,
	"mod+shift+down":  actSwapDown,
	"mod+shift+left":  actSwapLeft,
	"mod+shift+right": actSwapRight,
	// Flip the focused pane's enclosing split between row/column. Only fires
	// via the global-key path (isGlobalKey) — handleEditorKey checks
	// "mod+s"/"ctrl+s" itself before the editor overlay ever reaches here, so
	// there is no collision with the editor's own save binding.
	"mod+s": actFlipSplit,
	// Right-click equivalent: open the focused pane's context menu.
	"mod+w": actContext,
}

type direction int

const (
	dirUp direction = iota
	dirDown
	dirLeft
	dirRight
)

type node struct {
	direction string       // "" == leaf
	cfg       *config.Node // live split node — render re-reads Direction so the
	// editor can flip row/column at runtime
	children []child
	leaf     *leaf
}

type child struct {
	spec sizeSpec      // parsed at build time; the fallback
	cfg  *config.Child // live config slot — render re-parses Size from here so
	// the editor can change a split ratio at runtime
	node node
}

type leaf struct {
	title  string
	border bool
	w      widget.Widget

	// pane points at the live config node this leaf was built from, parent at
	// the enclosing split slot, and split at the enclosing split node (parent
	// and split are nil for the root leaf). The editor overlay reads current
	// values through these and mutates them in place; nothing else in the
	// engine writes to them.
	pane   *config.Pane
	parent *config.Child
	split  *config.Node

	// chain is the sequence of ancestor splits from the root down to (not
	// including) this leaf, each paired with the child index the path to
	// this leaf takes through it. Built fresh by every build() call (see
	// resize.go); used to find the nearest enclosing split whose axis
	// matches an arrow direction for resize / swap. Empty for the root leaf.
	chain []ancestorLink

	// contentZoom is a per-pane content-scale level (0 = unscaled), driven by
	// mod+=/-/0 and forwarded to the widget through the optional
	// widget.Zoomer interface. This is NOT the pane's rectangle size — that
	// is mod+arrows (resize.go). A widget that doesn't implement Zoomer
	// simply ignores it.
	contentZoom int

	// absolute rectangle from the last render, for mouse hit-testing and
	// directional focus movement.
	x, y, pw, ph int
}

// ancestorLink is one step in a leaf's chain: the split node and which of
// its children (by index) the path to the leaf goes through.
type ancestorLink struct {
	split *config.Node
	idx   int
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

	overlay       overlayKind
	overlayReturn overlayKind // where Esc from a nested overlay lands (one level)
	confirmKind   confirmKind
	confirmPrompt string
	confirmLeaf   int // leaf index a cfKill dialog targets, captured at open time

	ctrlRow int // F2 control-panel cursor row

	ctxLeaf int // leaf the context menu / editor targets, captured at open time
	ctxRow  int // context-menu cursor row

	// editor overlay state (editor.go)
	eb            editBuf
	edScope       editScope // pane (from the context menu) or whole config (from F2)
	edTab         int
	edRow         int
	edErr         string
	edNote        string // transient confirmation ("saved"), cleared on next key
	edDirty       bool   // scratch buffer differs from the live config
	edConfigDirty bool   // live config has unsaved edits (cleared on disk save)

	themeName string // resolved colour-palette name (see theme.go)
	styleName string // resolved styleset name (border + glyphs)
	themeMemo *theme // built lazily by theme(); invalidateTheme() drops it

	// F3 settings-menu state.
	setCol        int    // 0 = theme column, 1 = style column
	setRow        [2]int // cursor row per column
	setSavedTheme string // theme/style active when the menu opened (Esc restores)
	setSavedStyle string

	rectW, rectH int // body size the leaf rectangles were computed at

	// ovX/Y/W/H is the active overlay panel's bounding box, stamped by
	// splice() (chrome.go) every render — used by handleOverlayMouse to hit-
	// test clicks against it (outside = dismiss, inside = row/field/tab hit).
	ovX, ovY, ovW, ovH int

	// drag.go: mod+drag pane move/resize state.
	dragMode             dragMode
	dragLeaf             int // leaf grabbed when the drag started
	dragLastX, dragLastY int // last screen position dragResizeStep consumed

	// ovSplit state (split.go): the context menu's Split row and the "New
	// pane" row (right-click on dead space) both land here.
	spTarget int    // leaf index the split applies to; -1 = wrap the whole root ("New")
	spDir    string // "row" (side by side) or "column" (stacked)
	spPct    int    // percent kept by the original content; the new pane gets the rest
	spField  int    // 0 = direction control, 1 = percent control

	// F3 global settings/info panel state (global.go).
	glMod       string
	glStatusBar bool
	glRow       int
	glDirty     bool
	glNote      string

	cfg *config.Config // the config this Model was built from (editor target)
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
	m := &Model{
		keys:      map[string]string{},
		opts:      opts,
		mod:       opts.Mod,
		themeName: resolveThemeName(opts.Theme),
		styleName: resolveStyleName(opts.Style),
		cfg:       cfg,
	}

	m.rebuildKeyMap()

	root, err := m.build(&cfg.Layout, nil, nil, "layout", nil, nil, nil)
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

// rebuildKeyMap (re)resolves m.keys from baseBinds < opts.Keys < cfg.Keybindings
// (config wins), "mod+"-expanding every key. Called once by NewWithOptions and
// again whenever the Keybinds editor tab commits a change to cfg.Keybindings,
// so an edited or newly added binding works immediately — no restart needed.
func (m *Model) rebuildKeyMap() {
	m.keys = map[string]string{}
	put := func(src map[string]string) {
		for k, v := range src {
			m.keys[expandMod(k, m.mod)] = v
		}
	}
	put(baseBinds)
	put(m.opts.Keys)
	put(m.cfg.Keybindings)
	m.tuiKeys = m.cfg.Keybindings
}

// build walks a config subtree into an engine node/leaf tree, appending each
// leaf to m.leaves in tree order. carry (nil on the very first build, from
// NewWithOptions) is a *config.Pane -> old-leaf map used by rebuild() to
// carry a live widget over a structural edit instead of respawning it — a
// Split must not restart the pane it split, a resize must not touch a widget
// at all. When carry supplies no entry, a fresh widget is built and, if fresh
// is non-nil, appended to it so the caller can batch its Init command (the
// program-start path already Init()s every leaf itself; only a rebuild after
// the program is running needs this).
func (m *Model) build(n *config.Node, parent *config.Child, split *config.Node, path string, carry map[*config.Pane]*leaf, fresh *[]widget.Widget, chain []ancestorLink) (node, error) {
	if n.Pane != nil {
		old, carried := carry[n.Pane]
		var w widget.Widget
		if carried {
			w = old.w
		} else {
			var err error
			w, err = widget.New(n.Pane)
			if err != nil {
				return node{}, fmt.Errorf("%s: %w", path, err)
			}
			if fresh != nil {
				*fresh = append(*fresh, w)
			}
		}
		lf := &leaf{
			title:  n.Pane.Title,
			border: n.Pane.Border == nil || *n.Pane.Border,
			w:      w,
			pane:   n.Pane,
			parent: parent,
			split:  split,
			chain:  chain,
		}
		if carried {
			lf.contentZoom = old.contentZoom
		}
		m.leaves = append(m.leaves, lf)
		return node{leaf: lf}, nil
	}

	dir := n.Direction
	if dir == "" {
		dir = "column"
	}
	out := node{direction: dir, cfg: n}
	for i := range n.Children {
		ch := &n.Children[i]
		// Always copy: chain is shared with every sibling built so far, and
		// append() may otherwise write into a backing array a sibling's leaf
		// already stored a slice header into.
		childChain := make([]ancestorLink, len(chain)+1)
		copy(childChain, chain)
		childChain[len(chain)] = ancestorLink{split: n, idx: i}

		sub, err := m.build(&ch.Node, ch, n, fmt.Sprintf("%s.children[%d]", path, i), carry, fresh, childChain)
		if err != nil {
			return node{}, err
		}
		out.children = append(out.children, child{spec: parseSize(ch.Size), cfg: ch, node: sub})
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
		if m.overlayActive() {
			return m.handleOverlayMouse(msg)
		}
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
//   - A modal overlay (help, confirm dialog), when open, swallows every key.
//   - Global bindings (the "mod+…" set, F1) always win — even inside a locked
//     interactive pane. That is what makes "mod" a reliable escape hatch.
//   - Ctrl+C is never a plain quit: in NAV it raises the quit-confirm dialog;
//     while a pane is locked it interrupts that pane's process (with its own
//     confirm) and the program keeps running.
//   - Unlocked: arrows / Tab move focus; Enter locks into the focused pane.
//     Unbound keys still fall through to the focused widget (text-pane scroll).
//   - Locked: every key goes to the widget. Esc releases the lock; for a raw
//     pane the first Esc is buffered and a second Esc sends a literal Esc to the
//     child (Esc-Esc), so vim stays usable.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.overlay {
	case ovHelp:
		switch key {
		case "ctrl+c":
			return m.openConfirmQuit()
		case "f1", "esc", "q", "enter", " ":
			m.overlay = ovNone
		}
		return m, nil
	case ovConfirm:
		return m.handleConfirmKey(key)
	case ovSettings:
		return m.handleSettingsKey(msg)
	case ovControl:
		return m.handleControlKey(msg)
	case ovContext:
		return m.handleContextKey(msg)
	case ovEditor:
		return m.handleEditorKey(msg)
	case ovSplit:
		return m.handleSplitKey(msg)
	case ovGlobal:
		return m.handleGlobalKey(msg)
	}

	// Global bindings win unconditionally.
	if m.isGlobalKey(key) {
		if action, ok := m.keys[key]; ok {
			return m.runAction(action)
		}
	}

	// Ctrl+C: context-sensitive, never a bare quit.
	if key == "ctrl+c" {
		if m.locked {
			return m.lockedInterrupt(msg)
		}
		return m.openConfirmQuit()
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
	case "f3":
		// Deliberately NOT in isGlobalKey (unlike F1/F2): F3 only opens from
		// NAV, so a locked shell/vim keeps its own F3 (see isGlobalKey's doc).
		m.openGlobalPanel()
		return m, nil
	}
	if action, ok := m.keys[key]; ok {
		return m.runAction(action)
	}
	return m.forwardKey(msg)
}

// isGlobalKey reports whether a key is one the engine intercepts even inside a
// locked pane: anything carrying the modifier, plus F1 (help) and F2 (control
// panel). F3 is deliberately NOT here any more — it now falls through to the
// focused widget so a locked shell/editor keeps its F3. Ctrl+C is also absent
// on purpose — while locked it belongs to the pane's process.
func (m *Model) isGlobalKey(key string) bool {
	return key == "f1" || key == "f2" || strings.HasPrefix(key, m.mod+"+")
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
		if m.overlay == ovHelp {
			m.overlay = ovNone
		} else {
			m.overlay = ovHelp
		}
	case actSettings:
		m.overlayReturn = ovNone
		m.openSettings()
	case actControl:
		m.openControl()
	case actContext:
		m.openContext(m.focus)
	case actLock:
		m.locked = true
	case actUnlock:
		m.locked, m.pendingEsc = false, false
	case actZoomIn:
		m.setContentZoom(m.focus, m.leaves[m.focus].contentZoom+1)
	case actZoomOut:
		m.setContentZoom(m.focus, m.leaves[m.focus].contentZoom-1)
	case actZoomReset:
		m.setContentZoom(m.focus, 0)
	case actResizeUp:
		m.resizeFocused(dirUp)
	case actResizeDown:
		m.resizeFocused(dirDown)
	case actResizeLeft:
		m.resizeFocused(dirLeft)
	case actResizeRight:
		m.resizeFocused(dirRight)
	case actSwapUp:
		return m, m.swapFocused(dirUp)
	case actSwapDown:
		return m, m.swapFocused(dirDown)
	case actSwapLeft:
		return m, m.swapFocused(dirLeft)
	case actSwapRight:
		return m, m.swapFocused(dirRight)
	case actFlipSplit:
		m.flipFocusedSplit()
	}
	return m, nil
}

// setContentZoom clamps and applies a per-pane content-zoom level, forwarding
// it to the widget if it implements widget.Zoomer (see actZoomIn/-Out/-Reset
// above). Widgets that don't implement it simply ignore zoom — there is no
// generic way to scale a pty's own cell output.
func (m *Model) setContentZoom(idx, level int) {
	if idx < 0 || idx >= len(m.leaves) {
		return
	}
	lf := m.leaves[idx]
	lf.contentZoom = clampInt(level, 0, contentZoomMax)
	if z, ok := lf.w.(widget.Zoomer); ok {
		z.SetZoom(lf.contentZoom)
	}
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
	// mod+drag (move/resize, see drag.go) preempts everything below it,
	// including the plain right-click context menu — "holding mod disables
	// the normal context-menu function" is exactly this branch winning.
	// Once a drag is under way it keeps routing here even if a motion event
	// happens to not carry Alt (some terminals only stamp modifiers on the
	// initial press), so releasing the modifier mid-drag can't strand it.
	if m.dragMode != dragNone || msg.Alt {
		return m.handleDragMouse(msg)
	}
	i := m.leafAt(msg.X, msg.Y)
	if msg.Button == tea.MouseButtonRight && msg.Action == tea.MouseActionPress {
		// i < 0 is dead space (the status bar row is the only such spot —
		// the pane tree tiles the rest of the body exactly): open the "New
		// pane" menu instead of a per-pane one. openContext(-1) is a no-op
		// if that ever changes and there truly is nothing there.
		m.openContext(i) // target captured as ctxLeaf; focus is left where it is
		return m, nil
	}
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
	switch m.overlay {
	case ovHelp:
		out = m.renderHelp(out)
	case ovConfirm:
		out = m.renderConfirm(out)
	case ovSettings:
		out = m.renderSettings(out)
	case ovControl:
		out = m.renderControl(out)
	case ovContext:
		out = m.renderContext(out)
	case ovEditor:
		out = m.renderEditor(out)
	case ovSplit:
		out = m.renderSplit(out)
	case ovGlobal:
		out = m.renderGlobal(out)
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
		if c.cfg != nil {
			specs[i] = parseSize(c.cfg.Size) // reflect live edits to the ratio
		} else {
			specs[i] = c.spec
		}
	}
	dir := n.direction
	if n.cfg != nil && n.cfg.Direction != "" {
		dir = n.cfg.Direction // reflect a live row/column edit
	}
	row := dir == "row"
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
	if lf.title != "" {
		if sp, ok := lf.w.(widget.ScrollProgress); ok {
			if f := sp.ScrollFraction(); f > 0 {
				title = strings.TrimSpace(title) + fmt.Sprintf("  %.0f%%", f*100)
			}
		}
	}
	if focused && m.locked {
		title = strings.TrimSpace(title + "  " + th.bullet + " LOCKED")
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
		box = box.Border(th.border)
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
func (m *Model) Focused() string { return m.leafName(m.focus) }

// leafName is a leaf's title, or a positional fallback when it has none.
func (m *Model) leafName(i int) string {
	if i >= 0 && i < len(m.leaves) && m.leaves[i].title != "" {
		return m.leaves[i].title
	}
	return fmt.Sprintf("pane#%d", i)
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
