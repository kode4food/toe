---
title: "Language Servers"
weight: 50
---

# Language Servers

toe uses `gopls` for Go language features. Other language servers can be selected and configured in `languages.toml`; each server must be installed on your `PATH`.

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
| Go to definition | `gd` | `goto-definition` |
| Go to declaration | `gD` | `goto-declaration` |
| Go to type definition | `gy` | `goto-type-definition` |
| Go to implementation | `gi` | `goto-implementation` |
| Go to references | `gr` | `goto-reference` |
| Select all references | `Space+h` | `select-references-to-symbol-under-cursor` |
| Hover docs | `Space+k` | `hover` |
| Rename symbol | `Space+r` | `rename-symbol` |
| Code actions | `Space+a` | `code-action` |
| Format selection | `=` | `format-selections` |
| Format document | `:format` | `format` |
| Signature help | (auto in Insert) | `signature-help` |
| Completion | `Ctrl+x` (or auto) | `completion` |
| Document symbols | `Space+s` | `symbol-picker` |
| Workspace symbols | `Space+S` | `workspace-symbol-picker` |

Workspace symbol searches query every running language server, not only the server for the focused document.

Diagnostics (errors and warnings) appear as underlines in the document, markers in the gutter, counts in the status bar, and a popup when the cursor rests on a diagnostic.

## Restarting Servers

```
:lsp-restart             restart all servers for the current document
:lsp-restart gopls       restart a specific server
:lsp-stop                stop all servers
```

## Formatter

You can also configure a standalone formatter (runs when `auto-format = true` or when you invoke `=`):

```toml
[[language]]
name = "go"
auto-format = true
formatter = { command = "gofmt" }
```

If a language server and formatter are both configured, the language server's formatting is used.
