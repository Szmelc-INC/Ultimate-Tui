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

# DESIGN REQUIREMENTS:
- Full interactivity. both keyboard & mouse controls.
- Full support for unicode, ascii, ansii, fonts (especially nerd mono).
- Full support for Shell (zsh) scripting & integration (Other languages like python & node should also be a no problem if needed).
- Full support for using other interactive shell commands & processes inside a TUI like in a terminal multiplexer.
- Support for animations, visualizations and other localized non static rendering.
- Full and total modularity! If something could be modular, exchangeable, expandable, configurable, easily improvable in future etc ~ it shall be.
- Well written documentations. For humans (README.md): Not overly verbose, but informative, clear and with pretty markdown formatting. + Separate doc, optimized for AI (LLM.md).
- Auxilary tooling is welcomed. (like Makefile with devel commands and setup scripts etc)

# Nice projects you can use or get inspired by:
> Not all of them are mandatory
- https://github.com/charmbracelet/huh
- https://github.com/charmbracelet/lipgloss
- https://github.com/charmbracelet/bubbletea
- https://github.com/charmbracelet/bubbles
- https://github.com/charmbracelet/glamour
- https://github.com/charmbracelet/gum
- https://github.com/charmbracelet/glow
- https://github.com/charmbracelet/vhs
- https://github.com/Gaurav-Gosain/tuios
- https://github.com/marcusolsson/tui-go
