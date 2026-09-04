package config_test

import (
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestParseGlobalValid(t *testing.T) {
	g, err := config.ParseGlobal([]byte(`{"mod":"ctrl","statusbar":false,"theme":"dark","keybindings":{"mod+/":"help"}}`))
	if err != nil {
		t.Fatalf("ParseGlobal: %v", err)
	}
	if g.Mod != "ctrl" || g.Theme != "dark" {
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
	if g.Mod != "" || g.StatusBar != nil || g.Theme != "" {
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
