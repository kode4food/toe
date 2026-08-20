---
title: "Key Bindings"
weight: 30
---

# Key Bindings

## How Keys Work

Every key press is dispatched through the keymap for the **current mode**. Normal, Select, and Insert are the editing modes; terminal panes, image panes, binary panes, prompts, and pickers each have their own bindings. The same physical key can do different things in different contexts, which is why this page is grouped by mode and context rather than by key.

**Key sequences and prefixes.** Many commands are bound to a sequence of keys, not a single press, `g` then `d`, or `Ctrl+w` then `v`. The first key of a sequence is a *prefix*; pressing it opens a popup listing the keys that can follow. The main prefixes are `g` (goto), `m` (match and surround), `z` and `Z` (view, the two are interchangeable), `[` and `]` (previous/next), `Ctrl+w` (windows), and the leader.

**Leader.** `Space` opens the leader menu. `Ctrl+\` is equivalent and also works in terminal and image panes, where `Space` is passed through instead. See [Leader Menu](#leader-menu).

**Counts.** In Normal and Select mode, typing digits before a command repeats it that many times: `5j` moves down five lines, `10G` goes to line 10. A leading `0` is a command (line start), not a count. The pending count appears in the key popup and clears once the command runs. A count is only accepted before a command that takes one, and `Backspace` removes the last digit or key typed toward a command.

**Placeholders.** Some commands capture the next key(s) directly. In the tables these appear as:

- `<char>`: a literal character to act on, e.g. `f<char>`, `r<char>`, `ms<char>`.
- `<reg>`: a register letter, e.g. `"<reg>` before a yank or paste, `Ctrl+r <reg>` in insert mode.
- `<n>`: a count typed before the key.

**Registers.** `"<reg>` chooses the register the next yank or paste uses; without it they use the default register. The popup lists the registers currently holding a value, with a preview of each. Yanks and pastes that go through the system clipboard live under the leader (`Space+y`, `Space+p`).

**Insert mode.** Any printable key with no binding is inserted as text; the Insert Mode bindings below are the exceptions that edit or move instead.

The rest of this page covers the three editing modes (Normal, Select, Insert) first, then the global facilities that apply across them: window management, the leader menu, terminal, image, and binary panes, prompts, and picker navigation.

Keys can be rebound and new actions scripted in Ale, see [Scripting]({{< relref "/docs/scripting" >}}).

<a href="../../downloads/toe-cheatsheet.pdf" download>Download the printable Toe cheatsheet (PDF)</a>.

## Normal Mode

Normal mode is the default. Keys are commands, not text: you move the cursor, manipulate selections, and launch edits from here.

### Motion

| Key | Action |
|-----|--------|
| `h` / `←` | Move left |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `l` / `→` | Move right |
| `w` | Move to start of next word |
| `b` | Move to start of previous word |
| `e` | Move to end of next word |
| `W` | Move to start of next long word |
| `B` | Move to start of previous long word |
| `E` | Move to end of next long word |
| `f<char>` | Move to next occurrence of char |
| `t<char>` | Move till next occurrence of char |
| `F<char>` | Move to previous occurrence of char |
| `T<char>` | Move till previous occurrence of char |
| `Home` | Goto line start |
| `End` | Goto line end |

### Goto (`g`)

The `g` prefix jumps somewhere; a count changes what `gg` does.

| Key | Action |
|-----|--------|
| `gg` / `<n>gg` | Goto line number `<n>` else file start |
| `G` / `<n>G` | Goto line number `<n>` |
| `gs` | Goto first non-blank in line |
| `ge` | Goto last line |
| `g\|` / `<n>g\|` | Goto column |
| `gt` | Goto window top |
| `gc` | Goto window center |
| `gb` | Goto window bottom |
| `gd` | Goto definition |
| `gD` | Goto declaration |
| `gy` | Goto type definition |
| `gi` | Goto implementation |
| `gr` | Goto references |
| `gf` | Goto files/URLs in selections |
| `gn` | Goto next buffer |
| `gp` | Goto previous buffer |
| `ga` | Goto last accessed file |
| `gm` | Goto last modified file |
| `g.` | Goto last modification |

