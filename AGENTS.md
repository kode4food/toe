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

- **Threshold (hard rule)**: a function taking **5 or more arguments** must bundle them into an `Args` struct; a function returning **3 or more results** must bundle them into a `Res` struct. Below those counts, pass/return values directly.
- **Success indicators are exempt from the result count**: a trailing `bool` (the `ok` idiom) or `error` signals success/failure, not data, so it never counts toward the 3-result threshold. `(previewImageRes, bool)` and `(Foo, Bar, error)` are fine — count only the data values. So the correct shape here is a `Res` struct **plus** the bool: `func previewImage(...) (previewImageRes, bool)`, not an `ok` field stuffed inside the struct.
- **Same-type adjacency**: the hazard is exactly *two values of the same type next to each other* — nothing else. Distinct types self-disambiguate: `(int, bool)` is fine, `(*Foo, int)` is fine, because the type tells you which value is which. Two of the same type do not: `(bool, bool)` — which bool is which? `(int, int)` — width then height, or height then width? That order is a cultural convention (reversed in other cultures), not a fact the types enforce, so `f(w, h)` and `f(h, w)` are equally legal and a swap compiles silently — the same hazard as positional struct literals. When two adjacent params or results share a type, give them names: a `Res`/`Args` struct with named fields, or distinct named types (`type Width int`), so the meaning lives in the code, not in an assumed convention. `(*previewImageEntry, int, int, bool)` is the worst case: long *and* two nameless same-type `int`s stranded in the middle.
- **Lifetime**: an `Args` struct must not outlive the call site where it is passed; a `Res` struct must not outlive the call site where it is received. Neither may be stored, forwarded to another function, or returned further up the call chain.
- **If a struct crosses more than one call site** it is not an Args/Res struct — rename it to a plain descriptive name (no suffix) and use pointer currency (`*T`) when passing it.
- **Placement**: each Args/Res struct must be declared immediately before the function that accepts or returns it. If one function has both an Args and a Res struct, declare them together in a single `type (...)` block immediately before that function. Never group them with unrelated types at the top of the file.
- **Value passing**: at their single call site, Args/Res structs are passed and returned by value (no `*`). They are small, short-lived, and stack-allocated by design.
- **Field names must stand alone**: a name that worked as a positional function parameter (short, disambiguated by position and the surrounding call) does not automatically work as a named struct field read on its own at a call site. Spell out abbreviations that aren't immediately decodable without reading the function body: `st` → `style`, `w` → `width`, `bg` → `background`, `hOff` → `horizontalOffset`, `e` → `editor`, `v` → `view`. Two fields of the same type still need names that disambiguate them beyond position — `parent Id` next to `id Id` is exactly the same-type-adjacency hazard above; use `parent Id` / `viewID Id`.
- **Literal formatting**: if a struct literal fits on one line, leave it on one line. If it wraps, use exactly one field per line — never pack two or more fields onto a wrapped line.

