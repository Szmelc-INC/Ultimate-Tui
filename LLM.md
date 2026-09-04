# LLM.md — ULTIMATE TUI, compressed for coding agents

Status: MVP. Go module `github.com/silverx/ultimate-tui`, Go 1.26. Direct deps:
bubbletea, lipgloss, creack/pty, charmbracelet/x/vt. No plugin system by design.

## Package map

- `internal/config` — schema structs + `Parse([]byte)` / `Load(path)`. Strict
  JSON (`DisallowUnknownFields`). `Validate()` enforces tree invariants.
  `Parse` is the ONLY place the wire format is known — add YAML here, nowhere else.
- `internal/widget` — `Widget` interface, `Register(kind, Factory)` /
  `New(*config.Pane)` registry. Kinds self-register in `init()`: `text`,
  `command`, `shell`. `scrollBox` (scroll.go) = shared scroll/wrap/wheel.
  `keyToBytes` (keys.go) = tea.KeyMsg → xterm bytes. Optional ifaces:
  `RawKeyConsumer` (want every key), `Closer` (owns OS resources), `StreamMsg`
  (endless read-loop msg marker).
- `internal/engine` — `sizeSpec`/`parseSize`/`distribute` (layout.go), `Model`
  (bubbletea root, engine.go: prefix state machine + mouse hit-test), `Run(cfg)`
  (run.go, TTY + mouse), `Snapshot(cfg,w,h)` (snapshot.go, bounded headless frame).
- `main.go` — CLI: `run`, `validate`, `render`, `version`, `help`.

## Data model

`config.Node` = split XOR leaf, by presence:
- split: `Direction` ("row"|"column", ""→"column") + `Children []Child`
- leaf: `Pane *Pane`
`Child` embeds `Node` + `Size string`.
`Pane`: `Type`, `Title`, `Border *bool` (nil→true), then flat widget fields
(`Content`; `Command`, `Shell`).
`Config.Prefix` (json "prefix", default "ctrl+b") = leader key for raw-key panes.

Engine mirrors this into unexported `node`/`child`/`leaf`; `leaf.w` holds the
live `widget.Widget`. `m.leaves` is the tree-order focus ring; `m.focus` indexes it.

## Control flow

- `New(cfg)`: merge `defaultKeys` ← `cfg.Keybindings`; `build()` recurses,
  instantiating widgets and appending to `m.leaves`.
- `Update`: `WindowSizeMsg`→store size+`ready`.
  - `KeyMsg` → `handleKey`: if focused widget is a `RawKeyConsumer` and active,
    only `m.prefix` is caught (sets `pendingPrefix`); `<prefix><prefix>` →
    literal to child; `<prefix> X` → action for X. Non-raw pane: `m.keys` wins,
    else forward to focused widget.
  - `MouseMsg` → `handleMouse`: no-op unless `rectW/rectH == w/h` (rects fresh).
    Left press → focus `leafAt(X,Y)`. Wheel → forward to that leaf's widget, no
    focus change.
  - other msg → broadcast to all widgets.
- Mouse reporting ON via `tea.WithMouseCellMotion()` in `run.go`; `Run` calls
  `m.Close()` after the program ends to kill child processes.
- `View`/`render`: `render(n, x, y, w, h)` recurses tracking absolute origin;
  each leaf stores its rect (`x,y,pw,ph`) for mouse hit-testing. `View` stamps
  `rectW/rectH`. split→`distribute` axis cells→render children→
  `lipgloss.JoinHorizontal/Vertical`. leaf→`renderLeaf`.
- `renderLeaf`: inner = w-2/h-2 if bordered. Optional title line (costs 1 body
  row). Body = `widget.View(innerW, bodyH)`. Wrap in
  `lipgloss.NewStyle().Width(innerW).Height(innerH)` + border.
  **Do NOT add `MaxWidth/MaxHeight` on the bordered style** — lipgloss clips the
  final output incl. border and shears the bottom/right edge. lipgloss
  Width/Height = content area; border adds 2 outside.

## Widget contract

```go
Init() tea.Cmd
Update(tea.Msg) (Widget, tea.Cmd)
View(width, height int) string   // must return exactly width x height
```

`command` widget: `Init` returns a `tea.Cmd` running `exec.Command(shell,"-c",
cmd).CombinedOutput()`, returns `cmdDoneMsg{id, output}`. Each instance has an
`atomic` id so the broadcast matches the right widget. Run-once only.

`shell` widget (shell.go): `Init` synchronously `pty.StartWithSize`s
`shell -c command`, makes a `vt.SafeEmulator(w,h)`, returns `readCmd()` — a
`tea.Cmd` that blocks on `ptmx.Read` and returns `shellOutMsg` (a `StreamMsg`).
`Update`: `shellOutMsg` → `emu.Write(data)` then re-issue `readCmd()`; on read
err → `finish()` (mark exited, `Close()`, reap in a goroutine). `KeyMsg` →
`ptmx.Write(keyToBytes(k))`. `View(w,h)` resizes emu + `pty.Setsize` on size
change, then `normalizeGrid(emu.Render(), w, h)` → exactly h lines × w cells
(Render() emits "" for blank rows + a trailing newline). Implements
`RawKeyConsumer` (true until exit), `Closer`. ALL emulator calls stay on the
bubbletea goroutine — `x/vt` Read/Write/Close share an unsynchronised flag.

## Adding a widget kind

New file in `internal/widget`, `func init(){ Register("kind", factory) }`,
implement the 3 methods, reuse `scrollBox` for text bodies. Add fields to
`config.Pane` (flat) or, if it gets crowded, introduce a `json.RawMessage`
per-pane options blob and decode in the factory.

## Snapshot / test rules

- No TTY in CI. Assert on `Model.View()` or `engine.Snapshot`.
- `Snapshot.drain` = bounded work queue: each `cmd()` via `runWithTimeout`
  (250ms), caps `snapMaxSteps`/`snapMaxElapsed`, applies a `StreamMsg` once
  without following its next cmd. Always `defer m.Close()`.
- `shell` widgets ARE headless-testable: pty needs no TTY. Drive `Init()`'s cmd
  loop by hand (`for cmd != nil { msg := cmd(); w, cmd = w.Update(msg) }`) with a
  wall-clock deadline; assert `stripANSI(w.View(w,h))` contains expected text.
  See `shell_test.go` (printf/cat) and `shell_htop_test.go` (real ncurses app).
- `keyToBytes` has a table test (`keys_test.go`) — the likely spot for a wrong
  byte. Run `go test -race` (shell widget uses goroutines for reap).
- Width check: `lipgloss.Width(line)` (ANSI-aware). Height: `len(Split(v,"\n"))`.
- `make test-one T=TestName`.

## Invariants worth keeping

- `Parse` is the only format-aware code.
- Engine imports widget+config; widget imports config; config imports neither.
- Every split should contain a flex child; layout does not error on fixed-cell
  overflow, it clamps to 0 and may look wrong. `Validate` DOES reject a split
  whose child percentages sum > 100.
- Global keybindings win over widget keys for non-raw panes — a config can't
  currently *unbind* a default, only remap the same key.
- A focused `RawKeyConsumer` pane gets ALL keys except `Config.Prefix`; that is
  the only escape hatch, so `Prefix` must never be empty (`New` defaults it).
- Never call `emu.Read` off the bubbletea goroutine (x/vt race). Consequence:
  terminal-query responses are not forwarded to `shell` children.
- Leaf rects are only valid for the last-rendered size — `handleMouse` bails if
  the stamp is stale.
