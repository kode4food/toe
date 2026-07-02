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
| `w` | Next word start |
| `b` | Previous word start |
| `e` | Next word end |
| `W` | Next WORD start |
| `B` | Previous WORD start |
| `E` | Next WORD end |
| `f<char>` | Move to next occurrence of char (inclusive) |
| `t<char>` | Move till next occurrence of char |
| `F<char>` | Move to previous occurrence of char |
| `T<char>` | Move till previous occurrence of char |
| `Home` | Line start |
| `End` | Line end |
| `gg` / `<n>gg` | File start (or go to line `n`) |
| `G` / `<n>G` | Last line (or go to line `n`) |
| `gs` | First non-whitespace character |
| `ge` | Last line |
| `g\|` / `<n>g\|` | Go to column `n` |
| `]p` | Next paragraph |
| `[p` | Previous paragraph |

### Goto Prefix (`g`)

| Key | Action |
|-----|--------|
| `gd` | Go to definition |
| `gD` | Go to declaration |
| `gy` | Go to type definition |
| `gi` | Go to implementation |
| `gr` | Go to references |
| `gf` | Open file or URL in selection |
| `gn` | Next buffer |
| `gp` | Previous buffer |
| `ga` | Last accessed file |
| `gm` | Last modified file |
| `.` | Last modification location |
| `gt` | Window top |
| `gc` | Window center |
| `gb` | Window bottom |

### Jumplist

| Key | Action |
|-----|--------|
| `Ctrl+o` | Jump backward |
| `Ctrl+i` / `Tab` | Jump forward |
| `Ctrl+s` | Save current position to jumplist |

### Entering Other Modes

| Key | Action |
|-----|--------|
| `i` | Insert before selection |
| `I` | Insert at line start |
| `a` | Append after selection |
| `A` | Append at line end |
| `o` | Open line below and insert |
| `O` | Open line above and insert |
| `v` | Enter Select mode |

### Editing

| Key | Action |
|-----|--------|
| `d` | Delete selection (yanks) |
| `Alt+d` | Delete without yank |
| `c` | Change selection (delete and insert, yanks) |
| `Alt+c` | Change without yank |
| `r<char>` | Replace selection with char |
| `u` | Undo |
| `U` | Redo |
| `Alt+u` | Step to earlier history branch |
| `Alt+U` | Step to later history branch |
| `~` | Toggle case |
| `` ` `` | Lowercase |
| `Alt+`` ` ``  | Uppercase |
| `>` | Indent |
| `<` | Unindent |
| `J` | Join lines |
| `Alt+J` | Join lines with spaces |
| `&` | Align selections in column |
| `_` | Trim whitespace from selections |
| `Ctrl+a` | Increment number |
| `Ctrl+x` | Decrement number |
| `=` | Format selection |

### Yank and Paste

| Key | Action |
|-----|--------|
| `y` | Yank |
| `p` | Paste after |
| `P` | Paste before |
| `R` | Replace with yanked |
| `Space+y` | Yank to system clipboard |
| `Space+Y` | Yank primary selection to clipboard |
| `Space+p` | Paste from clipboard after |
| `Space+P` | Paste from clipboard before |
| `Space+R` | Replace with clipboard |
| `"<reg>` | Select register for next yank/paste |

### Search

| Key | Action |
|-----|--------|
| `/` | Search forward |
| `?` | Search backward |
| `n` | Next match |
| `N` | Previous match |
| `*` | Search word under cursor (whole word) |
| `Alt+*` | Search selection |

### Selection Manipulation

| Key | Action |
|-----|--------|
| `s` | Select regex matches within selection |
| `S` | Split selection by regex |
| `K` | Keep selections matching regex |
| `Alt+K` | Remove selections matching regex |
| `Alt+s` | Split selection on newlines |
| `;` | Collapse to single cursor |
| `Alt+;` | Flip cursor/anchor |
| `%` | Select all |
| `x` | Extend selection by line |
| `X` | Extend to line bounds |
| `Alt+x` | Shrink to line bounds |
| `,` | Keep only primary selection |
| `Alt+,` | Remove primary selection |
| `(` | Rotate selections backward |
| `)` | Rotate selections forward |
| `Alt+(` | Rotate contents backward |
| `Alt+)` | Rotate contents forward |
| `Alt+:` | Ensure selections are forward |
| `C` | Copy selection on next line |
| `Alt+C` | Copy selection on previous line |
| `Alt+-` | Merge selections |
| `Alt+_` | Merge consecutive selections |
| `Alt+.` | Repeat last motion |

### Match (`m` prefix)

| Key | Action |
|-----|--------|
| `mm` | Go to matching bracket |
| `ms<char>` | Add surround character |
| `mr<from><to>` | Replace surround character |
| `md<char>` | Delete surround character |
| `ma<char>` | Select around text object |
| `mi<char>` | Select inside text object |

