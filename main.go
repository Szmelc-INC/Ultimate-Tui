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
	// Let the editor overlay (F2 → Edit config, or a pane's context menu) write
	// the edited config back to the file it was loaded from.
	cfgPath := args[0]
	opts.SaveConfig = func(c *config.Config) error { return config.Save(cfgPath, c) }
	opts.ConfigPath = cfgPath
	opts.GlobalConfigPath = config.GlobalPath()
	if err := engine.RunWithOptions(cfg, opts); err != nil {
		fmt.Fprintln(os.Stderr, "ultui:", err)
		return 1
	}
	return 0
}

// runnerOptions loads the global runner config and folds it over the engine
// defaults. A missing file is fine; a malformed one — or an unknown theme /
// style name — is an error, consistent with the strict-JSON config rule.
func runnerOptions() (engine.Options, error) {
	gc, err := config.LoadGlobal()
	if err != nil {
		return engine.Options{}, err
	}
	opts := engine.OptionsFromGlobal(gc)
	if !engine.ValidThemeName(opts.Theme) {
		return engine.Options{}, fmt.Errorf("unknown theme %q (known: %v)", opts.Theme, engine.ThemeNames())
	}
	if !engine.ValidStyleName(opts.Style) {
		return engine.Options{}, fmt.Errorf("unknown style %q (known: %v)", opts.Style, engine.StyleNames())
	}
	// Let the theme/style picker (opened from the F2 control panel or F3)
	// persist a choice back to the global config, and the F3 panel persist
	// mod/statusbar (and whatever it saw for theme/style/keybindings).
	opts.SaveTheme = config.SaveGlobalThemeStyle
	opts.SaveGlobal = config.SaveGlobal
	return opts, nil
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
	fmt.Printf("effective: mod=%q statusbar=%v theme=%q style=%q\n",
		opts.Mod, opts.StatusBar, orDefault(opts.Theme, "auto"), orDefault(opts.Style, "rounded"))
	fmt.Println("themes:", strings.Join(engine.ThemeNames(), " "))
	fmt.Println("styles:", strings.Join(engine.StyleNames(), " "))
	if !engine.ValidThemeName(opts.Theme) {
		fmt.Fprintf(os.Stderr, "warning: theme %q is not a known name\n", opts.Theme)
	}
	if !engine.ValidStyleName(opts.Style) {
		fmt.Fprintf(os.Stderr, "warning: style %q is not a known name\n", opts.Style)
	}
	return 0
}

func orDefault(s, def string) string {
	if s == "" {
		return def
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
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ultui render <config.json> [WIDTHxHEIGHT] [theme=NAME] [style=NAME]")
		return 2
	}
	w, h := 100, 30
	var themeOv, styleOv string
	for _, a := range args[1:] {
		switch {
		case strings.HasPrefix(a, "theme="):
			themeOv = strings.TrimPrefix(a, "theme=")
		case strings.HasPrefix(a, "style="):
			styleOv = strings.TrimPrefix(a, "style=")
		default:
			if x, y, ok := parseWH(a); ok {
				w, h = x, y
			} else {
				fmt.Fprintf(os.Stderr, "ultui: bad arg %q (want WxH, theme=NAME or style=NAME)\n", a)
				return 2
			}
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
	// Optional per-invocation overrides — handy for eyeballing a theme/style
	// headlessly without editing the global config.
	if themeOv != "" {
		if !engine.ValidThemeName(themeOv) {
			fmt.Fprintf(os.Stderr, "ultui: unknown theme %q (known: %v)\n", themeOv, engine.ThemeNames())
			return 2
		}
		opts.Theme = themeOv
	}
	if styleOv != "" {
		if !engine.ValidStyleName(styleOv) {
			fmt.Fprintf(os.Stderr, "ultui: unknown style %q (known: %v)\n", styleOv, engine.StyleNames())
			return 2
		}
		opts.Style = styleOv
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
  ultui render <config.json> [WxH] [theme=N] [style=N]   print one headless frame
  ultui config                         show the global runner config + effective settings
  ultui version
  ultui help

global config: %s
widget types:  %v
`, version, config.GlobalPath(), widget.Kinds())
}
