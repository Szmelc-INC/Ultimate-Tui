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
