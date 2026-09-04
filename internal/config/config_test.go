package config_test

import (
	"strings"
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
)

func TestParseValidConfig(t *testing.T) {
	c, err := config.Parse([]byte(`{
      "name": "demo",
      "version": "1.0",
      "keybindings": { "r": "reload" },
      "layout": {
        "direction": "column",
        "children": [
          { "size": "3",  "pane": { "type": "text", "content": "header" } },
          { "size": "*",  "pane": { "type": "command", "command": "ls" } }
        ]
      }
    }`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "demo" || len(c.Leaves()) != 2 {
		t.Fatalf("bad parse: %+v", c)
	}
}

func TestRejectsUnknownKey(t *testing.T) {
	_, err := config.Parse([]byte(`{"name":"x","layuot":{}}`))
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("want parse error, got %v", err)
	}
}

func TestRejectsNodeWithBothForms(t *testing.T) {
	_, err := config.Parse([]byte(`{
      "name": "x",
      "layout": {
        "pane": { "type": "text" },
        "children": [ { "pane": { "type": "text" } } ]
      }
    }`))
	if err == nil || !strings.Contains(err.Error(), "pick one") {
		t.Fatalf("want both-forms error, got %v", err)
	}
}

func TestRejectsEmptyNode(t *testing.T) {
	_, err := config.Parse([]byte(`{"name":"x","layout":{}}`))
	if err == nil || !strings.Contains(err.Error(), "neither") {
		t.Fatalf("want empty-node error, got %v", err)
	}
}

func TestRejectsPercentOverflow(t *testing.T) {
	_, err := config.Parse([]byte(`{
      "name": "x",
      "layout": { "direction": "row", "children": [
        { "size": "60%", "pane": { "type": "text" } },
        { "size": "60%", "pane": { "type": "text" } }
      ]}
    }`))
	if err == nil || !strings.Contains(err.Error(), "100") {
		t.Fatalf("want percent-overflow error, got %v", err)
	}
}

func TestAllowsPercentExactly100(t *testing.T) {
	_, err := config.Parse([]byte(`{
      "name": "x",
      "layout": { "direction": "row", "children": [
        { "size": "50%", "pane": { "type": "text" } },
        { "size": "50%", "pane": { "type": "text" } }
      ]}
    }`))
	if err != nil {
		t.Fatalf("100%% total should be valid: %v", err)
	}
}

func TestRejectsMissingName(t *testing.T) {
	_, err := config.Parse([]byte(`{"layout":{"pane":{"type":"text"}}}`))
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("want name error, got %v", err)
	}
}
