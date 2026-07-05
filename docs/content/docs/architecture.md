---
title: "Architecture"
weight: 60
---

# Architecture

toe is a Go-native modal terminal editor built on Bubbletea, Lipgloss, and Tree-sitter. This page explains how the codebase is layered, how data flows through it, and where the extension seams are.

## Design Principles

- **toe edits Go projects, not the universe.** Features exist because Go development needs them, not because other editors have them.
- **Persistent data structures.** The core text and editing types are immutable; every operation returns a new value. Structural sharing keeps this efficient and makes undo/redo, previews, and concurrent workers safe by construction.
- **Modular ownership.** Each capability (LSP, VCS, pickers, themes) owns its configuration and state. The core editor exposes narrow seams; modules plug into them rather than reaching into editor internals.
- **Render once, cache everything.** The render path runs on every keystroke. Anything parsed or computed (config, themes, syntax, highlight spans) is cached and invalidated on change, never recomputed per frame.

## Package Layers

Dependencies point downward; lower layers never import higher ones.

### Core Model

Packages: `internal/core`.

Persistent text (`Rope`), selections, ranges, movement, transactions (`ChangeSet`, `Transaction`), and pure editing algorithms. No I/O, no UI, no editor state — everything here is a pure function over immutable values. All higher layers are built on these types.

### Editor State

Packages: `internal/view` and its subpackages.

The editor, documents, view tree (splits), history, and file I/O. A `Document` owns text, revision, language, and per-view selections; a `View` is a window onto a document; the `Editor` owns the document table, the split tree, and focus. Subpackages:

- `view/config` — editor configuration, runtime options, and EditorConfig support.
- `view/language` — language configuration, matching, and formatter/server metadata.
- `view/register` — registers and clipboard behavior.
- `view/action` — reusable editor actions invoked by commands and UI components.

`view.Options` is deliberately limited to innate editor behavior that core editor, document, action, or renderer code must consult directly. Module-owned settings live with their module.

### Command System

Packages: `internal/term/command`, `internal/term/defaults`.

`term/command` provides the machinery: command signatures, tokenization, completion, key parsing, key tries, keymaps, and the registry. `term/defaults` provides the content: built-in command modules and the immutable default key bindings. Commands are declarative registrations that resolve to `view/action` calls, so the same behavior is reachable from keys, the command prompt, and UI components.

### Terminal UI

Packages: `internal/term/ui`, `internal/tui`.

`term/ui` contains the Bubbletea model: the document renderer, prompt, pickers, completion popup, overlays, and event handling. `internal/tui` is the low-level terminal layer — cell buffer, styles, spans, and ANSI rendering.

Overlays implement one of two interfaces: `BufferOverlayComponent` (`RenderOverBuffer`) writes cells directly into the frame buffer and is the fast path for complex panels; `OverlayComponent` composes lipgloss layers and suits simple string-based overlays like the command prompt. Bordered popups share the `popup` helper so content and border render in one pass.

### Syntax And Themes

Packages: `internal/term/syntax`, `internal/term/highlight`, `internal/term/theme`.

The syntax runtime loads Tree-sitter grammars and queries for supported languages, `highlight` turns parse results into styled spans, and `theme` maps theme scopes onto terminal styles. Highlight spans are cached per document revision.

### Services

Packages: `internal/lsp`, `internal/vcs`.

- `internal/lsp` implements the language-server client: transport, lifecycle, capability negotiation, and feature surfaces (completion, hover, signature help, diagnostics, symbols, code actions, rename, navigation). Server metadata comes from `languages.toml`.
- `internal/vcs` implements version-control integration behind a provider registry. Git is the only provider today, shelling out to the git binary. A debounced per-document diff worker (`vcs.Differ`) computes line hunks; `vcs.Attach` wires a session into document lifecycle observers, and the UI subscribes to update events for gutter rendering.

### Support

Packages: `internal/loader`, `internal/glob`, `internal/health`, `internal/testutil`.

Runtime path lookup, embedded assets, TOML merge helpers, and workspace trust live in `loader`. `health` powers the runtime health checks; `testutil` holds shared test infrastructure.

## Data Flow

toe is a single Bubbletea program. One frame looks like:

1. Terminal input arrives as a Bubbletea message.
2. The model routes it: modal overlays (picker, completion, prompt) get first refusal; otherwise the key trie resolves it against the active mode's keymap.
3. The resolved command invokes a `view/action`, which builds a `Transaction` against the document's current `Rope`.
4. Applying the transaction produces a new document revision; history records it, selections are mapped through the `ChangeSet`, and observers (LSP document sync, VCS differ) are notified.
5. The renderer draws the visible viewport from cached highlight spans and gutter state into the cell buffer, and Bubbletea diffs it onto the screen.

Because documents are persistent values, background workers (diff computation, LSP sync) operate on the revision they were handed without locking or racing the UI.

## Extension Points

- **Languages and language servers** — add a `[[language]]` entry and a `[language-server.<name>]` section to `languages.toml`. No code changes are needed for a new server; grammar support requires bundling the Tree-sitter grammar and queries in the syntax runtime.
- **VCS providers** — implement the `vcs.Provider` interface and register it with the provider registry. The `view.VersionControl` seam is what the editor consumes, so providers never touch UI code.
- **Commands** — add a command module under `term/defaults` that registers signatures against the command registry. Registered commands automatically participate in key binding, prompt completion, and the command palette.
- **Actions** — put reusable editing behavior in `view/action` so commands, keymaps, and UI components can share it.
- **Themes** — themes are TOML scope-to-style maps loaded by `loader`; drop a theme file into the runtime path. The four Catppuccin variants ship embedded.
- **Clipboard** — clipboard access goes through the provider abstraction in `view/register`, so platform integrations (OSC 52, custom commands) are pluggable.
- **UI components** — new panels implement `BufferOverlayComponent` and register with the model's overlay handling; pickers share list interaction helpers for hit testing, scrolling, and cursor visibility.

## Testing Strategy

All tests are black-box (`package_test`), with a 90% coverage floor and nothing excluded — test helpers are tested too. Rendering changes are benchmarked (`-benchmem`) against the realistic worst cases: long single lines, far-right horizontal scroll, and split layouts. Service packages (LSP, VCS) are exercised end-to-end against real fixtures — temporary git repositories for VCS, a test language server for LSP.
