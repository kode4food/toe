---
title: "Key Bindings"
weight: 30
---

# Key Bindings

## Normal Mode

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
| `gg` / `<n>gg` | Goto line number `<n>` else file start |
| `G` / `<n>G` | Goto line |
| `gs` | Goto first non-blank in line |
| `ge` | Goto last line |
| `g\|` / `<n>g\|` | Goto column |
| `]p` | Goto next paragraph |
| `[p` | Goto previous paragraph |

### Goto Prefix (`g`)

| Key | Action |
|-----|--------|
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
| `gt` | Goto window top |
| `gc` | Goto window center |
| `gb` | Goto window bottom |

### Jumplist

| Key | Action |
|-----|--------|
| `Ctrl+o` | Jump backward on jumplist |
| `Ctrl+i` / `Tab` | Jump forward on jumplist |
| `Ctrl+s` | Save current selection to jumplist |

### Entering Other Modes

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
| `u` | Undo change |
| `U` | Redo change |
| `Alt+u` | Move backward in history |
| `Alt+U` | Move forward in history |
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

### Yank and Paste

| Key | Action |
|-----|--------|
| `y` | Yank selection |
| `p` | Paste after selection |
| `P` | Paste before selection |
| `R` | Replace with yanked text |
| `Space+y` | Yank selections to clipboard |
| `Space+Y` | Yank main selection to clipboard |
| `Space+p` | Paste clipboard after selections |
| `Space+P` | Paste clipboard before selections |
| `Space+R` | Replace selections by clipboard content |
| `"<reg>` | Select register |

### Search

| Key | Action |
|-----|--------|
| `/` | Search for regex pattern |
| `?` | Reverse search for regex pattern |
| `n` | Select next search match |
| `N` | Select previous search match |
| `*` | Use current selection as search pattern, word bounded |
| `Alt+*` | Use current selection as search pattern |

### Selection Manipulation

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

### Match (`m` prefix)

| Key | Action |
|-----|--------|
| `mm` | Goto matching bracket |
| `ms<char>` | Surround add |
| `mr<from><to>` | Surround replace |
| `md<char>` | Surround delete |
| `ma<char>` | Select around object |
| `mi<char>` | Select inside object |

### View / Scroll (`z` / `Z` prefix)

| Key | Action |
|-----|--------|
| `zz` / `zc` / `Zz` / `Zc` | Align view center |
| `zt` / `z.` / `Zt` / `Z.` | Align view top |
| `zb` / `Zb` | Align view bottom |
| `zk` / `z↑` / `Zk` / `Z↑` | Scroll view up |
| `zj` / `z↓` / `Zj` / `Z↓` | Scroll view down |
| `Ctrl+b` / `PageUp` | Move page up |
| `Ctrl+f` / `PageDown` | Move page down |
| `Ctrl+u` | Move page and cursor half up |
| `Ctrl+d` | Move page and cursor half down |

### Splits (`Ctrl+w` or `Space+w`)

| Key | Action |
|-----|--------|
| `Ctrl+w v` / `Ctrl+w Ctrl+v` | Vertical right split |
| `Ctrl+w s` / `Ctrl+w Ctrl+s` | Horizontal bottom split |
| `Ctrl+w n v` / `Ctrl+w n Ctrl+v` | Vertical right split scratch buffer |
| `Ctrl+w n s` / `Ctrl+w n Ctrl+s` | Horizontal bottom split scratch buffer |
| `Ctrl+w q` / `Ctrl+w Ctrl+q` | Close window |
| `Ctrl+w o` / `Ctrl+w Ctrl+o` | Close windows except current |
| `Ctrl+w w` / `Ctrl+w Ctrl+w` | Goto next window |
| `Ctrl+w t` / `Ctrl+w Ctrl+t` | Transpose splits |
| `Ctrl+w h/j/k/l` / `Ctrl+w Ctrl+h/j/k/l` | Jump to left/below/above/right split |
| `Ctrl+w H/J/K/L` | Swap with left/below/above/right split |

All `Ctrl+w` bindings also work with `Space+w`.

### Space Menu

| Key | Action |
|-----|--------|
| `Space+y` | Yank selections to clipboard |
| `Space+Y` | Yank main selection to clipboard |
| `Space+p` | Paste clipboard after selections |
| `Space+P` | Paste clipboard before selections |
| `Space+R` | Replace selections by clipboard content |
| `Space+w` | Window (see Splits) |
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
| `Space+.` | Open file explorer at current buffer's directory |
| `Space+b` | Open buffer picker |
| `Space+j` | Open jumplist picker |
| `Space+/` | Global search in workspace folder |
| `Space+?` | Open command palette |
| `Space+'` | Reopen the last picker |
| `Space+c` | Comment/uncomment selections |
| `Space+Alt+c` | Line comment/uncomment selections |
| `Space+C` | Block comment/uncomment selections |

### Prev/Next (`[` / `]`)

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

### Comments and Macros

| Key | Action |
|-----|--------|
| `Ctrl+c` | Comment/uncomment selections |
| `Q` | Record macro |
| `q` | Replay macro |

### Shell

| Key | Action |
|-----|--------|
| `\|` | Pipe selections through shell command |
| `Alt+\|` | Pipe selections into shell command ignoring output |
| `!` | Insert shell command output before selections |
| `Alt+!` | Append shell command output after selections |
| `$` | Filter selections with shell predicate |

---

## Insert Mode

| Key | Action |
|-----|--------|
| `Escape` | Enter normal mode |
| `←↓↑→` | Move cursor |
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
| `Tab` | Insert tab if all cursors have all whitespace to their left, else complete current word |
| `Ctrl+x` | Complete current word |
| `PageUp` / `Ctrl+b` | Move page up |
| `PageDown` / `Ctrl+f` | Move page down |

### Completion Popup

| Key | Action |
|-----|--------|
| `Return` / `Tab` | Accept completion |
| `Escape` | Cancel completion |
| `↑` / `Ctrl+p` | Previous completion |
| `↓` / `Ctrl+n` | Next completion |
| `PageUp` | Previous completion page |
| `PageDown` | Next completion page |
| `Home` | First completion |
| `End` | Last completion |

---

## Select Mode

Select mode extends the current selection. Motion keys move the selection's head rather than collapsing it.

| Key | Action |
|-----|--------|
| `h/j/k/l` | Extend selection |
| `w/b/e/W/B/E` | Extend by word |
| `f/t/F/T` | Extend to character |
| `Home` / `End` | Extend to line start/end |
| `x` | Select current line, if already selected, extend to next line |
| `gg` | Extend to line number `<n>` else file start |
| `ge` | Extend to last line |
| `g\|` | Extend to column |
| `n` / `N` | Add next/previous search match to selection |
| `Escape` | Exit selection mode |

All other Normal mode commands (editing, clipboard, search) work the same in
Select mode.

---

## Picker Navigation

When any picker is open (file picker, buffer picker, global search, etc.):

| Key | Action |
|-----|--------|
| `↑` / `Ctrl+p` | Move up |
| `↓` / `Ctrl+n` | Move down |
| `Return` | Open selected item |
| `Ctrl+s` | Open in horizontal split |
| `Ctrl+v` | Open in vertical split |
| `Escape` | Close picker |
