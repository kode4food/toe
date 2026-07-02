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
| `.toe/config.toml` | Workspace config |
| `.toe/languages.toml` | Workspace language config |

`$XDG_CONFIG_HOME` defaults to `~/.config` on Linux/macOS.

Open your user config directly: `:config_open`  
Open workspace config: `:config_open_workspace`  
Reload after editing: `:config_reload`

## Editor Options

Options can also be changed at runtime with `:set <key> <value>`, `:get <key>`,
and `:toggle <key>` (for booleans).

### Theme

```toml
theme = "frappe"   # frappe | latte | macchiato | mocha | default
```

### General

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.mouse` | bool | `true` | Enable mouse support |
| `editor.middle-click-paste` | bool | `true` | Paste on middle-click |
| `editor.insecure` | bool | `false` | Disable workspace trust checks |
| `editor.editor-config` | bool | `true` | Respect `.editorconfig` files |
| `editor.auto-session` | bool | `true` | Save/restore session automatically |
| `editor.default-line-ending` | string | (system) | `lf`, `crlf`, or `native` |

### Display

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.line-number` | string | `"absolute"` | `absolute`, `relative`, or `none` |
| `editor.cursorline` | bool | `false` | Highlight cursor line |
| `editor.cursorcolumn` | bool | `false` | Highlight cursor column |
| `editor.text-width` | int | `80` | Text width (used by rulers and reflow) |
| `editor.rulers` | int[] | `[]` | Column ruler positions, e.g. `[80, 120]` |
| `editor.bufferline` | string | `"never"` | Show buffer tabs: `never`, `always`, `multiple` |

### Soft Wrap

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.soft-wrap.enable` | bool | `false` | Enable soft wrap |
| `editor.soft-wrap.max-wrap` | int | `80` | Maximum visual indentation when wrapping |
| `editor.soft-wrap.max-indent-retain` | int | `16` | Max indent levels to retain |
| `editor.soft-wrap.wrap-indicator` | string | `"↳"` | Continuation indicator |
| `editor.soft-wrap.wrap-at-text-width` | bool | `false` | Wrap at `text-width` instead of window width |

### Whitespace and Indentation

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.whitespace.render` | string | `"none"` | `none`, `all`, or specific chars |
| `editor.indent-guides.render` | bool | `false` | Show indent guides |
| `editor.indent-guides.character` | string | `"│"` | Guide character |
| `editor.indent-guides.skip-levels` | int | `1` | Indent levels to skip |
| `editor.gutters.line-numbers.min-width` | int | `4` | Minimum gutter width |

### Editing

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.auto-pairs` | bool | `true` | Auto-insert closing brackets and quotes |
| `editor.continue-comments` | bool | `true` | Extend comment tokens on new lines |
| `editor.atomic-save` | bool | `true` | Write via temp file to prevent partial writes |
| `editor.insert-final-newline` | bool | `true` | Ensure file ends with a newline |
| `editor.trim-final-newlines` | bool | `false` | Remove trailing blank lines on save |
| `editor.trim-trailing-whitespace` | bool | `false` | Remove trailing spaces on save |

### Auto-Save

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.auto-save` | bool | `false` | Save on focus loss and delay |
| `editor.auto-save.focus-lost` | bool | `false` | Save when focus leaves the view |
| `editor.auto-save.after-delay.enable` | bool | `false` | Save after idle delay |
| `editor.auto-save.after-delay.timeout` | int | `1000` | Idle delay in milliseconds |

### Search

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.search.smart-case` | bool | `true` | Case-insensitive unless pattern has uppercase |
| `editor.search.wrap-around` | bool | `true` | Wrap search at end of file |

### Scrolling

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.scrolloff` | int | `3` | Lines of context kept above/below cursor |
| `editor.scroll-lines` | int | `3` | Lines moved per scroll step |

### Cursor Shape

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.cursor-shape.normal` | string | (terminal) | `block`, `bar`, or `underline` |
| `editor.cursor-shape.insert` | string | (terminal) | Cursor shape in Insert mode |
| `editor.cursor-shape.select` | string | (terminal) | Cursor shape in Select mode |

### Status Bar

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.statusline.separator` | string | `" "` | Separator between status items |
| `editor.statusline.mode.normal` | string | `"normal"` | Label for Normal mode |
| `editor.statusline.mode.insert` | string | `"insert"` | Label for Insert mode |
| `editor.statusline.mode.select` | string | `"select"` | Label for Select mode |

### Pickers

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor.buffer-picker.start_position` | string | `"top"` | `top` or `bottom` |
| `editor.file-explorer.hidden` | bool | `false` | Show hidden files |
| `editor.file-explorer.follow-symlinks` | bool | `false` | Follow symlinks |
| `editor.file-explorer.ignore` | bool | `true` | Respect `.ignore` files |
| `editor.file-explorer.git-ignore` | bool | `true` | Respect `.gitignore` |
| `editor.file-explorer.flatten-dirs` | bool | `false` | Collapse single-child directories |

### Shell

```toml
[editor]
shell = ["bash", "-c"]   # default: system shell
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
