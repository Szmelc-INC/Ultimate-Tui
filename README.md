# ULTIMATE TUI

> A config-driven TUI framework in Go. One file describes a whole terminal
> interface — layout, panes, keys, shell commands — and `ultui` renders it.

This is an **MVP**. The pieces below work end to end; the roadmap lists what is
deliberately not built yet.

---

## Quick start

```sh
make build                              # -> bin/ultui
./bin/ultui run examples/dashboard.json # live TUI
./bin/ultui render examples/hello.json 80x20   # one frame to stdout, no TTY
```

```
┌──────────────────────────┐┌────────────────────────────────┐┌────────────────────────────────┐
│nav                       ││host                            ││disk                            │
│ULTIMATE TUI              ││Linux 7.1.8-arch1-3             ││Filesystem      Size  Used Avail│
│                          ││ 09:31  up 17 days   load 3.02  ││/dev/nvme1n1p5  344G  285G   42G│
│- overview                ││                                ││/dev/nvme1n1p6  1.3T  1.2T   51G│
│- processes               │└────────────────────────────────┘└────────────────────────────────┘
│- disk                    │┌──────────────────────────────────────────────────────────────────┐
│tab: cycle panes          ││processes (top by rss)                                            │
│^C:  quit                 ││    PID   RSS COMMAND                                              │
└──────────────────────────┘└──────────────────────────────────────────────────────────────────┘
```

## Config format

A config is **JSON** (DESIGN.md's "markdown json" resolved to one strict
decoder; a second format is a decoder swap away). Unknown keys are rejected so
typos surface.

```jsonc
{
  "name": "system-dashboard",
  "keybindings": { "1": "focus-next", "mod+r": "quit" },  // merged over the defaults
  "layout": { /* a node */ }
}
```

Bindings map a key to `quit` / `focus-next` / `focus-prev` / `help` /
`settings` / `lock` / `unlock`. A `mod+` prefix expands to the modifier key
(default **Alt** — set by the global runner config, see below).

### Nodes

A layout **node** is either a **split** or a **leaf** — decided by which keys are
present, not a type tag.

```jsonc
// split: lay children out along one axis
{ "direction": "row" | "column",          // default "column"
  "children": [ { "size": "...", ...node } ] }

// leaf: one widget
{ "pane": { "type": "text", "title": "...", "border": true } }
```

### Sizing

`size` on each child of a split:

| form   | meaning                                  |
|--------|------------------------------------------|
| `40%`  | 40% of the split's axis                  |
| `20`   | 20 fixed cells                           |
| `2*`   | flex weight 2                            |
| `*`, ``| flex weight 1                            |

Fixed and percentage slots are taken first; flex slots divide the remainder by
weight. **Keep at least one `*` per split** so rounding and odd sizes stay
covered.

### Widgets

| type      | keys                      | notes |
|-----------|---------------------------|-------|
| `text`    | `content`                 | static text, scrolls if it overflows |
| `markdown`| `content` or `file`, `markdownStyle` (opt) | renders Markdown with **glamour** — headings, tables, blockquotes, syntax-highlighted code. Lock the pane to scroll. `markdownStyle`: `dark` (default, also what `auto`/`""` resolve to) · `light` · `dracula` · `tokyo-night` · `pink` · `ascii` · `notty`. A relative `file` is resolved against ultui's working directory, and `validate` fails if it is missing. |
| `command` | `command`, `shell` (opt)  | runs the command **once**, shows combined stdout+stderr, scrollable |
| `shell`   | `command`, `shell` (opt)  | a **real interactive process** on a pty — `htop`, `vim`, `cmatrix`, `$SHELL`. Output goes through a VT emulator; keystrokes are forwarded to the child |

`command` captures text; `shell` *is* a terminal. Pick `shell` when the program
draws its own screen or needs input. `markdown` is for documents — help pages,
READMEs, changelogs — rendered in place.

`text`, `command` and `markdown` panes scroll (backed by **bubbles/viewport**)
and show a `NN%` position marker in their title once the content overflows.

### Keys

Two modes, shown in the bottom status bar:

| mode       | keys |
|------------|------|
| **NAV** (default) | `↑ ↓ ← →` move focus to the pane in that direction · `Tab` / `Shift+Tab` cycle · `Enter` **locks** focus into the pane · `mod+= / mod+- / mod+0` grow / shrink / reset the focused pane · `mod+w` (or right-click) pane menu · `F1` help · `F2` controls · `Ctrl+C` quit (asks first) |
| **LOCKED** | every key goes to the pane's widget (scroll a `text` pane, drive `htop`/`vim` in a `shell` pane) · `Esc` releases · `Esc Esc` sends a literal `Esc` to a `shell` child · `Ctrl+C` kills **this pane's** process (asks first), program keeps running |

