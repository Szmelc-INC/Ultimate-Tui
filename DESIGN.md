# ULTIMATE TUI FRAMEWORK
> A Ultimate config driven TUI/CLI design & creator framework in Go!
This project is meant to create a robust, universal, quick and simple way for user to write, design, create and operate a Text User Interfaces for any need, application and use case.

### STACK: 
- Built with: `Go` + `markdown json` 
- Works with: `zsh/bash python node html/js/css php yaml xml`
### ENV:
- [OS]: Arch Linux
- [WM]: Hyprland (Wayland)
- [SHELL]: Zsh

# FEATURES:
- [Config driven] - Every TUI exists as it's config file with it's all parameters and functionality, it is human readable and can be easily modified & expanded in future. (configs can either be split between separate config files each something else, or consolidated into one massive file with all sub-configs as sections).
- [Works as Runtime & Compiled] - TUI's can work both in runtime (specify path to config file to start it up), as well as compiled into executable. Compilation has two modes of linking: dynamic (core engine + config), or static (bakes the core engine with a specified config & modules and dependencies required by the config).
- [Multiplexing] - Dividing, resizing, positioning, and managing all the available space freely, (something like a modern tmux but already configured and on steroids).

# DESIGN:
- Full interactivity. both keyboard & mouse controls.
- Full support for unicode, ascii, ansii, fonts (especially nerd mono).
- Full support for Shell (zsh) scripting & integration (Other languages like python & node should also be a no problem if needed).
- Full support for using other interactive shell commands & processes inside a TUI like in a terminal multiplexer.
- Support for animations, visualizations and other localized non static rendering.
- Full and total modularity! If something could be modular, exchangeable, expandable, configurable, easily improvable in future etc ~ it shall be.
- Well written documentations. For humans (README.md): Not overly verbose, but informative, clear and with pretty markdown formatting. + Separate doc, optimized for AI (LLM.md).
- Auxilary tooling is welcomed. (like Makefile with devel commands and setup scripts etc)

# Nice projects you can use or get inspired by:
> Those are reccomended to use projects, NOT a mandatory requirement. But they are well made and worthy of use. You can integrate all, some, or none ~ your call.
- https://github.com/charmbracelet/huh
- https://github.com/charmbracelet/lipgloss
- https://github.com/charmbracelet/bubbletea
- https://github.com/charmbracelet/bubbles
- https://github.com/charmbracelet/glamour
- https://github.com/charmbracelet/gum
- https://github.com/charmbracelet/glow
- https://github.com/charmbracelet/vhs
- https://github.com/hsaeed3/xnano
- https://github.com/fdehau/tui-rs
- https://github.com/Gaurav-Gosain/tuios
- https://github.com/marcusolsson/tui-go
- fzf
- ncurses

### Inspiration refferences:
- cli agentic harnesses (like claude code cli, codex cli, opencode etc)
- midnight commander (and other CLI/TUI commanders)
- lazy nvim & lazy git
- Archinstall

---

# GENERAL AGENT RULES:
1. If you are unsure about something important -> Hold up for a sec to ask user to clarify. If you are unsure about something not so important -> Use best effort guess & instinct to decide the best fitting action.
1. You should understand idea, concept and follow design notes closely, just as long as they make sense ~ If something about your instructions is impossible/false/difficult/pointless/dumb/inefficient/wastefull -> Do it your way if you know how to do something better, fullfilling user's wish as closely as possible at the same time.
2. You have full design, technical & creative control over the task & project, as long as you are gonna deliver at least what was requested or that + more. If you know how and can make something 'your way', then go ahead, as long as it's what user would want, even if initial instructions were different.
3. If you feel like something should be made -> do it, even if there was no mention about it in instructions.
4. Be carefull, you are running on live production system without a sandbox! Please don't nuke it. Always remember to use git, if there is a repo -> create or update branch, then ask if pr to main. if no repo detected, create local via git, and remember to use it.

# SOFTWARE DESIGN RULES:
> **The principles of thoughtful software design come down to common sense and minimalism:**
1. Simplicity and Logic: Create solutions that are simple and logical. Build exactly what is required with zero overhead.
2. Avoid Over-engineering: Do not overcomplicate things, and avoid using monolithic frameworks for trivial features.
3. Minimize Dependencies: If a heavy library is needed for just one small task, write the code yourself or cherry-pick the required snippet.
4. Question Your Code: Continuously ask yourself if the solution is overly complex. If it is, evaluate whether refactoring is worth the effort.
5. Simplicity is Elegance: Simple does not mean superficial. True elegance is straightforward.**
