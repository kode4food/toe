---
title: "Commands"
weight: 35
---

# Commands

Enter command mode with `:`. Any command below can be run by name; the Aliases column lists additional names. Descriptions come from each command's built-in documentation.

## File

| Command | Aliases | Description |
|---------|---------|-------------|
| `write` | `w` | Write changes to disk. Accepts an optional path (:write some/path.txt) |
| `write!` | `w!` | Force write changes to disk creating necessary subdirectories. Accepts an optional path (:write! some/path.txt) |
| `write-all` | `wa` | Write changes from all buffers to disk |
| `write-all!` | `wa!` | Forcefully write changes from all buffers to disk creating necessary subdirectories |
| `write-quit` | `wq`, `exit`, `x`, `xit` | Write changes to disk and close the current view. Accepts an optional path (:wq some/path.txt) |
| `write-quit!` | `wq!`, `exit!`, `x!`, `xit!` | Write changes to disk and close the current view forcefully. Accepts an optional path (:wq! some/path.txt) |
| `write-quit-all` | `wqa`, `xa` | Write changes from all buffers to disk and close all views |
| `write-quit-all!` | `wqa!`, `xa!` | Forcefully write changes from all buffers to disk, creating necessary subdirectories, and close all views (ignoring unsaved changes) |
| `write-buffer-close` | `wbc` | Write changes to disk and closes the buffer. Accepts an optional path (:write-buffer-close some/path.txt) |
| `write-buffer-close!` | `wbc!` | Force write changes to disk creating necessary subdirectories and closes the buffer. Accepts an optional path (:write-buffer-close! some/path.txt) |
| `update` | `u` | Write changes only if the file has been modified |
| `open` | `o`, `edit`, `e` | Open a file from disk into the current view |
| `new` | `n` | Create a new scratch buffer |
| `reload` | `rl` | Discard changes and reload from the source file |
| `reload-all` | `rla` | Discard changes and reload all documents from the source files |
| `move` | `mv` | Move the current buffer and its corresponding file to a different path |
| `move!` | `mv!` | Move the current buffer and its corresponding file to a different path creating necessary subdirectories |
| `read` | `r` | Load a file into buffer |

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

## Movement

| Command | Aliases | Description |
|---------|---------|-------------|
| `goto-line-end-newline` |  | Goto newline at line end |
| `move-char-left` |  | Move left |
| `move-visual-line-down` |  | Move down |
| `move-visual-line-up` |  | Move up |
| `move-char-right` |  | Move right |
| `move-next-word-start` |  | Move to start of next word |
| `move-prev-word-start` |  | Move to start of previous word |
| `move-next-word-end` |  | Move to end of next word |
| `move-prev-word-end` |  | Move to end of previous word |
| `move-next-long-word-start` |  | Move to start of next long word |
| `move-prev-long-word-start` |  | Move to start of previous long word |
| `move-next-long-word-end` |  | Move to end of next long word |
| `move-prev-long-word-end` |  | Move to end of previous long word |
| `move-next-sub-word-start` |  | Move to start of next sub-word |
| `move-prev-sub-word-start` |  | Move to start of previous sub-word |
| `move-next-sub-word-end` |  | Move to end of next sub-word |
| `move-prev-sub-word-end` |  | Move to end of previous sub-word |
| `goto-line-start` |  | Goto line start |
| `goto-line-end` |  | Goto line end |
| `find-next-char` |  | Move to next occurrence of char |
| `find-till-char` |  | Move till next occurrence of char |
| `find-prev-char` |  | Move to previous occurrence of char |
| `till-prev-char` |  | Move till previous occurrence of char |
| `goto-line` |  | Goto line |
| `goto-line-or-file-start` |  | Goto line number `<n>` else file start |
| `goto-file` |  | Goto files/URLs in selections |
| `goto-column` |  | Goto column |
| `goto-window-top` |  | Goto window top |
| `goto-window-center` |  | Goto window center |
| `goto-window-bottom` |  | Goto window bottom |
| `goto-last-line` |  | Goto last line |
| `goto-first-nonwhitespace` |  | Goto first non-blank in line |
| `goto-last-accessed-file` |  | Goto last accessed file |
| `goto-last-modified-file` |  | Goto last modified file |
| `goto-last-modification` |  | Goto last modification |
| `jump-forward` |  | Jump forward on jumplist |
| `jump-backward` |  | Jump backward on jumplist |
| `save-selection` |  | Save current selection to jumplist |
| `goto-next-paragraph` |  | Goto next paragraph |
| `goto-prev-paragraph` |  | Goto previous paragraph |
| `goto-line-or-extend-file-start` |  | Extend to line number `<n>` else file start |
| `repeat-last-motion` |  | Repeat last motion |
| `match-brackets` |  | Goto matching bracket |
| `goto` | `g` | Goto a path:line:col location |

