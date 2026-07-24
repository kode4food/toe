---
title: "toe"
---

# toe — Thom's Own Editor

**toe** is a modal terminal editor for Go development. toe edits Go projects, not the universe.

![toe screenshot](img/screenshot.png)

## Super Opinionated

toe is opinionated because it is built for one tight workflow: editing Go projects from a terminal without growing into a general-purpose IDE. It favors modal editing, `gopls`, TOML config, project-local state, Git diff gutters and a changed-file picker, and a small set of deliberate defaults over plugin sprawl or endless knobs.

- Modal editing: normal, insert, and selection modes; multi-cursor editing; undo and redo
- Project navigation: multiple buffers, split views, fuzzy file/buffer pickers, global search, file and diff previews, image panes, and an integrated terminal pane
- Go-focused language tooling: syntax highlighting, LSP completion, hover, signature help, formatting, symbols, code actions, rename, go-to navigation, and diagnostics
- Editor display: soft wrap, rulers, whitespace rendering, indent guides, gutters, configurable cursor shapes, and statusline elements
- Version control: git diff gutters, changed-hunk navigation and reset, and a changed-file picker with unified diff previews
- Project state: workspace trust, user/workspace TOML config, EditorConfig, session persistence, external file change detection, and clean-buffer reloads
- 4 Catppuccin themes: frappe, latte, macchiato, mocha

## Quick Start

```sh
make install        # install to $GOPATH/bin
toe path/to/file    # open a file
```

Press `i` to insert text. `Escape` returns to Normal mode. `:wq` saves and quits.

→ [Getting Started]({{< relref "/docs/getting-started" >}})

## Acknowledgements

toe is possible because of excellent terminal UI, parsing, syntax highlighting, and theme projects:

- [Christian Rocha](https://github.com/meowgorithm) and the [Charm team](https://charm.land/) for [Bubble Tea](https://github.com/charmbracelet/bubbletea), which gives toe its TUI runtime, input handling, and terminal output
- [Max Brunsfeld](https://github.com/maxbrunsfeld) and the [Tree-sitter project](https://tree-sitter.github.io/tree-sitter/) for the incremental parsing stack, official Go bindings, and grammars behind toe's Tree-sitter highlighting
- [Alec Thomas](https://github.com/alecthomas) and the [Chroma project](https://github.com/alecthomas/chroma), the pure-Go syntax highlighter toe uses as its highlighting fallback
- [Pocco](https://github.com/pocco81) and the [Catppuccin project](https://catppuccin.com/) for the Latte, Frappe, Macchiato, and Mocha palettes. toe ships only Catppuccin themes because I love them and I don't care if you don't ;-)
