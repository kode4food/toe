---
title: "toe"
---

# toe — Thom's Own Editor

**toe** is a modal terminal editor for Go development. toe edits Go projects, not the universe.

![toe screenshot](img/screenshot.png)

## Super Opinionated

- Modal editing with normal, insert, and selection modes
- Multiple buffers, split views, fuzzy pickers, and workspace search
- Go-focused LSP support via `gopls`, plus web language servers
- Diagnostics, syntax highlighting, formatting, and symbol navigation
- Git diff gutters, change navigation, changed-file picker, and statusline element
- Session persistence, external file reloads, user/workspace TOML config, and EditorConfig
- 4 Catppuccin themes: frappe, latte, macchiato, mocha

## Quick Start

```sh
make install        # install to $GOPATH/bin
toe path/to/file    # open a file
```

Press `i` to insert text. `Escape` returns to Normal mode. `:wq` saves and quits.

→ [Getting Started]({{< relref "/docs/getting-started" >}})
