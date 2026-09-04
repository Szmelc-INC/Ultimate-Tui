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
`settings` / `control` / `context-menu` / `lock` / `unlock` / `zoom-in` /
`zoom-out` / `zoom-reset` / `resize-up` / `resize-down` / `resize-left` /
`resize-right` / `swap-up` / `swap-down` / `swap-left` / `swap-right` /
`flip-split`. A `mod+` prefix expands to the modifier key (default **Alt** —
set by the global runner config, see below).

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
| `text`    | `content`                 | static text, scrolls if it overflows. Also the one widget that supports content zoom (`mod+=/-/0`) — big block-glyph rendering, not a resize |
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
| **NAV** (default) | `↑ ↓ ← →` move focus to the pane in that direction · `Tab` / `Shift+Tab` cycle · `Enter` **locks** focus into the pane · `mod+w` (or right-click) pane menu · `F1` help · `F2` this TUI's controls · `F3` global settings (NAV only) · `Ctrl+C` quit (asks first) |
| **LOCKED** | every key goes to the pane's widget (scroll a `text` pane, drive `htop`/`vim` in a `shell` pane) · `Esc` releases · `Esc Esc` sends a literal `Esc` to a `shell` child · `Ctrl+C` kills **this pane's** process (asks first), program keeps running |

`mod` is the modifier — **Alt** by default. Terminals cannot tell left Alt from
right Alt, so any Alt press counts. `mod+…`, `F1` and `F2` are caught even while
a pane is LOCKED (`F3` is not — it only opens from NAV, so a locked shell/vim
keeps its own F3). `Ctrl+C` is **not** a bare quit: in NAV it raises a `[Y/n]`
confirm dialog, and while LOCKED it belongs to the pane (kills just that
process, on confirm). The way out of a locked pane is `Esc` then `Ctrl+C`.

**Resize, swap, split orientation** — all `mod+`, so they work in either mode:

| keys | does |
|------|------|
| `mod+↑ ↓ ← →` | grow the focused pane toward that edge, taking the space from whichever sibling sits there |
| `mod+Shift+↑ ↓ ← →` | swap the focused pane with that same neighbor |
| `mod+s` | flip the focused pane's enclosing split between row / column |
| `mod+= / mod+- / mod+0` | **content** zoom — bigger/smaller/reset glyphs for the focused pane (currently `text` panes only; see below) — does **not** resize the pane |

Resize and content zoom are deliberately separate: put a pane full of tiny
print next to a pane with a couple of huge words, resize the tiny-print pane
bigger without touching its font, or blow up the other pane's text without
stealing an inch of screen from its neighbor.

**`F1`** opens a centered popup listing every active binding — the engine's plus
this TUI's own `keybindings`. **`F2`** opens the control panel for *this* TUI: a
**Theme & style** row opens the live picker (`↑↓` preview, `←→` switch column,
`Enter` applies and saves to the global config, `Esc` reverts), and an **Edit
config** row opens the editor over the whole config. **`F3`** opens the
**global** settings panel — the modifier key, the status bar toggle, a shortcut
into the same theme/style picker, and a read-only block of run info (config
path, global config path, resolved binary, pane count, mode); `mod+s` there
saves to the global runner config (see below) — this takes effect on the
*next* run, since `mod+` bindings are resolved once at startup.

