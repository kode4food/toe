---
title: "Language Servers"
weight: 50
---

# Language Servers

toe is a Go editor. LSP support is built around `gopls` for Go, with additional servers available for web languages (TypeScript, HTML, CSS) and other tools in the Go ecosystem. Each language specifies which server to use; you configure servers in your `languages.toml`.

## Configuring a Language Server

Add a `[language-server.<name>]` section and reference it from the language:

```toml
# $XDG_CONFIG_HOME/toe/languages.toml

[language-server.gopls]
command = "gopls"
args = ["-remote=auto"]          # optional
environment = { GOFLAGS = "-mod=mod" }  # optional
timeout = 30                     # optional, seconds

[[language]]
name = "go"
language-servers = ["gopls"]
```

## Workspace Config

Language server config in `.toe/languages.toml` is merged on top of user config, letting you override per-project without touching global settings.

## Available LSP Features

| Feature | Key | Command |
|---------|-----|---------|
| Go to definition | `gd` | `goto_definition` |
| Go to declaration | `gD` | `goto_declaration` |
| Go to type definition | `gy` | `goto_type_definition` |
| Go to implementation | `gi` | `goto_implementation` |
| Go to references | `gr` | `goto_reference` |
| Select all references | `Space+h` | `select_references_to_symbol_under_cursor` |
| Hover docs | `Space+k` | `hover` |
| Rename symbol | `Space+r` | `rename_symbol` |
| Code actions | `Space+a` | `code_action` |
| Format document | `=` | `format_selections` |
| Signature help | (auto in Insert) | `signature-help` |
| Completion | `Ctrl+x` (or auto) | `completion` |
| Document symbols | `Space+s` | `symbol_picker` |
| Workspace symbols | `Space+S` | `workspace_symbol_picker` |

Diagnostics (errors and warnings) appear as underlines in the document, markers in the gutter, counts in the status bar, and a popup when the cursor rests on a diagnostic.

## Restarting Servers

```
:lsp-restart             restart all servers for the current document
:lsp-restart gopls       restart a specific server
:lsp-stop                stop all servers
```

## Formatter

You can also configure a standalone formatter (runs when `auto-format = true`
or when you invoke `=`):

```toml
[[language]]
name = "go"
auto-format = true
formatter = { command = "gofmt" }
```

If a language server and formatter are both configured, the language server's
formatting is used.
