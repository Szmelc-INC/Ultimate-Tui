package engine

import (
	"sort"

	"github.com/charmbracelet/lipgloss"

	"github.com/silverx/ultimate-tui/internal/widget"
)

// theme is the fully-resolved palette + styleset for one run: concrete colours,
// a border, and the glyphs the chrome draws with. Built once per (themeName,
// styleName) pair and memoized on the Model.
type theme struct {
	borderIdle  lipgloss.TerminalColor
	borderFocus lipgloss.TerminalColor
	borderLock  lipgloss.TerminalColor

	border    lipgloss.Border
	glyphNav  string
	glyphLock string
	bullet    string
	sep       string
	ellipsis  string

	title      lipgloss.Style
	titleFocus lipgloss.Style

	bar     lipgloss.Style
	barMode lipgloss.Style

	helpBox     lipgloss.Style
	helpTitle   lipgloss.Style
	helpKey     lipgloss.Style
	helpDesc    lipgloss.Style
	helpSection lipgloss.Style
	helpDim     lipgloss.Style

	menuSel    lipgloss.Style
	menuItem   lipgloss.Style
	menuColHdr lipgloss.Style

	// chipPrimary/chipMuted are filled "pill" styles — Background, not just
	// Foreground, unlike everything above — for the app's few genuine
	// buttons/badges: a confirm dialog's Yes/Cancel (overlay.go) and the
	// status bar's key-hint badges (chrome.go). chipPrimary is the
	// affirmative/primary action; chipMuted is everything else that still
	// wants a filled chip but not the one loud accent color repeated
	// everywhere — see palette.surface's doc comment.
	chipPrimary lipgloss.Style
	chipMuted   lipgloss.Style

	// widgetStyles is pushed into every leaf's widget each frame (renderLeaf)
	// via widget.Styler, so list/form/input content is colored from the same
	// resolved palette as the chrome around it — see widget/styles.go.
	widgetStyles widget.Styles
}

// palette is the semantic colour set a named theme provides. Named themes use
// concrete lipgloss.Color; the "auto" palette uses AdaptiveColor so a light or
// dark terminal both read well.
type palette struct {
	subtle  lipgloss.TerminalColor
	accent  lipgloss.TerminalColor
	accent2 lipgloss.TerminalColor
	warn    lipgloss.TerminalColor
	success lipgloss.TerminalColor
	text    lipgloss.TerminalColor
	barBg   lipgloss.TerminalColor
	modeFg  lipgloss.TerminalColor // text drawn on the accent-coloured mode chip

	// surface/onSurface are a second, muted fill+foreground pair for a
	// neutral filled "chip" (a status-bar hint badge, a dialog's secondary
	// button) — distinct from barBg so a chip reads as its own element
	// against the status bar's own background, not just more of it, and
	// distinct from accent so a screen with several filled chips doesn't
	// become the single loud accent color repeated everywhere.
	surface   lipgloss.TerminalColor
	onSurface lipgloss.TerminalColor
}

// styleset is the non-colour half of a theme: box-drawing border + the glyphs
// the status bar / pane titles use. Presets range from Nerd-Font-rich to pure
// 7-bit ASCII for terminals without a patched font.
type styleset struct {
	border    lipgloss.Border
	glyphNav  string
	glyphLock string
	bullet    string
	sep       string
	ellipsis  string
}

const (
	defaultThemeName = "auto"
	defaultStyleName = "rounded"
)