`mod` is the modifier — **Alt** by default. Terminals cannot tell left Alt from
right Alt, so any Alt press counts. `mod+…`, `F1` and `F2` are caught even while
a pane is LOCKED (`F3` is not — it falls through to the widget). `Ctrl+C` is
**not** a bare quit: in NAV it raises a `[Y/n]` confirm dialog, and while LOCKED
it belongs to the pane (kills just that process, on confirm). The way out of a
locked pane is `Esc` then `Ctrl+C`.

**Per-pane zoom** — `mod+=` (or `mod++`) grows the focused pane, `mod+-` shrinks
it, `mod+0` resets. The bias is applied in every ancestor split and siblings give
up the space; the total never changes, so nothing shears.

**`F1`** opens a centered popup listing every active binding — the engine's plus
this TUI's own `keybindings`. **`F2`** opens the control panel: a summary of the
effective runner settings plus a **Theme & style** row that opens the live
picker (`↑↓` preview, `←→` switch column, `Enter` applies and saves to the global
config, `Esc` reverts — either way you land back on the F2 menu).

### Global runner config

`~/.config/ultui/config.json` (or `$XDG_CONFIG_HOME/ultui/config.json`) applies
to **every** TUI you run. All keys optional; a missing file is fine. Run
`ultui config` to see the resolved path and effective values.

```jsonc
{
  "mod": "alt",                 // modifier for "mod+" bindings: alt | ctrl | ...
  "statusbar": true,            // bottom mode/hint bar (auto-hidden under 6 rows)
  "theme": "auto",              // auto | tokyo-night | gruvbox | slate | midnight
  "style": "rounded",           // rounded | double | heavy | ascii  (border + glyphs)
  "keybindings": { "mod+/": "help" }   // merged under every TUI's own bindings
}
```

`theme` is the colour palette (`""` / `dark` / `light` all mean auto-detect);
`style` is the box-drawing + glyph set (`ascii` for terminals without a Nerd
Font). Both are also switchable at runtime from the **`F2`** control panel, and
`ultui render <cfg> theme=NAME style=NAME` previews a combo headlessly.

### Mouse

On by default. **Click** a pane to focus it; **right-click** opens its context
menu (Stop · Restart · Refresh · Edit — the process rows grey out for a `text` /
`markdown` pane); **wheel** scrolls the pane under the cursor without moving
focus. The menu itself is keyboard-driven (`↑↓` pick, `Enter` run, `Esc` close),
since a modal swallows the mouse. Mouse events are not yet forwarded *into* `shell`
children, so `htop`'s own mouse doesn't work — keyboard does. Enabling mouse
reporting disables the terminal's native text selection while ultui runs.

## Run modes

| mode              | how                                                        |
|-------------------|-----------------------------------------------------------|
| runtime           | `ultui run config.json`                                    |
| headless frame    | `ultui render config.json 120x40` — for CI, docs, `vhs`    |
| compiled (static) | `go:embed` the config + link the engine — see `examples/compiled/` |
| compiled (dynamic)| same wrapper, read the config from a path at startup       |

All modes call the same `engine.RunWithOptions`. Widgets are a registry compiled
into the engine; there is no plugin/`dlopen` mechanism, which is what makes
static linking coherent.

## Development

```sh
make test                        # all tests (headless — no TTY needed)
make test-one T=TestFocusTraversalWraps
make vet fmt
make render EXAMPLE=examples/hello.json SIZE=100x30
```

Layout:

```
internal/config   schema + strict loader (Parse is the only format seam);
                  global runner config (global.go)
internal/widget   Widget interface + kind registry; text · markdown · command ·
                  shell — scrollBox = bubbles/viewport (scroll.go),
                  keyToBytes (keys.go)
internal/engine   layout math, bubbletea model, focus-lock + mouse routing,
                  overlays: F1 help / Ctrl+C confirm (overlay.go) + F2 control
                  panel (control.go) + theme picker (settings.go) + pane menu
                  (context.go) + pane/config editor (editor.go),
                  per-pane zoom math (layout.go), palettes + stylesets (theme.go),
                  status bar (chrome.go), bounded headless Snapshot
main.go           run / validate / render / config CLI
examples/         hello · dashboard · responsive · keys · sysmon · pretty ·
                  devdash · multiplex · guide · compiled/  — see examples/README.md
```

## Roadmap

Not built yet, roughly in priority order:

- Mouse forwarded *into* `shell` children (SGR sequences to the pty)
- Drag-resize splits with the mouse
- Terminal-query responses (DA/CPR) forwarded to `shell` children — blocked on
  an `x/vt` concurrency limitation
- Live/streaming and periodic `command` refresh
- More widgets: `list`, `input` / `form` (huh — the next Charm dep a widget will pull), `table`
- `ultui compile` — codegen the wrapper + `go build` in one step
- Shared borders between adjacent panes
- Public `pkg/` API once it stabilises
