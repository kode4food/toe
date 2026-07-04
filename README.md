# Thom's Own Editor (toe) <img src="./docs/img/logo.png" align="right" height="100"/>

![Build Status](https://github.com/kode4food/toe/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/toe/coverage.svg)](https://qlty.sh/gh/kode4food/projects/toe) [![Maintainability](https://qlty.sh/gh/kode4food/projects/toe/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/toe) [![GitHub](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/kode4food/toe/blob/main/LICENSE)

**toe** is a modal terminal editor for Go development. toe edits Go projects, not the universe.

Work in progress. Assume it will lose your edits.

![toe screenshot](./docs/img/screenshot.png)

## What works

- Modal editing (normal, insert, selection)
- Multiple buffers and split views
- Fuzzy file, buffer, and global search pickers with live preview
- Syntax highlighting
- Persistent undo history
- Soft wrap, rulers, whitespace rendering, auto-pairs
- Language server support for completion, hover, signature help, formatting, symbols, code actions, rename, and go-to navigation
- Diagnostics with underlines, gutter markers, status counts, and cursor-scoped popup text
- User and workspace config in TOML
- Session persistence: open documents, split layout, cursor positions, and view modes survive restarts

## What's still being built

- Snippet expansion
- Tree-sitter textobjects, indentation queries, bracket matching, and surround matching
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
