# toe Finalization Plan

toe is a Go-native modal terminal editor for Go developers.

**Mission: toe edits Go projects, not the universe.**

This plan captures the remaining finalization work. Earlier phases (core text model, document model, rendering, TUI, commands, pickers, LSP, VCS, themes, config, sessions) are complete and documented under `docs/`. See the Architecture page in the docs for how the packages fit together.

## Working Rules

- Read this file before implementation work.
- Preserve modular ownership boundaries; use toe-native package boundaries and avoid speculative abstractions.
- Keep `view.Options` limited to innate editor behavior that core editor, document, action, or renderer code must consult directly. Module-owned config stays with the owning module.
- Do not mark a feature complete until it is implemented, tested, documented where applicable, and reflected here.


## Remaining Work

### Repository And Runtime

- [x] Add CI entry points.
- [x] Add a check that validates `PLAN.md` status against implemented package tests where practical.
- [x] Finish runtime asset layout validation for supported languages only.

### Tree-Sitter Editing Features

- [ ] Injection queries; overlay highlighting after injection/scope support lands.
- [ ] Textobject queries and textobject selection.
- [x] Syntax-aware selection expansion/shrinking.
- [x] Tree-sitter-aware bracket matching.
- [ ] Tree-sitter-aware surround pair finding.
- [x] Generated tests: every supported language entry parses; every supported runtime query file is discoverable.

### Command Mode

- [x] Finish register expansion coverage.
- [x] Finish variable expansion coverage.
- [x] Tokenizer/parser tests for expansions, flags, signatures, and raw-after behavior.

### Pickers

- [x] Diagnostics picker and workspace diagnostics picker.

### Registers And Clipboard

- [x] Black-hole register behavior.
- [x] OSC 52 clipboard support where practical.
- [x] Clipboard provider tests with fake providers.

### Theme And Config

- [x] Theme parse/style tests; generated tests that all four Catppuccin variants parse.
- [x] Full config parse/merge coverage for the modeled config surface.
- [x] Workspace trust startup prompt and workspace config guard.

### Generated And Behavior Tests

- [x] Every documented command is registered.
- [x] Every default keybinding resolves.
- [x] Static command behavior tests and typable command behavior tests.
- [x] Key parsing tests, key trie tests, default keymap coverage tests.
- [x] Full regex search command tests.
- [x] Split tree tests.

### UI

- [x] Picker preview split dragging and auto-session persistence.
- [x] Bubbletea model update tests.
- [x] Complete mouse behavior audit and tests for remaining gaps.

### Integration

- [x] Launch `toe` in a pseudo-terminal for integration coverage (`cmd/toe/integration_test.go`; builds the binary once in `TestMain`, drives it through a pty with real keystrokes, asserts on ANSI-stripped screen output and saved file bytes).
- [x] Exercise open/edit/save/reload flows.
- [x] Exercise normal/insert/select mode transitions.
- [x] Exercise multiple cursors (copy-on-next-line, insert at both cursors, verify saved bytes).
- [x] Exercise search and replace flows.
- [x] Exercise splits and buffers.
- [x] Exercise LSP end-to-end with a small test server.

### VCS

- [x] Refresh diff bases on external head movement.

### Final Audit

- [ ] Confirm the final supported surface is documented and tested.

## Completion Definition

A remaining item is complete only when:

- The Go implementation exists.
- Black-box tests cover normal and edge cases.
- Public commands and keybindings are registered where applicable.
- Errors/status messages are documented or tested when user-visible.
- `go test ./...` passes.
- `PLAN.md` is updated in the same change.