```go
// Good — declared immediately before its function, used only at one call site
type renderPaneArgs struct {
    doc     *view.Document
    view    *view.View
    buf     *tui.Buffer
    yOffset int
    focused bool
}

func (r *renderPass) renderPane(args renderPaneArgs) { ... }

// Good — fits on one line, stays on one line
r.renderStatus(renderStatusArgs{doc: doc, view: v, buf: buf})

// Good — wraps, so one field per line
r.renderStatus(renderStatusArgs{
    doc:     doc,
    view:    v,
    buf:     buf,
    at:      geom.Point{X: a.X, Y: yOffset + a.Y + contentH},
    width:   a.Width,
    focused: focused,
})

// Bad — wrapped but multiple fields share a line
r.renderStatus(renderStatusArgs{
    doc: doc, view: v, buf: buf,
    at: geom.Point{X: a.X, Y: yOffset + a.Y + contentH},
    width: a.Width, focused: focused,
})

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

## Modularity and Package Boundaries

toe's package layers, top to bottom: `internal/core` → `internal/view`
(+ `view/action`, `view/config`, `view/language`, `view/register`) →
`internal/term/command` → `internal/term/builtin` → `internal/term/ui`, with
`internal/lsp` and `internal/vcs` as services plugged in through `view`-owned
interfaces, and `cmd/toe/internal/app.go` as the composition root. Dependencies
point downward, toward more stable semantics — `core`'s editing semantics
change far less often than `term/ui`'s rendering details.

**Rules:**

1. **One authoritative owner per concept.** Every concept (selections,
   diagnostics, diffs, completions) has exactly one package that owns its
   state and invariants. Other packages consume it through that owner, never
   reimplement or shadow it.
2. **Dependencies point toward stability.** `core` depends on nothing else in
   the editor. `view` depends only on `core`. Commands and UI depend on
   `view`, never the reverse.
3. **Boundaries follow authority and reasons to change, not file/line count.**
   Split a package because two parts change for different reasons and are
   owned by different concerns — not because a file got long (see "Do not
   split packages solely because they are large" below).
4. **State stays with the module that preserves its invariants.** See
   Configuration Boundaries above for the config-specific version of this
   rule; it applies equally to runtime state, caches, and lifecycle state.
5. **`view.Editor` holds capability seams, not module implementation state.**
   `Editor` may hold a `VersionControl`, `LanguageServerController`, or
   similar interface value (see `SetVersionControl`/`SetLanguageServerController`
   in `internal/view`). It must not grow fields that belong to `lsp` or `vcs`
   internals (client transports, provider state, differs).
6. **Interfaces are consumer-defined and minimal.** `view.VersionControl` and
   `view.LanguageServerController` are declared in `view` because `view` is
   the consumer; they expose only what `view`/commands/UI need, not the full
   surface of `vcs.Provider` or the LSP protocol.
7. **Don't add an interface just because a package boundary is crossed.**
   A concrete type passed and used directly is fine. Introduce an interface
   only when there is a real substitutable implementation or the consumer
   needs to decouple from a concrete lifecycle.
8. **Boundary values speak the receiving package's language.** `vcs.Session`
   returns `view.DiffHunk`/`view.FileChange`; `lsp` results are normalized
   into `view.CompletionItem`, `view.Location`, `view.Symbol`, etc. before
   crossing into `view`. Provider/protocol-shaped types (raw LSP structs, git
   plumbing types) never leak past their owning package.
9. **Generic mechanisms don't import concrete registrations.** `term/command`
   (signatures, tokenizer, registry, keymaps) must not import `term/builtin`
   or any specific command module. `vcs.NewRegistry` installing `Git` directly
   is the one accepted exception today (see Extension Points in
   `docs/content/docs/architecture.md`); new providers should still register
   through the app composition root where practical, not by having the
   mechanism import every provider.
10. **Concrete assembly belongs in `cmd/toe/internal/app.go`.** Wiring
    `lsp.Attach`, `vcs.Attach`, `builtin.Register`, and clipboard providers
    together is `app.go`'s job. Packages below it should not know about each
    other's concrete constructors.
11. **Commands orchestrate; they don't implement.** A `term/builtin` command
    handler calls `view/action`, `view`, `lsp`, or `vcs` APIs — it must not
    contain substantial editing, rendering, LSP protocol, VCS diffing, or
    persistence logic inline. If a handler is doing real work, that work
    belongs in the owning package.
12. **Reusable editing lives in `view/action`; pure text semantics live in
    `core`.** `core` never depends on `view` or terminal packages. Anything
    that needs a `Document`/`Editor`/`View` but is reusable across commands
    and UI belongs in `view/action`, not duplicated per command module.
13. **Calls request; observers/events announce.** `view.DocumentObserver`
    methods (`DocumentOpened`, `DocumentChanged`, `DocumentSaved`,
    `DocumentClosed`) report facts that already happened — implementations
    must not treat them as a place to request further mutation of the same
    document mid-notification. A direct method call (`SetVersionControl`,
    `DiffHunks`) requests behavior and expects a synchronous answer.
14. **No dumping-ground packages.** Do not create `util`, `common`, `helpers`,
    `models`, or similarly named packages. A shared helper needs a name that
    describes the concept it owns (`internal/glob`, `internal/loader`), not
    the fact that it's shared.
15. **Exported surface stays much smaller than the implementation.** If a
    package's exported API is nearly as large as its unexported internals,
    that's a sign the boundary is in the wrong place or too much is exported
    by default.
16. **Large is not a reason to split.** `internal/view/action` and
    `internal/lsp` are both large because they own a large, cohesive concept
    (reusable editing operations; the LSP client surface). Split only per
    rule 3.
17. **Don't move code to satisfy a layering ideal.** Preserve cohesion.
    Moving a function to a "more correct" layer that adds indirection
    (forwarding wrappers, an interface with one implementation) without
    changing an actual dependency problem is a net loss.
18. **Proposing a new package requires stating:**
    - what concept it owns;
    - which invariants it preserves;
    - what it may import;
    - what may import it;
    - why the existing owner is incorrect.

**Dependency guide:**

- `internal/core` must not import `view`, any `term/*` package, `lsp`, or
  `vcs`.
- `internal/view` (and subpackages) must not import `term/ui`, `term/builtin`,
  or `cmd/toe/internal`.
- `internal/lsp` and `internal/vcs` must not import `term/ui` or
  `term/builtin`; they depend on `core` and `view` only.
- `internal/term/command` must not import `term/builtin` or concrete service
  packages (`lsp`, `vcs`); it is the generic command mechanism.
- `internal/term/builtin` may import `term/command`, `term/ui`, `view`,
  `view/action`, `lsp`, and `vcs` — it is where commands bridge to services.
- `cmd/toe/internal/app.go` may import any concrete module; it is the only
  place allowed to wire everything together.

**Before moving code, answer:**

- Who owns this concept today?
- What independent reason to change justifies moving it?
- Does the move reduce the number of packages a caller must understand?
- Does the proposed package have a coherent name and responsibility?
- Will the move introduce forwarding wrappers, dependency inversion with no
  substitutable implementation, or a generic helper package?
- Can the boundary be enforced through imports alone or a narrow
  consumer-owned interface, without new indirection?

See also Configuration Boundaries and Args/Res Structs above, and Interface
Names / Interface Compliance below, for the naming and shape rules that apply
within these boundaries.

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

### User Documentation

`README.md` and user-facing pages under `docs/content` should help competent users operate toe:

- Include what a feature does, how to use it, and choices or consequences that affect a user's workflow.
- State the general rule first, followed by meaningful exceptions.
- Keep command, keybinding, and configuration references complete and accurate.
- Use kebab-case command names in user documentation; underscore names are internal identifiers.
- Omit implementation details, internal mechanics, incidental behavior, tuning constants, and obvious facts.
- Do not document a change merely because it is observable; include it only when knowing it materially helps someone use toe.
- Avoid patronizing explanations and negative-space descriptions that force readers to infer the main behavior.
- Keep prose concise and avoid repeating information already clear from a table or another appropriate page.

The architecture page is developer-facing and may describe internals, but its details must serve architectural understanding rather than catalog implementation trivia.

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

## No Cross-Package Var Aliasing

**Never declare `var Foo = otherpkg.Foo` to re-export another package's
identifier under a local name.** Go's `var x = y` exists for local
refactoring inside a package, not as a general-purpose re-export or
aliasing mechanism between packages. If a package needs a value another
package already owns, import that package and reference the value directly
— `view.ErrNoLanguageServer`, not a same-named local copy that happens to
equal it.

```go
// Bad — lsp package re-exports view's sentinel under its own name
var ErrNoLanguageServer = view.ErrNoLanguageServer
...
return ErrNoLanguageServer

// Good — call sites use the owning package's identifier directly
return view.ErrNoLanguageServer
```

This applies to sentinel errors, constants, and any other exported value:
if `view` owns it (see Modularity and Package Boundaries above — interfaces
and their errors are consumer-owned), every package that needs it imports
`view` and writes `view.X`. Don't introduce a second name for the same
value just because a package boundary is in the way.

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

# i18n Policy

Any user-facing English prose — status messages, prompts, hints shown during
an interactive mode — must go through `internal/i18n`, not a hardcoded Go
string constant. Add a `Key` in `internal/i18n/keys.go` and a translation
entry in each locale file (`en.json`, `de.json`, `fr.json`, `it.json`) under
`internal/i18n/translations/`, then reference it with `i18n.Text(key, ...)`.

`internal/i18n/translations/common.json` is reserved for values shared
identically across all locales (e.g. the `:` command prompt) — not a catch-all
or a place to skip translating a new message into the other languages.

The one exception is a hint that echoes a literal keystroke sequence back at
the user (`"ms ..."`, `"r ..."`, `"^r ..."`) — that's not language, so it
stays a plain Go string. A hint that also contains descriptive prose (e.g.
`"h/j/k/l or ←/↓/↑/→ resize, esc/enter exits"`) is not exempt and must be
translated.

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
