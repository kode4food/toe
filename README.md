# Thom's Own Editor (toe) <img src="./docs/img/logo.png" align="right" height="100"/>

![Build Status](https://github.com/kode4food/toe/actions/workflows/build.yml/badge.svg) [![Code Coverage](https://qlty.sh/gh/kode4food/projects/toe/coverage.svg)](https://qlty.sh/gh/kode4food/projects/toe) [![Maintainability](https://qlty.sh/gh/kode4food/projects/toe/maintainability.svg)](https://qlty.sh/gh/kode4food/projects/toe) [![GitHub](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/kode4food/toe/blob/main/LICENSE)

**toe** is a modal terminal editor for Go development. toe edits Go projects, not the universe.

Work in progress. Assume it will lose your edits.

![toe screenshot](./docs/img/screenshot.png)

## Super Opinionated

toe is opinionated because it is built for one tight workflow: editing Go projects from a terminal without growing into a general-purpose IDE. It favors modal editing, `gopls`, TOML config, project-local state, Git-aware navigation, and a small set of deliberate defaults over plugin sprawl or endless knobs.

- Modal editing: normal, insert, and selection modes; multi-cursor editing; persistent undo history
- Project navigation: multiple buffers, split views, fuzzy file/buffer pickers, global search, and live previews
- Go-focused language tooling: syntax highlighting, LSP completion, hover, signature help, formatting, symbols, code actions, rename, go-to navigation, and diagnostics
- Editor display: soft wrap, rulers, whitespace rendering, indent guides, gutters, configurable cursor shapes, and statusline elements
- Version control: git diff gutters, change navigation, reset-diff-change, changed-file picker, and statusline element
- Project state: user/workspace TOML config, EditorConfig, session persistence, external file change detection, and clean-buffer reloads
- 4 Catppuccin themes: frappe, latte, macchiato, mocha

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
