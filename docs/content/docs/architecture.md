---
title: "Architecture"
weight: 60
---

# Architecture

toe is a Go-native modal terminal editor built on Bubbletea, Lipgloss, Tree-sitter, and Chroma. This page explains the current package layout, the main data flow, and the integration points that exist in the code today.

## Design Principles

- **toe edits Go projects, not the universe.** Features exist because Go development needs them, not because other editors have them.
- **Persistent editing values.** The core text and edit values (`Rope`, `ChangeSet`, `Transaction`, `Selection`, and `Range`) return new values rather than mutating their inputs. `History` is the exception in the current implementation: it is owned by a document and mutates its revision cursor and revision list while storing immutable transactions.
- **Modular ownership.** LSP, VCS, pickers, themes, and command modules keep their state and configuration close to the module that owns the behavior. The editor exposes narrow interfaces where decoupled services need to plug in.
- **Render once, cache everything expensive.** The render path runs on every keystroke. Parsed syntax queries, syntax caches, raw document text, highlight spans, search spans, preview entries, and line-prefix scans are cached and invalidated by revision or input changes.

## Package Layers

Dependencies point downward; lower layers never import higher ones.

### Core Model

Packages: `internal/core`.

Persistent text (`Rope`), selections, ranges, movement, transactions (`ChangeSet`, `Transaction`), search, comments, indentation, brackets, surround helpers, text objects, wrapping, and undo/redo history. This package has no terminal UI or file I/O. Most editing values are immutable value types; `History` is stateful and lives inside a `Document`.

### Editor State

Packages: `internal/view` and its subpackages.

The editor, documents, view tree (splits), sessions, file I/O, overlays, diagnostics, and service interfaces. A `Document` owns text, revision, language, history, diagnostics, LSP overlay state, and per-view selections; a `View` is a window onto a document; the `Editor` owns the document table, split tree, focus, runtime options, registers, document observers, and optional service controllers. Subpackages:

- `view/config` — raw editor config loading/merging and EditorConfig support.
- `view/language` — language configuration, matching, formatter metadata, server metadata, indentation, auto-pair, and soft-wrap settings.
- `view/register` — the in-memory register store, including the default and black-hole registers.
- `view/action` — reusable editor actions invoked by commands and UI components.

`view.Options` is deliberately limited to innate editor behavior that core editor, document, action, or renderer code must consult directly. Module-owned settings live with their module.

### Command System

Packages: `internal/term/command`, `internal/term/defaults`.

`term/command` provides the machinery: command signatures, argument parsing, tokenization and expansion, completion, key parsing, key tries, keymaps, option registration, config sections, and the registry. `term/defaults` provides the content: built-in command modules, default key bindings, module-owned config structs, and live option handlers. Many commands resolve directly to `view/action` calls; others bridge to the UI model, LSP, VCS, shell commands, sessions, or config reload.

### Terminal UI

Packages: `internal/term/ui`, `internal/tui`.

`term/ui` contains the Bubbletea model: the document renderer, status line, prompt, pickers, completion popup, hover and signature popups, overlays, macro handling, mouse handling, and event routing. `internal/tui` is the low-level terminal layer: cell buffers, styles, graphics primitives, spans, and ANSI rendering.

Overlays implement one of two interfaces: `BufferOverlayComponent` (`RenderOverBuffer`) writes cells directly into the frame buffer and is the fast path for complex panels; `OverlayComponent` composes lipgloss layers and suits simple string-based overlays like the command prompt. Bordered popups share the `popup` helper so content and border render in one pass.

### Syntax And Themes

Packages: `internal/term/syntax`, `internal/term/highlight`, `internal/term/theme`.

`syntax` owns the Tree-sitter language registry, embedded highlight, injection, and textobject queries, query inheritance, parser/query caches, Tree-sitter tokenization, syntax-aware selection, bracket matching, and surround-pair lookup for the supported languages. `highlight` is the Chroma fallback and also provides language detection and fallback styles. `theme` decodes embedded Catppuccin themes and maps scope names onto Lipgloss styles. Editor rendering caches highlight spans per document revision; picker previews cache spans for open documents by revision and file previews by path.

