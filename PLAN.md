# toe Finalization Plan

toe is a Go-native modal terminal editor for Go developers.

**Mission: toe edits Go projects, not the universe.**

This plan captures the remaining finalization work. Earlier phases (core text model, document model, rendering, TUI, commands, pickers, LSP, VCS, themes, config, sessions) are complete and documented under `docs/`. See the Architecture page in the docs for how the packages fit together.

## Working Rules

- Read this file before implementation work.
- Preserve modular ownership boundaries; use toe-native package boundaries and avoid speculative abstractions.
- Keep `view.Options` limited to innate editor behavior that core editor, document, action, or renderer code must consult directly. Module-owned config stays with the owning module.
- Do not mark a feature complete until it is implemented, tested, documented where applicable, and reflected here.

## Out Of Scope

- **Snippets** — decided 2026-07-05. A snippet engine (parser, tabstops, placeholder rendering, variable resolution, regex transforms, navigation state) is large surface for marginal UX now that AI completion covers boilerplate expansion, and signature help already covers post-acceptance call parameters. The LSP client advertises `snippetSupport: false` so servers send plain-text completions, and snippet workspace edits are rejected as unsupported. Revisit only if a concrete need emerges.
- **DAP** — on hold until finalization is complete. Debug adapter registry, transport, client lifecycle, execution control, pickers, breakpoints, and a test adapter all land together when resumed.

## Remaining Work

### Repository And Runtime

- [ ] Add CI entry points.
- [ ] Add a check that validates `PLAN.md` status against implemented package tests where practical.
- [ ] Finish runtime asset layout validation for supported languages only.

### Tree-Sitter Queries And Syntax Features

The largest remaining chunk. Bundle and load each query kind, then the features that consume them:

- [ ] Injection queries; overlay highlighting after injection/scope support lands.
- [ ] Locals queries.
- [ ] Textobject queries; textobject selection and syntax-aware selection expansion/shrinking.
- [ ] Indent queries; Tree-sitter indentation heuristics and exact comment-continuation behavior.
- [ ] Tags queries; revisit Tree-sitter fallback symbol pickers after they land.
- [ ] Folds queries.
- [ ] Rainbow bracket queries.
- [ ] Tree-sitter-aware bracket matching.
- [ ] Tree-sitter-aware surround pair finding.
- [ ] Generated tests: every supported language entry parses; every supported runtime query file is discoverable.

### Command Mode

- [ ] Finish register expansion coverage.
- [ ] Finish variable expansion coverage.
- [ ] Tokenizer/parser tests for expansions, flags, signatures, and raw-after behavior.

### Pickers

- [ ] Diagnostics picker and workspace diagnostics picker.

### Registers And Clipboard

- [ ] Black-hole register behavior.
- [ ] OSC 52 clipboard support where practical.
- [ ] Clipboard provider tests with fake providers.

### Theme And Config

- [ ] Theme parse/style tests; generated tests that all four Catppuccin variants parse.
- [ ] Full config parse/merge coverage for the modeled config surface.
- [ ] Interactive workspace trust prompts.
- [ ] Config event fanout to VCS and any future LSP reload needs.
- [ ] Terminal-info/backend capability detection beyond current true-color checks.

### Generated And Behavior Tests

- [ ] Every documented command is registered.
- [ ] Every default keybinding resolves.
- [ ] Static command behavior tests and typable command behavior tests.
- [ ] Key parsing tests, key trie tests, default keymap coverage tests.
- [ ] Full regex search command tests.
- [ ] Split tree tests.

### UI

- [ ] Rendering golden tests.
- [ ] Bubbletea model update tests.
- [ ] Complete mouse behavior audit and tests for remaining gaps.

### Integration

- [x] Launch `toe` in a pseudo-terminal for integration coverage (`cmd/toe/integration_test.go`; builds the binary once in `TestMain`, drives it through a pty with real keystrokes, asserts on ANSI-stripped screen output and saved file bytes).
- [x] Exercise open/edit/save/reload flows.
- [x] Exercise normal/insert/select mode transitions.
- [x] Exercise multiple cursors (copy-on-next-line, insert at both cursors, verify saved bytes).
- [x] Exercise search and replace flows.
- [x] Exercise splits and buffers.
- [ ] Exercise LSP end-to-end with a small test server.

### VCS

- [ ] Refresh diff bases on external head movement (commits made outside toe are picked up on save/reload only).

### Conditional Items

Implement only if a remaining feature above needs them; otherwise prune during the final audit:

- Tendril/compact string semantics; remaining visual offset helpers; remaining range helper utilities; `UrlConversionErrorKind` detail.
- Range filtering/mapping helpers beyond `Selection.Map`, `ChangeSet.MapPos`, `ChangeSet.MapRange`.
- Mark mapping over edits; `ChangeIterator`.
- Encodings beyond UTF-8/UTF-8-BOM; Unicode line endings; save cleanup in transaction/history; file lock/read-only behavior; expansion variables.
- Custom command clipboard provider.
- Dedicated menu/select widgets; progress spinner UI; word jump labels.
- Events And Jobs framework (event registration, hooks, debouncing, task cancellation, redraw locks, job queue) — current direct wiring works; adopt only if a feature demands decoupling.
- `ProgressStatus`, `LspProgressMap`, `ApplyEditError`, `ApplyEditErrorKind` public abstractions.
- `DiagnosticTag`, `NumberOrString` in diagnostics display.

### Final Audit

- [ ] Remove any stale checklist item that no longer maps to a toe capability or concrete behavior.
- [ ] Confirm the final supported surface is documented and tested.

## Completion Definition

A remaining item is complete only when:

- The Go implementation exists.
- Black-box tests cover normal and edge cases.
- Public commands and keybindings are registered where applicable.
- Errors/status messages are documented or tested when user-visible.
- `go test ./...` passes.
- `PLAN.md` is updated in the same change.
