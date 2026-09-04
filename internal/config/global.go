package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GlobalConfig is the runner-wide config that applies to every TUI regardless of
// which config file is loaded. It lives at $XDG_CONFIG_HOME/ultui/config.json
// (falling back to ~/.config/ultui/config.json). Every field is optional; a
// missing file is not an error.
type GlobalConfig struct {
	// Mod is the modifier a keybinding's "mod+" prefix expands to. Default
	// "alt". Terminals cannot distinguish left from right Alt.
	Mod string `json:"mod,omitempty"`
	// StatusBar toggles the bottom mode/hint bar. nil == true.
	StatusBar *bool `json:"statusbar,omitempty"`
	// Theme picks the colour palette: "auto" (default), "tokyo-night",
	// "gruvbox", "slate", "midnight" (or "" / "dark" / "light" = auto).
	Theme string `json:"theme,omitempty"`
	// Style picks the styleset (border + glyphs): "rounded" (default),
	// "double", "heavy", "ascii".
	Style string `json:"style,omitempty"`
	// Keybindings are global bindings merged under each TUI's own keybindings.
	// Keys may use the "mod+" prefix.
	Keybindings map[string]string `json:"keybindings,omitempty"`
}

// GlobalPath is the resolved location of the global runner config.
func GlobalPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "ultui", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ultui", "config.json")
}

// LoadGlobal reads the global runner config. A missing file returns (nil, nil):
// callers fall back to engine defaults. A present-but-malformed file is an
// error so typos surface.
func LoadGlobal() (*GlobalConfig, error) {
	path := GlobalPath()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseGlobal(data)
}

// SaveGlobalThemeStyle writes theme and style into the global runner config,
// leaving every other field (mod, statusbar, keybindings) untouched. It loads
// the current file, mutates only those two keys, and swaps the file in via a
// temp file + rename so a crash mid-write cannot truncate the config. The
// directory is created if the file does not exist yet. An empty value clears
// that key.
func SaveGlobalThemeStyle(theme, style string) error {
	path := GlobalPath()
	if path == "" {
		return fmt.Errorf("global config: no resolvable path")
	}
	g, err := LoadGlobal()
	if err != nil {
		return err
	}
	if g == nil {
		g = &GlobalConfig{}
	}
	g.Theme, g.Style = theme, style

	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SaveGlobal writes gc to the global runner config path via a temp file +
// rename, creating the directory if needed. Unlike SaveGlobalThemeStyle
// (which load-mutates only theme/style, preserving everything else on disk),
// this replaces the whole file with gc as given — the F3 global-settings
// panel edits every field it cares about into gc itself before calling this,
// carrying forward anything it doesn't edit (e.g. Keybindings) from what it
// was constructed with.
func SaveGlobal(gc *GlobalConfig) error {
	path := GlobalPath()
	if path == "" {
		return fmt.Errorf("global config: no resolvable path")
	}
	data, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ParseGlobal decodes global-config bytes (strict JSON — unknown keys rejected).
func ParseGlobal(data []byte) (*GlobalConfig, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var g GlobalConfig
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("global config: %w", err)
	}
	return &g, nil
}
