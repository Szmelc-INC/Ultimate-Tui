# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

MVP built. Go module `github.com/silverx/ultimate-tui` (Go 1.26). A config renders end to end: recursive row/column splits; `text`, `markdown`, `command`, and interactive `shell` (pty) widgets; focus ring; a **NAV/LOCKED** focus model (arrows = directional focus, Enter locks a pane, Esc releases, `Esc Esc` = literal Esc to a child); `mod+`-prefixed keybindings (`mod` = Alt by default); **Ctrl+C = confirm-quit in NAV / kill-this-pane's-process when LOCKED** (no bare quit); an F1 help popup; an **F3 live theme & style picker** (named colour palettes × border/glyph stylesets, persisted to the global config); a bottom status bar; a **global runner config** at `~/.config/ultui/config.json` (`ultui config` to inspect); mouse (click-to-focus, wheel-scroll); three run modes (runtime / headless / go:embed-static). `DESIGN.md` is the vision; `README.md` and `LLM.md` document what actually exists; `optimal-mode.md` is a token-budget ruleset.

## Commands

```sh
make build                              # -> bin/ultui
make test                               # full suite — headless, no TTY required
make test-one T=TestFocusTraversalWraps # single test (go test ./... -run '<T>' -v)
make vet ; make fmt
./bin/ultui run examples/dashboard.json      # live TUI (needs a TTY)
./bin/ultui render examples/hello.json 100x30 # print ONE frame to stdout — how to verify rendering headless
./bin/ultui render examples/pretty.json 120x40 theme=gruvbox style=double  # preview a theme/style combo headlessly
./bin/ultui validate examples/dashboard.json
./bin/ultui config                          # resolved global-config path + effective mod/statusbar/theme/style + known names
make compiled                           # build the go:embed static demo (examples/compiled/)
```

**This is a background job with no TTY** — never verify by launching `run`. Assert on `engine.Model.View()` / `engine.Snapshot(cfg, w, h)` in `_test.go`, or eyeball `ultui render`. `shell` panes ARE testable headlessly: a pty spawns without a TTY, so drive `widget.Init()`'s command loop by hand and assert on `View()` (see `internal/widget/shell_test.go`, `shell_htop_test.go`). Run `go test -race` — the shell widget uses goroutines.

## Code map