Highlight queries are bundled for every supported Tree-sitter language. Injection and textobject queries are bundled where toe has behavior that consumes them.

### Services

Packages: `internal/lsp`, `internal/vcs`.

- `internal/lsp` implements the language-server client: transport, lifecycle, capability negotiation, dynamic file-watch registration, document sync, diagnostics, completion, hover, signature help, navigation, symbols, code actions, rename, formatting, document links, inlay hints, document colors, progress, and workspace edits. Server metadata comes from merged `languages.toml` data.
- `internal/vcs` implements version-control integration behind a provider registry. Git is the only provider today, shelling out to the git binary. A debounced per-document diff worker (`vcs.Differ`) computes line hunks; `vcs.Attach` wires a session into document lifecycle observers, installs the editor's `view.VersionControl` implementation, and exposes update events for gutter/status/picker rendering.

### Support

Packages: `internal/loader`, `internal/glob`, `internal/health`, `internal/testutil`.

Runtime path lookup, embedded assets, TOML merge helpers, theme loading, and workspace trust live in `loader`. `glob` provides glob matching used by language/config behavior. `health` powers the runtime health checks; `testutil` holds shared test infrastructure.

## Data Flow

toe is a single Bubbletea program. One frame looks like:

1. Terminal input arrives as a Bubbletea message.
2. The model routes it: modal overlays (picker, completion, prompt) get first refusal; otherwise the key trie resolves it against the active mode's keymap.
3. The resolved command runs its handler. Editing handlers usually call `view/action` helpers that build a `Transaction` against the document's current `Rope`.
4. For an edit, applying the transaction produces a new `Rope`, increments the document revision, records history unless the edit is being accumulated for insert mode, maps selections through the `ChangeSet`, and notifies observers such as LSP document sync and the VCS differ.
5. The renderer draws the visible viewport from cached highlight spans and gutter state into the cell buffer, and Bubbletea diffs it onto the screen.

Because document text is a persistent `Rope`, background workers can keep the text snapshot they were handed. Mutable document snapshot fields are protected by document locks where async LSP goroutines need to read or update them.

## Extension Points

- **Languages and language servers** — add or override `[[language]]` entries and `[language-server.<name>]` sections in the merged `languages.toml` data. No code changes are needed for a new server. Tree-sitter highlighting for a new language requires adding the grammar import to `internal/term/syntax`, registering it in the language registry, and bundling a highlight query.
- **VCS providers** — implement the `vcs.Provider` interface. The registry currently installs Git directly in `NewRegistry`, so adding another provider also requires wiring it into that constructor. The editor consumes only the `view.VersionControl` seam.
- **Commands** — add a command module under `term/defaults` that registers signatures against the command registry. Registered commands automatically participate in key binding, prompt completion, and the command palette.
- **Actions** — put reusable editing behavior in `view/action` so commands, keymaps, and UI components can share it.
- **Themes** — themes are TOML scope-to-style maps decoded by `internal/term/theme` and loaded through `loader`. The four embedded Catppuccin variants (`latte`, `frappe`, `macchiato`, `mocha`) are the supported theme names today.
- **Clipboard** — register yanks and pastes use `view/register`. System clipboard actions detect external tools (`pbcopy`/`pbpaste`, `xclip`, `xsel`, or `wl-copy`/`wl-paste`) directly in `view/action`; OSC 52 and custom command providers are not implemented yet.
- **UI components** — complex overlays implement `BufferOverlayComponent` and draw directly into the frame buffer. Simple string overlays can implement `OverlayComponent`. Pickers share source/list/render helpers for matching, hit testing, scrolling, preview caching, and cursor visibility.

## Testing Strategy

Tests are black-box (`package_test`) throughout the repo, and shared test helpers have their own tests. CI entry points, runtime asset validation, command/keybinding registration tests, behavior tests, integration tests, and render benchmarks cover the supported surface.

Rendering-sensitive code has benchmarks with `-benchmem` coverage for long single lines, picker previews, large highlighted files, scrolling, and visual-column calculations. Service packages are exercised with real fixtures where practical: temporary git repositories for VCS and an in-process test language server for LSP.
