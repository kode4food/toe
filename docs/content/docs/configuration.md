---
title: "Configuration"
weight: 20
---

# Configuration

## Config Files

toe reads config in this order (later values override earlier ones):

| File | Purpose |
|------|---------|
| `$XDG_CONFIG_HOME/toe/config.toml` | User config |
| `$XDG_CONFIG_HOME/toe/languages.toml` | User language config |
| `.toe/config.toml` | Workspace config (trusted workspaces only) |
| `.toe/languages.toml` | Workspace language config (trusted workspaces only) |

`$XDG_CONFIG_HOME` defaults to `~/.config`.

Open your user config directly: `:config-open`
Open workspace config: `:config-open-workspace`
Reload after editing: `:config-reload`

## Interface Language

toe selects English, German, French, or Italian from `LC_ALL`, `LC_MESSAGES`, then `LANG`. Unsupported locales use English.

## Workspace Trust

toe treats a directory with `.git` or `.toe` as a workspace. Workspace trust is the explicit opt-in that lets workspace-controlled config and tooling affect toe.

Until a workspace is trusted:

- normal file editing, `:write`, `:write-all`, and `:move` still work
- automatic workspace session restore/save is skipped
- workspace-local config files (`.toe/config.toml` and `.toe/languages.toml`) are not loaded
- workspace-configured language servers and formatter commands are not started from workspace config
- `:config-open-workspace` refuses to create or open workspace config until you trust the workspace

User config still applies in untrusted workspaces.

Trust the current workspace:

```
:workspace-trust
```

Remove trust:

```
:workspace-untrust
```

Trusted workspaces are stored in `$XDG_DATA_HOME/toe/trusted_workspaces` (normally `~/.local/share/toe/trusted_workspaces`). If a workspace is untrusted at startup, toe shows a status message asking you to run `:workspace-trust`.

To bypass trust checks entirely, set in your user config:

```toml
[editor]
insecure = true
```

## Editor Configuration

Except for the top-level `theme` key, the settings below belong under `[editor]` in `config.toml`. They can also be changed for the current session with `:set <key> <value>`, `:get <key>`, and `:toggle <key>` for booleans. Lists and tables use TOML syntax. Completion after `:set ` shows the available keys.

Examples:

```text
:set gutters.layout ["diagnostics", "line-numbers", "diff"]
:set statusline.left ["mode!", "file-name"]
:set whitespace.render.tab all
:set whitespace.characters.tab "→"
:set auto-pairs {"(" = ")", "[" = "]"}
:set buffer-picker.start-position previous
:toggle file-explorer.hidden
```

### Theme

```toml
theme = "mocha"   # frappe | latte | macchiato | mocha
```

### General

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `mouse` | bool | `true` | Enable mouse support |
| `middle-click-paste` | bool | `true` | Paste on middle-click |
| `insecure` | bool | `false` | Disable workspace trust checks |
| `editor-config` | bool | `true` | Respect `.editorconfig` files |
| `auto-session` | bool | `true` | Save/restore session automatically |
| `nerd-fonts` | bool | `true` | Use Nerd Font icons in pickers and completion |
| `default-line-ending` | string | (system) | `lf`, `crlf`, or `native` |

### Display

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `line-number` | string | `"absolute"` | `absolute` or `relative` |
| `cursorline` | bool | `true` | Highlight cursor line |
| `cursorcolumn` | bool | `false` | Highlight cursor column |
| `text-width` | int | `80` | Text width (used by rulers and reflow) |
| `rulers` | int[] | `[]` | Column ruler positions, e.g. `[80, 120]` |
| `bufferline` | string | `"never"` | Show buffer tabs: `never`, `always`, `multiple` |

### Soft Wrap

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `soft-wrap.enable` | bool | `false` | Enable soft wrap |
| `soft-wrap.max-wrap` | int | `20` | Maximum visual indentation when wrapping |
| `soft-wrap.max-indent-retain` | int | `40` | Max indent levels to retain |
| `soft-wrap.wrap-indicator` | string | `"↪ "` | Continuation indicator |
| `soft-wrap.wrap-at-text-width` | bool | `false` | Wrap at `text-width` instead of window width |

### Whitespace and Indentation

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `whitespace.render` | string/table | `"none"` | `none` or `all`, globally or by whitespace type |
| `indent-guides.render` | bool | `false` | Show indent guides |
| `indent-guides.character` | string | `"│"` | Guide character |
| `indent-guides.skip-levels` | int | `0` | Indent levels to skip |
| `gutters.layout` | string[] | built-in | Ordered list of `diagnostics`, `line-numbers`, `diff`, and `spacer` gutters |
| `gutters.line-numbers.min-width` | int | `3` | Minimum gutter width |

Whitespace rendering can be set separately for `space`, `nbsp`, `nnbsp`, `tab`, and `newline`, and each display character can be replaced:

```toml
[editor.whitespace]
render = { default = "none", tab = "all", newline = "all" }

[editor.whitespace.characters]
space = "·"
nbsp = "⍽"
nnbsp = "␣"
tab = "→"
tabpad = " "
newline = "⏎"
```

