# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

MVP built. Go module `github.com/silverx/ultimate-tui` (Go 1.26). A config renders end to end: recursive row/column splits, `text` and `command` widgets, focus ring, borders, keybindings, three run modes (runtime / headless / go:embed-static). `DESIGN.md` is the vision; `README.md` and `LLM.md` document what actually exists; `optimal-mode.md` is a token-budget ruleset.

## Commands

```sh
make build                              # -> bin/ultui
make test                               # full suite — headless, no TTY required
make test-one T=TestFocusTraversalWraps # single test (go test ./... -run '<T>' -v)
make vet ; make fmt
./bin/ultui run examples/dashboard.json      # live TUI (needs a TTY)
./bin/ultui render examples/hello.json 100x30 # print ONE frame to stdout — how to verify rendering headless
./bin/ultui validate examples/dashboard.json
make compiled                           # build the go:embed static demo (examples/compiled/)
```

**This is a background job with no TTY** — never verify by launching `run`. Assert on `engine.Model.View()` / `engine.Snapshot(cfg, w, h)` in `_test.go`, or eyeball `ultui render`.

## Code map

| path | role |
|------|------|
| `internal/config` | schema structs; `Parse([]byte)` / `Load(path)`. Strict JSON. **The only format-aware code** — a second format (YAML/TOML) is a decoder swap here and nowhere else. |
| `internal/widget` | `Widget` interface + `Register`/`New` kind registry (self-register in `init()`). `text.go`, `command.go`, shared `scrollBox` in `scroll.go`. |
| `internal/engine` | `parseSize`/`distribute` layout math (`layout.go`), bubbletea root `Model` (`engine.go`), `Run` TTY entry (`run.go`), headless `Snapshot` (`snapshot.go`). |
| `main.go` | `run` / `validate` / `render` CLI. |
| `examples/` | `hello`, `dashboard`, `responsive`, `keys`, `sysmon`, `pretty`, `devdash` + `compiled/` (go:embed wrapper); `examples/README.md` maps each to what it demonstrates and a render size. |

Import direction: `engine` → `widget` → `config` → (nothing). Keep it acyclic.

## Architecture decisions (do not undo)

- **Widgets are a compile-time registry, not plugins.** `widget.Register(kind, factory)` + config selecting a kind by name. No `plugin.Open`, no `dlopen`, no subprocess-as-module. This is what makes "runtime, dynamic-linked, and static-linked" the *same engine* — runtime reads config from disk, static `go:embed`s it, dynamic reads a path; all call `engine.Run`. "Total modularity" from `DESIGN.md` will tempt a rewrite toward plugins — resist; add a widget file instead.
- **Config is strict JSON.** `DESIGN.md`'s "markdown json" was ambiguous; resolved to one strict decoder (`DisallowUnknownFields`, so typos error). Nodes are split-XOR-leaf by key presence, not a discriminator field.
- **`command` widget ≠ interactive terminal.** It runs a command once and streams combined output into a scroll pane. Interactive process panes (pty + ANSI cell-grid emulator + key forwarding) are a separate, unbuilt subsystem — `DESIGN.md`'s "like a terminal multiplexer" line. Don't let the schema imply the pane hosts `vim`.
- **lipgloss sizing in `renderLeaf`:** `Width/Height` set the content area, a border adds 2 outside. Never put `MaxWidth/MaxHeight` on a bordered style — it clips the final output including the border and shears an edge.

When you create auxiliary tooling `DESIGN.md` asks for (setup scripts etc.), keep the `Makefile` the single source of dev commands and update the block above.

## What this project is

A config-driven TUI/CLI framework in **Go**. A "TUI" is authored entirely as a config file (markdown/JSON, human-readable, splittable across files or consolidated into one file with sub-config sections). The framework turns that config into a running interface.

Three architectural pillars from `DESIGN.md`:

1. **Config-driven** — every TUI *is* its config: layout, behavior, keybindings, embedded scripts. No recompilation to change a TUI.
2. **Runtime and compiled execution** — a TUI runs by pointing the engine at a config path, *or* it compiles to an executable. Compilation has two linking modes:
   - *dynamic* — core engine + config kept separate
   - *static* — engine baked together with a specific config plus the modules/dependencies that config requires
3. **Multiplexing** — split, resize, position and manage screen space like a pre-configured tmux; run interactive shell commands and long-lived processes inside panes.

Cross-cutting requirements: full keyboard + mouse interactivity; unicode/ascii/ansi and Nerd Font support; first-class shell (zsh) scripting and integration, with python/node as secondary; animations and localized non-static rendering; and **total modularity** — anything that could reasonably be modular, swappable or config-driven should be.

## Recommended stack

`DESIGN.md` recommends (does not mandate) the Charm ecosystem — integrate all, some, or none: `bubbletea` (runtime/event loop), `lipgloss` (styling/layout), `bubbles` (widgets), `huh` (forms), `glamour`/`glow` (markdown), `gum`, `vhs`. Reference points for behavior: tmux, midnight commander, lazygit/lazyvim, archinstall, and CLI agent harnesses.

## Documentation deliverables

The project expects two parallel docs, kept in sync with the code:
- `README.md` — for humans: clear, pretty markdown, informative without being verbose.
- `LLM.md` — for AI: optimized for LLM consumption.

## Working constraints

- **Runs on a live production Arch Linux system with no sandbox.** Be careful with destructive operations.
- **Use git.** Work on a branch; never commit straight to `main` or merge without asking.
- **Minimalism over frameworks.** Build exactly what's required with zero overhead. Don't pull a heavy dependency for one small task — write the snippet or cherry-pick it. Continuously question whether a solution is over-complex.
- **Design authority.** Follow `DESIGN.md`'s intent closely *where it makes sense*. If an instruction is impossible, wrong, or wasteful, do it the better way while still delivering at least what was asked. If something clearly should exist, build it even if unmentioned.
- **Clarify only high-stakes unknowns.** For low-stakes ambiguity, make a best-effort call and proceed.

## Token discipline

`optimal-mode.md` is an opt-in "eco mode" ruleset: pinpoint context with `rg`/`fd`/`sed` before reading; never `cat` a >250-line file just for context; keep reasoning effort low unless the task genuinely needs more; terminal-style output with zero filler; take the explicit `[Bypass: requires deep context]` escape hatch rather than looping on retries.
