package config_test

import (
	"path/filepath"
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestSaveRoundTrips(t *testing.T) {
	src := `{"name":"rt","layout":{"direction":"row","children":[
	  {"size":"30%","pane":{"type":"text","title":"A","content":"x","border":false}},
	  {"size":"*","pane":{"type":"command","command":"echo hi"}}
	]}}`
	c, err := config.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "c.json")
	if err := config.Save(p, c); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c2, err := config.Load(p)
	if err != nil {
		t.Fatalf("reload after Save: %v", err)
	}
	if c2.Name != "rt" {
		t.Fatalf("name = %q", c2.Name)
	}
	leaves := c2.Leaves()
	if len(leaves) != 2 || leaves[0].Title != "A" || leaves[1].Command != "echo hi" {
		t.Fatalf("round-trip lost pane data: %+v", leaves)
	}
	if leaves[0].Border == nil || *leaves[0].Border != false {
		t.Fatalf("round-trip lost border=false")
	}
	if c2.Layout.Children[0].Size != "30%" {
		t.Fatalf("round-trip lost size: %q", c2.Layout.Children[0].Size)
	}
}

func TestSaveRejectsInvalid(t *testing.T) {
	c := &config.Config{Layout: config.Node{Pane: &config.Pane{Type: "text"}}} // no name
	if err := config.Save(filepath.Join(t.TempDir(), "c.json"), c); err == nil {
		t.Fatal("Save accepted a config with an empty name")
	}
}
