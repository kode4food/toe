# toe — Thom's Own Editor

## Project

Thom's Own Editor: a Go-native modal terminal editor using Go 1.26, Bubbletea, Lipgloss, and Chroma. Module: `github.com/kode4food/toe`

## CRITICAL: Do Exactly What Is Asked

When the user asks for a specific thing, do that thing and nothing else. Do not take liberties rewriting, refactoring, or "improving" code that wasn't part of the request. The risk of breaking something or introducing unwanted changes is not worth it, and unilateral decisions like that are not mine to make.

---

## Configuration Boundaries

`view.Options` is only for innate editor behavior that the core editor, documents, actions, or renderer must consult directly at runtime. It must not be used as a dumping ground for configuration owned by optional or decoupled modules.

Module-owned configuration must be colocated with the module that owns the behavior:

- Default command module TOML section structs live in `internal/term/defaults` with the command module that loads them.
- UI component behavior options live with the UI component in `internal/term/ui`.
- A module command passes its parsed options into the module/component factory explicitly.

Do not add picker, explorer, LSP, VCS, DAP, or other pluggable capability settings to `view.Options` unless the core editor itself must own that behavior to function correctly.

---

## Args and Res Structs

Structs with an `Args` suffix are parameter bundles for a single function. Structs with a `Res` suffix are result bundles returned by a single function. Both have strict rules:

- **Lifetime**: an `Args` struct must not outlive the call site where it is passed; a `Res` struct must not outlive the call site where it is received. Neither may be stored, forwarded to another function, or returned further up the call chain.
- **If a struct crosses more than one call site** it is not an Args/Res struct — rename it to a plain descriptive name (no suffix) and use pointer currency (`*T`) when passing it.
- **Placement**: each Args/Res struct must be declared immediately before the function that accepts or returns it. If one function has both an Args and a Res struct, declare them together in a single `type (...)` block immediately before that function. Never group them with unrelated types at the top of the file.
- **Value passing**: at their single call site, Args/Res structs are passed and returned by value (no `*`). They are small, short-lived, and stack-allocated by design.

```go
// Good — declared immediately before its function, used only at one call site
type renderPaneArgs struct {
    doc     *view.Document
    view    *view.View
    buf     *tui.Buffer
    y0      int
    focused bool
}

func (r *renderPass) renderPane(args renderPaneArgs) { ... }

// Bad — declared in a top-level type block far from its function
type (
    renderPaneArgs struct { ... }  // ← wrong place
    someOtherType  struct { ... }
)

// Bad — forwarded to a second function (no longer a single call site)
func outer(args myArgs) {
    args.x = 0        // mutates
    inner(args)       // forwarded — rename and use *myType instead
}
```

---

## Persistent Data Structures

All core data structures — `Rope`, `ChangeSet`, `Transaction`, `Selection`, `Range`, `History` — are **persistent (immutable)**. Every operation must return a new value; the original must never be modified.

**In-place node mutation is absolutely forbidden.** This applies even inside helper functions. Rotation, rebalancing, and any structural change must allocate and return new nodes — never mutate fields on existing ones. Shared nodes are the norm (structural sharing is how persistence stays efficient), so a mutated node corrupts every data structure that references it.

```go
// Bad — mutates a shared node
func rotateRight(n *node) *node {
    p := n.left
    n.left = p.right  // mutates n (may be shared)
    p.right = n       // mutates p (definitely shared)
    refresh(p)
    return p
}

// Good — allocates new nodes, originals untouched
func rotateRight(n *node) *node {
    p := n.left
    newN := &node{left: p.right, right: n.right}
    refresh(newN)
    newP := &node{left: p.left, right: newN}
    refresh(newP)
    return newP
}
```

---

# Go Style Guide

## Naming Conventions

### Receiver Names

Single lowercase letter, first letter of type name:

```go
// Good
func (h *History) Undo() {}
func (r *rowRender) rows() {}
func (e *Editor) OpenFile(path string) {}
func (p *Picker) moveBy(n int) {}

// Bad
func (history *History) Undo() {}
func (self *History) Undo() {}
func (this *History) Undo() {}
```

### Variable Names

**Prefer short names.** The closer a variable is used to where it's declared, the shorter it can be. Loop variables can be single letters.

