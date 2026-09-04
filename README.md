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
│q:   quit                 ││    PID   RSS COMMAND                                              │
└──────────────────────────┘└──────────────────────────────────────────────────────────────────┘
```

## Config format

A config is **JSON** (DESIGN.md's "markdown json" resolved to one strict
decoder; a second format is a decoder swap away). Unknown keys are rejected so
typos surface.

```jsonc
{
  "name": "system-dashboard",
  "keybindings": { "1": "focus-next" },   // merged over the defaults
  "prefix": "ctrl+b",                     // leader key for interactive panes
  "layout": { /* a node */ }
}
```

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
| `command` | `command`, `shell` (opt)  | runs the command **once**, shows combined stdout+stderr, scrollable |
| `shell`   | `command`, `shell` (opt)  | a **real interactive process** on a pty — `htop`, `vim`, `cmatrix`, `$SHELL`. Output goes through a VT emulator; keystrokes are forwarded to the child |

`command` captures text; `shell` *is* a terminal. Pick `shell` when the program
draws its own screen or needs input.

### Keys

Defaults: `tab` / `shift+tab` focus next/prev, `q` / `ctrl+c` quit. Scroll keys
(`j`/`k`, arrows, `pgup`/`pgdn`, `g`) go to the focused widget. `keybindings` in
the config remap any key to `quit` / `focus-next` / `focus-prev`.

**Interactive (`shell`) panes** take over the keyboard while focused so the child
program gets everything — including `q` and `ctrl+c`. To reach ultui's own keys,
press the **prefix** first (`"prefix"` in the config, default `ctrl+b`), tmux
style:

| keys | effect |
|------|--------|
| `<prefix>` then `tab` / `h` / `l` / `q` … | run that key's action |
| `<prefix>` `<prefix>` | send a literal prefix byte to the child |

The focused interactive pane shows `⌨ PREFIX` in its title while it waits for the
second key.

### Mouse

On by default. **Click** a pane to focus it; **wheel** scrolls the pane under the
cursor without moving focus. Mouse events are not yet forwarded *into* `shell`
children, so `htop`'s own mouse doesn't work — keyboard does. Enabling mouse
reporting disables the terminal's native text selection while ultui runs.

## Run modes

| mode              | how                                                        |
|-------------------|-----------------------------------------------------------|
| runtime           | `ultui run config.json`                                    |
| headless frame    | `ultui render config.json 120x40` — for CI, docs, `vhs`    |
| compiled (static) | `go:embed` the config + link the engine — see `examples/compiled/` |
| compiled (dynamic)| same wrapper, read the config from a path at startup       |

All modes call the same `engine.Run`. Widgets are a registry compiled into the
engine; there is no plugin/`dlopen` mechanism, which is what makes static
linking coherent.

## Development

```sh
make test                        # all tests (headless — no TTY needed)
make test-one T=TestFocusTraversalWraps
make vet fmt
make render EXAMPLE=examples/hello.json SIZE=100x30
```

Layout:

```
internal/config   schema + strict loader (Parse is the only format seam)
internal/widget   Widget interface + kind registry; text · command · shell
                  scrollBox (scroll.go), keyToBytes (keys.go)
internal/engine   layout math, bubbletea model, prefix + mouse routing,
                  bounded headless Snapshot
main.go           run / validate / render CLI
examples/         hello · dashboard · responsive · keys · sysmon · pretty ·
                  devdash · multiplex · compiled/   —  see examples/README.md
```

## Roadmap

Not built yet, roughly in priority order:

- Mouse forwarded *into* `shell` children (SGR sequences to the pty)
- Drag-resize splits with the mouse
- Terminal-query responses (DA/CPR) forwarded to `shell` children — blocked on
  an `x/vt` concurrency limitation
- Live/streaming and periodic `command` refresh
- More widgets: `markdown` (glamour), `list`, `viewport`, `input`
- `ultui compile` — codegen the wrapper + `go build` in one step
- Shared borders between adjacent panes
- Public `pkg/` API once it stabilises
