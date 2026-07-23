---
title: "Commands"
weight: 35
---

# Commands

Enter command mode with `:`. The Alias column lists additional names.

## File

| Command | Aliases | Description |
|---------|---------|-------------|
| `write` | `w` | Write changes to disk. Accepts an optional path |
| `write!` | `w!` | Force write, creating necessary subdirectories. Accepts an optional path |
| `write-all` | `wa` | Write changes from all buffers to disk |
| `write-all!` | `wa!` | Forcefully write all buffers, creating necessary subdirectories |
| `write-quit` | `wq`, `exit`, `x`, `xit` | Write changes and close the current view. Accepts an optional path |
| `write-quit!` | `wq!`, `exit!`, `x!`, `xit!` | Write and close the current view forcefully. Accepts an optional path |
| `write-quit-all` | `wqa`, `xa` | Write all buffers and close all views |
| `write-quit-all!` | `wqa!`, `xa!` | Forcefully write all buffers, creating necessary subdirectories, and close all views |
| `write-buffer-close` | `wbc` | Write changes and close the buffer. Accepts an optional path |
| `write-buffer-close!` | `wbc!` | Force write and close the buffer, creating necessary subdirectories. Accepts an optional path |
| `update` | `u` | Write changes only if the file has been modified |
| `open` | `o`, `edit`, `e` | Open a file from disk into the current view |
| `new` | `n` | Create a new scratch buffer |
| `reload` | `rl` | Reload from the source file, preserving undo history and selections |
| `reload-all` | `rla` | Reload all documents from their source files, preserving undo history and selections |
| `move` | `mv` | Move the current buffer and its corresponding file to a different path |
| `move!` | `mv!` | Move the current buffer and file, creating necessary subdirectories |
| `read` | `r` | Load a file into buffer at the cursor |

Clean buffers reload automatically after external changes. Dirty buffers remain untouched; use `reload`, `reload-all`, or `write` to resolve them.

## Buffer

| Command | Aliases | Description |
|---------|---------|-------------|
| `buffer-close` | `bc`, `bclose` | Close the current buffer |
| `buffer-close-force` | `buffer-close!`, `bc!`, `bclose!` | Close the current buffer forcefully, ignoring unsaved changes |
| `buffer-close-others` | `bco`, `bcloseother` | Close all buffers but the currently focused one |
| `buffer-close-all` | `bca`, `bcloseall` | Close all buffers without quitting |
| `buffer-next` | `bn`, `bnext` | Goto next buffer |
| `buffer-previous` | `bp`, `bprev` | Goto previous buffer |

## Window

| Command | Aliases | Description |
|---------|---------|-------------|
| `vsplit` | `vs` | Vertical right split |
| `split` | `hs`, `sp` | Horizontal bottom split |
| `vsplit-new` | `vnew` | Vertical right split scratch buffer |
| `hsplit-new` | `hnew` | Horizontal bottom split scratch buffer |
| `terminal` | | Open the user's shell in the focused pane |
| `terminal-search` | | Search the focused terminal's scrollback |
| `transpose-view` | | Transpose splits |
| `resize-view` | | Enter interactive resize mode |
| `wclose` | `wc` | Close window |
| `wclose!` | `wc!` | Force close window |
| `wonly` | `wo` | Close windows except current |

Splitting a document or image pane creates another view of the same document or image. Splitting a terminal starts a new shell.

## Image

| Command | Aliases | Description |
|---------|---------|-------------|
| `image-zoom-in` | `zoom-in` | Zoom image in |
| `image-zoom-out` | `zoom-out` | Zoom image out |
| `image-zoom-reset` | `zoom-reset` | Fit image to pane |

## Quit

| Command | Aliases | Description |
|---------|---------|-------------|
| `quit` | `q` | Close the current view |
| `quit!` | `q!` | Force close the current view, ignoring unsaved changes |
| `quit-all` | `qa` | Close all views |
| `quit-all!` | `qa!` | Force close all views ignoring unsaved changes |
| `cquit` | `cq` | Quit with exit code (default 1) |
| `cquit!` | `cq!` | Force quit with exit code (default 1) ignoring unsaved changes |

## Navigation

| Command | Aliases | Description |
|---------|---------|-------------|
| `goto` | `g` | Goto line number |

## Directory

| Command | Aliases | Description |
|---------|---------|-------------|
| `change-directory` | `change-current-directory`, `cd` | Change the current working directory |
| `show-directory` | `pwd` | Show the current working directory |
| `show-directory-stack` | | Show the directory stack as a space delimited string |
| `push-directory` | `pushd` | Save and then change the current directory |
| `pop-directory` | `popd` | Remove the top entry from the directory stack and cd to the new top directory |

## Config

| Command | Aliases | Description |
|---------|---------|-------------|
| `get-option` | `get` | Get the current value of a config option |
| `set-option` | `set` | Set a config option at runtime |
| `toggle-option` | `toggle` | Toggle a config option at runtime |
| `config-open` | | Open the user config.toml file |
| `config-open-workspace` | | Open the workspace config.toml file in a trusted workspace |
| `config-reload` | | Refresh user config |
| `theme` | | Change the editor theme (show current theme if no name specified) |
| `log-open` | | Open the editor log file |
| `set-language` | `lang` | Set or show the current buffer's language |
| `set-line-ending` | `line-ending` | Set or show the current buffer's line ending |
| `indent-style` | | Set or show the current buffer's indentation style |
| `encoding` | | Show the current buffer's encoding |

