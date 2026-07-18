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

Open your user config directly: `:config_open`
Open workspace config: `:config_open_workspace`
Reload after editing: `:config_reload`

## Workspace Trust

toe treats a directory with `.git` or `.toe` as a workspace. Workspace trust is the explicit opt-in that lets workspace-controlled config and tooling affect toe.

Until a workspace is trusted:

- normal file editing, `:write`, `:write_all`, and `:move` still work
- automatic workspace session restore/save is skipped
- workspace-local config files (`.toe/config.toml` and `.toe/languages.toml`) are not loaded
- workspace-configured language servers and formatter commands are not started from workspace config
- `:config_open_workspace` refuses to create or open workspace config until you trust the workspace

User config still applies in untrusted workspaces.

Trust the current workspace:

```
:workspace_trust
```

Remove trust:

```
:workspace_untrust
```

Trusted workspaces are stored in `$DATA_DIR/trusted_workspaces` (`~/.local/share/toe/trusted_workspaces` on Linux/macOS). If a workspace is untrusted at startup, toe shows a status message asking you to run `:workspace_trust`.

To bypass trust checks entirely, set in your user config:

```toml
[editor]
insecure = true
```

## Editor Options

Options can be changed at runtime with `:set <key> <value>`, `:get <key>`,
and `:toggle <key>` (for booleans).

### Theme

```toml
theme = "frappe"   # frappe | latte | macchiato | mocha
```

### General

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mouse` | bool | `true` | Enable mouse support |
| `middle-click-paste` | bool | `true` | Paste on middle-click |
| `insecure` | bool | `false` | Disable workspace trust checks |
| `editor-config` | bool | `true` | Respect `.editorconfig` files |
| `auto-session` | bool | `true` | Save/restore session automatically |
| `nerd-fonts` | bool | `true` | Use Nerd Font icons in pickers and completion |
| `default-line-ending` | string | (system) | `lf`, `crlf`, or `native` |

### Display

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `line-number` | string | `"absolute"` | `absolute` or `relative` |
| `cursorline` | bool | `false` | Highlight cursor line |
| `cursorcolumn` | bool | `false` | Highlight cursor column |
| `text-width` | int | `80` | Text width (used by rulers and reflow) |
| `rulers` | int[] | `[]` | Column ruler positions, e.g. `[80, 120]` |
| `bufferline` | string | `"never"` | Show buffer tabs: `never`, `always`, `multiple` |

### Soft Wrap

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `soft-wrap.enable` | bool | `false` | Enable soft wrap |
| `soft-wrap.max-wrap` | int | `20` | Maximum visual indentation when wrapping |
| `soft-wrap.max-indent-retain` | int | `40` | Max indent levels to retain |
| `soft-wrap.wrap-indicator` | string | `"↪ "` | Continuation indicator |
| `soft-wrap.wrap-at-text-width` | bool | `false` | Wrap at `text-width` instead of window width |

### Whitespace and Indentation

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `whitespace.render` | string | `"none"` | `none` or `all` |
| `indent-guides.render` | bool | `false` | Show indent guides |
| `indent-guides.character` | string | `"│"` | Guide character |
| `indent-guides.skip-levels` | int | `0` | Indent levels to skip |
| `gutters.line-numbers.min-width` | int | `3` | Minimum gutter width |

### Editing

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto-pairs` | bool | `true` | Auto-insert closing brackets and quotes |
| `continue-comments` | bool | `true` | Extend comment tokens on new lines |
| `atomic-save` | bool | `true` | Write via temp file to prevent partial writes |
| `insert-final-newline` | bool | `true` | Ensure file ends with a newline |
| `trim-final-newlines` | bool | `false` | Remove trailing blank lines on save |
| `trim-trailing-whitespace` | bool | `false` | Remove trailing spaces on save |

### Auto-Save

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto-save` | bool | `false` | Save when focus is lost (alias for `auto-save.focus-lost`) |
| `auto-save.focus-lost` | bool | `false` | Save when focus leaves the view |
| `auto-save.after-delay.enable` | bool | `false` | Save after idle delay |
| `auto-save.after-delay.timeout` | int | `3000` | Idle delay in milliseconds |

### Search

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `search.smart-case` | bool | `true` | Case-insensitive unless pattern has uppercase |
| `search.wrap-around` | bool | `true` | Wrap search at end of file |

### Scrolling

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `scrolloff` | int | `5` | Lines of context kept above/below cursor |
| `scroll-lines` | int | `3` | Lines moved per scroll step |

### Cursor Shape

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `cursor-shape.normal` | string | (terminal) | `block`, `bar`, or `underline` |
| `cursor-shape.insert` | string | (terminal) | Cursor shape in Insert mode |
| `cursor-shape.select` | string | (terminal) | Cursor shape in Select mode |

### Status Bar

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `statusline.left` | string[] | built-in | Left-aligned statusline elements |
| `statusline.right` | string[] | built-in | Right-aligned statusline elements |
| `statusline.separator` | string | `"│"` | Separator between status items |
| `statusline.mode.normal` | string | `"normal"` | Label for Normal mode |
| `statusline.mode.insert` | string | `"insert"` | Label for Insert mode |
| `statusline.mode.select` | string | `"select"` | Label for Select mode |

Valid statusline elements: `mode`, `file-name`, `file-base-name`, `file-absolute-path`, `file-modified-indicator`, `read-only-indicator`, `file-encoding`, `file-line-ending`, `file-indent-style`, `file-type`, `diagnostics`, `selections`, `primary-selection-length`, `position`, `position-percentage`, `total-line-numbers`, `separator`, `spacer`, `register`, and `version-control`.

When the pane is too narrow to fit everything, sections shed elements from their inner edge, so items anchored at the bar's edges survive longest: the right section drops from its left end, the left section from its right end. The right section sheds first, then the left. Suffix an element with `!` (for example `"mode!"` or `"position!"`) to pin it so it never drops. The default configuration pins `mode` and `position`.

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

Picker options are module-owned UI settings. Split ratios are saved per picker. They can also be changed at runtime with keys like `:set editor.picker.split-ratios.diagnostics 0.65` and are persisted by auto-session when changed.

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
