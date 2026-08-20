# Changelog

Notable changes to toe.

## 0.3.3

### Interface

- Added `Ctrl` + click to jump to the definition of the symbol under the pointer
- Completions no longer start selected: `Tab` takes the top match or the highlighted one, and `Return` submits the prompt or inserts a newline instead of accepting
- Removed middle-click paste and its `middle-click-paste` option

## 0.3.2

### Interface

- Added desktop notifications from terminal panes
- Added `Shift+Tab` as the backward jump, mirroring `Tab` forward
- Jump list now works across documents: jumps follow you between files, pickers record where you jumped from, and the jump list picker walks the history instead of rewriting it
- Every navigation scrolls its destination into view, horizontally as well as vertically

## 0.3.1

### Interface

- Added border dragging to resize pickers and prompt popups, from 30% to 100% of the screen per axis, always centered and remembered per picker and prompt by auto-session

## 0.3.0

### Interface

- Removed the command line. Messages appear as a toast in the bottom-right corner, colored by severity and dismissed by a click or a timer
- Moved commands, search, and the other prompts into a centered popup. Completions list beside the input with the first one selected, so `Tab` or `Enter` accepts it and a second `Enter` runs the line
- Moved pending keys and their hints into a centered popup listing what can follow, with backspace to undo the last key or count digit
- Added a read-only `[messages]` buffer holding every message, reachable from the buffer picker or the `messages` command
- Added colored Nerd Font file and directory icons to pickers and previews

### Commands

- Made yank and paste use the system clipboard by default
- Added argument validation and completion across commands
- Documented every command and option

## 0.2.4

### Pickers

- Showed picker rows name-first, with the location trailing in a dimmed style, across the file, buffer, search, jump list, symbol, diagnostic, and changed-file pickers
- Scored only the newly streamed rows while a picker is still loading, so a large workspace walk no longer rescores everything already loaded on each batch
- Shortened package-qualified type names in diagnostic messages, in the diagnostics pickers and in hover popups
- Brightened the dimmed secondary text in the mocha theme

### Terminal

- Held terminal pane resizes until the layout settles before telling the shell its new size, so dragging a split no longer floods the shell with intermediate sizes

## 0.2.3

### Options

- Marked the current value when completing `:set` and `:theme` arguments, using a check icon with Nerd Fonts enabled and `*` otherwise

### Sessions

- Saved only the options that differ from the startup configuration with `:save-session`, matching what automatic session saving already wrote

## 0.2.2

### Sessions

- Switched workspace session storage from `.toe/session.toml` to `.toe/session.json`; existing TOML sessions are not migrated
- Remembered each document's scroll position per pane and restored it when switching back, including across sessions

## 0.2.1

### Localization

- Added German, French, and Italian translations for command descriptions throughout the command palette and keybinding help
- Colocated translations with their owning modules, with automatic English fallback for untranslated command descriptions
- Improved editor-specific terminology across existing translations
- Standardized user-facing command names on kebab-case and clarified `extend-to-line-end-newline`

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
