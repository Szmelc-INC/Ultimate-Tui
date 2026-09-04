// Package engine turns a parsed config into a running bubbletea program: it
// builds the layout tree, owns focus and global keybindings, and renders each
// pane's widget into its computed rectangle.
package engine

import (
	"fmt"

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
}

// Model is the root bubbletea model.
type Model struct {
	root   node
	leaves []*leaf
	focus  int
	keys   map[string]string
	w, h   int
	ready  bool
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

// Update routes messages: window size and global keys are handled here, other
// keys go to the focused widget, everything else is broadcast.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h, m.ready = msg.Width, msg.Height, true
		return m, nil

	case tea.KeyMsg:
		if action, ok := m.keys[msg.String()]; ok {
			switch action {
			case actQuit:
				return m, tea.Quit
			case actFocusNext:
				m.focus = (m.focus + 1) % len(m.leaves)
				return m, nil
			case actFocusPrev:
				m.focus = (m.focus - 1 + len(m.leaves)) % len(m.leaves)
				return m, nil
			}
		}
		nw, cmd := m.leaves[m.focus].w.Update(msg)
		m.leaves[m.focus].w = nw
		return m, cmd

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

// View renders the whole tree.
func (m *Model) View() string {
	if !m.ready {
		return "loading…"
	}
	return m.render(m.root, m.w, m.h)
}

func (m *Model) render(n node, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	if n.leaf != nil {
		return m.renderLeaf(n.leaf, w, h)
	}

	specs := make([]sizeSpec, len(n.children))
	for i, c := range n.children {
		specs[i] = c.spec
	}
	axis := h
	if n.direction == "row" {
		axis = w
	}
	sizes := distribute(specs, axis)

	parts := make([]string, len(n.children))
	for i, c := range n.children {
		if n.direction == "row" {
			parts[i] = m.render(c.node, sizes[i], h)
		} else {
			parts[i] = m.render(c.node, w, sizes[i])
		}
	}
	if n.direction == "row" {
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

	bodyH := innerH
	var head string
	if lf.title != "" {
		head = titleStyle.Width(innerW).MaxWidth(innerW).Render(lf.title)
		bodyH = innerH - 1
	}
	body := lf.w.View(innerW, bodyH)
	content := body
	if head != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, head, body)
	}

	// lipgloss Width/Height size the content area; a border adds 2 cells on
	// each axis outside that. So the bordered box uses innerW/innerH (== w-2 /
	// h-2) and lands at exactly w x h. MaxWidth/MaxHeight would clip the final
	// output *including* the border, shearing off the bottom/right edge.
	box := lipgloss.NewStyle().Width(innerW).Height(innerH)
	if lf.border {
		box = box.Border(lipgloss.NormalBorder())
		if m.leaves[m.focus] == lf {
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
