package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestParseGlobalValid(t *testing.T) {
	g, err := config.ParseGlobal([]byte(`{"mod":"ctrl","statusbar":false,"theme":"gruvbox","style":"double","keybindings":{"mod+/":"help"}}`))
	if err != nil {
		t.Fatalf("ParseGlobal: %v", err)
	}
	if g.Mod != "ctrl" || g.Theme != "gruvbox" || g.Style != "double" {
		t.Fatalf("fields = %+v", g)
	}
	if g.StatusBar == nil || *g.StatusBar {
		t.Fatalf("statusbar: want non-nil false, got %v", g.StatusBar)
	}
	if g.Keybindings["mod+/"] != "help" {
		t.Fatalf("keybindings not parsed: %v", g.Keybindings)
	}
}

func TestParseGlobalEmptyObject(t *testing.T) {
	g, err := config.ParseGlobal([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if g.Mod != "" || g.StatusBar != nil || g.Theme != "" || g.Style != "" {
		t.Fatalf("zero value expected, got %+v", g)
	}
}

func TestParseGlobalRejectsUnknownKey(t *testing.T) {
	if _, err := config.ParseGlobal([]byte(`{"statusbar":true,"nope":1}`)); err == nil {
		t.Fatal("unknown key should be rejected (DisallowUnknownFields)")
	}
}

func TestParseGlobalRejectsBadJSON(t *testing.T) {
	if _, err := config.ParseGlobal([]byte(`{"mod":`)); err == nil {
		t.Fatal("malformed JSON should error")
	}
}

func TestSaveGlobalThemeStylePreservesOtherFields(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Seed a config that carries fields the F3 save must not clobber.
	seed := `{"mod":"ctrl","statusbar":false,"keybindings":{"mod+n":"focus-next"}}`
	if err := os.MkdirAll(filepath.Join(dir, "ultui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ultui", "config.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := config.SaveGlobalThemeStyle("gruvbox", "double"); err != nil {
		t.Fatalf("SaveGlobalThemeStyle: %v", err)
	}

	g, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if g.Theme != "gruvbox" || g.Style != "double" {
		t.Fatalf("theme/style not written: %+v", g)
	}
	if g.Mod != "ctrl" || g.StatusBar == nil || *g.StatusBar {
		t.Fatalf("mod/statusbar clobbered: %+v", g)
	}
	if g.Keybindings["mod+n"] != "focus-next" {
		t.Fatalf("keybindings clobbered: %v", g.Keybindings)
	}
}

func TestSaveGlobalThemeStyleCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := config.SaveGlobalThemeStyle("midnight", "ascii"); err != nil {
		t.Fatalf("SaveGlobalThemeStyle on missing file: %v", err)
	}
	g, err := config.LoadGlobal()
	if err != nil || g == nil {
		t.Fatalf("reload after create: g=%v err=%v", g, err)
	}
	if g.Theme != "midnight" || g.Style != "ascii" {
		t.Fatalf("fields = %+v", g)
	}
}

func TestLoadGlobalMissingFileIsNilNil(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // guaranteed to hold no ultui/config.json
	g, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if g != nil {
		t.Fatalf("missing file should yield nil, got %+v", g)
	}
}
