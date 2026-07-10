---
title: "Commands"
weight: 35
---

# Commands

Enter command mode with `:`. All commands can be used by their full name or any listed alias.

## File

| Command | Aliases | Description |
|---------|---------|-------------|
| `write` | `w` | Write changes to disk. Accepts an optional path |
| `write!` | `w!` | Force write, creating necessary subdirectories. Accepts an optional path |
| `write_all` | `write-all`, `wa` | Write changes from all buffers to disk |
| `write-all!` | `wa!` | Forcefully write all buffers, creating necessary subdirectories |
| `write_quit` | `write-quit`, `wq`, `exit`, `x`, `xit` | Write changes and close the current view. Accepts an optional path |
| `write-quit!` | `wq!`, `exit!`, `x!`, `xit!` | Write and close the current view forcefully. Accepts an optional path |
| `write_quit_all` | `write-quit-all`, `wqa`, `xa` | Write all buffers and close all views |
| `write-quit-all!` | `wqa!`, `xa!` | Forcefully write all buffers, creating necessary subdirectories, and close all views |
| `write_buffer_close` | `write-buffer-close`, `wbc` | Write changes and close the buffer. Accepts an optional path |
| `write-buffer-close!` | `wbc!` | Force write and close the buffer, creating necessary subdirectories. Accepts an optional path |
| `update` | `u` | Write changes only if the file has been modified |
| `open` | `o`, `edit`, `e` | Open a file from disk into the current view |
| `new` | `n` | Create a new scratch buffer |
| `reload` | `rl` | Reload from the source file, preserving undo history and selections |
| `reload_all` | `reload-all`, `rla` | Reload all documents from their source files, preserving undo history and selections |
| `move` | `mv` | Move the current buffer and its corresponding file to a different path |
| `move!` | `mv!` | Move the current buffer and file, creating necessary subdirectories |
| `read` | `r` | Load a file into buffer at the cursor |

toe watches file-backed buffers for external changes. Clean buffers reload automatically; dirty buffers keep their in-memory text and can be resolved explicitly with `reload`, `reload_all`, or `write`. Reload applies the on-disk diff to the open document, so undo history and cursor/selection positions are preserved where possible.

## Buffer

| Command | Aliases | Description |
|---------|---------|-------------|
| `buffer_close` | `buffer-close`, `bc`, `bclose` | Close the current buffer |
| `buffer_close_force` | `buffer-close!`, `bc!`, `bclose!` | Close the current buffer forcefully, ignoring unsaved changes |
| `buffer_close_others` | `buffer-close-others`, `bco`, `bcloseother` | Close all buffers but the currently focused one |
| `buffer_close_all` | `buffer-close-all`, `bca`, `bcloseall` | Close all buffers without quitting |
| `buffer_next` | `buffer-next`, `bn`, `bnext` | Goto next buffer |
| `buffer_previous` | `buffer-previous`, `bp`, `bprev` | Goto previous buffer |

## Window

| Command | Aliases | Description |
|---------|---------|-------------|
| `vsplit` | `vs` | Vertical right split |
| `split` | `hs`, `sp` | Horizontal bottom split |
| `vsplit_new` | `vnew` | Vertical right split scratch buffer |
| `hsplit_new` | `hnew` | Horizontal bottom split scratch buffer |
| `transpose_view` | | Transpose splits |
| `wclose` | `wc` | Close window |
| `wclose!` | `wc!` | Force close window |
| `wonly` | `wo` | Close windows except current |

## Quit

| Command | Aliases | Description |
|---------|---------|-------------|
| `quit` | `q` | Close the current view |
| `quit!` | `q!` | Force close the current view, ignoring unsaved changes |
| `quit_all` | `quit-all`, `qa` | Close all views |
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
| `change_directory` | `change-current-directory`, `cd` | Change the current working directory |
| `show_directory` | `show-directory`, `pwd` | Show the current working directory |
| `show_directory_stack` | `show-directory-stack` | Show the directory stack as a space delimited string |
| `push_directory` | `push-directory`, `pushd` | Save and then change the current directory |
| `pop_directory` | `pop-directory`, `popd` | Remove the top entry from the directory stack and cd to the new top directory |

## Config

| Command | Aliases | Description |
|---------|---------|-------------|
| `get_option` | `get-option`, `get` | Get the current value of a config option |
| `set_option` | `set-option`, `set` | Set a config option at runtime |
| `toggle_option` | `toggle-option`, `toggle` | Toggle a config option at runtime |
| `config_open` | `config-open` | Open the user config.toml file |
| `config_open_workspace` | `config-open-workspace` | Open the workspace config.toml file in a trusted workspace |
| `config_reload` | `config-reload` | Refresh user config |
| `theme` | | Change the editor theme (show current theme if no name specified) |
| `log_open` | `log-open` | Open the editor log file |

