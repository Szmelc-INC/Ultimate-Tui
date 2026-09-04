// This is the "compiled / static" build mode from DESIGN.md: the config is
// baked into the binary with go:embed and the engine is linked in. The result
// is a single self-contained executable — no config file needed at runtime.
//
//	go build -o myapp ./examples/compiled
//	./myapp
//
// "Dynamic" linking is just this file reading the config from a path/flag
// instead of embedding it. Same engine either way — there is no plugin system.
package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
)

//go:embed app.json
var appConfig []byte

func main() {
	cfg, err := config.Parse(appConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "embedded config invalid:", err)
		os.Exit(1)
	}
	if err := engine.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