### Previous / Next (`[` / `]`)

The `[` and `]` prefixes step backward and forward through the same kind of target.

| Key | Action |
|-----|--------|
| `[p` | Goto previous paragraph |
| `]p` | Goto next paragraph |
| `[g` | Goto previous change |
| `]g` | Goto next change |
| `[G` | Goto first change |
| `]G` | Goto last change |
| `[␣` | Add newline above |
| `]␣` | Add newline below |

### Jumplist

| Key | Action |
|-----|--------|
| `Shift+Tab` / `Ctrl+o` | Jump backward on jumplist |
| `Tab` / `Ctrl+i` | Jump forward on jumplist |
| `Ctrl+s` | Push current selection onto jumplist |

### Switching Modes

| Key | Action |
|-----|--------|
| `:` | Enter command mode |
| `i` | Insert before selection |
| `I` | Insert at start of line |
| `a` | Append after selection |
| `A` | Insert at end of line |
| `o` | Open new line below selection |
| `O` | Open new line above selection |
| `v` | Enter selection extend mode |

### Editing

| Key | Action |
|-----|--------|
| `d` | Delete selection |
| `Alt+d` | Delete selection without yanking |
| `c` | Change selection |
| `Alt+c` | Change selection without yanking |
| `r<char>` | Replace with new char |
| `~` | Switch (toggle) case |
| `` ` `` | Switch to lowercase |
| `Alt+`` ` `` | Switch to uppercase |
| `>` | Indent selection |
| `<` | Unindent selection |
| `J` | Join lines inside selection |
| `Alt+J` | Join lines inside selection and select spaces |
| `&` | Align selections in column |
| `_` | Trim whitespace from selections |
| `Ctrl+a` | Increment item under cursor |
| `Ctrl+x` | Decrement item under cursor |
| `=` | Format selection |
| `u` | Undo change |
| `U` | Redo change |
| `Alt+u` | Move backward in history |
| `Alt+U` | Move forward in history |

### Yank and Paste

| Key | Action |
|-----|--------|
| `y` | Yank selection |
| `p` | Paste after selection |
| `P` | Paste before selection |
| `R` | Replace with yanked text |
| `"<reg>` | Select register for the next yank or paste |