## Selection

| Command | Aliases | Description |
|---------|---------|-------------|
| `extend-char-left` |  | Extend left |
| `extend-visual-line-down` |  | Extend down |
| `extend-visual-line-up` |  | Extend up |
| `extend-char-right` |  | Extend right |
| `extend-next-word-start` |  | Extend to start of next word |
| `extend-prev-word-start` |  | Extend to start of previous word |
| `extend-next-word-end` |  | Extend to end of next word |
| `extend-prev-word-end` |  | Extend to end of previous word |
| `extend-next-long-word-start` |  | Extend to start of next long word |
| `extend-prev-long-word-start` |  | Extend to start of previous long word |
| `extend-next-long-word-end` |  | Extend to end of next long word |
| `extend-prev-long-word-end` |  | Extend to end of previous long word |
| `extend-next-sub-word-start` |  | Extend to start of next sub-word |
| `extend-prev-sub-word-start` |  | Extend to start of previous sub-word |
| `extend-next-sub-word-end` |  | Extend to end of next sub-word |
| `extend-prev-sub-word-end` |  | Extend to end of previous sub-word |
| `extend-next-char` |  | Extend to next occurrence of char |
| `extend-till-char` |  | Extend till next occurrence of char |
| `extend-prev-char` |  | Extend to previous occurrence of char |
| `extend-till-prev-char` |  | Extend till previous occurrence of char |
| `extend-to-line-start` |  | Extend to line start |
| `extend-to-line-end` |  | Extend to line end |
| `extend-to-line-end-newline` |  | Extend to line end |
| `extend-to-first-nonwhitespace` |  | Extend to first non-blank in line |
| `extend-to-column` |  | Extend to column |
| `extend-to-last-line` |  | Extend to last line |
| `extend-to-file-end` |  | Extend to file end |
| `copy-on-next-line` |  | Copy selection on next line |
| `copy-on-prev-line` |  | Copy selection on previous line |
| `select-within-regex` |  | Select all regex matches inside selections |
| `split-selection-by-regex` |  | Split selections on regex matches |
| `keep-selections-matching` |  | Keep selections matching regex |
| `remove-selections-matching` |  | Remove selections matching regex |
| `split-selection-on-newline` |  | Split selection on newlines |
| `merge-selections` |  | Merge selections |
| `merge-consecutive-selections` |  | Merge consecutive selections |
| `collapse-selection` |  | Collapse selection into single cursor |
| `flip-selections` |  | Flip selection cursor and anchor |
| `select-all` |  | Select whole document |
| `select-line-above` |  | Select line above |
| `select-line-below` |  | Select line below |
| `extend-line-below` |  | Select current line, if already selected, extend to next line |
| `extend-to-line-bounds` |  | Extend selection to line bounds |
| `shrink-to-line-bounds` |  | Shrink selection to line bounds |
| `expand-selection` |  | Expand selection to syntax node |
| `shrink-selection` |  | Shrink selection to syntax node |
| `keep-primary-selection` |  | Keep primary selection |
| `remove-primary-selection` |  | Remove primary selection |
| `surround-add` |  | Surround add |
| `surround-replace` |  | Surround replace |
| `surround-delete` |  | Surround delete |
| `select-textobject-around` |  | Select around object |
| `select-textobject-inner` |  | Select inside object |

## Editing