## Workspace Trust

| Command | Aliases | Description |
|---------|---------|-------------|
| `workspace-trust` | | Trust the current workspace, enabling workspace config, configured tooling, and auto-session |
| `workspace-untrust` | | Remove current workspace trust |

## Session

| Command | Aliases | Description |
|---------|---------|-------------|
| `save-session` | | Save session to the workspace session file |
| `restore-session` | | Restore session from the workspace session file |

## Pickers

| Command | Aliases | Description |
|---------|---------|-------------|
| `file-picker` | | Open file picker |
| `file-picker-in-current-dir` | | Open file picker at current working directory |
| `file-explorer` | | Open file explorer at workspace root |
| `file-explorer-in-current-pane-dir` | | Open file explorer at current pane's directory |
| `buffer-picker` | | Open buffer picker |
| `diagnostic-picker` | | Open diagnostic picker |
| `workspace-diagnostics-picker` | | Open workspace diagnostic picker |
| `global-search` | | Global search in workspace folder |
| `command-palette` | | Open command palette |
| `last-picker` | | Reopen the last picker |
| `jumplist-picker` | | Open jumplist picker |

File pickers preview text, directories, and supported images. The changed-file picker previews unified diffs.

## Format

| Command | Aliases | Description |
|---------|---------|-------------|
| `format` | `fmt` | Format the file using an external formatter or language server |
| `format-selections` | | Format the current selection |
| `reflow` | | Hard-wrap the current selection of lines to a given width. Accepts an optional width argument (defaults to `text-width`) |
| `sort` | | Sort ranges in selection. Flags: `-r`/`--reverse`, `-i`/`--insensitive` |

## LSP

| Command | Aliases | Description |
|---------|---------|-------------|
| `goto-declaration` | | Goto declaration |
| `goto-definition` | | Goto definition |
| `goto-type-definition` | | Goto type definition |
| `goto-implementation` | | Goto implementation |
| `goto-reference` | | Goto references |
| `select-references-to-symbol-under-cursor` | | Select symbol references |
| `code-action` | | Perform code action |
| `hover` | | Show docs for item under cursor |
| `rename-symbol` | | Rename symbol |
| `signature-help` | | Show signature help |
| `symbol-picker` | | Open symbol picker |
| `workspace-symbol-picker` | | Open workspace symbol picker |
| `lsp-restart` | | Restart language servers for the current document |
| `lsp-stop` | | Stop language servers for the current document |
| `lsp-workspace-command` | | Execute a language server workspace command |

## Version Control

| Command | Aliases | Description |
|---------|---------|-------------|
| `changed-file-picker` | | Open changed file picker |
| `goto-next-change` | | Goto next change |
| `goto-prev-change` | | Goto previous change |
| `goto-first-change` | | Goto first change |
| `goto-last-change` | | Goto last change |
| `reset-diff-change` | `diff-reset` | Reset the diff changes under the selections |

## Clipboard

| Command | Aliases | Description |
|---------|---------|-------------|
| `yank` | `clipboard-yank` | Yank selection |
| `paste-after` | | Paste after selection |
| `paste-before` | | Paste before selection |
| `replace-with-yanked` | | Replace with yanked text |
| `yank-to-clipboard` | | Yank selections to clipboard |
| `yank-main-selection-to-clipboard` | | Yank main selection to clipboard |
| `paste-clipboard-after` | `clipboard-paste-after` | Paste clipboard after selections |
| `paste-clipboard-before` | `clipboard-paste-before` | Paste clipboard before selections |
| `clipboard-paste-replace` | | Replace selections by clipboard content |
| `yank-joined-to-clipboard` | `yank-join` | Yank joined selections. Accepts an optional separator argument |
| `yank-to-primary-clipboard` | `primary-clipboard-yank` | Yank selections to primary clipboard |
| `paste-primary-clipboard-after` | `primary-clipboard-paste-after` | Paste primary clipboard after selections |
| `paste-primary-clipboard-before` | `primary-clipboard-paste-before` | Paste primary clipboard before selections |
| `primary-clipboard-paste-replace` | | Replace selections by primary clipboard |
| `clear-register` | | Clear given register. If no argument is given, clear all registers |

## Search

| Command | Aliases | Description |
|---------|---------|-------------|
| `search-forward` | | Search for regex pattern (forward) |
| `search-backward` | | Reverse search for regex pattern |
| `search-next` | | Select next search match |
| `search-prev` | | Select previous search match |
| `search-selection-word` | | Use current selection as search pattern, word bounded |
| `make-search-word-bounded` | | Modify current search to make it word bounded |
| `search-selection` | | Use current selection as search pattern |

## Support

| Command | Aliases | Description |
|---------|---------|-------------|
| `character-info` | `char` | Get info about the character under the primary cursor |
| `echo` | | Prints the given arguments to the statusline |
| `redraw` | | Clear and re-render the whole UI |
