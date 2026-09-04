# LLM.md — ULTIMATE TUI, compressed for coding agents

Status: MVP. Go module `github.com/silverx/ultimate-tui`, Go 1.26. Deps:
bubbletea, lipgloss, bubbles (bubbles not used yet). No plugin system by design.

## Package map

- `internal/config` — schema structs + `Parse([]byte)` / `Load(path)`. Strict
  JSON (`DisallowUnknownFields`). `Validate()` enforces tree invariants.
  `Parse` is the ONLY place the wire format is known — add YAML here, nowhere else.
- `internal/widget` — `Widget` interface, `Register(kind, Factory)` /
  `New(*config.Pane)` registry. Kinds self-register in `init()`. `scrollBox`
  (scroll.go) is shared scroll/wrap logic. Kinds: `text`, `command`.
- `internal/engine` — `sizeSpec`/`parseSize`/`distribute` (layout.go), `Model`
  (bubbletea root, engine.go), `Run(cfg)` (run.go, TTY), `Snapshot(cfg,w,h)`
  (snapshot.go, headless one frame).
- `main.go` — CLI: `run`, `validate`, `render`, `version`, `help`.

## Data model

`config.Node` = split XOR leaf, by presence:
- split: `Direction` ("row"|"column", ""→"column") + `Children []Child`
- leaf: `Pane *Pane`
`Child` embeds `Node` + `Size string`.
`Pane`: `Type`, `Title`, `Border *bool` (nil→true), then flat widget fields
(`Content`; `Command`, `Shell`).

Engine mirrors this into unexported `node`/`child`/`leaf`; `leaf.w` holds the
live `widget.Widget`. `m.leaves` is the tree-order focus ring; `m.focus` indexes it.

## Control flow

- `New(cfg)`: merge `defaultKeys` ← `cfg.Keybindings`; `build()` recurses,
  instantiating widgets and appending to `m.leaves`.
- `Update`: `WindowSizeMsg`→store size+`ready`. `KeyMsg`→ if key in `m.keys`
  run action (`quit`/`focus-next`/`focus-prev`); else forward to focused widget.
  Any other msg → broadcast to all widgets (async results, mouse).
- `View`/`render`: recurse; split→`distribute` axis cells→render children→
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

## Adding a widget kind

New file in `internal/widget`, `func init(){ Register("kind", factory) }`,
implement the 3 methods, reuse `scrollBox` for text bodies. Add fields to
`config.Pane` (flat) or, if it gets crowded, introduce a `json.RawMessage`
per-pane options blob and decode in the factory.

## Snapshot / test rules

- No TTY in CI. Assert on `Model.View()` or `engine.Snapshot`.
- `Snapshot` drains `Init()` commands synchronously incl. `tea.BatchMsg`, so
  `command` output appears in the frame. Only handles run-once cmds.
- Width check: `lipgloss.Width(line)` (ANSI-aware). Height: `len(Split(v,"\n"))`.
- `make test-one T=TestName`.

## Invariants worth keeping

- `Parse` is the only format-aware code.
- Engine imports widget+config; widget imports config; config imports neither.
- Every split should contain a flex child; layout does not error on overflow, it
  clamps to 0 and may look wrong.
- Global keybindings win over widget keys — a config can't currently *unbind* a
  default, only remap the same key.