| path | role |
|------|------|
| `internal/config` | schema structs; `Parse([]byte)` / `Load(path)`. Strict JSON. **The only format-aware code** — a second format (YAML/TOML) is a decoder swap here and nowhere else. `global.go` = `GlobalConfig` + `LoadGlobal()` for the runner-wide config. |
| `internal/widget` | `Widget` interface + `Register`/`New` kind registry (self-register in `init()`). Kinds: `text`, `markdown` (glamour), `command`, `shell`. Shared `scrollBox` (bubbles/viewport-backed) in `scroll.go`; `keyToBytes` (xterm key encoding) in `keys.go`. Optional interfaces: `RawKeyConsumer`, `Closer`, `Interrupter` (kill one pane's process on locked Ctrl+C), `StreamMsg`, `ScrollProgress` (→ `NN%` in the pane title). |
| `internal/engine` | `parseSize`/`distribute` layout math (`layout.go`), bubbletea root `Model` + focus-lock state machine + directional focus + mouse hit-testing (`engine.go`), runner `Options` (`options.go`), palettes + stylesets + memoized `theme` (`theme.go`), status bar (`chrome.go`), overlay enum + F1 help + Ctrl+C confirm dialog + per-pane interrupt (`overlay.go`), F3 live theme/style picker (`settings.go`), `Run`/`RunWithOptions` TTY entry (`run.go`), headless bounded `Snapshot`/`SnapshotWithOptions` (`snapshot.go`). `engine.New/Run/Snapshot` stay pure — the global config is read only in `main.go`. |
| `main.go` | `run` / `validate` / `render` (opt `theme=`/`style=`) / `config` CLI. |
| `examples/` | `hello`, `dashboard`, `responsive`, `keys`, `sysmon`, `pretty`, `devdash`, `multiplex`, `guide` (markdown widget) + `compiled/` (go:embed wrapper); `examples/README.md` maps each to what it demonstrates and a render size. |

Deps: `bubbletea`, `lipgloss` (UI); `bubbles` (`viewport` behind `scrollBox`); `glamour` (`markdown` widget — pulls `chroma` for code highlighting); `charmbracelet/x/ansi` (ANSI-safe help-overlay splice); `creack/pty` + `charmbracelet/x/vt` (interactive `shell` panes — pty + VT emulator).

Import direction: `engine` → `widget` → `config` → (nothing). Keep it acyclic.

## Architecture decisions (do not undo)

- **Widgets are a compile-time registry, not plugins.** `widget.Register(kind, factory)` + config selecting a kind by name. No `plugin.Open`, no `dlopen`, no subprocess-as-module. This is what makes "runtime, dynamic-linked, and static-linked" the *same engine* — runtime reads config from disk, static `go:embed`s it, dynamic reads a path; all call `engine.Run`. "Total modularity" from `DESIGN.md` will tempt a rewrite toward plugins — resist; add a widget file instead.
- **Config is strict JSON.** `DESIGN.md`'s "markdown json" was ambiguous; resolved to one strict decoder (`DisallowUnknownFields`, so typos error). Nodes are split-XOR-leaf by key presence, not a discriminator field.
- **Two process widgets, on purpose.** `command` runs once and captures text into a scroll pane. `shell` hosts a *real interactive process* (htop, vim, cmatrix, `$SHELL`) on a pty, feeds its output through a `charmbracelet/x/vt` emulator, and forwards keystrokes via `keyToBytes`. Don't merge them; don't make `command` grow a pty.
- **Focus lock replaces the tmux prefix.** Two modes. **NAV**: arrows do directional focus, `Tab` cycles, `Enter` locks the focused pane, bound keys run their action, unbound keys fall through to the widget. **LOCKED**: every key goes to the focused widget — this is when a `RawKeyConsumer` (`shell`) pane actually receives raw keys, so htop/vim get their own `q`/F-keys. `Esc` releases; on a raw pane the first `Esc` buffers and `Esc Esc` sends a literal `Esc` to the child. `isGlobalKey` — anything starting `mod+` (mod = Alt by default, set in the global config), plus `F1` and `F3` — is intercepted in **both** modes. `Config.Prefix` is parsed-but-ignored (kept so old configs don't fail `DisallowUnknownFields`). Don't reintroduce a leader-key state machine.
- **Ctrl+C is never a bare quit.** It is handled directly in `handleKey`, not the bindings map, and is deliberately *not* in `isGlobalKey`. NAV: raises the `ovConfirm` quit dialog (`y`/`Enter`/second `Ctrl+C` = quit, `n`/`Esc` = cancel). LOCKED: `lockedInterrupt` — a pane implementing `widget.Interrupter` gets a "kill this pane's process?" confirm and, on yes, only that child dies (`shell` drops to its exit overlay, program keeps running); a plain pane just receives the key. `mod+q` was removed — the way out of a locked pane is `Esc` then `Ctrl+C`. Don't re-add an unconditional quit key.
- **Overlays are one enum, one splice.** `m.overlay` ∈ `ovNone/ovHelp/ovConfirm/ovSettings`; at most one is active. `handleKey` routes every key to it first, and `Update` drops `tea.MouseMsg` entirely while `overlayActive()` — a click behind a modal must not move focus or clear the lock. The `cfKill` dialog captures its target leaf index (`confirmLeaf`) at open time, not by re-reading `m.focus` at confirm. `View` splices exactly one panel (`renderHelp`/`renderConfirm`/`renderSettings`) via `overlayCenter` (ANSI-safe, `x/ansi` `Truncate`/`TruncateLeft` — does not blank the background). Add a new modal as an enum member, not a parallel bool.
- **Theming = palette × styleset, resolved once.** `theme.go` has a `palettes` registry (`auto` adaptive; `tokyo-night`/`gruvbox`/`slate`/`midnight` concrete `lipgloss.Color` — named themes must NOT be adaptive) and a `stylesets` registry (`rounded`/`double`/`heavy`/`ascii` = border + status-bar/lock glyphs; `ascii` is pure 7-bit for non-Nerd-Font terminals). `(*Model).theme()` builds + memoizes; `invalidateTheme()` drops the memo (F3 live preview). Unknown names fall back to the default in the engine but `main.go` rejects them up front (strict-config parity). The F1/confirm/settings panels, status bar and pane borders all read from the resolved `theme` — no hardcoded colours or box chars outside `theme.go` (the one known exception: `shell.go`'s exit bar, which is in the `widget` package and can't import `engine`).
- **F3 settings menu persists via a callback.** `Options.SaveTheme func(theme, style string) error` is nil in `DefaultOptions`/tests (engine never writes disk); `main.go` wires it to `config.SaveGlobalThemeStyle`, which load-mutates-renames the global config touching only `theme`+`style`. The menu previews live on cursor move, commits only on `Enter`, restores `setSavedTheme/setSavedStyle` on `Esc`.
- **Engine stays config-file-agnostic.** `engine.New/Run/Snapshot` use `DefaultOptions()`; `*WithOptions` variants take an `engine.Options`. The global runner config (`config.LoadGlobal`) is read **only in `main.go`** and folded in via `OptionsFromGlobal` — so tests never depend on `~/.config`.
- **Status bar owns the last row.** When shown (`Options.StatusBar` && height ≥ 6) the layout tree gets `height-1`; `rectW/rectH` are stamped to that body size and mouse/`focusDir` staleness checks compare against `bodyDims()`, not the raw terminal size.
- **Mouse.** `render` records each leaf's absolute rect (stamped with the size it was computed at); `Update` hit-tests `tea.MouseMsg` against them — left-press focuses, wheel scrolls the pane *under the cursor* without moving focus. Mouse events are ignored when the rect stamp ≠ current size (stale) or before the first `View`.
- **`Snapshot` is bounded.** A `shell` read loop is infinite by design, so `drain` runs each command with a timeout + step/elapsed caps and applies a `StreamMsg` once without following it. `Snapshot` always `m.Close()`s to kill spawned children. Interactive apps that paint on a timer (htop) won't have painted in a headless frame — that's expected; `run` gives them real time.
- **lipgloss sizing in `renderLeaf`:** `Width/Height` set the content area, a border adds 2 outside. Never put `MaxWidth/MaxHeight` on a bordered style — it clips the final output including the border and shears an edge.
- **Status bar exact width is `lipgloss.Width`-measured.** `statusBar` computes `gap = w - Width(left) - Width(hint)`; the mode-chip glyph comes from the styleset, and Nerd Font glyph cell-widths are exactly where lipgloss and the terminal can disagree — any new styleset glyph needs the `TestStatusBarWidthHoldsAcrossStylesets` table to stay green (both NAV and LOCKED glyphs, several widths).
- **`x/vt` caveat:** its `Read`/`Close`/`Write` share an unsynchronised `closed` flag. Never call `emu.Read` from a second goroutine — that's why terminal-query responses are *not* forwarded to children yet. Keep all emulator calls on the bubbletea goroutine.

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

Actually imported today: `bubbletea` + `lipgloss` + `bubbles` (`viewport`) + `charmbracelet/x/{ansi,vt}` + `creack/pty` + `glamour` (see `go.mod`). `glamour` is pulled by the `markdown` widget only; `bubbles/viewport` backs the shared `scrollBox`. Both live behind the widget layer, not in the engine. `huh` (forms) is *not* wired in; a `form`/`input`/`list` widget would be the first to pull it — same rule: widget file, not engine.

## Documentation deliverables

The project expects two parallel docs, kept in sync with the code:
- `README.md` — for humans: clear, pretty markdown, informative without being verbose.
- `LLM.md` — for AI: optimized for LLM consumption.

## Working constraints

- **Runs on a live production Arch Linux system with no sandbox.** Be careful with destructive operations.
- **Use git.** Work on a branch; never commit straight to `main` or merge without asking.
- **Minimalism over frameworks.** Build exactly what's required with zero overhead. Don't pull a heavy dependency for one small task — write the snippet or cherry-pick it. Continuously question whether a solution is over-complex.
- **Design authority.** Follow `DESIGN.md`'s intent closely *where it makes sense*. If an instruction is impossible, wrong, or wasteful, do it the better way while still delivering at least what was asked. If something clearly should exist, build it even if unmentioned. (`DESIGN.md` is now vision + design requirements only — the agent/software-design rules it used to carry live in this section now.)
- **Clarify only high-stakes unknowns.** For low-stakes ambiguity, make a best-effort call and proceed.

## Token discipline

`optimal-mode.md` is an opt-in "eco mode" ruleset: pinpoint context with `rg`/`fd`/`sed` before reading; never `cat` a >250-line file just for context; keep reasoning effort low unless the task genuinely needs more; terminal-style output with zero filler; take the explicit `[Bypass: requires deep context]` escape hatch rather than looping on retries.
