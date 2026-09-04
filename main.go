// Command ultui is the runtime for the ULTIMATE TUI framework: point it at a
// config file and it renders that TUI.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/silverx/ultimate-tui/internal/config"
	"github.com/silverx/ultimate-tui/internal/engine"
	"github.com/silverx/ultimate-tui/internal/widget"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "validate":
		os.Exit(cmdValidate(os.Args[2:]))
	case "render":
		os.Exit(cmdRender(os.Args[2:]))
	case "config":
		os.Exit(cmdConfig(os.Args[2:]))
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "ultui: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func cmdRun(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ultui run <config.json>")
		return 2
	}
	cfg, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	opts, err := runnerOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	if err := engine.RunWithOptions(cfg, opts); err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	return 0
}

// runnerOptions loads the global runner config and folds it over the engine
// defaults. A missing file is fine; a malformed one is an error.
func runnerOptions() (engine.Options, error) {
	gc, err := config.LoadGlobal()
	if err != nil {
		return engine.Options{}, err
	}
	return engine.OptionsFromGlobal(gc), nil
}

func cmdConfig(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: ultui config")
		return 2
	}
	path := config.GlobalPath()
	fmt.Println("global config:", path)
	gc, err := config.LoadGlobal()
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid:", err)
		return 1
	}
	if gc == nil {
		fmt.Println("status: not present — using built-in defaults")
	} else {
		fmt.Println("status: loaded")
	}
	opts := engine.OptionsFromGlobal(gc)
	fmt.Printf("effective: mod=%q statusbar=%v theme=%q\n", opts.Mod, opts.StatusBar, orAuto(opts.Theme))
	return 0
}

func orAuto(s string) string {
	if s == "" {
		return "auto"
	}
	return s
}

func cmdValidate(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ultui validate <config.json>")
		return 2
	}
	cfg, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid:", err)
		return 1
	}
	// Building the model catches widget-level errors that pure schema
	// validation cannot (unknown type, missing command, ...).
	m, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid:", err)
		return 1
	}
	fmt.Printf("ok: %q — %d pane(s)\n", cfg.Name, m.PaneCount())
	return 0
}

func cmdRender(args []string) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(os.Stderr, "usage: ultui render <config.json> [WIDTHxHEIGHT]")
		return 2
	}
	w, h := 100, 30
	if len(args) == 2 {
		if x, y, ok := parseWH(args[1]); ok {
			w, h = x, y
		} else {
			fmt.Fprintf(os.Stderr, "ultui: bad size %q (want e.g. 120x40)\n", args[1])
			return 2
		}
	}
	cfg, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	opts, err := runnerOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	frame, err := engine.SnapshotWithOptions(cfg, w, h, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	fmt.Println(frame)
	return 0
}

func parseWH(s string) (int, int, bool) {
	a, b, ok := strings.Cut(strings.ToLower(s), "x")
	if !ok {
		return 0, 0, false
	}
	w, err1 := strconv.Atoi(a)
	h, err2 := strconv.Atoi(b)
	if err1 != nil || err2 != nil || w < 1 || h < 1 {
		return 0, 0, false
	}
	return w, h, true
}

func usage() {
	fmt.Fprintf(os.Stderr, `ultui %s — config-driven TUI runtime

usage:
  ultui run <config.json>              render and run a TUI
  ultui validate <config.json>         parse and type-check a config
  ultui render <config.json> [WxH]     print one headless frame to stdout
  ultui config                         show the global runner config + effective settings
  ultui version
  ultui help

global config: %s
widget types:  %v
`, version, config.GlobalPath(), widget.Kinds())
}
