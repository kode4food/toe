---
title: "Getting Started"
weight: 10
---

# Getting Started

toe is a modal terminal editor built for Go development. toe edits Go projects, not the universe.

## Requirements

- Go 1.26 or later
- A terminal with ANSI color support

## Installing

```sh
git clone https://github.com/kode4food/toe
cd toe
make install   # installs to $GOPATH/bin
```

Or build without installing:

```sh
make build   # writes to dist/toe
```

## Opening Files

```sh
toe                        # open in current directory
toe path/to/file.go        # open a single file
toe file1.go file2.go      # open multiple files
```

## Modes

toe is a modal editor. Every key press means something different depending on the current mode.

| Mode | How to enter | Purpose |
|------|-------------|---------|
| **Normal** | `Escape` (or start here) | Navigation, commands, editing operations |
| **Insert** | `i`, `a`, `o`, `A`, `I`, `O` | Type text |
| **Select** | `v` | Extend and manipulate selections |

The mode is shown in the status bar at the bottom of the screen.

## First Steps

After opening a file:

```
h j k l       move left / down / up / right
i             enter Insert mode before cursor
Escape        return to Normal mode
u             undo
U             redo
```

### Saving and Quitting

```
:w            save (write)
:q            quit
:wq           save and quit
:q!           quit without saving
:wq!          force save and quit
```

### Finding Files

```
Space+f       open file picker
Space+b       open buffer picker
Space+g       open changed-file picker
Space+/       global search
```

Start typing to filter. `Enter` opens the selection.

### Splits

```
Ctrl+w s      horizontal split
Ctrl+w v      vertical split
Ctrl+w q      close current split
Ctrl+w h/j/k/l  navigate between splits
```

### Command Mode

Press `:` in Normal mode to type a command. Commands support aliases and tab completion via the command palette (`Space+?`).

→ See [Key Bindings]({{< relref "/docs/key-bindings" >}}) for the complete reference.