**Context menu** — right-click a pane (or `mod+w` on the focused one): **Stop ·
Restart · Refresh · Split · Close · Edit** — the process rows grey out for a
`text` / `markdown` pane, and Close greys out on the last remaining pane.
Right-clicking empty space (the status bar row — the only spot the pane tree
doesn't already fill) opens a one-row **New pane** menu instead. Both **Split**
and **New** open the same prompt: pick a direction (side-by-side / stacked) and
how much of the space the original content keeps; confirming creates a fresh
blank `text` pane in the other half. **Close** asks first, then removes the
pane (a two-pane split collapses back to a single pane rather than leaving an
empty slot).

**Editor** — reached from the context menu's **Edit** row (one pane), or
**F2 → Edit config** (the whole file). A tab bar (`General` / `Layout` /
`Widget`, plus `Config` and `Keybinds` in whole-config mode) over typed
controls: pick-lists for `type` / `markdownStyle` / size mode, free text for
`title` / `content` / `command`, an on/off for `border`, a slider for a numeric
size. The **Keybinds** tab lists this TUI's own `keybindings`, one key + action
pair per entry — `ctrl+d` removes the entry under the cursor, the trailing
**+ add keybind** row appends a blank one. `↑↓` move between fields, `←→`
change a value, type to edit a text field, `Tab` switches tabs. **`Enter`**
applies the change to the running UI (an edited/added keybinding works
immediately, no restart); **`mod+s`** also writes it back to the config file
(`ultui run` only); **`Esc`** leaves, asking first if there are unsaved edits.
Changing `type` rebuilds the pane's widget (the old process, if any, is stopped).

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
Font). Both are also switchable at runtime from the **`F2`** control panel (or
**`F3`**), and `ultui render <cfg> theme=NAME style=NAME` previews a combo
headlessly. `F3` also edits `mod` and `statusbar` — saved changes there take
effect on the *next* run, not the one currently open.

### Mouse

On by default, everywhere — the pane tree *and* every overlay.

- **Click** a pane to focus it; **right-click** opens its context menu;
  **wheel** scrolls the pane under the cursor without moving focus.
- **`mod`+left-drag** grabs the pane under the cursor and, on release, swaps it
  with whatever pane it's dropped on. **`mod`+right-drag** grabs it to resize —
  from anywhere on the pane, drag any direction to resize that way, live as
  you drag: it grows the pane toward the edge you're dragging when there's a
  neighbor to grow into, and shrinks it from the opposite edge otherwise, so
  a pane flush against one side of the layout still resizes when dragged
  toward that side. Holding `mod` while right-clicking resizes instead of
  opening the context menu.
- **Inside any open overlay** (help, a confirm dialog, the context menu, F2,
  F3, the theme/style picker, the split prompt, the pane/config editor):
  clicking a row selects it (a menu row like Close or Restart runs
  immediately; a form field like a text box or checkbox just gets focus/
  toggles), clicking a tab switches it, wheel adjusts a slider or moves a
  list selection. **Clicking outside the panel dismisses it** — the same as
  pressing Esc, so it cancels/closes rather than falling through to whatever
  is behind it.

Mouse events are not yet forwarded *into* `shell` children, so `htop`'s own
mouse doesn't work inside a `shell` pane — keyboard does. Enabling mouse
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
                  identity-preserving tree rebuild (rebuild.go), resize /
                  swap / flip-split (resize.go), Split / Close / New pane
                  (split.go), mod+drag move/resize (drag.go),
                  overlays: F1 help / Ctrl+C confirm (overlay.go) + F2 control
                  panel (control.go) + F3 global settings (global.go) + theme
                  picker (settings.go) + pane menu (context.go) + pane/config
                  editor incl. Keybinds tab (editor.go), mouse routing for
                  every overlay (overlay_mouse.go), palettes + stylesets
                  (theme.go), status bar (chrome.go), bounded headless Snapshot
main.go           run / validate / render / config CLI
examples/         hello · dashboard · responsive · keys · sysmon · pretty ·
                  devdash · multiplex · guide · compiled/  — see examples/README.md
```

## Roadmap

Not built yet, roughly in priority order:

- Mouse forwarded *into* `shell` children (SGR sequences to the pty)
- Content zoom (`mod+=/-/0`) for widgets other than `text` — `markdown`/
  `command`/`shell` currently ignore it
- Terminal-query responses (DA/CPR) forwarded to `shell` children — blocked on
  an `x/vt` concurrency limitation
- Live/streaming and periodic `command` refresh
- More widgets: `list`, `input` / `form` (huh — the next Charm dep a widget will pull), `table`
- `ultui compile` — codegen the wrapper + `go build` in one step
- Shared borders between adjacent panes
- Public `pkg/` API once it stabilises
