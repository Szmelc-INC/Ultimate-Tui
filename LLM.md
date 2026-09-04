# LLM.md — ULTIMATE TUI, compressed for coding agents

Status: MVP. Go module `github.com/silverx/ultimate-tui`, Go 1.26. Direct deps:
bubbletea, lipgloss, bubbles (viewport), glamour, creack/pty,
charmbracelet/x/{vt,ansi}. No plugin system by design.

## Package map

- `internal/config` — schema structs + `Parse([]byte)` / `Load(path)`. Strict
  JSON (`DisallowUnknownFields`). `Validate()` enforces tree invariants.
  `Parse` is the ONLY place the wire format is known — add YAML here, nowhere else.
  `global.go`: `GlobalConfig` + `LoadGlobal()` / `ParseGlobal()` — the runner-wide
  config at `$XDG_CONFIG_HOME/ultui/config.json` (missing file → `nil,nil`).
- `internal/widget` — `Widget` interface, `Register(kind, Factory)` /
  `New(*config.Pane)` registry. Kinds self-register in `init()`: `text`,
  `markdown` (glamour, markdown.go), `command`, `shell`. `scrollBox` (scroll.go)
  = shared scroll/wrap/wheel, backed by `bubbles/viewport` (it truncates lines
  to width in `View`, so exact-size is safe); `g`/`home` → top, `G`/`end` →
  bottom, everything else delegates to the viewport KeyMap.
  `keyToBytes` (keys.go) = tea.KeyMsg → xterm bytes. Optional ifaces:
  `RawKeyConsumer` (want every key), `Closer` (owns OS resources, freed on
  program exit), `Interrupter` (`Interrupt() error` — kill just this pane's
  process on a confirmed locked Ctrl+C; shell impl sets `exited` + "process
  killed" note), `StreamMsg` (endless read-loop msg marker), `ScrollProgress`
  (`ScrollFraction() float64` 0..1 → engine appends `NN%` to the pane title).
- `internal/engine` — `sizeSpec`/`parseSize`/`distribute` (layout.go), `Model`
  (bubbletea root, engine.go: focus-lock state machine + directional focus +
  mouse hit-test), `Options`/`DefaultOptions`/`OptionsFromGlobal` (options.go),
  palettes + stylesets + memoized `theme` (theme.go), status bar (chrome.go),
  overlay enum + F1 help + Ctrl+C confirm dialog + per-pane interrupt
  (overlay.go), F2 control panel (control.go), theme/style picker (settings.go),
  pane context menu (context.go), pane/config editor (editor.go),
  per-pane zoom math `zoomCells`/`shiftZoom` (layout.go),
  `Run`/`RunWithOptions` (run.go), `Snapshot`/`SnapshotWithOptions` (snapshot.go).
- `main.go` — CLI: `run`, `validate`, `render`, `config`, `version`, `help`.
  `run`/`render` load the global config via `config.LoadGlobal` →
  `engine.OptionsFromGlobal` and pass it to `*WithOptions`.

## Data model

`config.Node` = split XOR leaf, by presence:
- split: `Direction` ("row"|"column", ""→"column") + `Children []Child`
- leaf: `Pane *Pane`
`Child` embeds `Node` + `Size string`.
`Pane`: `Type`, `Title`, `Border *bool` (nil→true), then flat widget fields
(`Content` = text/markdown source; `File` + `MarkdownStyle` = markdown; `Command`,
`Shell` = command/shell).
`Config.Prefix` (json "prefix") is **deprecated and ignored** — still parsed so
old configs don't fail `DisallowUnknownFields`.

`GlobalConfig`: `Mod` (""→"alt"), `StatusBar *bool` (nil→true), `Theme`
(auto|tokyo-night|gruvbox|slate|midnight; ""/"dark"/"light"→auto), `Style`
(rounded|double|heavy|ascii; ""→rounded), `Keybindings map[string]string`.
Folded into `engine.Options` by `OptionsFromGlobal`. `config.SaveGlobalThemeStyle`
persists just theme+style (temp-file + rename, other fields untouched) — wired to
`Options.SaveTheme` in main.go so the engine never writes disk itself. Likewise
`config.Save(path, *Config)` (Validate → marshal-indent → tmp+rename) is wired to
`Options.SaveConfig` by `cmdRun`; the editor overlay calls it on `mod+s`.

Engine mirrors the layout into unexported `node`/`child`/`leaf`; `leaf.w` holds
the live `widget.Widget`. `m.leaves` is the tree-order focus ring; `m.focus`
indexes it. `m.locked` = focus lock; `m.pendingEsc` = one Esc buffered inside a
locked raw pane. `m.overlay` (`ovNone|ovHelp|ovConfirm|ovSettings|ovControl`) =
the active modal layer (`ovContext` = pane menu, `ovEditor` = pane/config
editor); `m.overlayReturn` is a one-level back target (F2 control panel → picker
→ Esc/Enter returns to F2). `m.ctxLeaf` = the leaf the context menu / editor
targets, captured at open time (not re-read from `m.focus`). `leaf.zoom` = per-pane size bias in
steps, folded into every ancestor split by `shiftZoom` (sum-preserving).
`m.themeMemo` caches the resolved `theme` (`invalidateTheme()` drops it — picker
live preview).

## Control flow

- `NewWithOptions(cfg, opts)` (`New` = defaults): key precedence
  `baseBinds` < `opts.Keys` < `cfg.Keybindings`, each run through `expandMod`
  ("mod+X" → "<opts.Mod>+X"). `baseBinds` = `tab`/`shift+tab`→focus-next/prev,
  `f1`→help, `f2`→control, `mod+w`→context-menu, `mod+=`/`mod++`/`mod+-`/`mod+_`/
  `mod+0`→zoom in/in/out/out/reset (focused pane). Ctrl+C, arrows, Enter, Esc are handled in
  code, not the map (Ctrl+C must never be a plain quit binding). `build()` recurses,
  instantiating widgets into `m.leaves`.
- `Update`: `WindowSizeMsg`→store size+`ready`.
  - `KeyMsg` → `handleKey`:
    1. `m.overlay` non-none swallows the key: help → Esc/F1/q/Enter/Space close
       (Ctrl+C → quit-confirm); confirm → y/Y/Enter/Ctrl+C confirm, n/N/Esc
       cancel (`cfDiscard`'s cancel returns to `ovEditor`, not `ovNone`);
       editor → `handleEditorKey` (Tab tabs, ↑↓ fields, ←→ change, type edits
       text, Enter applies live, `mod+s` applies+`SaveConfig`, Esc exits with a
       `cfDiscard` confirm when `edDirty`/`edConfigDirty`);
       settings → `handleSettingsKey` (arrows preview, Enter commits +
       `opts.SaveTheme`, Esc restores — both return to `overlayReturn`);
       control → `handleControlKey` (↑↓ pick, Enter opens the row, Esc closes).
    2. `isGlobalKey` (starts `mod+`, or `f1`/`f2` — NOT `f3`, which now falls
       through to the widget) → run bound action even when locked. Ctrl+C is
       deliberately NOT global.
    3. `ctrl+c` → locked: `lockedInterrupt` (pane has `widget.Interrupter` →
       kill-confirm dialog, else forward the key); unlocked: `openConfirmQuit`.
    4. locked → `esc` releases (raw pane: first `esc` buffers, `esc esc` forwards
       a literal Esc, any other key after a lone buffered esc releases + re-runs);
       otherwise forward the key to the focused widget.
    5. unlocked → arrows = `focusDir` (nearest pane that way, ring-order fallback
       when rects are stale); `enter` locks; else bound action; else forward.
  - `MouseMsg` → ignored outright while `overlayActive()`; else `handleMouse`:
    no-op unless `rectW/rectH == bodyDims()`. Right press → `openContext(leafAt)`
    (target captured, focus unchanged). Left press → focus `leafAt`, clears lock
    if the pane changed. Wheel → forward to that leaf, no focus change. Because
    mouse is dropped once an overlay is up, the context menu is keyboard-driven
    (↑↓ skip greyed rows, Enter runs, Esc / mod+w close).
  - other msg → broadcast to all widgets.
- `View`: `bodyDims()` = full size minus 1 row when the status bar shows
  (`opts.StatusBar && h>=6`). `render` the tree into the body, stamp
  `rectW/rectH` to the body size, append `statusBar(w)`, then splice the active
  overlay (`renderHelp`/`renderConfirm`/`renderSettings`/`renderControl`) centered
  via `overlayCenter` (ANSI-safe, `x/ansi` Truncate + TruncateLeft).
  `render(n,x,y,w,h)` recurses tracking absolute origin; each leaf stores its rect
  (`x,y,pw,ph`). Split nodes: `distribute` then, if any child subtree holds a
  zoomed leaf, `shiftZoom` rebalances the sizes (total held constant).
- `renderLeaf`: inner = w-2/h-2 if bordered. Optional title line (costs 1 body
  row) — when `lf.title != ""` it gets a `ScrollProgress` `NN%` suffix and, if
  focused+locked, a `<bullet> LOCKED` suffix; border + glyphs from the resolved
  styleset, colours from the palette (`theme.go`). Body = `widget.View(innerW,
  bodyH)`. Wrap in
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

`markdown` widget (markdown.go): source = `Content` or `os.ReadFile(File)`.
source read in `newMarkdown` (`Content`, or `os.ReadFile(File)` — so a bad `file`
path fails `engine.New`/`validate`, and a relative path needs the right cwd).
`View(w,h)` lazily `glamour.NewTermRenderer(WithStandardStyle(style),
WithWordWrap(w))` — style pinned (`MarkdownStyle`; ""/"auto" → "dark") so `render`
output is deterministic — caches the rendered lines per width, scrolls them as
opaque ANSI lines (no re-wrap), forces exact w×h via `normalizeGrid` (shared with
shell.go), and implements `ScrollProgress`. `Init` is nil; not a `RawKeyConsumer`.

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
- A config can remap a key but can't *unbind* a default (same-key remap only).
- Unlocked, the engine owns arrows/Tab/Enter/Ctrl+C and every bound key; other
  keys fall through to the focused widget. Locked, only `isGlobalKey` (mod+…,
  F1, F2 — NOT F3), `ctrl+c` (→ per-pane interrupt) and `esc` are intercepted —
  everything else goes to the widget. A `RawKeyConsumer` pane only receives raw
  keys **while locked**, so the user must press Enter to drive `htop`/`vim`.
  Ctrl+C never quits outright (NAV: confirm dialog; LOCKED: kills that pane's
  process); the way out of a locked pane is Esc then Ctrl+C.
- `engine.New/Run/Snapshot` stay pure (default options). The global config is
  loaded in `main.go` only, so tests never touch `~/.config`.
- Status bar eats the last row (`bodyDims`); `rectW/rectH` are stamped to the
  body size, and `handleMouse`/`focusDir` compare against `bodyDims()`, not
  `w/h`.
- Never call `emu.Read` off the bubbletea goroutine (x/vt race). Consequence:
  terminal-query responses are not forwarded to `shell` children.
- Leaf rects are only valid for the last-rendered size — `handleMouse` bails if
  the stamp is stale.
