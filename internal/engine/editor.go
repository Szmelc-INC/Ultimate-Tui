package engine

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The editor overlay. This is the read-only first cut: it shows every editable
// parameter of a pane and its current value. Interactive editing (tabbed
// categories, select / text / slider / checkbox controls, save/discard) lands
// in the following commits. Opened from the pane context menu's "Edit" row.

func (m *Model) openPaneEditor(idx int) {
	if idx < 0 || idx >= len(m.leaves) {
		return
	}
	m.overlay = ovEditor
	m.ctxLeaf = idx
}

func (m *Model) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.overlay = m.overlayReturn
		m.overlayReturn = ovNone
	}
	return m, nil
}

func (m *Model) renderEditor(frame string) string {
	return overlayCenter(frame, m.editorPanel(), m.w, m.h)
}

func (m *Model) editorPanel() string {
	th := m.theme()

	var b strings.Builder
	b.WriteString(th.helpTitle.Render("edit · " + m.leafName(m.ctxLeaf)))
	b.WriteString("\n\n")

	row := func(k, v string) {
		b.WriteString(th.helpKey.Render(padRight(k, 14)))
		b.WriteString(th.helpDesc.Render(v))
		b.WriteString("\n")
	}

	if m.ctxLeaf < 0 || m.ctxLeaf >= len(m.leaves) {
		b.WriteString(th.helpDesc.Render("(no pane)"))
		return th.helpBox.Render(b.String())
	}
	lf := m.leaves[m.ctxLeaf]
	p := lf.pane

	row("type", p.Type)
	row("title", quoteOrDash(p.Title))
	row("border", boolWord(p.Border == nil || *p.Border))
	if lf.parent != nil {
		row("size", quoteOrDash(lf.parent.Size))
	}
	switch p.Type {
	case "text":
		row("content", quoteOrDash(oneLine(p.Content)))
	case "markdown":
		row("content", quoteOrDash(oneLine(p.Content)))
		row("file", quoteOrDash(p.File))
		row("markdownStyle", quoteOrDash(p.MarkdownStyle))
	case "command", "shell":
		row("command", quoteOrDash(p.Command))
		row("shell", quoteOrDash(p.Shell))
	}

	b.WriteString("\n")
	b.WriteString(th.helpDim.Render("read-only preview · Esc close · interactive editing next"))
	return th.helpBox.Render(b.String())
}

func quoteOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return fmt.Sprintf("%q", s)
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", "⏎")
	if len(s) > 48 {
		s = s[:47] + "…"
	}
	return s
}