## Workspace Trust

| Command | Aliases | Description |
|---------|---------|-------------|
| `workspace_trust` | `workspace-trust` | Trust the current workspace, enabling workspace config, configured tooling, and auto-session |
| `workspace_untrust` | `workspace-untrust` | Remove current workspace trust |

## Session

| Command | Aliases | Description |
|---------|---------|-------------|
| `save_session` | `save-session` | Save session to the workspace session file |
| `restore_session` | `restore-session` | Restore session from the workspace session file |

## Format

| Command | Aliases | Description |
|---------|---------|-------------|
| `format` | `fmt` | Format the file using an external formatter or language server |
| `reflow` | | Hard-wrap the current selection of lines to a given width. Accepts an optional width argument (defaults to `text-width`) |
| `sort` | | Sort ranges in selection. Flags: `-r`/`--reverse`, `-i`/`--insensitive` |

## LSP

| Command | Aliases | Description |
|---------|---------|-------------|
| `goto_declaration` | | Goto declaration |
| `goto_definition` | | Goto definition |
| `goto_type_definition` | | Goto type definition |
| `goto_implementation` | | Goto implementation |
| `goto_reference` | | Goto references |
| `select_references_to_symbol_under_cursor` | | Select symbol references |
| `code_action` | | Perform code action |
| `hover` | | Show docs for item under cursor |
| `rename_symbol` | | Rename symbol |
| `signature-help` | | Show signature help |
| `symbol_picker` | | Open symbol picker |
| `workspace_symbol_picker` | | Open workspace symbol picker |
| `lsp-restart` | | Restart language servers for the current document |
| `lsp-stop` | | Stop language servers for the current document |
| `lsp-workspace-command` | | Execute a language server workspace command |

## Version Control

| Command | Aliases | Description |
|---------|---------|-------------|
| `changed_file_picker` | | Open changed file picker |
| `goto_next_change` | | Goto next change |
| `goto_prev_change` | | Goto previous change |
| `goto_first_change` | | Goto first change |
| `goto_last_change` | | Goto last change |
| `reset_diff_change` | `reset-diff-change`, `diff-reset` | Reset the diff changes under the selections |

## Clipboard

| Command | Aliases | Description |
|---------|---------|-------------|
| `yank` | `clipboard-yank` | Yank selection |
| `paste_after` | | Paste after selection |
| `paste_before` | | Paste before selection |
| `replace_with_yanked` | | Replace with yanked text |
| `yank_to_clipboard` | | Yank selections to clipboard |
| `yank_main_selection_to_clipboard` | | Yank main selection to clipboard |
| `paste_clipboard_after` | `clipboard-paste-after` | Paste clipboard after selections |
| `paste_clipboard_before` | `clipboard-paste-before` | Paste clipboard before selections |
| `clipboard_paste_replace` | `clipboard-paste-replace` | Replace selections by clipboard content |
| `yank_joined_to_clipboard` | `yank-join` | Yank joined selections. Accepts an optional separator argument |
| `yank_to_primary_clipboard` | `primary-clipboard-yank` | Yank selections to primary clipboard |
| `paste_primary_clipboard_after` | `primary-clipboard-paste-after` | Paste primary clipboard after selections |
| `paste_primary_clipboard_before` | `primary-clipboard-paste-before` | Paste primary clipboard before selections |
| `primary_clipboard_paste_replace` | `primary-clipboard-paste-replace` | Replace selections by primary clipboard |
| `clear_register` | `clear-register` | Clear given register. If no argument is given, clear all registers |
| `show_clipboard_provider` | `show-clipboard-provider` | Show clipboard provider name in status bar |

## Search

| Command | Aliases | Description |
|---------|---------|-------------|
| `search_forward` | | Search for regex pattern (forward) |
| `search_backward` | | Reverse search for regex pattern |
| `search_next` | | Select next search match |
| `search_prev` | | Select previous search match |
| `search_selection_word` | | Use current selection as search pattern, word bounded |
| `make_search_word_bounded` | | Modify current search to make it word bounded |
| `search_selection` | | Use current selection as search pattern |

## Support

| Command | Aliases | Description |
|---------|---------|-------------|
| `character_info` | `character-info`, `char` | Get info about the character under the primary cursor |
| `echo` | | Prints the given arguments to the statusline |
| `redraw` | | Clear and re-render the whole UI |
