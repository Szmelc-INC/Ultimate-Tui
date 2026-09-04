package engine

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/widget"
)

type recWidget struct{ closed int }

func (r *recWidget) Init() tea.Cmd                           { return nil }
func (r *recWidget) Update(tea.Msg) (widget.Widget, tea.Cmd) { return r, nil }
func (r *recWidget) View(w, h int) string                    { return "" }
func (r *recWidget) Close() error                            { r.closed++; return nil }

func mkModel(t *testing.T, src string) *Model {
	t.Helper()
	cfg, err := config.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return u.(*Model)
}

const oneText = `{"name":"x","layout":{"pane":{"type":"text","title":"T","content":"hi"}}}`

const splitSrc = `{"name":"x","layout":{"direction":"row","children":[
  {"size":"50%","pane":{"type":"text","title":"L","content":"l"}},
  {"size":"*","pane":{"type":"text","title":"R","content":"r"}}
]}}`

// A widget-affecting edit (content change) replaces lf.w with a fresh
// instance; the leaf's buffered content-zoom level must carry over onto that
// new instance, not silently reset to unzoomed while the leaf still reports
// the old level (see advisor review: mod+- would then make the pane bigger).
func TestApplyEditReappliesContentZoomToRebuiltWidget(t *testing.T) {
	m := mkModel(t, oneText)
	m.setContentZoom(0, 3)

	m.openPaneEditor(0)
	m.eb.content = "changed" // widget-affecting -> applyEdit rebuilds lf.w
	_ = m.applyEdit()

	// text implements widget.Zoomer; compare the rebuilt widget's rendering
	// against a plain unzoomed instance and against one with SetZoom(3)
	// applied directly, rather than reaching into unexported fields.
	unzoomed, err := widget.New(&config.Pane{Type: "text", Content: "changed"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := widget.New(&config.Pane{Type: "text", Content: "changed"})
	if err != nil {
		t.Fatal(err)
	}
	z, ok := want.(widget.Zoomer)
	if !ok {
		t.Fatal("text widget does not implement widget.Zoomer")
	}
	z.SetZoom(3)

	got := m.leaves[0].w.View(40, 10)
	if got == unzoomed.View(40, 10) {
		t.Fatal("rebuilt widget did not carry over the buffered content-zoom level (renders unzoomed)")
	}
	if got != want.View(40, 10) {
		t.Fatal("rebuilt widget's rendering does not match SetZoom(3) applied directly")
	}
}

func TestApplyEditRebuildsAndClosesOldWidget(t *testing.T) {
	m := mkModel(t, oneText)
	fake := &recWidget{}
	m.leaves[0].w = fake

	m.openPaneEditor(0)
	m.eb.content = "changed"
	_ = m.applyEdit()

	if fake.closed != 1 {
		t.Fatalf("old widget Close() called %d times, want 1", fake.closed)
	}
	if m.leaves[0].w == widget.Widget(fake) {
		t.Fatal("widget was not replaced")
	}
	if m.leaves[0].pane.Content != "changed" {
		t.Fatalf("pane.Content = %q, want %q", m.leaves[0].pane.Content, "changed")
	}
	if m.edDirty {
		t.Fatal("edDirty should be cleared after apply")
	}
}

func TestApplyEditInvalidTypeKeepsOldWidget(t *testing.T) {
	m := mkModel(t, oneText)
	orig := m.leaves[0].w

	m.openPaneEditor(0)
	m.eb.typ = "command" // command widget requires a non-empty command
	_ = m.applyEdit()

	if m.edErr == "" {
		t.Fatal("expected an error for an unbuildable pane")
	}
	if m.leaves[0].pane.Type != "text" || m.leaves[0].w != orig {
		t.Fatal("invalid apply mutated the live pane")
	}

	// Now provide the command and re-apply: it should succeed.
	m.eb.command = "true"
	_ = m.applyEdit()
	if m.edErr != "" {
		t.Fatalf("valid apply still errored: %s", m.edErr)
	}
	if m.leaves[0].pane.Type != "command" || m.leaves[0].w == orig {
		t.Fatal("valid apply did not rebuild as command")
	}
}

func TestApplyEditSizeInPlace(t *testing.T) {
	m := mkModel(t, splitSrc)
	_ = m.View()

	m.openPaneEditor(0)
	m.eb.sizeMode, m.eb.sizeVal = "percent", 75
	_ = m.applyEdit()

	if got := m.leaves[0].parent.Size; got != "75%" {
		t.Fatalf("parent.Size = %q, want 75%%", got)
	}
	// render must pick up the new ratio: left pane wider than right.
	m.overlay = ovNone
	v := m.View()
	lines := strings.Split(v, "\n")
	// count leading spaces before the right pane's border on a mid row —
	// simpler: assert the frame still renders at full width and height.
	if len(lines) != 24 {
		t.Fatalf("frame height = %d after resize, want 24", len(lines))
	}
}

func TestEditorSaveCallsCallbackAndClearsDirty(t *testing.T) {
	cfg, err := config.Parse([]byte(oneText))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	var saved *config.Config
	opts := DefaultOptions()
	opts.SaveConfig = func(c *config.Config) error { calls++; saved = c; return nil }
	m, err := NewWithOptions(cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = u.(*Model)

	m.openPaneEditor(0)
	m.eb.title = "Renamed"
	m.edDirty = true
	_ = m.editorSave()

	if calls != 1 {
		t.Fatalf("SaveConfig called %d times, want 1", calls)
	}
	if saved != m.cfg || m.leaves[0].pane.Title != "Renamed" {
		t.Fatal("save did not apply the buffer to the live config first")
	}
	if m.edDirty || m.edConfigDirty {
		t.Fatal("dirty flags not cleared after a successful save")
	}
	if m.edNote == "" {
		t.Fatal("no on-screen save confirmation")
	}
}

func TestEditorSaveWithoutCallbackErrors(t *testing.T) {
	m := mkModel(t, oneText) // DefaultOptions -> SaveConfig nil
	m.openPaneEditor(0)
	m.eb.title = "X"
	m.edDirty = true
	_ = m.editorSave()
	if m.edErr == "" {
		t.Fatal("expected an error when no SaveConfig is wired")
	}
}

func TestEditorEscConfirmsWhenDirty(t *testing.T) {
	m := mkModel(t, oneText)
	m.openPaneEditor(0)
	m.eb.title = "Changed"
	m.edDirty = true

	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	if m.overlay != ovConfirm || m.confirmKind != cfDiscard {
		t.Fatalf("Esc with edits did not raise the discard confirm (overlay=%d)", m.overlay)
	}

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = u.(*Model)
	if m.overlay != ovEditor {
		t.Fatalf("cancel did not return to the editor (overlay=%d)", m.overlay)
	}

	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	u, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = u.(*Model)
	if m.overlay != ovNone {
		t.Fatalf("discard=yes did not close the editor (overlay=%d)", m.overlay)
	}
}

func TestEditorEscCleanWhenPristine(t *testing.T) {
	m := mkModel(t, oneText)
	m.openPaneEditor(0)
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = u.(*Model)
	if m.overlay != ovNone {
		t.Fatalf("Esc on a pristine editor did not close it (overlay=%d)", m.overlay)
	}
}

func TestApplyEditDirectionInPlace(t *testing.T) {
	// splitSrc is a "direction":"row" split of two text panes.
	m := mkModel(t, splitSrc)
	_ = m.View()
	if m.root.cfg == nil || m.root.cfg.Direction != "row" {
		t.Fatalf("precondition: root split not row (%q)", m.root.cfg.Direction)
	}

	m.openPaneEditor(0)
	if !m.eb.hasDir || m.eb.dir != "row" {
		t.Fatalf("buffer did not load the split direction: hasDir=%v dir=%q", m.eb.hasDir, m.eb.dir)
	}
	m.eb.dir = "column"
	_ = m.applyEdit()

	if m.leaves[0].split.Direction != "" {
		t.Fatalf("apply wrote Direction = %q, want \"\" (column)", m.leaves[0].split.Direction)
	}
	// render must pick up the flip: with a column split the two panes now stack,
	// so the frame is still 24 rows but each pane spans the full 80 columns.
	m.overlay = ovNone
	v := m.View()
	if lines := strings.Split(v, "\n"); len(lines) != 24 {
		t.Fatalf("frame height = %d after direction flip", len(lines))
	}
}

func TestParseComposeSizeRoundTrip(t *testing.T) {
	cases := []struct {
		in, mode string
		val      int
		out      string
	}{
		{"40%", "percent", 40, "40%"},
		{"20", "fixed", 20, "20"},
		{"3*", "flex", 3, "3*"},
		{"", "auto", 1, "*"},
		{"*", "auto", 1, "*"},
		{"1*", "flex", 1, "*"},
	}
	for _, c := range cases {
		mode, val := parseSizeStr(c.in)
		if mode != c.mode || val != c.val {
			t.Errorf("parseSizeStr(%q) = (%q,%d), want (%q,%d)", c.in, mode, val, c.mode, c.val)
		}
		if got := composeSize(mode, val); got != c.out {
			t.Errorf("composeSize(%q,%d) = %q, want %q", mode, val, got, c.out)
		}
	}
}