| Command | Aliases | Description |
|---------|---------|-------------|
| `commit-undo-checkpoint` |  | Commit changes to new checkpoint |
| `delete-word-backward` |  | Delete previous word |
| `delete-word-forward` |  | Delete next word |
| `kill-to-line-start` |  | Delete till start of line |
| `kill-to-line-end` |  | Delete till end of line |
| `delete-char-backward` |  | Delete previous char |
| `delete-char-forward` |  | Delete next char |
| `insert-newline` |  | Insert newline char |
| `smart-tab` |  | Insert tab in leading whitespace; otherwise move past the enclosing syntax node |
| `insert-tab` |  | Insert tab at each cursor |
| `replace` |  | Replace with new char |
| `delete-selection` |  | Delete selection |
| `delete-selection-noyank` |  | Delete selection without yanking |
| `change-selection` |  | Change selection |
| `change-selection-noyank` |  | Change selection without yanking |
| `switch-case` |  | Switch (toggle) case |
| `switch-to-lowercase` |  | Switch to lowercase |
| `switch-to-uppercase` |  | Switch to uppercase |
| `indent` |  | Indent selection |
| `unindent` |  | Unindent selection |
| `join-selections` |  | Join lines inside selection |
| `join-selections-space` |  | Join lines inside selection and select spaces |
| `align-selections` |  | Align selections in column |
| `trim-selections` |  | Trim whitespace from selections |
| `rotate-selections-backward` |  | Rotate selections backward |
| `rotate-selections-forward` |  | Rotate selections forward |
| `rotate-contents-backward` |  | Rotate selections contents backward |
| `rotate-contents-forward` |  | Rotate selection contents forward |
| `ensure-forward` |  | Ensure all selections face forward |
| `increment` |  | Increment item under cursor |
| `decrement` |  | Decrement item under cursor |
| `add-newline-above` |  | Add newline above |
| `add-newline-below` |  | Add newline below |

## Undo History

| Command | Aliases | Description |
|---------|---------|-------------|
| `undo` |  | Undo change |
| `redo` |  | Redo change |
| `earlier` |  | Move backward in history |
| `later` |  | Move forward in history |

## Modes

| Command | Aliases | Description |
|---------|---------|-------------|
| `select-mode` |  | Enter selection extend mode |
| `normal-mode` |  | Enter normal mode |
| `insert-mode` |  | Insert before selection |
| `insert-at-line-start` |  | Insert at start of line |
| `append-mode` |  | Append after selection |
| `append-to-line` |  | Insert at end of line |
| `open-below` |  | Open new line below selection |
| `open-above` |  | Open new line above selection |
| `enter-command-mode` |  | Enter command mode |

## Comments

| Command | Aliases | Description |
|---------|---------|-------------|
| `toggle-comments` |  | Comment/uncomment selections |
| `toggle-line-comments` |  | Line comment/uncomment selections |
| `toggle-block-comments` |  | Block comment/uncomment selections |

## Search

| Command | Aliases | Description |
|---------|---------|-------------|
| `search-forward` |  | Search for regex pattern |
| `search-backward` |  | Reverse search for regex pattern |
| `search-next` |  | Select next search match |
| `search-prev` |  | Select previous search match |
| `search-selection-word` |  | Use current selection as the search pattern, automatically wrapping with `\b` on word boundaries |
| `make-search-word-bounded` |  | Modify current search to make it word bounded |
| `search-selection` |  | Use current selection as search pattern |
| `extend-search-next` |  | Add next search match to selection |
| `extend-search-prev` |  | Add previous search match to selection |

## Completion

| Command | Aliases | Description |
|---------|---------|-------------|
| `completion` |  | Complete current word |
| `completion-accept` |  | Accept completion |
| `completion-cancel` |  | Cancel completion |
| `completion-previous` |  | Previous completion |
| `completion-next` |  | Next completion |
| `completion-page-up` |  | Previous completion page |
| `completion-page-down` |  | Next completion page |
| `completion-first` |  | First completion |
| `completion-last` |  | Last completion |

## Registers

| Command | Aliases | Description |
|---------|---------|-------------|
| `insert-register` |  | Insert register |
| `select-register` |  | Select register |
| `clear-register` |  | Clear given register. If no argument is provided, clear all registers |

## Macros

| Command | Aliases | Description |
|---------|---------|-------------|
| `record-macro` |  | Record macro |
| `replay-macro` |  | Replay macro |

## Clipboard