```go
// Good - short names, close usage
for i, r := range ranges {
    if ok := r.Valid(); !ok {
        continue
    }
}

for _, span := range spans {
    process(span)
}

// Good - map access always uses 'ok'
if entry, ok := spanCache[id]; ok {
    return entry.spans
}

if doc, ok := e.docs[id]; ok {
    return doc.Path()
}

// Bad - verbose names for tight scope
for rangeIndex, currentRange := range ranges {
    if exists := currentRange.Valid(); !exists {  // Use 'ok', not 'exists'
        continue
    }
}
```

Avoid local names that merely restate the type. Prefer the semantic subject, not the full noun phrase:

```go
// Good
for id := range e.docs {
    doc := e.docs[id]
    if doc.Lang() == "go" {
        return openFile(id, doc)
    }
}

rope := doc.Text()
sel := doc.Selection(vid)
tx := core.NewTransaction(rope)
entry, ok := p.spanCache[id]

// Bad
for documentID := range editor.docs {
    documentValue := editor.docs[documentID]
    if documentValue.Lang() == "go" {
        return openFile(documentID, documentValue)
    }
}

ropeValue := doc.Text()
selectionValue := doc.Selection(vid)
transactionValue := core.NewTransaction(ropeValue)
spanCacheEntry, ok := pickerState.spanCache[documentID]
```

Use longer names only when the broader scope really needs them, such as struct fields, exported APIs, or tests where the variable itself is the subject under test.

**Longer names for wider scope** (exported functions, struct fields):

```go
// Good - clear at API boundaries
func (e *Editor) OpenFile(path string) (*View, error)

func NewChangeSetFromChanges(
    doc Rope, changes []Change,
) (ChangeSet, error)

// Good - descriptive struct fields
type previewDocRender struct {
    text   core.Rope
    spans  []highlight.Span
    format *config.TextFormat
    cfg    *config.Config
    th     *theme.Theme
    w, h   int
    hlFrom int
    hlTo   int
}
```

**Idiomatic short names**:

| Name                   | Usage                                       |
| ---------------------- | ------------------------------------------- |
| `i`, `j`, `k`          | Loop indices                                |
| `n`                    | Count or length                             |
| `ok`                   | Boolean from map/type assertion             |
| `err`                  | Error values                                |
| `ctx`                  | context.Context                             |
| `b`                    | bytes or buffer                             |
| `r`, `w`               | io.Reader, io.Writer                        |
| `t`                    | \*testing.T                                 |
| `s`                    | String (when scope is tiny)                 |
| `idx`                  | Index (when `i` is ambiguous)               |
| `pfx`, `sfx`           | Prefix, suffix                              |
| `cfg`                  | Config struct                               |
| `opts`                 | Options struct                              |

Examples:

```go
h := core.NewHistory()
st := core.State{Doc: core.NewRope("hello"), Selection: core.PointSelection(0)}
tx := core.NewTransaction(st.Doc)
sel, err := core.NewSelection(ranges, 0)
entry, ok := p.spanCache[id]
rope, spans := entry.rope, entry.spans
lang := doc.Lang()
rev := doc.Revision()
```

### Function Signature Wrapping

When a function signature is too long for one line, keep as many parameters as fit on the first line and wrap the remainder on the next line(s). Do not put one parameter per line unless the line would still exceed the limit.

Example with more parameters:

```go
func NewChangeSetFromChanges(
	doc Rope, changes []Change,
) (ChangeSet, error) {

func renderPreviewDocInto(
	buf *tui.Buffer, x0, y0 int, args *previewDocRender,
) {
```

### Function Names

Verb + noun. Get/Set only when accessing fields:

```go
// Good
func (e *Editor) OpenFile(path string) (*View, error)
func (e *Editor) SwitchBuffer(did DocumentId) bool
func (h *History) CommitRevision(tx Transaction, st State) error
func (r Rope) SliceString(from, to int) string

// Bad - Get/Set for non-field access
func (e *Editor) GetFileFromDisk(path string)     // Use Open
func (r Rope) GetSubstringFromTree(from, to int)  // Use Slice
```

### Constructor Names

`New` prefix, return pointer:

