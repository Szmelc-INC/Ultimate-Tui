package widget

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// scrollBox is a minimal vertically-scrollable text body shared by widgets that
// display static or captured text. It wraps to the render width and shows a
// height-sized window into the content.
type scrollBox struct {
	raw string
	off int
}

func (s *scrollBox) setText(text string) {
	s.raw = text
	s.off = 0
}

// handleKey consumes navigation keys and reports whether it did.
func (s *scrollBox) handleKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up", "k":
		if s.off > 0 {
			s.off--
		}
	case "down", "j":
		s.off++
	case "pgup", "b":
		s.off -= 10
		if s.off < 0 {
			s.off = 0
		}
	case "pgdown", "f", " ":
		s.off += 10
	case "home", "g":
		s.off = 0
	default:
		return false
	}
	return true
}

// handleMouse consumes wheel events and reports whether it did.
func (s *scrollBox) handleMouse(msg tea.MouseMsg) bool {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if s.off -= 3; s.off < 0 {
			s.off = 0
		}
		return true
	case tea.MouseButtonWheelDown:
		s.off += 3
		return true
	}
	return false
}

func (s *scrollBox) view(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(s.raw)
	lines := strings.Split(wrapped, "\n")

	maxOff := len(lines) - height
	if maxOff < 0 {
		maxOff = 0
	}
	if s.off > maxOff {
		s.off = maxOff
	}

	end := s.off + height
	if end > len(lines) {
		end = len(lines)
	}
	window := append([]string(nil), lines[s.off:end]...)
	for len(window) < height {
		window = append(window, "")
	}
	return strings.Join(window, "\n")
}
