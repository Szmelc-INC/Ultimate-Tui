package engine

import "github.com/silverx/ultimate-tui/internal/config"

// Options are runner-wide settings that apply to every TUI, independent of the
// individual config file. They come from the global runner config
// (~/.config/ultui/config.json); the engine itself never reads that file so
// tests stay hermetic — main.go loads it and passes the result here.
type Options struct {
	// Mod is the modifier token that a keybinding's "mod+" prefix expands to.
	// Default "alt" (bubbletea cannot tell left Alt from right Alt — any Alt
	// press counts).
	Mod string
	// StatusBar draws a one-line mode/hint bar along the bottom edge. Skipped
	// automatically when the terminal is shorter than 6 rows.
	StatusBar bool
	// Theme selects the colour palette: "auto" (default), "tokyo-night",
	// "gruvbox", "slate", "midnight" — or "" / "dark" / "light" for auto.
	Theme string
	// Style selects the styleset (border + glyphs): "rounded" (default),
	// "double", "heavy", "ascii".
	Style string
	// Keys are extra global bindings (already "mod+"-expanded or literal),
	// layered under the TUI config's own keybindings.
	Keys map[string]string
	// SaveTheme, when non-nil, persists a theme/style pair chosen in the
	// theme/style picker (opened from the F2 control panel). The engine never
	// writes to disk itself — main.go wires this to the global runner config
	// so tests stay hermetic.
	SaveTheme func(themeName, styleName string) error
	// SaveConfig, when non-nil, persists the whole (edited) TUI config from the
	// editor overlay's save key. nil in DefaultOptions and tests; main.go wires
	// it to config.Save for `run`.
	SaveConfig func(*config.Config) error
	// SaveGlobal, when non-nil, persists the F3 global-settings panel's
	// buffered mod/statusbar/theme/style/keybindings. nil in DefaultOptions
	// and tests; main.go wires it to config.SaveGlobal.
	SaveGlobal func(*config.GlobalConfig) error
	// ConfigPath and GlobalConfigPath are purely informational — shown in the
	// F3 panel's "this run" block. Engine never reads either path itself;
	// main.go resolves them (the TUI config it loaded, config.GlobalPath())
	// and hands over the strings, the same boundary SaveConfig/SaveTheme/
	// SaveGlobal already draw. Both are "" for go:embed-static/Snapshot runs
	// that never had a path to begin with.
	ConfigPath       string
	GlobalConfigPath string
}

// DefaultOptions is the baseline used by New/Run/Snapshot when no global config
// is supplied.
func DefaultOptions() Options {
	return Options{Mod: "alt", StatusBar: true}
}

// OptionsFromGlobal folds a parsed global config over the defaults. A nil gc
// yields DefaultOptions.
func OptionsFromGlobal(gc *config.GlobalConfig) Options {
	o := DefaultOptions()
	if gc == nil {
		return o
	}
	if gc.Mod != "" {
		o.Mod = gc.Mod
	}
	if gc.StatusBar != nil {
		o.StatusBar = *gc.StatusBar
	}
	if gc.Theme != "" {
		o.Theme = gc.Theme
	}
	if gc.Style != "" {
		o.Style = gc.Style
	}
	if len(gc.Keybindings) > 0 {
		o.Keys = map[string]string{}
		for k, v := range gc.Keybindings {
			o.Keys[k] = v
		}
	}
	return o
}
