# Icon Credits

## Redistributed here

These SVGs are copied from upstream unmodified, resolved by codepoint from the
glyphs toe renders:

| Files | Source | License |
|-------|--------|---------|
| `symbol-*.svg`, `file.svg`, `folder.svg`, `go-to-file.svg` | [Codicons](https://github.com/microsoft/vscode-codicons) — Microsoft Corporation | [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) (artwork; the repo's code is MIT) |
| `diff-*-16.svg`, `question-16.svg`, `alert-16.svg` | [Octicons](https://github.com/primer/octicons) — GitHub Inc. | MIT |

## Referenced, not redistributed

| What | Source | License |
|------|--------|---------|
| The glyph codepoints toe renders in the terminal (`internal/term/ui/completion-icons.go`, `picker-vcs.go`, `picker-buffer.go`), and the `glyphnames.json` mapping used to resolve them | [Nerd Fonts](https://github.com/ryanoasis/nerd-fonts) — Ryan L McIntyre | Multi-licensed: project source MIT, fonts and glyph SVGs [SIL OFL 1.1](https://openfontlicense.org/) |

No Nerd Font files are bundled with toe or with these docs — the editor emits
the codepoints and the user's terminal font supplies the glyphs.
