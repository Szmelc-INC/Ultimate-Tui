package widget

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// scrollBox is a vertically-scrollable text body shared by widgets that display
// static or captured text. It wraps content to the render width and shows a
// height-sized window into it, backed by bubbles/viewport for scroll physics
// and a scroll-position fraction.
type scrollBox struct {
	vp     viewport.Model
	raw    string
	inited bool
	wrapW  int // width the current viewport content was wrapped at
}

func (s *scrollBox) setText(text string) {
	s.raw = text
	s.wrapW = 0 // force a re-wrap + SetContent on the next view()
	if s.inited {
		s.vp.SetYOffset(0)
	}
}

func (s *scrollBox) ensure(width, height int) {
	if !s.inited {
		s.vp = viewport.New(width, height)
		s.vp.MouseWheelEnabled = true
		s.inited = true
	}
	if s.vp.Width != width || s.vp.Height != height {
		s.vp.Width, s.vp.Height = width, height
		s.wrapW = 0
	}
	if s.wrapW != width {
		s.vp.SetContent(lipgloss.NewStyle().Width(width).Render(s.raw))
		s.wrapW = width
	}
}

// handleKey consumes navigation keys and reports whether the offset moved.
func (s *scrollBox) handleKey(msg tea.KeyMsg) bool {
	if !s.inited {
		return false
	}
	switch msg.String() {
	case "g", "home":
		s.vp.GotoTop()
		return true
	case "G", "end":
		s.vp.GotoBottom()
		return true
	}
	before := s.vp.YOffset
	s.vp, _ = s.vp.Update(msg)
	return s.vp.YOffset != before
}

// handleMouse consumes wheel events and reports whether the offset moved.
func (s *scrollBox) handleMouse(msg tea.MouseMsg) bool {
	if !s.inited {
		return false
	}
	before := s.vp.YOffset
	s.vp, _ = s.vp.Update(msg)
	return s.vp.YOffset != before
}

func (s *scrollBox) view(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	s.ensure(width, height)
	return s.vp.View()
}

// scrollFraction is 0..1 — how far down the content is scrolled, or 0 when it
// all fits. Feeds the pane-title position indicator.
func (s *scrollBox) scrollFraction() float64 {
	if !s.inited || s.vp.TotalLineCount() <= s.vp.Height {
		return 0
	}
	return s.vp.ScrollPercent()
}
