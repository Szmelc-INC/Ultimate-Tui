package engine

import (
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/silverx/ultimate-tui/internal/config"
)

// The F3 overlay is the runner-wide global settings + diagnostics panel —
// distinct from F2 (this TUI's own config, via the pane editor) and the
// theme/style picker (reachable from either F2 or here). It edits the
// Options-level settings that apply to every TUI regardless of which config
// file is loaded (the modifier key, the status bar toggle), and shows a
// read-only "this run" block: config paths, the resolved binary, pane count,
// mode, permissions on the loaded config file — whatever helps answer "what
// is this program doing and where does its state live". F3 is deliberately
// NOT in isGlobalKey (engine.go): it only opens from NAV, so a locked
// shell/vim keeps its own F3, matching F1/F2's original design intent before
// F3 existed.

const (
	glRowMod = iota
	glRowStatusBar
	glRowTheme
	glRowCount
)

var glModChoices = []string{"alt", "ctrl"}

func (m *Model) openGlobalPanel() {
	m.overlay = ovGlobal
	m.glMod = m.mod
	m.glStatusBar = m.opts.StatusBar
	m.glRow = 0
	m.glDirty, m.glNote = false, ""
}

func (m *Model) closeGlobalPanel() {
	m.overlay = ovNone
	m.glDirty, m.glNote = false, ""
}

func (m *Model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.glNote = ""
	switch msg.String() {
	case "esc", "q", "f3":
		m.closeGlobalPanel()
	case "up", "k":
		m.glRow = (m.glRow - 1 + glRowCount) % glRowCount
	case "down", "j":
		m.glRow = (m.glRow + 1) % glRowCount
	case "left", "h":
		m.globalCycle(-1)
	case "right", "l":
		m.globalCycle(1)
	case " ":
		if m.glRow == glRowStatusBar {
			m.glStatusBar = !m.glStatusBar
			m.glDirty = true
		}
	case "enter":
		if m.glRow == glRowTheme {
			m.overlayReturn = ovGlobal
			m.openSettings()
		} else {
			m.globalCycle(1)
		}
	case m.mod + "+s", "ctrl+s":
		return m, m.applyGlobal()
	}
	return m, nil
}

func (m *Model) globalCycle(dir int) {
	switch m.glRow {
	case glRowMod:
		i := (indexOf(glModChoices, m.glMod) + dir + len(glModChoices)) % len(glModChoices)
		m.glMod = glModChoices[i]
		m.glDirty = true
	case glRowStatusBar:
		m.glStatusBar = !m.glStatusBar
		m.glDirty = true
	}
}

// globalMouseClick maps a click's content-relative row to one of the three
// fixed editable rows (see globalPanel: title, blank, mod(2), blank,
// statusbar(4), blank, theme(6)) — the read-only info block below them isn't
// interactive, so rows past glRowTheme are ignored.
func (m *Model) globalMouseClick(row int) (tea.Model, tea.Cmd) {
	switch row {
	case 2:
		m.glRow = glRowMod
	case 4:
		m.glRow = glRowStatusBar
		m.glStatusBar = !m.glStatusBar
		m.glDirty = true
	case 6:
		m.glRow = glRowTheme
		m.overlayReturn = ovGlobal
		m.openSettings()
	}
	return m, nil
}

// applyGlobal persists the buffered mod/statusbar, carrying the currently
// resolved theme/style and whatever global keybindings this run loaded
// (m.opts.Keys — see OptionsFromGlobal) forward unchanged, so saving here
// can never silently drop a hand-edited global keybinding. This does NOT
// take effect on the running program — "mod+" bindings are expanded once at
// NewWithOptions time — so it's saved for the *next* run; the panel says so.
func (m *Model) applyGlobal() tea.Cmd {
	if m.opts.SaveGlobal == nil {
		m.glNote = "no save target wired"
		return nil
	}
	sb := m.glStatusBar
	gc := &config.GlobalConfig{
		Mod:         m.glMod,
		StatusBar:   &sb,
		Theme:       m.themeName,
		Style:       m.styleName,
		Keybindings: m.opts.Keys,
	}
	if err := m.opts.SaveGlobal(gc); err != nil {
		m.glNote = "save failed: " + err.Error()
		return nil
	}
	m.glDirty = false
	m.glNote = "saved — takes effect next run"
	return nil
}

func (m *Model) renderGlobal(frame string) string {
	return m.splice(frame, m.globalPanel())
}

func (m *Model) globalPanel() string {
	th := m.theme()
	var b strings.Builder

	b.WriteString(th.helpTitle.Render("global settings — F3"))
	b.WriteString("\n\n")

	row := func(i int, label, value string) string {
		mark := "  "
		v := th.menuItem.Render(value)
		if i == m.glRow {
			mark = th.helpKey.Render("› ")
			v = th.helpKey.Render("‹ ") + value + th.helpKey.Render(" ›")
		}
		return mark + th.helpDesc.Render(padRight(label, 14)) + v
	}
	sbWord := "off"
	if m.glStatusBar {
		sbWord = "on"
	}
	b.WriteString(row(glRowMod, "modifier", m.glMod))
	b.WriteString("\n\n")
	b.WriteString(row(glRowStatusBar, "status bar", sbWord))
	b.WriteString("\n\n")

	themeMark := "  "
	if m.glRow == glRowTheme {
		themeMark = th.helpKey.Render("› ")
	}
	b.WriteString(themeMark + th.helpDesc.Render(padRight("theme & style", 14)) +
		th.menuItem.Render(m.themeName+" / "+m.styleName) + th.helpDim.Render("  (Enter to change)"))
	b.WriteString("\n\n")

	b.WriteString(th.helpSection.Render("this run"))
	b.WriteString("\n")
	info := func(k, v string) string {
		return th.helpKey.Render(padRight(k, 16)) + th.helpDesc.Render(v)
	}
	if m.opts.ConfigPath != "" {
		b.WriteString(info("config", m.opts.ConfigPath) + "\n")
		if fi, err := os.Stat(m.opts.ConfigPath); err == nil {
			b.WriteString(info("permissions", fi.Mode().String()) + "\n")
		}
	}
	if m.opts.GlobalConfigPath != "" {
		b.WriteString(info("global config", m.opts.GlobalConfigPath) + "\n")
	}
	if bin, err := os.Executable(); err == nil {
		b.WriteString(info("binary", bin) + "\n")
	}
	b.WriteString(info("panes", strconv.Itoa(len(m.leaves))) + "\n")
	b.WriteString(info("mode", m.Mode()) + "\n")
	b.WriteString(info("keybindings", strconv.Itoa(len(m.keys))+" resolved, "+strconv.Itoa(len(m.tuiKeys))+" from this TUI") + "\n")

	b.WriteString("\n")
	switch {
	case m.glNote != "":
		b.WriteString(th.helpKey.Render(m.glNote) + "\n")
	case m.glDirty:
		b.WriteString(th.helpDim.Render("● unsaved — "+m.mod+"+s to write to disk") + "\n")
	}
	b.WriteString(th.helpDim.Render("↑↓ field · ←→ change · Enter apply/open · " + m.mod + "+s save · Esc close"))
	return th.helpBox.Render(padBlockWidth(b.String(), edMinWidth))
}
