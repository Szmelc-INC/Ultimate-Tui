package engine

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestShiftZoomPreservesSum(t *testing.T) {
	cases := []struct {
		name  string
		sizes []int
		bias  []int
	}{
		{"grow first of two", []int{40, 60}, []int{9, 0}},
		{"shrink middle of three", []int{30, 30, 40}, []int{0, -6, 0}},
		{"two zoomed opposite", []int{25, 25, 25, 25}, []int{9, 0, 0, -9}},
		{"bias exceeds room", []int{50, 6}, []int{40, 0}},
		{"single child no-op", []int{80}, []int{9}},
		{"tiny container", []int{4, 4, 4}, []int{6, 0, 0}},
	}
	for _, tc := range cases {
		want := 0
		for _, s := range tc.sizes {
			want += s
		}
		got := shiftZoom(tc.sizes, tc.bias)
		total := 0
		for _, v := range got {
			total += v
			if v < 0 {
				t.Errorf("%s: negative slot in %v", tc.name, got)
			}
		}
		if total != want {
			t.Errorf("%s: sum(%v) = %d, want %d", tc.name, got, total, want)
		}
	}
}

func TestShiftZoomActuallyGrows(t *testing.T) {
	got := shiftZoom([]int{50, 50}, []int{9, 0})
	if got[0] <= 50 || got[1] >= 50 {
		t.Fatalf("zoom did not grow slot 0 at slot 1's expense: %v", got)
	}
}

const zoomGridSrc = `{
  "name": "grid",
  "layout": { "direction": "column", "children": [
    { "size": "50%", "direction": "row", "children": [
      { "pane": { "type": "text", "title": "TL", "content": "a" } },
      { "pane": { "type": "text", "title": "TR", "content": "b" } }
    ]},
    { "size": "*", "direction": "row", "children": [
      { "pane": { "type": "text", "title": "BL", "content": "c" } },
      { "pane": { "type": "text", "title": "BR", "content": "d" } }
    ]}
  ]}
}`

// The whole zoom feature rests on bubbletea encoding Alt+= as "alt+=" so the
// "mod+=" baseBind resolves. Drive the real key path, not just the helper.
func TestZoomBindingBiasesFocusedPane(t *testing.T) {
	cfg, err := config.Parse([]byte(zoomGridSrc))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewWithOptions(cfg, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = u.(*Model)

	press := func(r rune) {
		u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true})
		m = u.(*Model)
	}

	press('=')
	press('=')
	if z := m.leaves[m.focus].zoom; z != 2 {
		t.Fatalf("alt+= x2: focused zoom = %d, want 2", z)
	}
	press('-')
	if z := m.leaves[m.focus].zoom; z != 1 {
		t.Fatalf("alt+-: focused zoom = %d, want 1", z)
	}
	press('0')
	if z := m.leaves[m.focus].zoom; z != 0 {
		t.Fatalf("alt+0: focused zoom = %d, want 0", z)
	}

	// The zoomed frame must still be exactly 80x24 — no shearing.
	press('=')
	press('=')
	press('=')
	v := m.View()
	lines := splitLines(v)
	if len(lines) != 24 {
		t.Fatalf("zoomed frame height = %d, want 24", len(lines))
	}
}

func splitLines(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