Use `whitespace.render.<type>` and `whitespace.characters.<type>` to change individual values at runtime.

To change gutter order or visibility:

```toml
[editor.gutters]
layout = ["diagnostics", "spacer", "line-numbers", "spacer", "diff"]

[editor.gutters.line-numbers]
min-width = 3
```

### Editing

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `auto-pairs` | bool/table | `true` | Auto-insert closing brackets and quotes |
| `continue-comments` | bool | `true` | Extend comment tokens on new lines |
| `atomic-save` | bool | `true` | Write via temp file to prevent partial writes |
| `insert-final-newline` | bool | `true` | Ensure file ends with a newline |
| `trim-final-newlines` | bool | `false` | Remove trailing blank lines on save |
| `trim-trailing-whitespace` | bool | `false` | Remove trailing spaces on save |

`auto-pairs` also accepts a table of custom opening and closing characters:

```toml
auto-pairs = { "(" = ")", "[" = "]", "{" = "}" }
```

At runtime, `auto-pairs` accepts a boolean or an inline table of custom pairs.

### Auto-Save

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `auto-save` | bool | `false` | Save when focus is lost (alias for `auto-save.focus-lost`) |
| `auto-save.focus-lost` | bool | `false` | Save when focus leaves the view |
| `auto-save.after-delay.enable` | bool | `false` | Save after idle delay |
| `auto-save.after-delay.timeout` | int | `3000` | Idle delay in milliseconds |

### Search

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `search.smart-case` | bool | `true` | Case-insensitive unless pattern has uppercase |
| `search.wrap-around` | bool | `true` | Wrap search at end of file |

### Scrolling

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `scrolloff` | int | `5` | Lines of context kept above/below cursor |
| `scroll-lines` | int | `3` | Lines moved per scroll step |

### Cursor Shape

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `cursor-shape.normal` | string | `"block"` | `block`, `bar`, `underline`, or `hidden` |
| `cursor-shape.insert` | string | `"bar"` | Cursor shape in Insert mode |
| `cursor-shape.select` | string | `"underline"` | Cursor shape in Select mode |

### Status Bar

| Config key | Type | Default | Description |
|--------|------|---------|-------------|
| `statusline.left` | string[] | built-in | Left-aligned statusline elements |
| `statusline.right` | string[] | built-in | Right-aligned statusline elements |
| `statusline.separator` | string | `"│"` | Separator between status items |
| `statusline.mode.normal` | string | `"NOR"` | Label for Normal mode |
| `statusline.mode.insert` | string | `"INS"` | Label for Insert mode |
| `statusline.mode.select` | string | `"SEL"` | Label for Select mode |

Valid statusline elements: `mode`, `file-name`, `file-base-name`, `file-absolute-path`, `file-modified-indicator`, `read-only-indicator`, `file-encoding`, `file-line-ending`, `file-indent-style`, `file-type`, `diagnostics`, `selections`, `primary-selection-length`, `position`, `position-percentage`, `total-line-numbers`, `separator`, `spacer`, `spinner`, `register`, and `version-control`.

When the pane is too narrow, toe drops unpinned status items from the right section and then the left. Suffix an element with `!` (for example `"mode!"` or `"position!"`) to keep it visible.

```toml
[editor.statusline]
left = ["mode!", "file-name", "read-only-indicator", "file-modified-indicator"]
right = ["diagnostics", "selections", "register", "position!"]
```

Example:

```toml
[editor.statusline]
left = ["mode", "file-name", "file-modified-indicator"]
right = ["version-control", "diagnostics", "position"]
```

### Pickers

Picker split ratios can be changed at runtime with commands such as `:set editor.picker.split-ratios.diagnostics 0.65` and are saved by auto-session.

| TOML key | Type | Default | Description |
|----------|------|---------|-------------|
| `picker.split-ratios` | table | `{}` | Picker list/preview split ratios by picker id, from `0.2` to `0.8`; missing pickers use `0.5` |
| `buffer-picker.start-position` | string | `"top"` | `top` or `previous` |
| `file-explorer.hidden` | bool | `false` | Show hidden files |
| `file-explorer.follow-symlinks` | bool | `false` | Follow symlinks |
| `file-explorer.parents` | bool | `false` | Include parent directories |
| `file-explorer.ignore` | bool | `false` | Respect `.ignore` files |
| `file-explorer.git-ignore` | bool | `false` | Respect `.gitignore` |
| `file-explorer.git-global` | bool | `false` | Respect global gitignore |
| `file-explorer.git-exclude` | bool | `false` | Respect git exclude rules |
| `file-explorer.flatten-dirs` | bool | `true` | Collapse single-child directories |

### Shell

```toml
[editor]
shell = ["sh", "-c"]   # default on Unix; ["cmd", "/C"] on Windows
```

## Config Example

```toml
theme = "mocha"

[editor]
cursorline = true
soft-wrap.enable = true
auto-session = true
rulers = [80, 120]

[editor.cursor-shape]
normal = "block"
insert = "bar"
```

## Language Config

See [Language Servers]({{< relref "/docs/lsp" >}}) for `languages.toml` configuration.
