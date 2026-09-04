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

// StreamMsg marks messages emitted by a widget's continuous read loop (see the
// shell widget). engine.Snapshot uses it to avoid draining an endless stream.
type StreamMsg interface {
	stream()
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
