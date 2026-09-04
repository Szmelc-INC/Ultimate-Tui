package widget

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/silverx/ultimate-tui/internal/config"
)

func init() { Register("markdown", newMarkdown) }

// markdownWidget renders a Markdown document with charmbracelet/glamour —
// headings, lists, tables, blockquotes and syntax-highlighted code blocks — and
// scrolls it. Source is either inline `content` or a `file` path.
//
// glamour pre-wraps its output to a fixed width and embeds ANSI styling, so the
// rendered text is cached per render width and scrolled as opaque lines (never
// re-wrapped, unlike the plain scrollBox).
type markdownWidget struct {
	src   string
	style string

	off    int
	cacheW int
	lines  []string
	rerr   error
}

func newMarkdown(p *config.Pane) (Widget, error) {
	src := p.Content
	if p.File != "" {
		b, err := os.ReadFile(p.File)
		if err != nil {
			return nil, fmt.Errorf("markdown widget: %w", err)
		}
		src = string(b)
	}
	if strings.TrimSpace(src) == "" {
		return nil, fmt.Errorf("markdown widget: needs \"content\" or \"file\"")
	}
	style := p.MarkdownStyle
	if style == "" {
		style = "dark" // pinned: keeps `ultui render` output deterministic
	}
	return &markdownWidget{src: src, style: style, cacheW: -1}, nil
}

func (w *markdownWidget) Init() tea.Cmd { return nil }

func (w *markdownWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "up", "k":
			if w.off > 0 {
				w.off--
			}
		case "down", "j":
			w.off++
		case "pgup", "b":
			w.off -= 10
		case "pgdown", "f", " ":
			w.off += 10
		case "home", "g":
			w.off = 0
		}
		if w.off < 0 {
			w.off = 0
		}
	case tea.MouseMsg:
		switch m.Button {
		case tea.MouseButtonWheelUp:
			if w.off -= 3; w.off < 0 {
				w.off = 0
			}
		case tea.MouseButtonWheelDown:
			w.off += 3
		}
	}
	return w, nil
}

func (w *markdownWidget) render(width int) {
	if width == w.cacheW {
		return
	}
	w.cacheW = width
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(w.style),
		glamour.WithWordWrap(max(width, 1)),
	)
	if err != nil {
		w.lines, w.rerr = nil, err
		return
	}
	out, err := r.Render(w.src)
	if err != nil {
		w.lines, w.rerr = nil, err
		return
	}
	w.rerr = nil
	w.lines = strings.Split(strings.Trim(out, "\n"), "\n")
}

func (w *markdownWidget) View(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	w.render(width)
	if w.rerr != nil {
		return normalizeGrid("markdown: "+w.rerr.Error(), width, height)
	}

	maxOff := len(w.lines) - height
	if maxOff < 0 {
		maxOff = 0
	}
	if w.off > maxOff {
		w.off = maxOff
	}
	end := w.off + height
	if end > len(w.lines) {
		end = len(w.lines)
	}
	window := strings.Join(w.lines[w.off:end], "\n")
	return normalizeGrid(window, width, height)
}
