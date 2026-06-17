# toe

A modal terminal editor written in Go. Helix-inspired editing with normal, insert, and selection modes. Multiple buffers, split views, and a fuzzy picker for files, buffers, and search results.

Work in progress. Assume it will lose your edits.

## What works

- Modal editing (normal, insert, selection)
- Multiple buffers and split views
- Fuzzy file, buffer, and global search pickers with live preview
- Syntax highlighting
- Persistent undo history
- Soft wrap, rulers, whitespace rendering, auto-pairs
- User and workspace config in TOML

## What's still being built

- Code completion and hover docs
- Go-to definition, find references, rename
- Inline errors and warnings
- Git change indicators in the gutter
- Debugger integration

## Requirements

- Go 1.26
- A terminal with ANSI color support

## Build and install

```sh
make build    # writes to dist/toe
make install  # installs to $GOPATH/bin
```

## Usage

```sh
toe
toe path/to/file.go
toe file1 file2
```

## Configuration

```text
$XDG_CONFIG_HOME/toe/config.toml
$XDG_CONFIG_HOME/toe/languages.toml
```

Workspace config goes in `.toe/config.toml` and `.toe/languages.toml` at the project root.

## Development

```sh
make pre-commit   # run this before committing
make test
make coverage
```
