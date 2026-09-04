// Package widget defines the Widget contract and a registry of widget kinds
// compiled into the engine. Config selects a kind by name and configures it;
// there is deliberately no plugin/dlopen mechanism, so runtime, dynamic-linked
// and go:embed-static builds all use the identical engine.
package widget

import (
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// Widget is a self-contained pane body. It is a bubbletea component with an
// explicit render size — the engine owns layout, the widget owns content.
type Widget interface {
	// Init returns an optional command to run when the program starts
	// (e.g. kick off a subprocess).
	Init() tea.Cmd
	// Update handles a message and returns the (possibly new) widget. The
	// engine forwards key messages only to the focused widget; every other
	// message is broadcast to all widgets.
	Update(msg tea.Msg) (Widget, tea.Cmd)
	// View renders the body into exactly width x height cells.
	View(width, height int) string
}

// RawKeyConsumer is an optional interface. A widget that returns true wants
// every key (except the engine's prefix) delivered verbatim while it is
// focused — an interactive child process, typically. The engine checks this
// per keypress, so a widget can drop out of raw mode when its process exits.
type RawKeyConsumer interface {
	ConsumesRawKeys() bool
}

// Closer is an optional interface for widgets holding OS resources (a child
// process, a pty). The engine calls Close on every leaf when the program ends.
type Closer interface {
	Close() error
}

// Interrupter is an optional interface for a widget that hosts a killable
// process. The engine calls Interrupt when the user confirms Ctrl+C on a
// locked pane: it stops that pane's process without ending the program.
// Distinct from Closer, which fires only on program exit.
type Interrupter interface {
	Interrupt() error
}

// Restarter is an optional interface for a widget that can tear its process
// down and spawn it again from scratch. The engine calls Restart from the pane
// context menu; the returned command is the fresh process's first read.
type Restarter interface {
	Restart() tea.Cmd
}

// Refresher is an optional interface for a widget that can force a repaint of
// its content without a full restart (a shell pane sends Ctrl+L to its child;
// a command pane re-runs). The engine calls Refresh from the pane context menu.
type Refresher interface {
	Refresh() tea.Cmd
}

// StreamMsg marks messages emitted by a widget's continuous read loop (see the
// shell widget). engine.Snapshot uses it to avoid draining an endless stream.
type StreamMsg interface {
	stream()
}

// ScrollProgress is an optional interface. A widget that reports how far its
// content is scrolled (0..1, and 0 when nothing overflows) gets a compact
// "NN%" indicator appended to its pane title.
type ScrollProgress interface {
	ScrollFraction() float64
}

// Zoomer is an optional interface for a widget that can scale its own
// content independent of its pane's rectangle — mod+=/-/0 on the focused
// pane, distinct from mod+arrows which resizes the pane itself. Level 0 is
// the unscaled baseline and is never negative ("smaller than one cell" has
// no meaning); the engine clamps and calls SetZoom only when the level
// actually changes. A widget that doesn't implement this simply ignores
// zoom — there is no generic way to magnify a pty's own cell output.
type Zoomer interface {
	SetZoom(level int)
}

// Factory builds a widget from its pane config.
type Factory func(*config.Pane) (Widget, error)

var registry = map[string]Factory{}

// Register adds a widget kind. Call from an init function in the kind's file.
func Register(kind string, f Factory) {
	if _, dup := registry[kind]; dup {
		panic("widget: duplicate registration for " + kind)
	}
	registry[kind] = f
}

// New instantiates the widget named by p.Type.
func New(p *config.Pane) (Widget, error) {
	f, ok := registry[p.Type]
	if !ok {
		return nil, fmt.Errorf("unknown widget type %q (known: %v)", p.Type, Kinds())
	}
	return f(p)
}

// Kinds lists registered widget names, sorted.
func Kinds() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
