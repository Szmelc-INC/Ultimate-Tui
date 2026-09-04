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
	// Theme picks the palette: "dark", "light", or "" (auto).
	Theme string `json:"theme,omitempty"`
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
