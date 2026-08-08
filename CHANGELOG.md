# Changelog

Notable changes to toe.

## 0.2

First public release.

### Editing

- Modal editing with normal, insert, and selection modes
- Multi-cursor editing, undo and redo, macros, and registers
- Text objects, surround pairs, bracket matching, and comment toggling
- Soft wrap, indent guides, rulers, whitespace rendering, and configurable cursor shapes

### Navigation

- Multiple buffers and split views
- Fuzzy pickers for files, buffers, and the file explorer
- Global search across the workspace, with previews
- A jump list kept per pane
- Image panes, binary/hex panes, and an integrated terminal pane

### Language tooling

- Tree-sitter syntax highlighting, with Chroma as a fallback
- Language server completion, hover, signature help, and diagnostics
- Go-to definition, declaration, type definition, implementation, and references
- Document and workspace symbol pickers, code actions, rename, and formatting
- Go with `gopls` is the target workflow; other servers are configurable in `languages.toml`

### Version control

- Git diff gutters, changed-hunk navigation, and hunk reset
- A changed-file picker with unified diff previews, grouped by staged and unstaged

### Project state

- Workspace trust gating for project-local configuration
- User and workspace TOML config, plus EditorConfig support
- Session persistence, external file change detection, and clean-buffer reloads

### Appearance

- Four Catppuccin themes: latte, frappe, macchiato, mocha
- Nerd Font glyphs by default, with ASCII fallbacks

### Scope

- Version control is Git
- Extend toe through commands, key bindings, and conditional bindings
- Image panes need a terminal supporting the Kitty graphics protocol
- The system clipboard uses detected external tools, with an OSC 52 fallback