### View / Scroll (`z` prefix)

| Key | Action |
|-----|--------|
| `zz` / `zcz` | Center view on cursor |
| `zt` / `z.` | Align view top to cursor |
| `zb` | Align view bottom to cursor |
| `zk` / `z↑` | Scroll up |
| `zj` / `z↓` | Scroll down |
| `Ctrl+b` / `PageUp` | Page up |
| `Ctrl+f` / `PageDown` | Page down |
| `Ctrl+u` | Half-page up |
| `Ctrl+d` | Half-page down |

### Splits (`Ctrl+w` or `Space+w`)

| Key | Action |
|-----|--------|
| `Ctrl+w v` | Vertical split |
| `Ctrl+w s` | Horizontal split |
| `Ctrl+w n v` | Vertical split with new buffer |
| `Ctrl+w n s` | Horizontal split with new buffer |
| `Ctrl+w q` | Close current split |
| `Ctrl+w o` | Close other splits |
| `Ctrl+w w` | Cycle to next split |
| `Ctrl+w t` | Transpose splits |
| `Ctrl+w h/j/k/l` | Jump to split in direction |
| `Ctrl+w H/J/K/L` | Swap split in direction |

All `Ctrl+w` bindings also work with `Space+w`.

### Space Menu

| Key | Action |
|-----|--------|
| `Space+f` | File picker |
| `Space+F` | File picker in current directory |
| `Space+e` | File explorer |
| `Space+.` | File explorer in buffer's directory |
| `Space+b` | Buffer picker |
| `Space+j` | Jumplist picker |
| `Space+/` | Global search |
| `Space+?` | Command palette |
| `Space+'` | Reopen last picker |
| `Space+k` | Hover docs (LSP) |
| `Space+a` | Code action (LSP) |
| `Space+r` | Rename symbol (LSP) |
| `Space+h` | Select all references (LSP) |
| `Space+s` | Document symbol picker (LSP) |
| `Space+S` | Workspace symbol picker (LSP) |
| `Space+c` | Toggle comment |
| `Space+C` | Toggle block comment |
| `Space+Alt+c` | Toggle line comment |
| `Space+y` | Yank to clipboard |
| `Space+p` | Paste from clipboard |

### Prev/Next (`[` / `]`)

| Key | Action |
|-----|--------|
| `[p` | Previous paragraph |
| `]p` | Next paragraph |
| `[␣` | Add newline above |
| `]␣` | Add newline below |

### Comments and Macros

| Key | Action |
|-----|--------|
| `Ctrl+c` | Toggle comment |
| `Q` | Record macro |
| `q` | Replay macro |

### Shell

| Key | Action |
|-----|--------|
| `\|` | Pipe selection through shell command |
| `Alt+\|` | Pipe to shell (discard output) |
| `!` | Insert shell output before selection |
| `Alt+!` | Append shell output after selection |
| `$` | Filter selection with shell predicate |

---

## Insert Mode

| Key | Action |
|-----|--------|
| `Escape` | Return to Normal mode |
| `←↓↑→` | Move cursor |
| `Home` | Line start |
| `End` | Line end |
| `Ctrl+r <reg>` | Insert register contents |
| `Ctrl+s` | Commit undo checkpoint |
| `Ctrl+h` / `Backspace` | Delete character backward |
| `Ctrl+d` / `Delete` | Delete character forward |
| `Ctrl+w` / `Alt+Backspace` | Delete word backward |
| `Alt+d` / `Alt+Delete` | Delete word forward |
| `Ctrl+u` | Delete to line start |
| `Ctrl+k` | Delete to line end |
| `Ctrl+j` / `Return` | Insert newline |
| `Tab` | Smart indent |
| `Ctrl+x` | Show completion popup |
| `PageUp` / `Ctrl+b` | Page up |
| `PageDown` / `Ctrl+f` | Page down |

### Completion Popup

| Key | Action |
|-----|--------|
| `Return` / `Tab` | Accept completion |
| `Escape` | Cancel |
| `↑` / `Ctrl+p` | Previous item |
| `↓` / `Ctrl+n` | Next item |
| `PageUp` | Page up |
| `PageDown` | Page down |
| `Home` | First item |
| `End` | Last item |

---

## Select Mode

Select mode extends the current selection. Motion keys move the selection's
head rather than collapsing it.

| Key | Action |
|-----|--------|
| `h/j/k/l` | Extend selection |
| `w/b/e/W/B/E` | Extend by word |
| `f/t/F/T` | Extend to character |
| `Home` / `End` | Extend to line start/end |
| `x` | Extend by line |
| `n` / `N` | Add next/previous search match to selection |
| `Escape` | Return to Normal mode |

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