| Command | Aliases | Description |
|---------|---------|-------------|
| `yank` | `clipboard-yank` | Yank selection |
| `paste-after` |  | Paste after selection |
| `paste-before` |  | Paste before selection |
| `replace-with-yanked` |  | Replace with yanked text |
| `yank-to-clipboard` |  | Yank selections to clipboard |
| `yank-main-selection-to-clipboard` |  | Yank main selection to clipboard |
| `paste-clipboard-after` | `clipboard-paste-after` | Paste clipboard after selections |
| `paste-clipboard-before` | `clipboard-paste-before` | Paste clipboard before selections |
| `clipboard-paste-replace` |  | Replace selections by clipboard content |
| `yank-joined-to-clipboard` | `yank-join` | Yank joined selections. A separator can be provided as first argument. Default value is newline |
| `yank-to-primary-clipboard` | `primary-clipboard-yank` | Yank selections to primary clipboard |
| `paste-primary-clipboard-after` | `primary-clipboard-paste-after` | Paste primary clipboard after selections |
| `paste-primary-clipboard-before` | `primary-clipboard-paste-before` | Paste primary clipboard before selections |
| `primary-clipboard-paste-replace` |  | Replace selections by primary clipboard |

## Shell

| Command | Aliases | Description |
|---------|---------|-------------|
| `shell-pipe` |  | Pipe selections through shell command |
| `shell-insert-output` |  | Insert shell command output before selections |
| `shell-keep-pipe` |  | Filter selections with shell predicate |
| `shell-pipe-to` |  | Pipe selections into shell command ignoring output |
| `shell-append-output` |  | Append shell command output after selections |

## Format

| Command | Aliases | Description |
|---------|---------|-------------|
| `format` | `fmt` | Format the file using an external formatter or language server |
| `format-selections` |  | Format selection |
| `reflow` |  | Hard-wrap the current selection of lines to a given width |
| `sort` |  | Sort ranges in selection |

## Windows and Splits

| Command | Aliases | Description |
|---------|---------|-------------|
| `vsplit` | `vs` | Vertical right split |
| `split` | `hs`, `sp` | Horizontal bottom split |
| `vsplit-new` | `vnew` | Vertical right split scratch buffer |
| `hsplit-new` | `hnew` | Horizontal bottom split scratch buffer |
| `transpose-view` |  | Transpose splits |
| `wclose` | `wc` | Close window |
| `wclose!` | `wc!` | Force close window |
| `wonly` | `wo` | Close windows except current |
| `rotate-view` |  | Goto next window |
| `toggle-pane-maximized` |  | Toggle focused pane maximized |
| `jump-view-left` |  | Jump to left split |
| `jump-view-down` |  | Jump to split below |
| `jump-view-up` |  | Jump to split above |
| `jump-view-right` |  | Jump to right split |
| `swap-view-left` |  | Swap with left split |
| `swap-view-down` |  | Swap with split below |
| `swap-view-up` |  | Swap with split above |
| `swap-view-right` |  | Swap with right split |
| `resize-view` |  | Resize split |

Splitting a document or image pane creates another view of the same document or image. Splitting a terminal starts a new shell. `Ctrl+w z` temporarily maximizes the focused pane; press it again to restore the preserved split layout.

## View and Scroll

| Command | Aliases | Description |
|---------|---------|-------------|
| `page-up` |  | Move page up |
| `page-down` |  | Move page down |
| `page-cursor-half-up` |  | Move page and cursor half up |
| `page-cursor-half-down` |  | Move page and cursor half down |
| `half-page-up` |  | Move half page up |
| `half-page-down` |  | Move half page down |
| `page-cursor-up` |  | Move page and cursor up |
| `page-cursor-down` |  | Move page and cursor down |
| `center-cursor-line` |  | Align view center |
| `align-view-top` |  | Align view top |
| `align-view-bottom` |  | Align view bottom |
| `scroll-up` |  | Scroll view up |
| `scroll-down` |  | Scroll view down |

## Terminal

| Command | Aliases | Description |
|---------|---------|-------------|
| `paste-clipboard-into-pane` |  | Paste clipboard into terminal |
| `terminal` |  | Open a new terminal |
| `terminal-search` |  | Search focused terminal's scrollback |

## Image

| Command | Aliases | Description |
|---------|---------|-------------|
| `image-zoom-in` | `zoom-in` | Zoom image in |
| `image-zoom-out` | `zoom-out` | Zoom image out |
| `image-zoom-reset` | `zoom-reset` | Fit image to pane |
| `image-pan-left` | `pan-left` | Pan image left |
| `image-pan-down` | `pan-down` | Pan image down |
| `image-pan-up` | `pan-up` | Pan image up |
| `image-pan-right` | `pan-right` | Pan image right |

## Pickers