```go
// Good
func NewHistory() History
func NewPicker(e *view.Editor, source PickerSource) *Picker

// Bad
func CreateHistory() History
func MakeHistory() History
```

### Interface Names

Single-method interfaces use `-er` suffix. Capabilities, not implementations:

```go
// Good - describes what it does
type CharMatcher interface {
    MatchChar(ch rune) bool
}

type BufferRenderer interface {
    RenderBuffer(width, height int, cx *Context) *tui.Buffer
}

// Bad - describes what it is
type CharMatcherInterface interface { ... }
type ICharMatcher interface { ... }
```

### Constant Names

`Default` prefix for defaults. `Max`/`Min` for limits:

```go
// Good
const (
    DefaultTabWidth    = 4
    DefaultScrollLines = 3
    MaxIndent          = 16
)

// Bad - unclear what 4 means
const TabWidth = 4
```

### Error Names

`Err` prefix, grouped in `var` block:

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrInvalidState = errors.New("invalid state")
    ErrTimeout      = errors.New("operation timed out")
)
```

### Boolean Names

Avoid `is`/`has` prefix (redundant in Go):

```go
// Good
if atRoot { ... }
if doc.Modified() { ... }
if hasOpenDocuments(e) { ... }  // Functions can use has/is

// Acceptable in struct fields when clarity needed
type Config struct {
    Enabled bool
    Ready   bool
}

// Bad - redundant prefix
if isAtRoot { ... }
if doc.IsModified() { ... }
```

### Acronyms

All caps for acronyms, even in camelCase:

```go
// Good
type HTTPClient struct {}
func (c *Client) GetURL() string
type DocumentID string
var xmlParser Parser

