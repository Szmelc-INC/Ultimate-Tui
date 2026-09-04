# LLM.md — ULTIMATE TUI, compressed for coding agents

Status: MVP. Go module `github.com/silverx/ultimate-tui`, Go 1.26. Direct deps:
bubbletea, lipgloss, bubbles (viewport, textinput, list), huh, glamour,
creack/pty, charmbracelet/x/{vt,ansi}. No plugin system by design.

## Package map

- `internal/config` — schema structs + `Parse([]byte)` / `Load(path)`. Strict
  JSON (`DisallowUnknownFields`). `Validate()` enforces tree invariants.
  `Parse` is the ONLY place the wire format is known — add YAML here, nowhere else.
  `global.go`: `GlobalConfig` + `LoadGlobal()` / `ParseGlobal()` — the runner-wide
  config at `$XDG_CONFIG_HOME/ultui/config.json` (missing file → `nil,nil`);
  `SaveGlobalThemeStyle()` (load-mutate theme+style, preserve the rest) and
  `SaveGlobal()` (replace the whole file — F3) both temp-file+rename.
- `internal/widget` — `Widget` interface, `Register(kind, Factory)` /
  `New(*config.Pane)` registry. Kinds self-register in `init()`: `text`,
  `markdown` (glamour, markdown.go), `command`, `shell`, `input`
  (bubbles/textinput, input.go), `list` (bubbles/list, list.go), `form`
  (huh, form.go). `scrollBox` (scroll.go) = shared scroll/wrap/wheel, backed
  by `bubbles/viewport` (it truncates lines to width in `View`, so exact-size
  is safe); `g`/`home` → top, `G`/`end` → bottom, everything else delegates
  to the viewport KeyMap. `input`/`list`/`form` embed their bubbles/huh
  component directly instead — it already owns scrolling/editing/filtering,
  so `scrollBox` would be redundant. Three gotchas from hosting a
  third-party bubbletea component as a *sub*-model rather than the whole
  program (see also the matching CLAUDE.md Architecture decisions bullet):
  (1) `tea.WindowSizeMsg` never reaches a widget — `engine.Model.Update`
  consumes it to size the layout tree and returns before the `default:`
  broadcast to leaves runs — so `list.SetSize`/`huh.Form.WithWidth,WithHeight`/
  `textinput.Width` are set every `View(w,h)` call instead, same pattern as
  `renderLeaf` re-reading `Size`/`Direction` every frame elsewhere; (2)
  `bubbles/list`'s default keymap binds `q`/`ctrl+c` to `tea.Quit` —
  `newList` calls `DisableQuitKeybindings()` or a keypress in one pane kills
  the whole multi-pane program; (3) cursor blink (and `huh.Form`'s own
  internal field messages) needs every `tea.Msg` forwarded to
  `Update`, not just `KeyMsg`/`MouseMsg` the way `text`/`command` filter —
  `inputWidget`/`listWidget`/`formWidget.Update` forward unconditionally, and
  engine.go's `default:` broadcast is what delivers those ticks. `keyToBytes`
  (keys.go) = tea.KeyMsg → xterm bytes. Optional ifaces: `RawKeyConsumer`
  (want every key), `Closer` (owns OS resources, freed on program exit),
  `Interrupter` (`Interrupt() error` — kill just this pane's process on a
  confirmed locked Ctrl+C; shell impl sets `exited` + "process killed"
  note), `StreamMsg` (endless read-loop msg marker), `ScrollProgress`
  (`ScrollFraction() float64` 0..1 → engine appends `NN%` to the pane title),
  `Zoomer` (`SetZoom(level int)` — per-pane content scale, `mod+=/-/0`; `text`
  implements it via `bigfont.go`'s 3x5 block-glyph font, nothing else does),
  `Styler` (`SetStyles(widget.Styles)` — resolved theme colors, pushed every
  frame by `renderLeaf` before `View`, same idiom as size above but for
  color; `widget.Styles`/`Styler` live in `widget` since it can't import
  `engine`'s theme.go; `list`/`input`/`form`/`shell` implement it — `list`
  recolors bubbles/list's default pink delegate + filter/pagination styles,
  `input` recolors textinput's prompt/text/placeholder/cursor, `form` builds
  a `*huh.Theme` from `huh.ThemeBase()` applied via `WithTheme` in `View`,
  `shell` recolors its "process exited" bar (formerly hardcoded red/white,
  now `Warn`/`ModeFg` with that hardcode kept only as the pre-first-frame
  fallback); `text`/`command`/`markdown` don't implement it — that's user
  content, not chrome).
- `internal/engine` — `sizeSpec`/`parseSize`/`distribute` (layout.go), `Model`
  (bubbletea root, engine.go: focus-lock state machine + directional focus +
  mouse hit-test), identity-preserving tree rebuild (rebuild.go: `rebuild()`,
  `indexOfPane`, `paneOf`), directional resize/swap/flip-split (resize.go),
  Split/Close/New-pane structural edits (split.go), mod+drag move/resize
  (drag.go), `Options`/`DefaultOptions`/`OptionsFromGlobal` (options.go),
  palettes + stylesets + memoized `theme` (theme.go — each palette:
  `subtle`/`accent`/`accent2`/`warn`/`success`/`text`/`barBg`/`modeFg` plus a
  `surface`/`onSurface` muted fill+fg pair, all filled concretely in every
  named palette, adaptive only in `auto`; `chipPrimary`/`chipMuted` are the
  only two `theme` styles that set `Background`, not just `Foreground` — used
  by the confirm dialog's Yes/Cancel buttons and the status bar's key-hint
  badges, both `Padding(0,N)` horizontal-only since row *count* is
  load-bearing there), status bar + `splice()`/
  `overlayGeometry` (chrome.go), overlay enum + F1 help + Ctrl+C confirm
  dialog + per-pane interrupt (overlay.go), mouse routing for every overlay
  (overlay_mouse.go), F2 control panel (control.go), F3 global
  settings/diagnostics panel (global.go), theme/style picker (settings.go),
  pane context menu (context.go), pane/config editor incl. Keybinds tab
  (editor.go), `Run`/`RunWithOptions` (run.go), `Snapshot`/`SnapshotWithOptions`
  (snapshot.go).
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
`Options.SaveTheme` in main.go so the engine never writes disk itself.
`config.SaveGlobal` replaces the whole file — wired to `Options.SaveGlobal`, F3's
save target; `applyGlobal` (global.go) always carries `m.opts.Keys` (this run's
already-resolved global keybindings) into what it saves, so it can never
silently drop a hand-edited one it never touched. Likewise `config.Save(path,
*Config)` (Validate → marshal-indent → tmp+rename) is wired to
`Options.SaveConfig` by `cmdRun`; the editor overlay calls it on `mod+s`.
`Options.ConfigPath`/`GlobalConfigPath` are plain strings `main.go` resolves and
hands over for F3's info block — engine still never touches either path itself.

Engine mirrors the layout into unexported `node`/`child`/`leaf`; `leaf.w` holds
the live `widget.Widget`. Each carries a back-pointer to its live config source
(`node.cfg *config.Node`, `child.cfg *config.Child`, `leaf.pane`/`leaf.parent`/
`leaf.split`); `render` re-reads `Direction`/`Size` from those each frame so the
editor's edits aren't frozen at build time. `leaf.chain []ancestorLink` (each
`{split *config.Node, idx int}`) is the root-to-parent path `build()` threads
down while recursing — `resize.go` walks it backward to find the nearest
enclosing split whose axis matches an arrow direction. `m.leaves` is the
tree-order focus ring; `m.focus` indexes it — but any structural edit
(Split/Close/New/swap) invalidates every such index, so re-resolve via
`indexOfPane(capturedPanePointer)` after calling `rebuild()`, never keep an
index across the call. `m.locked` = focus lock; `m.pendingEsc` = one Esc
buffered inside a locked raw pane. `m.overlay`
(`ovNone|ovHelp|ovConfirm|ovSettings|ovControl|ovContext|ovEditor|ovSplit|ovGlobal`)
= the active modal layer; `m.overlayReturn` is a one-level back target (F2 or F3
→ theme picker → Esc/Enter returns to whichever opened it). `m.ctxLeaf` = the
leaf the context menu / editor targets, captured at open time (not re-read from
`m.focus`); `-1` is a valid value meaning "dead space" (the New-pane menu).
`leaf.contentZoom` = per-pane content-scale level (0=unscaled), forwarded to
the widget via the optional `widget.Zoomer` interface — nothing to do with
layout. `m.ovX/Y/W/H` = the active overlay panel's bounding box, stamped by
`splice()` every render, used by `handleOverlayMouse` to hit-test clicks.
`m.dragMode` (`dragNone|dragMove|dragResize`) + `m.dragLeaf`/`dragLastX`/
`dragLastY` = mod+drag state (drag.go). `m.themeMemo` caches the resolved
`theme` (`invalidateTheme()` drops it — picker live preview).

## Control flow

- `NewWithOptions(cfg, opts)` (`New` = defaults) calls `rebuildKeyMap()`: key
  precedence `baseBinds` < `opts.Keys` < `cfg.Keybindings`, each run through
  `expandMod` ("mod+X" → "<opts.Mod>+X"). `baseBinds` = `tab`/`shift+tab`→
  focus-next/prev, `f1`→help, `f2`→control, `mod+w`→context-menu, `mod+=`/
  `mod++`/`mod+-`/`mod+_`/`mod+0`→content-zoom in/in/out/out/reset,
  `mod+↑↓←→`→resize-{up,down,left,right}, `mod+shift+↑↓←→`→swap-{…},
  `mod+s`→flip-split. Ctrl+C, arrows, Enter, Esc, F3 are handled in code, not
  the map (Ctrl+C must never be a plain quit binding; F3 must NOT be in
  `isGlobalKey`). `build()` recurses, instantiating widgets into `m.leaves`
  and threading `leaf.chain` (nil `carry`/`fresh` on this first build — see
  `rebuild()` below for what those are for). `rebuildKeyMap()` is also called
  by `applyKeybinds` (editor.go) whenever the Keybinds tab's apply actually
  changes `cfg.Keybindings`, so an edited/added binding works immediately.
- `Update`: `WindowSizeMsg`→store size+`ready`.
  - `KeyMsg` → `handleKey`:
    1. `m.overlay` non-none swallows the key: help → Esc/F1/q/Enter/Space close
       (Ctrl+C → quit-confirm); confirm → y/Y/Enter/Ctrl+C confirm, n/N/Esc
       cancel (`cfDiscard`'s cancel returns to `ovEditor`, not `ovNone`);
       editor → `handleEditorKey` (Tab tabs, ↑↓ fields, ←→ change, type edits
       text, `ctrl+d` removes a Keybinds-tab row, Enter applies live (or, on
       the "+ add keybind" row, appends one), `mod+s` applies+`SaveConfig`,
       Esc exits with a `cfDiscard` confirm when `edDirty`/`edConfigDirty`);
       split → `handleSplitKey` (↑↓/Tab field, ←→ change direction/percent,
       Enter performs the split/wrap-root and calls `rebuild()`, Esc cancels);
       global → `handleGlobalKey` (↑↓ row, ←→/Space cycle mod or toggle
       statusbar, Enter on the theme row opens the picker, `mod+s` calls
       `applyGlobal`→`opts.SaveGlobal`, Esc/F3 close);
       settings → `handleSettingsKey` (arrows preview, Enter commits +
       `opts.SaveTheme`, Esc restores — both return to `overlayReturn`);
       control → `handleControlKey` (↑↓ pick, Enter opens the row, Esc closes);
       context → `handleContextKey` (↑↓/`home`/`end` skip disabled rows, Enter
       runs — Split/New open `ovSplit`, Close opens a `cfClose` confirm, Stop a
       `cfKill` confirm — Esc/`mod+w` close).
    2. `isGlobalKey` (starts `mod+`, or `f1`/`f2` — NOT `f3`) → run bound
       action even when locked. Ctrl+C is deliberately NOT global.
    3. `ctrl+c` → locked: `lockedInterrupt` (pane has `widget.Interrupter` →
       kill-confirm dialog, else forward the key); unlocked: `openConfirmQuit`.
    4. locked → `esc` releases (raw pane: first `esc` buffers, `esc esc` forwards
       a literal Esc, any other key after a lone buffered esc releases + re-runs);
       otherwise forward the key to the focused widget.
    5. unlocked → arrows = `focusDir` (nearest pane that way, ring-order fallback
       when rects are stale); `enter` locks; `f3` opens the global panel
       (hardcoded here, not `isGlobalKey`, so it's NAV-only); else bound
       action; else forward.
  - `MouseMsg` → `overlayActive()` routes to `handleOverlayMouse`
    (overlay_mouse.go): outside the panel's stamped bounding box
    (`m.ovX/Y/W/H`), a left/right press is re-dispatched as `handleKey(Esc)`
    — dismiss through the SAME per-overlay Esc handling, never a generic
    close; inside, content-relative coords (`- ovX/ovY - panelContentOffsetX/Y`,
    one constant for every panel since they all use `th.helpBox`) go to that
    overlay's `xxxMouseClick`. Else `handleMouse`: `m.dragMode != dragNone ||
    msg.Alt` → `handleDragMouse` (drag.go) FIRST, preempting everything below
    (this is how holding mod disables the plain right-click menu) — press
    starts `dragMove`/`dragResize` on the leaf under the cursor and focuses
    it, `dragResize` motion calls `dragResizeAxis` (growDir first, falls back
    to shrinking the opposite edge if growDir has no neighbor —
    `resizeFocusedBy` in resize.go, the signed generalization of
    `resizeFocused`) live per `dragCellsPerStep` cells moved, release either
    ends a resize or (`dragMove`) swaps
    `nodeOfLeaf(from).Pane` with `nodeOfLeaf(to).Pane` and `rebuild()`s. Non-
    drag: no-op unless `rectW/rectH == bodyDims()`. Right press → `openContext
    (leafAt)` (`-1` for dead space → the New-pane menu; target captured, focus
    unchanged). Left press → focus `leafAt`, clears lock if the pane changed.
    Wheel → forward to that leaf, no focus change.
  - other msg → broadcast to all widgets.
- `View`: `bodyDims()` = full size minus 1 row when the status bar shows
  (`opts.StatusBar && h>=6`). `render` the tree into the body, stamp
  `rectW/rectH` to the body size, append `statusBar(w)`, then splice the active
  overlay via `m.splice(frame, panel)` (chrome.go) — calls `overlayGeometry`
  to stamp `m.ovX/Y/W/H` before compositing through `overlayCenter` (ANSI-safe,
  `x/ansi` Truncate + TruncateLeft). `render(n,x,y,w,h)` recurses tracking
  absolute origin; each leaf stores its rect (`x,y,pw,ph`). Split nodes: just
  `distribute` — no post-pass any more (that was `shiftZoom`, deleted with the
  old zoom-as-size feature; resize/swap write directly into `Child.Size` /
  swap `Node` content instead, see `resize.go`/`split.go`).
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
- Unlocked, the engine owns arrows/Tab/Enter/Ctrl+C/F3 and every bound key;
  other keys fall through to the focused widget. Locked, only `isGlobalKey`
  (mod+…, F1, F2 — NOT F3, which is a hardcoded NAV-only case, not in
  `isGlobalKey` at all), `ctrl+c` (→ per-pane interrupt) and `esc` are
  intercepted — everything else goes to the widget. A `RawKeyConsumer` pane
  only receives raw keys **while locked**, so the user must press Enter to
  drive `htop`/`vim`. Ctrl+C never quits outright (NAV: confirm dialog;
  LOCKED: kills that pane's process); the way out of a locked pane is Esc
  then Ctrl+C.
- `engine.New/Run/Snapshot` stay pure (default options). The global config is
  loaded in `main.go` only, so tests never touch `~/.config`. Same rule for
  `Options.ConfigPath`/`GlobalConfigPath` — plain strings `main.go` hands
  over, never resolved by the engine itself.
- A structural edit to the pane tree (Split/Close/New/swap — anything that
  changes `Node.Children`'s length or moves a `*config.Pane` between `Child`
  slots) MUST call `rebuild()` immediately after and MUST NOT hold a leaf
  index (`m.focus`, `m.ctxLeaf`, …) across the call — capture a
  `*config.Pane` before, re-resolve via `indexOfPane` after. A pure
  `Child.Size`/`Node.Direction` mutation (resize, flip-split) needs neither —
  `render()` re-reads both every frame and the `Children` slice itself is
  untouched. `New`/Split's blank pane defaults to `type: "text"` — a pane
  needs a non-empty `Type` or `config.Save`'s `Validate()` rejects it.
- Status bar eats the last row (`bodyDims`); `rectW/rectH` are stamped to the
  body size, and `handleMouse`/`focusDir` compare against `bodyDims()`, not
  `w/h`.
- Never call `emu.Read` off the bubbletea goroutine (x/vt race). Consequence:
  terminal-query responses are not forwarded to `shell` children.
- Leaf rects are only valid for the last-rendered size — `handleMouse` bails if
  the stamp is stale (`m.rectW/rectH` vs the current body size). `m.ovX/Y/W/H`
  (the overlay bounding box, stamped by `splice()`) has a lighter guard:
  `handleOverlayMouse` bails on `!m.ready` (no `View()` has run yet, so the
  box is still zero) but does not re-check the box against the current
  terminal size the way `handleMouse` does — a resize between one `View()`
  and the next click is a one-frame window, not one this guard closes.
