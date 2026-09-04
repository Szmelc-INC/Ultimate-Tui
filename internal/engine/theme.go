package engine

import "github.com/charmbracelet/lipgloss"

// theme is the resolved palette + styles for one run. Colours are
// lipgloss.AdaptiveColor so a dark or light terminal both read well; the
// engine picks the variant automatically unless Options.Theme forces one.
type theme struct {
	borderIdle  lipgloss.TerminalColor
	borderFocus lipgloss.TerminalColor
	borderLock  lipgloss.TerminalColor

	title      lipgloss.Style
	titleFocus lipgloss.Style

	bar     lipgloss.Style
	barKey  lipgloss.Style
	barMode lipgloss.Style

	helpBox     lipgloss.Style
	helpTitle   lipgloss.Style
	helpKey     lipgloss.Style
	helpDesc    lipgloss.Style
	helpSection lipgloss.Style
	helpDim     lipgloss.Style
}

var (
	colSubtle  = lipgloss.AdaptiveColor{Light: "#9AA0A6", Dark: "#5C6370"}
	colAccent  = lipgloss.AdaptiveColor{Light: "#6C4CE0", Dark: "#B4A0FF"}
	colAccent2 = lipgloss.AdaptiveColor{Light: "#0B7285", Dark: "#63D2E0"}
	colWarn    = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#F2A65A"}
	colText    = lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#E6E6E6"}
	colBarBg   = lipgloss.AdaptiveColor{Light: "#ECECFF", Dark: "#2A2740"}
)

func (m *Model) theme() theme {
	// Options.Theme could pin light/dark here later; lipgloss already adapts to
	// the detected background, which covers the common case.
	t := theme{
		borderIdle:  colSubtle,
		borderFocus: colAccent,
		borderLock:  colWarn,
	}
	t.title = lipgloss.NewStyle().Bold(true).Foreground(colSubtle)
	t.titleFocus = lipgloss.NewStyle().Bold(true).Foreground(colAccent)

	t.bar = lipgloss.NewStyle().Background(colBarBg).Foreground(colText)
	t.barKey = lipgloss.NewStyle().Background(colBarBg).Foreground(colAccent2).Bold(true)
	t.barMode = lipgloss.NewStyle().Background(colAccent).Foreground(lipgloss.Color("#101010")).Bold(true)

	t.helpBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colAccent).
		Padding(1, 3)
	t.helpTitle = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	t.helpSection = lipgloss.NewStyle().Bold(true).Foreground(colAccent2)
	t.helpKey = lipgloss.NewStyle().Bold(true).Foreground(colText)
	t.helpDesc = lipgloss.NewStyle().Foreground(colText)
	t.helpDim = lipgloss.NewStyle().Foreground(colSubtle).Italic(true)
	return t
}
