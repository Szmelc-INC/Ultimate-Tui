package engine_test

import (
	"strings"
	"testing"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

func TestSnapshotRunsCommandWidgets(t *testing.T) {
	cfg, err := config.Parse([]byte(`{
      "name": "snap",
      "layout": {
        "direction": "row",
        "children": [
          { "size": "50%", "pane": { "type": "text",    "title": "note", "content": "STATIC" } },
          { "size": "*",    "pane": { "type": "command", "title": "cmd",  "command": "printf SNAPOUT" } }
        ]
      }
    }`))
	if err != nil {
		t.Fatal(err)
	}

	frame, err := engine.Snapshot(cfg, 50, 10)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(frame, "STATIC") {
		t.Errorf("text widget body missing:\n%s", frame)
	}
	if !strings.Contains(frame, "SNAPOUT") {
		t.Errorf("command output not drained into snapshot:\n%s", frame)
	}
	if got := len(strings.Split(frame, "\n")); got != 10 {
		t.Errorf("snapshot height = %d, want 10", got)
	}
}
