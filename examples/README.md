# Example configs

Run any of these live:

```sh
make build
./bin/ultui run examples/<name>.json
```

…or print a single frame with no TTY (great for diffing and screenshots):

```sh
./bin/ultui render examples/<name>.json 120x36
make render EXAMPLE=examples/<name>.json SIZE=120x36
```

| config | what it demonstrates | try it at |
|--------|----------------------|-----------|
| **hello.json** | the smallest possible TUI — one `text` pane | `80x20` |
| **dashboard.json** | nested `row` / `column` / `row` splits, `command` panes with live `ps` / `df` | `100x28` |
| **responsive.json** | the sizing model side by side: `25%` fraction, `30` fixed cells, `*` flex; borderless header + footer strips. Reflows to any size — run it at three. | `70x20`, `120x30`, `200x50` |
| **keys.json** | config-driven interactivity: custom `keybindings` (`h`/`l`, arrows, `ctrl+n`/`ctrl+p` → focus), per-pane scrolling (`j`/`k`, `pgdn`, `g`), focus ring with the highlighted border | `120x28` |
| **sysmon.json** | dense real-world composition — CPU / memory / disk / net / processes in one grid, each a shell snapshot taken at launch. ANSI colour from `ip -c` passes straight through. | `130x34` |
| **pretty.json** | pretty-print: truecolor `figlet | lolcat` banner, 256-colour and truecolor ramps, `bat` syntax highlighting, `cowsay`, plus unicode box-drawing / blocks / braille in a plain `text` pane | `120x36` |
| **devdash.json** | a dev cockpit for this repo: `git status` + `git log --graph` (colour), `go test ./...`, an `eza`/`tree` source tree, and a TODO/FIXME grep | `130x40` |
| **multiplex.json** | the multiplexer: `htop`, an interactive `$SHELL`, and `cmatrix` each in a live `shell` pane. Click to focus, wheel to scroll; inside a pane press `ctrl+b` then `h`/`l`/`q`. **Use `ultui run`** — under `ultui render` the panes look empty because htop/cmatrix paint on a timer and the headless drain stops after ~250 ms; that is not a bug. | run it |
| **compiled/** | the `go:embed` static build — `go build -o baked ./examples/compiled`, then `./baked` needs no config file | — |

## Notes / current limits

- **`command` panes run once, at launch.** There is no live refresh yet — re-run
  to update. Streaming and periodic reload are on the roadmap. Use a **`shell`**
  pane for anything that redraws itself (`watch`, `htop`, `top -d1`).
- **`shell` panes are real terminals.** They forward the keyboard to the child,
  so use the prefix (`ctrl+b`) then a nav key to get back to ultui. Mouse is not
  forwarded into the child yet. Apps that block waiting for a terminal-query
  reply may misbehave; `htop`/`vim`/`less`/`cmatrix`/`btop` are fine.
- **ANSI colour survives** in `command` output; force it on (`--color=always`,
  `CLICOLOR_FORCE=1`, `lolcat -f`) since the child's stdout is a pipe, not a tty.
  Very long coloured lines can wrap mid-escape and look slightly ragged.
- **ANSI escapes cannot be written into a `text` pane's `content`** — JSON has no
  `\e`/`\033`. Use unicode for static pretty-print, or a `command` pane
  (`printf '\\033[..'`) for colour.
- **ASCII-art / figlet panes need to be at least as wide as the art** or lines
  wrap. Keep banner text short.
- Nerd-font glyphs (`eza --icons`, the ` `/` ` in titles) need a Nerd Font in
  your terminal.