Clipboard yanks and pastes are under the [Leader Menu](#leader-menu) (`Space+y`, `Space+p`, …).

### Search

| Key | Action |
|-----|--------|
| `/` | Search for regex pattern |
| `?` | Reverse search for regex pattern |
| `n` | Select next search match |
| `N` | Select previous search match |
| `*` | Use current selection as search pattern, word bounded |
| `Alt+*` | Use current selection as search pattern |

### Selection

| Key | Action |
|-----|--------|
| `s` | Select all regex matches inside selections |
| `S` | Split selections on regex matches |
| `K` | Keep selections matching regex |
| `Alt+K` | Remove selections matching regex |
| `Alt+s` | Split selection on newlines |
| `;` | Collapse selection into single cursor |
| `Alt+;` | Flip selection cursor and anchor |
| `%` | Select whole document |
| `x` | Select current line, if already selected, extend to next line |
| `X` | Extend selection to line bounds |
| `Alt+x` | Shrink selection to line bounds |
| `Alt+o` | Expand selection to syntax node |
| `Alt+i` | Shrink selection to syntax node |
| `,` | Keep primary selection |
| `Alt+,` | Remove primary selection |
| `(` | Rotate selections backward |
| `)` | Rotate selections forward |
| `Alt+(` | Rotate selections contents backward |
| `Alt+)` | Rotate selection contents forward |
| `Alt+:` | Ensure all selections face forward |
| `C` | Copy selection on next line |
| `Alt+C` | Copy selection on previous line |
| `Alt+-` | Merge selections |
| `Alt+_` | Merge consecutive selections |
| `Alt+.` | Repeat last motion |

### Match and Surround (`m`)

| Key | Action |
|-----|--------|
| `mm` | Goto matching bracket |
| `ms<char>` | Surround add |
| `mr<from><to>` | Surround replace |
| `md<char>` | Surround delete |
| `ma<char>` | Select around object |
| `mi<char>` | Select inside object |

### View and Scroll (`z` / `Z`)

`z` and `Z` are the same prefix; either works.

| Key | Action |
|-----|--------|
| `zz` / `zc` | Align view center |
| `zt` / `z.` | Align view top |
| `zb` | Align view bottom |
| `zk` / `z↑` | Scroll view up |
| `zj` / `z↓` | Scroll view down |
| `Ctrl+b` / `PageUp` | Move page up |
| `Ctrl+f` / `PageDown` | Move page down |
| `Ctrl+u` | Move page and cursor half up |
| `Ctrl+d` | Move page and cursor half down |

### Comments and Macros

| Key | Action |
|-----|--------|
| `Ctrl+c` | Comment/uncomment selections |
| `Q` | Record macro |
| `q` | Replay macro |

Comment variants (line, block) are on the [Leader Menu](#leader-menu) under `Space+c`.

### Shell

| Key | Action |
|-----|--------|
| `\|` | Pipe selections through shell command |
| `Alt+\|` | Pipe selections into shell command ignoring output |
| `!` | Insert shell command output before selections |
| `Alt+!` | Append shell command output after selections |
| `$` | Filter selections with shell predicate |

## Select Mode

Select mode extends the current selection: motion keys move the selection's head instead of collapsing it. Enter it with `v`, leave it with `Escape`. Every Normal-mode command (editing, yank, search, and the rest) works the same here.

| Key | Action |
|-----|--------|
| `h/j/k/l` | Extend selection |
| `w/b/e/W/B/E` | Extend by word |
| `f/t/F/T<char>` | Extend to character |
| `Home` / `End` | Extend to line start/end |
| `x` | Select current line, if already selected, extend to next line |
| `gg` / `<n>gg` | Extend to line number `<n>` else file start |
| `ge` | Extend to last line |
| `g\|` | Extend to column |
| `n` / `N` | Add next/previous search match to selection |
| `Escape` | Exit selection mode |

## Insert Mode

Printable keys type text. The bindings below edit or move instead; anything unbound is inserted.

| Key | Action |
|-----|--------|
| `Escape` | Enter normal mode |
| `←` / `→` | Move by character |
| `↑` / `↓` | Move by line |
| `Home` | Goto line start |
| `End` | Goto newline at line end |
| `Ctrl+r <reg>` | Insert register |
| `Ctrl+s` | Commit changes to new checkpoint |
| `Ctrl+h` / `Backspace` | Delete previous char |
| `Ctrl+d` / `Delete` | Delete next char |
| `Ctrl+w` / `Alt+Backspace` | Delete previous word |
| `Alt+d` / `Alt+Delete` | Delete next word |
| `Ctrl+u` | Delete till start of line |
| `Ctrl+k` | Delete till end of line |
| `Ctrl+j` / `Return` | Insert newline char |
| `Tab` | Insert tab in leading whitespace, otherwise move past the enclosing syntax node |
| `Shift+Tab` | Insert tab |
| `Ctrl+x` | Complete current word |
| `PageUp` / `Ctrl+b` | Move page up |
| `PageDown` / `Ctrl+f` | Move page down |

### Completion Popup

While the completion popup is open:

| Key | Action |
|-----|--------|
| `Tab` | Accept completion |
| `Return` | Insert a newline and close the popup |
| `Escape` | Cancel completion |
| `↑` / `Ctrl+p` | Previous completion |
| `↓` / `Ctrl+n` | Next completion |
| `PageUp` | Previous completion page |
| `PageDown` | Next completion page |
| `Home` | First completion |
| `End` | Last completion |

## Mouse

In a document pane, with `mouse` enabled:

| Action | Result |
|--------|--------|
| Click | Focus the pane and move the cursor |
| Drag | Select text; dragging past an edge scrolls |
| `Alt` + click | Add a cursor at the click |
| `Ctrl` + click | Goto definition of the symbol under the click |

## Windows and Splits

Window management works from any editing mode. The `Ctrl+w` chord and the leader's `w` menu (`Space+w` or `Ctrl+\ w`) bind the same commands.

| Key | Action |
|-----|--------|
| `Ctrl+w n` | Create a new scratch buffer |
| `Ctrl+w x` | Open a new terminal |
| `Ctrl+w /` | Search the focused terminal's scrollback |
| `Ctrl+w v` / `Ctrl+w Ctrl+v` | Vertical right split |
| `Ctrl+w s` / `Ctrl+w Ctrl+s` | Horizontal bottom split |
| `Ctrl+w t` / `Ctrl+w Ctrl+t` | Transpose splits |
| `Ctrl+w q` / `Ctrl+w Ctrl+q` | Close window |
| `Ctrl+w o` / `Ctrl+w Ctrl+o` | Close windows except current |
| `Ctrl+w w` / `Ctrl+w Ctrl+w` | Goto next window |
| `Ctrl+w z` | Toggle focused pane maximized |
| `Ctrl+w h/j/k/l` / `Ctrl+w ←/↓/↑/→` | Jump to left/below/above/right split |
| `Ctrl+w H/J/K/L` | Swap with left/below/above/right split |
| `Ctrl+w r` | Enter resize mode |

Splitting a document or image pane opens another view of the same document or image; splitting a terminal starts a new shell. Splits can also be resized by dragging a separator with the mouse.

`Ctrl+w z` temporarily gives the focused pane the full editor area while preserving the split layout. The status line shows `MAX` while active. Press the key again to restore the layout.

### Resize Mode

`Ctrl+w r` enters resize mode.

| Key | Action |
|-----|--------|
| `h` / `←` | Push the left border left |
| `l` / `→` | Push the right border right |
| `j` / `↓` | Push the bottom border down |
| `k` / `↑` | Push the top border up |
| `Escape` / `Enter` | Exit resize mode |

## Leader Menu

`Space` (or `Ctrl+\`) opens the leader menu, a shared set of commands reachable from every mode. Pressing the leader shows a popup of the entries below.

| Key | Action |
|-----|--------|
| `Space+y` | Yank selections to clipboard |
| `Space+Y` | Yank main selection to clipboard |
| `Space+p` | Paste clipboard after selections |
| `Space+P` | Paste clipboard before selections |
| `Space+R` | Replace selections by clipboard content |
| `Space+w` | Window menu (see [Windows and Splits](#windows-and-splits)) |
| `Space+h` | Select symbol references |
| `Space+a` | Perform code action |
| `Space+k` | Show docs for item under cursor |
| `Space+r` | Rename symbol |
| `Space+s` | Open symbol picker |
| `Space+S` | Open workspace symbol picker |
| `Space+f` | Open file picker |
| `Space+F` | Open file picker at current working directory |
| `Space+g` | Open changed-file picker |
| `Space+e` | Open file explorer at workspace root |
| `Space+.` | Open file explorer at current pane's directory |
| `Space+b` | Open buffer picker |
| `Space+j` | Open jumplist picker |
| `Space+d` | Open diagnostic picker |
| `Space+D` | Open workspace diagnostic picker |
| `Space+/` | Global search in workspace folder |
| `Space+?` | Open command palette |
| `Space+'` | Reopen the last picker |
| `Space+c` | Comment/uncomment selections |
| `Space+Alt+c` | Line comment/uncomment selections |
| `Space+C` | Block comment/uncomment selections |

## Terminal Panes

While a terminal pane has focus, keys pass straight to the shell. The exceptions are `Ctrl+w`, which opens the window menu, and `Ctrl+\`, which opens the leader menu (`Space` is not intercepted here).

| Key | Action |
|-----|--------|
| `Ctrl+w` | Open the window menu |
| `Ctrl+w /` | Search the focused terminal's scrollback |
| `Ctrl+w q` | Close the pane and kill its shell |
| `Ctrl+\` | Open the terminal's leader menu |
| `Ctrl+\ p` | Paste the clipboard into the terminal |
| `Ctrl+\ f` / `Ctrl+\ b` | Open the file / buffer picker |
| Mouse wheel | Scroll into scrollback; any keypress returns to live output |
| Mouse click/drag | Select and copy terminal text when mouse tracking is off |
| Mouse click/drag/wheel | Forwarded to the shell when it enables mouse tracking (e.g. vim, htop) |

## Image Panes

Image panes support the command prompt, window menu, and leader.

| Key | Action |
|-----|--------|
| `:` | Enter command mode |
| `+` / `=` | Zoom image in |
| `-` | Zoom image out |
| `0` | Fit image to pane and recenter |
| `h` `j` `k` `l` / arrows | Pan a zoomed-in image |
| `g a` / `g m` | Goto last accessed / modified file |
| Mouse click | Zoom image in |
| Right click / `Mod` + click | Zoom image out |
| Mouse wheel / two-finger swipe | Pan a zoomed-in image |
| `Mod` + wheel | Zoom image in or out |
| `Ctrl+w` / `Space+w` | Window menu |

## Binary Panes

Files that are neither text nor a supported image open as a read-only
hexadecimal dump. Binary panes support the command prompt, window menu, and
leader.

| Key | Action |
|-----|--------|
| `:` | Enter command mode |
| `j` / `k` / arrows | Scroll one row |
| `PgDn` / `PgUp` | Scroll one page |
| `Home` | Jump to start |
| `G` / `End` | Jump to end |
| `g a` / `g m` | Goto last accessed / modified file |
| Mouse wheel | Scroll |
| `Ctrl+w` / `Space+w` | Window menu |

## Prompts

Commands (`:`), search (`/`, `?`), and other text prompts open a popup in the centre of the frame. Command completions list beside the input as you type, matched on the command name and annotated with what it does. Nothing is selected until you reach for it: `Tab` takes the top match into the input, the arrow keys highlight another one for `Tab` to take, and `Return` always submits the line as typed. There is no command line: messages appear as notifications in the bottom-right corner, and the selected register and macro-recording indicator sit on the corner statusline.

| Key | Action |
|-----|--------|
| `←` / `→` / `Ctrl+b` / `Ctrl+f` | Move by character |
| `Ctrl+←` / `Ctrl+→` / `Alt+b` / `Alt+f` | Move by word |
| `Home` / `Ctrl+a` | Move to start |
| `End` / `Ctrl+e` | Move to end |
| `Backspace` / `Ctrl+h` | Delete char before caret |
| `Delete` / `Ctrl+d` | Delete char after caret |
| `Ctrl+w` / `Alt+Backspace` | Delete word before caret |
| `Alt+d` / `Ctrl+Delete` | Delete word after caret |
| `Ctrl+u` | Delete to start |
| `Ctrl+k` | Delete to end |
| `Tab` | Accept the highlighted completion, else the top match |
| `↑` / `↓` / `Ctrl+p` / `Ctrl+n` | Highlight a completion |
| `Return` | Submit the line as typed |
| `Escape` | Cancel |

## Picker Navigation

When any picker is open (file picker, buffer picker, global search, etc.):

| Key | Action |
|-----|--------|
| `↑` / `Ctrl+p` / `Shift+Tab` | Move up |
| `↓` / `Ctrl+n` / `Tab` | Move down |
| `PageUp` / `Ctrl+u` | Move page up |
| `PageDown` / `Ctrl+d` | Move page down |
| `Home` | Jump to first item |
| `End` | Jump to last item |
| `Return` | Open selected item |
| `Ctrl+s` | Open in horizontal split |
| `Ctrl+v` | Open in vertical split |
| `Escape` | Close picker |