| Command | Aliases | Description |
|---------|---------|-------------|
| `symbol-picker` |  | Open symbol picker |
| `workspace-symbol-picker` |  | Open workspace symbol picker |
| `file-picker` |  | Open file picker |
| `file-picker-in-current-dir` |  | Open file picker at current working directory |
| `file-explorer` |  | Open file explorer at workspace root |
| `file-explorer-in-current-pane-dir` |  | Open file explorer at current pane's directory |
| `buffer-picker` |  | Open buffer picker |
| `jumplist-picker` |  | Open jumplist picker |
| `diagnostic-picker` |  | Open diagnostic picker |
| `workspace-diagnostics-picker` |  | Open workspace diagnostic picker |
| `global-search` |  | Global search in workspace folder |
| `command-palette` |  | Open command palette |
| `last-picker` |  | Reopen the last picker |
| `changed-file-picker` |  | Open changed file picker |

## LSP

| Command | Aliases | Description |
|---------|---------|-------------|
| `goto-declaration` |  | Goto declaration |
| `goto-definition` |  | Goto definition |
| `goto-type-definition` |  | Goto type definition |
| `goto-implementation` |  | Goto implementation |
| `goto-reference` |  | Goto references |
| `select-references-to-symbol-under-cursor` |  | Select symbol references |
| `code-action` |  | Perform code action |
| `hover` |  | Show docs for item under cursor |
| `rename-symbol` |  | Rename symbol |
| `signature-help` |  | Show signature help |
| `lsp-restart` |  | Restart language servers for the current document |
| `lsp-stop` |  | Stop language servers for the current document |
| `lsp-workspace-command` |  | Execute a language server workspace command |

## Version Control

| Command | Aliases | Description |
|---------|---------|-------------|
| `goto-next-change` |  | Goto next change |
| `goto-prev-change` |  | Goto previous change |
| `goto-first-change` |  | Goto first change |
| `goto-last-change` |  | Goto last change |
| `reset-diff-change` | `diff-reset` | Reset the diff changes under the selections |

## Directory

| Command | Aliases | Description |
|---------|---------|-------------|
| `change-directory` | `change-current-directory`, `cd` | Change the current working directory |
| `show-directory` | `pwd` | Show the current working directory |
| `show-directory-stack` |  | Show the directory stack as a space delimited string |
| `push-directory` | `pushd` | Save and then change the current directory |
| `pop-directory` | `popd` | Remove the top entry from the directory stack and cd to the new top directory |

## Config

| Command | Aliases | Description |
|---------|---------|-------------|
| `get-option` | `get` | Get the current value of a config option |
| `set-option` | `set` | Set a config option at runtime |
| `toggle-option` | `toggle` | Toggle a config option at runtime |
| `config-open` |  | Open the user config.toml file |
| `config-open-workspace` |  | Open the workspace config.toml file |
| `config-reload` |  | Refresh user config |
| `log-open` |  | Open the editor log file |
| `workspace-trust` |  | Add current workspace to the list of trusted workspaces |
| `workspace-untrust` |  | Remove current workspace from the list of trusted workspaces |
| `theme` |  | Change the editor theme (show current theme if no name specified) |
| `set-language` | `lang` | Set the language of current buffer (show current language if no value specified) |
| `set-line-ending` | `line-ending` | Set the document's default line ending. Options: crlf, lf, native |
| `indent-style` |  | Set the indentation style for editing. ('t' for tabs or 1-16 for number of spaces) |
| `encoding` |  | Set encoding |

## Session

| Command | Aliases | Description |
|---------|---------|-------------|
| `save-session` |  | Save session to the workspace session file |
| `restore-session` |  | Restore session from the workspace session file |

## Quit

| Command | Aliases | Description |
|---------|---------|-------------|
| `quit` | `q` | Close the current view |
| `quit!` | `q!` | Force close the current view, ignoring unsaved changes |
| `quit-all` | `qa` | Close all views |
| `quit-all!` | `qa!` | Force close all views ignoring unsaved changes |
| `cquit` | `cq` | Quit with exit code (default 1) |
| `cquit!` | `cq!` | Force quit with exit code (default 1) ignoring unsaved changes |

## Support

| Command | Aliases | Description |
|---------|---------|-------------|
| `character-info` | `char` | Get info about the character under the primary cursor |
| `echo` |  | Prints the given arguments to the status line |
| `redraw` |  | Clear and re-render the whole UI |
