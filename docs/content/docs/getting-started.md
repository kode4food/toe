---
title: "Getting Started"
weight: 10
---

# Getting Started

toe is a modal terminal editor built for Go development. toe edits Go projects, not the universe.

## Requirements

- Go 1.26 or later when building from source
- A terminal with ANSI color support
- `gopls` on `PATH` for Go language features
- A Kitty graphics capable terminal
- Nerd Font glyphs for enhanced UI

## Installing

Install the latest stable release with Homebrew (recommended):

```sh
brew install kode4food/tap/toe
```

To build and install the current source:

```sh
git clone https://github.com/kode4food/toe
cd toe
make install   # installs to $GOPATH/bin
```

To build without installing:

```sh
make build   # writes to dist/toe
```

## Opening Files

```sh
toe                        # open in current directory
toe path/to/file.go        # open a single file
toe file1.go file2.go      # open multiple files
toe path/to/project        # use a directory as the project root
```

Only the first positional argument may be a directory.

## Modes

toe is a modal editor. Every key press means something different depending on the current mode.

| Mode | How to enter | Purpose |
|------|-------------|---------|
| **Normal** | `Escape` (or start here) | Navigation, commands, editing operations |
| **Insert** | `i`, `a`, `o`, `A`, `I`, `O` | Type text |
| **Select** | `v` | Extend and manipulate selections |
| **Image** | Open or focus an image pane | Zoom and window commands |
| **Terminal** | Open or focus a terminal pane | Shell input and window commands |

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

Use `Space` or `Ctrl+\` to open the leader menu. In terminal panes, use `Ctrl+\`.

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

Image files open in image panes when the terminal supports the Kitty graphics protocol. File picker previews also show PNG, JPEG, and GIF images in the preview pane.

### Splits

```
Ctrl+w s      horizontal split
Ctrl+w v      vertical split
Ctrl+w q      close current split
Ctrl+w h/j/k/l  navigate between splits
```

### Command Mode

Press `:` in Normal mode to type a command. Use `Tab` and `Shift+Tab` to cycle completions. `Space+?` opens the command palette.

→ See [Key Bindings]({{< relref "/docs/key-bindings" >}}) for the complete reference.
