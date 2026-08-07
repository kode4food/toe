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

## Available Language Server Features

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

Completion is requested three ways: `Ctrl+x` at any time, automatically after a trigger character the language server advertises (`.` for most servers), and automatically once typing pauses — see [Completion]({{< relref "/docs/configuration" >}}#completion) for the delay and trigger length.

Diagnostics (errors and warnings) appear as underlines in the document, markers in the gutter, counts in the status bar, and a popup when the cursor rests on a diagnostic.

## Completion Kinds

The completion popup marks each candidate with its kind, colored by the theme scope in the last column. Terminals without a Nerd Font show the short label instead:

| Icon | Kind | Plain | Theme scope |
|------|------|-------|-------------|
| {{< glyph "symbol-key" "kind-string" >}} | `text` | `Txt` | `string` |
| {{< glyph "symbol-method" "kind-function" >}} | `function` | `Fun` | `function` |
| {{< glyph "symbol-method" "kind-function" >}} | `method` | `Mth` | `function` |
| {{< glyph "symbol-method" "kind-function" >}} | `constructor` | `Ctr` | `function` |
| {{< glyph "symbol-field" "kind-variable-other-member" >}} | `field` | `Fld` | `variable.other.member` |
| {{< glyph "symbol-variable" "kind-variable-parameter" >}} | `variable` | `Var` | `variable.parameter` |
| {{< glyph "symbol-class" "kind-type" >}} | `class` | `Cls` | `type` |
| {{< glyph "symbol-interface" "kind-type" >}} | `interface` | `Ifc` | `type` |
| {{< glyph "symbol-namespace" "kind-namespace" >}} | `module` | `Mod` | `namespace` |
| {{< glyph "symbol-property" "kind-variable-other-member" >}} | `property` | `Prp` | `variable.other.member` |
| {{< glyph "symbol-ruler" "kind-type" >}} | `unit` | `Unt` | `type` |
| {{< glyph "symbol-enum" "kind-constant" >}} | `value` | `Val` | `constant` |
| {{< glyph "symbol-enum" "kind-constant" >}} | `enum` | `Enm` | `constant` |
| {{< glyph "symbol-keyword" "kind-keyword" >}} | `keyword` | `Kwd` | `keyword` |
| {{< glyph "symbol-snippet" "kind-string" >}} | `snippet` | `Snp` | `string` |
| {{< glyph "symbol-color" "kind-string" >}} | `color` | `Clr` | `string` |
| {{< glyph "file" "kind-string" >}} | `file` | `Fil` | `string` |
| {{< glyph "go-to-file" "kind-variable-other-member" >}} | `reference` | `Ref` | `variable.other.member` |
| {{< glyph "folder" "kind-string" >}} | `folder` | `Dir` | `string` |
| {{< glyph "symbol-constant" "kind-constant" >}} | `constant` | `Cst` | `constant` |
| {{< glyph "symbol-structure" "kind-type" >}} | `struct` | `Sct` | `type` |
| {{< glyph "symbol-event" "kind-string" >}} | `event` | `Evt` | `string` |
| {{< glyph "symbol-operator" "kind-operator" >}} | `operator` | `Opr` | `operator` |
| {{< glyph "symbol-parameter" "kind-type" >}} | `type param` | `Tpm` | `type` |
| {{< glyph "symbol-enum-member" "kind-constant" >}} | `enum member` | `Emb` | `constant` |

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