var palettes = map[string]palette{
	"auto": {
		subtle:  lipgloss.AdaptiveColor{Light: "#9AA0A6", Dark: "#5C6370"},
		accent:  lipgloss.AdaptiveColor{Light: "#6C4CE0", Dark: "#B4A0FF"},
		accent2: lipgloss.AdaptiveColor{Light: "#0B7285", Dark: "#63D2E0"},
		warn:    lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#F2A65A"},
		success: lipgloss.AdaptiveColor{Light: "#2F9E44", Dark: "#8CE99A"},
		text:    lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#E6E6E6"},
		barBg:   lipgloss.AdaptiveColor{Light: "#ECECFF", Dark: "#2A2740"},
		modeFg:  lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#101010"},
		// Deliberately desaturated relative to accent/accent2 — a purple-
		// tinted dark tone here (e.g. #332F4D) is close enough to equal
		// R/G/B that it downsamples to plain ANSI magenta on a 16-color
		// terminal, which is exactly the "loud accent repeated everywhere"
		// look palette.surface exists to avoid landing on.
		surface:   lipgloss.AdaptiveColor{Light: "#E9E7F0", Dark: "#3A3944"},
		onSurface: lipgloss.AdaptiveColor{Light: "#3D3A47", Dark: "#D7D5DE"},
	},
	"tokyo-night": {
		subtle:    lipgloss.Color("#565f89"),
		accent:    lipgloss.Color("#7aa2f7"),
		accent2:   lipgloss.Color("#7dcfff"),
		warn:      lipgloss.Color("#ff9e64"),
		success:   lipgloss.Color("#9ece6a"),
		text:      lipgloss.Color("#c0caf5"),
		barBg:     lipgloss.Color("#24283b"),
		modeFg:    lipgloss.Color("#1a1b26"),
		surface:   lipgloss.Color("#292e42"),
		onSurface: lipgloss.Color("#a9b1d6"),
	},
	"gruvbox": {
		subtle:    lipgloss.Color("#928374"),
		accent:    lipgloss.Color("#fabd2f"),
		accent2:   lipgloss.Color("#8ec07c"),
		warn:      lipgloss.Color("#fe8019"),
		success:   lipgloss.Color("#b8bb26"),
		text:      lipgloss.Color("#ebdbb2"),
		barBg:     lipgloss.Color("#3c3836"),
		modeFg:    lipgloss.Color("#282828"),
		surface:   lipgloss.Color("#504945"),
		onSurface: lipgloss.Color("#d5c4a1"),
	},
	"slate": {
		subtle:    lipgloss.Color("#64748b"),
		accent:    lipgloss.Color("#38bdf8"),
		accent2:   lipgloss.Color("#2dd4bf"),
		warn:      lipgloss.Color("#fb7185"),
		success:   lipgloss.Color("#34d399"),
		text:      lipgloss.Color("#e2e8f0"),
		barBg:     lipgloss.Color("#1e293b"),
		modeFg:    lipgloss.Color("#0f172a"),
		surface:   lipgloss.Color("#334155"),
		onSurface: lipgloss.Color("#cbd5e1"),
	},
	"midnight": {
		subtle:    lipgloss.Color("#6b7280"),
		accent:    lipgloss.Color("#8b5cf6"),
		accent2:   lipgloss.Color("#22d3ee"),
		warn:      lipgloss.Color("#f472b6"),
		success:   lipgloss.Color("#4ade80"),
		text:      lipgloss.Color("#e5e7eb"),
		barBg:     lipgloss.Color("#14142b"),
		modeFg:    lipgloss.Color("#0a0a1a"),
		surface:   lipgloss.Color("#242444"),
		onSurface: lipgloss.Color("#c7c7e6"),
	},
}

// themeAliases map legacy / convenience names onto a real palette.
var themeAliases = map[string]string{
	"":      "auto",
	"dark":  "auto",
	"light": "auto",
}

var asciiBorder = lipgloss.Border{
	Top: "-", Bottom: "-", Left: "|", Right: "|",
	TopLeft: "+", TopRight: "+", BottomLeft: "+", BottomRight: "+",
}

var stylesets = map[string]styleset{
	"rounded": {
		border:    lipgloss.RoundedBorder(),
		glyphNav:  "\U000f0208 ", // nf-md-eye
		glyphLock: " ",          // nf-fa-lock
		bullet:    "●",           // ●
		sep:       "  ·  ",       // ·
		ellipsis:  "…",           // …
	},
	"double": {
		border:    lipgloss.DoubleBorder(),
		glyphNav:  "◈ ", // ◈
		glyphLock: "◆ ", // ◆
		bullet:    "◆",  // ◆
		sep:       "  ·  ",
		ellipsis:  "…",
	},
	"heavy": {
		border:    lipgloss.ThickBorder(),
		glyphNav:  "● ", // ●
		glyphLock: "■ ", // ■
		bullet:    "■",  // ■
		sep:       "  ·  ",
		ellipsis:  "…",
	},
	"ascii": {
		border:    asciiBorder,
		glyphNav:  "> ",
		glyphLock: "# ",
		bullet:    "*",
		sep:       "  |  ",
		ellipsis:  "...",
	},
}

var styleAliases = map[string]string{
	"":       "rounded",
	"normal": "rounded",
	"thick":  "heavy",
	"bold":   "heavy",
}

