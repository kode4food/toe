---
title: "toe"
---

# toe — Thom's Own Editor

**toe** is a modal terminal editor for Go development. toe edits Go projects, not the universe.

- Normal, insert, and selection modes
- Multiple buffers and split views
- Fuzzy file, buffer, and global search pickers with live preview
- Syntax highlighting via tree-sitter
- Persistent undo history with branching
- LSP support via `gopls` (plus TypeScript, HTML, CSS)
- Session persistence (open files, layout, cursor positions)
- 4 Catppuccin themes: frappe, latte, macchiato, mocha
- User and workspace config in TOML
- EditorConfig support

## Quick Start

```sh
make install        # install to $GOPATH/bin
toe path/to/file    # open a file
```

Press `i` to insert text. `Escape` returns to Normal mode. `:wq` saves and quits.

→ [Getting Started]({{< relref "/docs/getting-started" >}})
