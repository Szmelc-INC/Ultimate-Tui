# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

Pre-implementation. The repo currently holds only design documents — `DESIGN.md` (the vision) and `optimal-mode.md` (a token-budget ruleset). There is no Go module, no source tree, no build. The first substantive task is scaffolding the framework described in `DESIGN.md`.

Because nothing is built yet, there are no build/lint/test commands to document. When you create the module, add a `Makefile` with the dev commands (`DESIGN.md` explicitly asks for auxiliary tooling like a Makefile and setup scripts) and update this file with the real invocations, including how to run a single test.

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
- **Use git.** No repo exists yet — create one (`git init`) before making changes, work on a branch, and ask before merging to main.
- **Minimalism over frameworks.** Build exactly what's required with zero overhead. Don't pull a heavy dependency for one small task — write the snippet or cherry-pick it. Continuously question whether a solution is over-complex.
- **Design authority.** Follow `DESIGN.md`'s intent closely *where it makes sense*. If an instruction is impossible, wrong, or wasteful, do it the better way while still delivering at least what was asked. If something clearly should exist, build it even if unmentioned.
- **Clarify only high-stakes unknowns.** For low-stakes ambiguity, make a best-effort call and proceed.

## Token discipline

`optimal-mode.md` is an opt-in "eco mode" ruleset: pinpoint context with `rg`/`fd`/`sed` before reading; never `cat` a >250-line file just for context; keep reasoning effort low unless the task genuinely needs more; terminal-style output with zero filler; take the explicit `[Bypass: requires deep context]` escape hatch rather than looping on retries.
