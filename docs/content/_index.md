---
title: "toe"
---

# toe — Thom's Own Editor

**toe** is a modal terminal editor for Go development. toe edits Go projects, not the universe.

![toe screenshot](img/screenshot.png)

## Super Opinionated

toe is opinionated because it is built for one tight workflow: editing Go projects from a terminal without growing into a general-purpose IDE. It favors modal editing, `gopls`, TOML config, project-local state, Git-aware navigation, and a small set of deliberate defaults over plugin sprawl or endless knobs.

- Modal editing: normal, insert, and selection modes; multi-cursor editing; persistent undo history
- Project navigation: multiple buffers, split views, fuzzy file/buffer pickers, global search, and live previews
- Go-focused language tooling: syntax highlighting, LSP completion, hover, signature help, formatting, symbols, code actions, rename, go-to navigation, and diagnostics
- Editor display: soft wrap, rulers, whitespace rendering, indent guides, gutters, configurable cursor shapes, and statusline elements
- Version control: git diff gutters, change navigation, reset-diff-change, changed-file picker, and statusline element
- Project state: user/workspace TOML config, EditorConfig, session persistence, external file change detection, and clean-buffer reloads
- 4 Catppuccin themes: frappe, latte, macchiato, mocha

## Quick Start

```sh
make install        # install to $GOPATH/bin
toe path/to/file    # open a file
```

Press `i` to insert text. `Escape` returns to Normal mode. `:wq` saves and quits.

→ [Getting Started]({{< relref "/docs/getting-started" >}})
