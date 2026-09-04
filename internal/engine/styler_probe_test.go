package engine_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
	"github.com/silverx/ultimate-tui/internal/widget"
)

// lastProbeStyles records whatever widget.Styles a "probe" pane most
// recently received via SetStyles. ANSI color itself is invisible under
// go test's non-TTY lipgloss profile, so this is how the renderLeaf ->
// widget.Styler wiring gets checked programmatically instead of by eye
// (CLAUDE.md's "verify programmatically, not by eye") — a dropped push, or
// a Styler method rename that still compiles (see the widget package's own
// `var _ Styler = ...` assertions for the compile-time half of that), would
// otherwise revert every themed widget to its stock colors with a still-
// green suite.
var lastProbeStyles widget.Styles

type probeWidget struct{}

func (probeWidget) Init() tea.Cmd                               { return nil }
func (probeWidget) Update(msg tea.Msg) (widget.Widget, tea.Cmd) { return probeWidget{}, nil }
func (probeWidget) View(w, h int) string                        { return "" }
func (probeWidget) SetStyles(s widget.Styles)                   { lastProbeStyles = s }

func init() {
	widget.Register("probe", func(*config.Pane) (widget.Widget, error) {
		return probeWidget{}, nil
	})
}

const probeSrc = `{"name":"probe","layout":{"pane":{"type":"probe","title":"p"}}}`

func TestRenderLeafPushesThemeStylesIntoWidgets(t *testing.T) {
	cfg, err := config.Parse([]byte(probeSrc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	snap := func(theme string) widget.Styles {
		lastProbeStyles = widget.Styles{}
		opts := engine.DefaultOptions()
		opts.Theme, opts.Style = theme, "rounded"
		if _, err := engine.SnapshotWithOptions(cfg, 40, 10, opts); err != nil {
			t.Fatalf("snapshot(%s): %v", theme, err)
		}
		return lastProbeStyles
	}

	gruvbox := snap("gruvbox")
	if gruvbox.Accent == nil {
		t.Fatal("SetStyles never reached the probe widget (Accent is nil) — renderLeaf's Styler push is broken")
	}

	slate := snap("slate")
	if gruvbox.Accent == slate.Accent {
		t.Fatalf("widget.Styles.Accent identical for gruvbox and slate (%#v) — theme changes aren't reaching the widget", gruvbox.Accent)
	}
}