// Bad
type HttpClient struct {}
func (c *Client) GetUrl() string
type DocumentId string
```

## Formatting

### Markdown

Markdown files should expect to be soft-wrapped. Do not hard-wrap prose to the code line-width limit; keep paragraphs as readable logical lines and let the editor wrap them. Preserve deliberate line breaks in lists, tables, code fences, quoted text, and other Markdown structures where the newline carries meaning.

### Line Width

Maximum 80 characters per line (tabs count as 4 spaces). This applies to code _and_ comments, not Markdown prose. Keep short argument lists on a single line when they fit; only break lines when the 80-character limit would be exceeded. When wrapping function signatures or call arguments, pack as many arguments per line as will fit under the limit before wrapping again. When you must wrap, break after the opening paren:

```go
func NewChangeSetFromChanges(
	doc Rope, changes []Change,
) (ChangeSet, error) {
```

```go
c, err := client.NewClient("embedded://", client.WithEmbedded(tr))
```

### Multi-line Calls with \*testing.T

When a function call wraps and the first argument is the test instance (`t`), keep `t` on the first line and break immediately after it. Do not place `t` alone on the next line.

```go
applyAll(t,
	h.Earlier(core.UndoSteps(3)), &st,
)
```

```go
assert.Equal(t,
	"a b c d\n", st.Doc.String(),
)
```

## File Organization

### Imports

Run `goimports` on all files. It handles grouping and sorting automatically.

### No Function-Scoped Type or Const Declarations

**Never declare `type` or `const` inside a function body.** All type and constant declarations must be at package level, in the appropriate block with the rest of the package's types and constants.

```go
// Bad — type declared inside a function
func process() {
    type work struct{ id int }   // FORBIDDEN
    const limit = 100            // FORBIDDEN
    const (                      // FORBIDDEN
        kindA = iota
        kindB
    )
}

// Good — all at package level
type work struct{ id int }

const limit = 100

const (
    kindA = iota
    kindB
)

func process() { ... }
```

This applies to test files as well.

### Top-Level Declaration Order

1. `type` declarations (must use a block when declaring multiple types). Ordering rule: if a type uses another type, the using type goes first.
2. `const` declarations (must use a block when declaring multiple constants)
3. `var` declarations (must use a block when declaring multiple vars; exception: errors always use a `var` block)
4. Exported functions (including constructors like `New...`)
5. Exported methods
6. Unexported methods
7. Unexported helper functions

```go
package core

type (
	History  struct { ... }
	UndoKind struct { ... }
	revision struct { ... }
)

const MaxIndent = 16

var (
	ErrEmptySelection       = errors.New("empty selection")
	ErrPrimaryIndexNotFound = errors.New("primary index not found")
)

func NewHistory() History { ... }

func (h *History) CurrentRevision() int { ... }          // exported
func (h *History) CommitRevision(...) error { ... }      // exported

func (h *History) jumpTo(to int) []Transaction { ... }   // unexported
func indentWidth(s string, tabWidth int) int { ... }     // unexported helper
```

### Method Ordering

1. Constructor (`New...`)
2. Exported methods grouped by functionality
3. Unexported methods that support the exported ones
4. Pure helper functions (non-methods) at the bottom

Related methods stay together. Within each group, order by call chain or first use. Unexported helpers appear after the exported methods that use them.

### Concern Grouping

Within a package, organize files around real concerns, not arbitrary helper categories. Prefer concern-oriented grouping when that matches the code's behavior:

- `picker.go`, `picker-component.go`, `picker-render.go`
- `picker-files.go`, `picker-search.go`, `picker-commands.go`
- `render-document.go`, `render-status.go`
- `model-action.go`, `model-types.go`

Do not introduce wrapper files that just forward calls to another package or rename errors.

## Struct Literals

**NEVER construct a struct using positional field order.** Always use named fields. Positional literals are fragile: a field reorder or insertion silently compiles and corrupts data.

```go
// Good
Separator{Layout: LayoutVertical, X: a.X + a.Width, Y: c.area.Y, W: 1, H: c.area.Height}

// Bad — positional, breaks silently on field reorder
Separator{LayoutVertical, a.X + a.Width, c.area.Y, 1, c.area.Height}
```

The only exception is single-field structs where the field name adds no information (e.g. `Point{3}` when `Point` wraps a single `int`).

## Control Flow

### Early Returns

Use guard clauses to minimize nesting. No else when early return works:

```go
// Good
func processStep(step *StepInfo) error {
	if step == nil {
		return ErrNilStep
	}
	if !step.IsValid() {
		return ErrInvalid
	}
	// main logic
	return nil
}

// Bad
func processStep(step *StepInfo) error {
	if step != nil {
		if step.IsValid() {
			// main logic
			return nil
		} else {
			return ErrInvalid
		}
	} else {
		return ErrNilStep
	}
}
```

### Nesting Limit

Maximum one level of conditional nesting. Exception: when early return would cause code duplication.

```go
// Acceptable nesting to avoid duplicating the return
func (e *Editor) focusedDoc() (*Document, bool) {
	if v, ok := e.focusedView(); ok {
		if doc, ok := e.docs[v.DocID()]; ok {
			return doc, true
		}
	}
	return nil, false
}
```

## Testing

### Coverage Target

Minimum 90% test coverage.

### Black-Box Testing Only

All tests use `package_test` suffix:

```go
package engine_test  // Good
package engine       // Bad
```

### Test Naming

Function names must be short labels for the unit under test. They should hold the related subtests and identify the subject, not describe every scenario. Put scenario detail in `t.Run()` names, not in the function name.

**`t.Run()` descriptions must be short and concise — never more than ~40 characters.** They label the scenario, not document it. Drop the subject (it's in the function name), drop "with", drop "without", drop the function name itself. Think: what's different here?

```go
// Good — concise, fits in one glance
t.Run("undoes and redoes edits", ...)
t.Run("navigates by steps", ...)
t.Run("empty selection returns error", ...)
t.Run("clips long lines", ...)

// Bad — too long, restates the subject
t.Run("History undoes and redoes edits correctly", ...)
t.Run("MoveRight with empty selection returns error", ...)
t.Run("PickerPreview clips long lines to width", ...)
```

```go
// Good - short function name
func TestHistory(t *testing.T) {
    t.Run("undoes and redoes edits", func(t *testing.T) {
        // ...
    })
    t.Run("navigates by steps", func(t *testing.T) {
        // ...
    })
}

// Bad - underscores are extraneous
func TestHistory_Undo(t *testing.T) { ... }
func TestRope_SliceString(t *testing.T) { ... }

// Bad - function name is a novel
func TestHistoryUndoesAndRedoesEditsCorrectly(t *testing.T) { ... }
func TestPickerPreviewClipsLongLinesToPreviewWidth(t *testing.T) { ... }
```

### Assertions

Use `testify/assert` only. Never `testify/require`. Never include message args:

```go
// Good
assert.NoError(t, err)
assert.Equal(t, expected, actual)
assert.True(t, ok)

// Bad - require stops test early
require.NoError(t, err)

// Bad - no message arguments
assert.NoError(t, err, "should not error")
assert.Equal(t, expected, actual, "values should match")
```

### Test Organization

- Table-driven tests for multiple scenarios
- Subtest descriptions with `t.Run()`
- `t.Helper()` in test utilities
- Keep test files aligned with source concerns when the split is clear

If the source is grouped by concern, the tests should mirror that grouping:

- `picker-files_test.go`
- `picker-scroll_test.go`
- `picker-preview_test.go`
- `picker-match_test.go`
- `movement_test.go`
- `selection_test.go`
- `history_test.go`

Do not keep broad mixed test files once the source has been split cleanly.

## Comments

### Godoc

**Exported** funcs, methods, types, consts, and vars always need godoc — but no more than 3 lines. Describe what it does, not how. If it takes more than 3 lines to say what something does, it isn't coded well:

```go
// History stores committed document revisions and supports undo/redo
// navigation by step count or time period
type History struct {
```

Skip godoc when the name is self-documenting:

```go
func NewHistory() History {
```

**Unexported** funcs and methods get no godoc by default. Only add one — capped at 2 lines — when the behavior is genuinely non-trivial and needs explanation:

```go
// unexported, self-explanatory — no comment
func clampSelection(sel core.Selection, maxChars int) core.Selection {

// unexported, but the "why" isn't obvious from the signature — 2 lines max
// diffDebounce: single async debounce; the gutter trails a keystroke by this
const diffDebounce = 50 * time.Millisecond
```

Godoc rule: the last sentence of a comment should not end with a period.

### Inline Comments

Avoid comments that restate the code. Never comment code that's already
self-describing. When a comment is warranted, it explains WHY, capped at 2
lines:

```go
// Bad
bucket, err := blob.OpenBucket(ctx, url)  // Open the bucket
return err                                 // Return the error

// Good - explains WHY, 2 lines max
// Missing key is not an error; deletion is idempotent by design
if gcerrors.Code(err) == gcerrors.NotFound {
	return nil
}
```

## Global State

**Mutable package-level variables are absolutely forbidden.** This includes counters, caches, registries, or any other state that can be mutated after initialization.

```go
// Bad — mutable global state
var idCounter atomic.Int64

// Good — state lives on the owning struct
type Editor struct {
    nextID int
}

func (e *Editor) newThing() *Thing {
    e.nextID++
    return &Thing{id: e.nextID}
}
```

Package-level `var` declarations are permitted only for:

- Sentinel error values (`var ErrNotFound = errors.New(...)`)
- Compile-time interface assertions (`var _ Foo = (*Bar)(nil)`)
- Truly immutable lookup tables that are never reassigned (treat them as constants; document if a slice element could be mutated)

## Interface Compliance

Compile-time interface checks:

```go
var _ CharMatcher = (*RuneMatcher)(nil)
```

## Error Handling

- **Never panic** - always return errors
- **Typed errors only** - All production code must use package-level vars with `Err` prefix
- **Pattern: `%w: context`** — wrapped error first, then context variable
- Plain error messages acceptable only in examples/documentation
- Handle errors immediately, early return

**Production Code - Always Use Typed Errors:**

```go
var (
	ErrEmptySelection       = errors.New("empty selection")
	ErrPrimaryIndexNotFound = errors.New("primary index not found")
	ErrChangeOrder          = errors.New("change order invalid")
)

// Good - %w: %d pattern with typed error
if primaryIndex >= len(ranges) {
    return fmt.Errorf("%w: %d", ErrPrimaryIndexNotFound, primaryIndex)
}

// Good - typed error with multiple context values
if from > to {
    return fmt.Errorf("%w: %d -> %d", ErrChangeOrder, from, to)
}

// Good - return typed error directly
if len(ranges) == 0 {
    return Selection{}, ErrEmptySelection
}

// Bad - plain message in production code (no typed error)
if len(ranges) == 0 {
    return Selection{}, fmt.Errorf("empty selection")  // NO! Use typed error
}

// Bad - context before wrapped error
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to apply: %w", err)  // Wrong order
}

// Bad - never panic
if doc == nil {
    panic("doc is nil")  // NO!
}
```

**Testing - Use errors.Is() to Check Typed Errors:**

Tests should use `errors.Is()` to check for specific error types, not `strings.Contains()`:

```go
// Good - use errors.Is for typed errors
_, err := core.NewSelection(nil, 0)
assert.True(t, errors.Is(err, core.ErrEmptySelection))

// Bad - fragile string matching
assert.True(t, strings.Contains(err.Error(), "empty selection"))
```

This enables robust error checking without brittle string comparisons. Typed errors are also easier to handle programmatically.

**Examples/Documentation Only - Plain Messages OK:**

```go
// Only acceptable in README examples, not in engine code
return fmt.Errorf("invalid configuration: %s", reason)
```

## Constants

- No magic numbers
- Group related constants
- Use typed constants when meaningful

```go
const (
	DefaultTabWidth    = 4
	DefaultScrollLines = 3
	MaxIndent          = 16
)
```

---

# UI Library Policy

**Always prefer Bubbletea and Lipgloss over home-grown alternatives.** Before writing custom terminal UI code, check whether the library already handles it:

- Use `lipgloss.Style` for padding, truncation, colour, borders — not manual ANSI or `strings.Repeat`.
- Use `lipgloss.Wrap` for ANSI-aware word wrapping.
- Use `ansi.StringWidth` / `ansi.Truncate` when you need raw cell widths without wrapping (e.g. clipping a single preview row).
- Use `lipgloss.JoinVertical` / `JoinHorizontal` / `PlaceHorizontal` for layout instead of manual gap calculations.
- For overlay panels, implement `BufferOverlayComponent` (`RenderOverBuffer`) rather than `OverlayComponent` (`RenderOver` with lipgloss layers). The buffer-native path skips the ANSI round-trip and is significantly faster for complex overlays. Use `OverlayComponent` / `lipgloss.NewLayer` + `lipgloss.NewCompositor` only for simple string-based overlays (e.g. the command prompt).
- Use the `popup` struct (`internal/term/ui/popup.go`) for any bordered popup window — it fills the box with the content style and draws the border in one pass, so callers write per-cell content without worrying about ANSI background resets.
- Use `tea.View.Cursor` for cursor shape and position instead of raw DECSCUSR escapes in content strings.

The only valid reason to roll your own is when the library genuinely has no equivalent (e.g. tab expansion, custom fuzzy-match highlight, per-character cursor/selection colouring in the editor viewport).

---

# Tools

Use the **Serena MCP** for all code navigation and editing tasks: symbol lookup, find references, rename, go-to-definition, diagnostics. Prefer it over shell commands (`grep`, `find`, `sed`) for anything code-structural.

---

# CRITICAL: Rendering Performance

**ALWAYS benchmark when changing editor or preview rendering.** The render path runs on every keystroke and every frame; a per-render regression (re-parsing config, re-allocating per character, building off-screen content) makes the editor lag and back up input events. Do not reason about performance from first principles — measure.

- Before and after any change to the editor content renderer or the picker preview renderer, run a Go benchmark with `-benchmem` and compare `ns/op`, `B/op`, and `allocs/op`. Profile with `-cpuprofile` / `-memprofile` and `go tool pprof` to locate the actual hotspot.
- Benchmark the realistic worst case: a long single line, the cursor scrolled far right, a split layout. See `BenchmarkRenderLongLine` (`internal/term/ui/bench_test.go`) and `BenchmarkVisualColumn` (`internal/view/bench_internal_test.go`).
- Rendering must do work **once** and re-do it only when its inputs change. Anything parsed, decoded, or loaded (config TOML, language definitions, themes, syntax) must be cached and invalidated on change — never re-parsed per render. Per-character work in the row loop must avoid allocation (use the ASCII fast paths) and must not be performed for off-screen columns.

---

# CRITICAL: Git Commits

**NEVER COMMIT. EVER.** Do not use `git commit` under any circumstances unless explicitly and directly instructed by the user in that exact session. Do not ask permission. Do not commit. Period.

The only exception is if the user explicitly says "commit" or "create a commit" in their current message.