// resolveThemeName / resolveStyleName fold aliases and fall back to the default
// for an unknown name (main.go rejects unknown names up front; the engine stays
// defensive so a hand-set Model never panics).
func resolveThemeName(name string) string {
	if a, ok := themeAliases[name]; ok {
		name = a
	}
	if _, ok := palettes[name]; !ok {
		return defaultThemeName
	}
	return name
}

func resolveStyleName(name string) string {
	if a, ok := styleAliases[name]; ok {
		name = a
	}
	if _, ok := stylesets[name]; !ok {
		return defaultStyleName
	}
	return name
}

// ValidThemeName / ValidStyleName report whether a user-supplied name is
// accepted (including "" and aliases). main.go uses these to fail loudly on a
// typo, consistent with the strict-JSON config decision.
func ValidThemeName(name string) bool {
	if _, ok := themeAliases[name]; ok {
		return true
	}
	_, ok := palettes[name]
	return ok
}

func ValidStyleName(name string) bool {
	if _, ok := styleAliases[name]; ok {
		return true
	}
	_, ok := stylesets[name]
	return ok
}

// ThemeNames / StyleNames list the canonical (non-alias) names, sorted, with
// the default first. Used by `ultui config` and the F3 settings menu.
func ThemeNames() []string { return namesWithDefaultFirst(palettes, defaultThemeName) }
func StyleNames() []string { return namesWithDefaultFirst(stylesets, defaultStyleName) }

func namesWithDefaultFirst[V any](m map[string]V, first string) []string {
	rest := make([]string, 0, len(m))
	for k := range m {
		if k != first {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append([]string{first}, rest...)
}

// theme resolves + memoizes the styles for the Model's current theme/style
// names. invalidateTheme() drops the memo so the F3 menu's live preview redraws.
func (m *Model) theme() theme {
	if m.themeMemo == nil {
		t := buildTheme(m.themeName, m.styleName)
		m.themeMemo = &t
	}
	return *m.themeMemo
}

func (m *Model) invalidateTheme() { m.themeMemo = nil }

func buildTheme(themeName, styleName string) theme {
	pal := palettes[resolveThemeName(themeName)]
	ss := stylesets[resolveStyleName(styleName)]

	t := theme{
		borderIdle:  pal.subtle,
		borderFocus: pal.accent,
		borderLock:  pal.warn,

		border:    ss.border,
		glyphNav:  ss.glyphNav,
		glyphLock: ss.glyphLock,
		bullet:    ss.bullet,
		sep:       ss.sep,
		ellipsis:  ss.ellipsis,
	}

	t.title = lipgloss.NewStyle().Bold(true).Foreground(pal.subtle)
	t.titleFocus = lipgloss.NewStyle().Bold(true).Foreground(pal.accent)

	t.bar = lipgloss.NewStyle().Background(pal.barBg).Foreground(pal.text)
	t.barMode = lipgloss.NewStyle().Background(pal.accent).Foreground(pal.modeFg).Bold(true)

	t.helpBox = lipgloss.NewStyle().Border(ss.border).BorderForeground(pal.accent).Padding(1, 3)
	t.helpTitle = lipgloss.NewStyle().Bold(true).Foreground(pal.accent)
	t.helpSection = lipgloss.NewStyle().Bold(true).Foreground(pal.accent2)
	t.helpKey = lipgloss.NewStyle().Bold(true).Foreground(pal.text)
	t.helpDesc = lipgloss.NewStyle().Foreground(pal.text)
	t.helpDim = lipgloss.NewStyle().Foreground(pal.subtle).Italic(true)

	t.menuColHdr = lipgloss.NewStyle().Bold(true).Foreground(pal.accent2)
	t.menuItem = lipgloss.NewStyle().Foreground(pal.text)
	t.menuSel = lipgloss.NewStyle().Bold(true).Foreground(pal.modeFg).Background(pal.accent)

	t.chipPrimary = lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(pal.modeFg).Background(pal.accent)
	t.chipMuted = lipgloss.NewStyle().Padding(0, 1).Foreground(pal.onSurface).Background(pal.surface)

	t.widgetStyles = widget.Styles{
		Accent:  pal.accent,
		Accent2: pal.accent2,
		Text:    pal.text,
		Subtle:  pal.subtle,
		Warn:    pal.warn,
		Success: pal.success,
		ModeFg:  pal.modeFg,
	}

	return t
}
