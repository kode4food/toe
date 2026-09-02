# toe: Thom's Own Editor

## Project

Thom's Own Editor: a Go-native modal terminal editor. Module: `github.com/kode4food/toe`

## CRITICAL: Do Exactly What Is Asked

When the user asks for a specific thing, do that thing and nothing else. Leave the rest of the code exactly as you found it: rewriting, refactoring, or "improving" code outside the request risks breaking something, and that call belongs to the user.

## CRITICAL: Do Not Reframe What I Asked For

Answer the request I actually made, at its full size. Take my words as written rather than telling me what I meant, and describe what you hand back exactly as it is: a partial job is reported as partial.

Every line in this repository is your own output from some session, so "pre-existing", "not mine", "not part of this diff", and "out of scope" are never true and never a reason to leave something unfixed. When I ask you to fix violations, fix all of them, however old, in one list regardless of who wrote them.

If something genuinely blocks the work (a generated file a tool rewrites, a decision only I can make), say exactly that, say what is left undone, and stop. Name the blocker itself: scope, authorship, and summaries of my intent are not blockers.

---

## Configuration Boundaries

`view.Options` is only for innate editor behavior that the core editor, documents, actions, or renderer must consult directly at runtime. Configuration owned by optional or decoupled modules lives with those modules.

Module-owned configuration must be colocated with the module that owns the behavior:

- Default command module TOML section structs live in `internal/term/defaults` with the command module that loads them.
- UI component behavior options live with the UI component in `internal/term/ui`.
- A module command passes its parsed options into the module/component factory explicitly.

Picker, explorer, LSP, VCS, DAP, and other pluggable capability settings stay with their own modules; they reach `view.Options` only when the core editor itself must own that behavior to function correctly.

---

## Args and Res Structs

Structs with an `Args` suffix are parameter bundles for a single function. Structs with a `Res` suffix are result bundles returned by a single function. Both have strict rules:

- **Threshold (hard rule)**: a function taking **5 or more arguments** must bundle them into an `Args` struct; a function returning **3 or more results** must bundle them into a `Res` struct. Below those counts, pass/return values directly.
- **Success indicators are exempt from the result count**: a trailing `bool` (the `ok` idiom) or `error` signals success/failure, not data, so it never counts toward the 3-result threshold. `(previewImageRes, bool)` and `(Foo, Bar, error)` are fine: count only the data values. So the correct shape here is a `Res` struct **plus** the bool: `func previewImage(...) (previewImageRes, bool)`, with `ok` staying outside the struct.
- **Same-type adjacency**: the hazard is exactly *two values of the same type next to each other*, and nothing else. Distinct types self-disambiguate: `(int, bool)` is fine, `(*Foo, int)` is fine, because the type tells you which value is which. Two of the same type do not. `(bool, bool)`: which bool is which? `(int, int)`: width then height, or height then width? That order is a cultural convention (reversed in other cultures), not a fact the types enforce, so `f(w, h)` and `f(h, w)` are equally legal and a swap compiles silently; the same hazard as positional struct literals. When two adjacent params or results share a type, give them names: a `Res`/`Args` struct with named fields, or distinct named types (`type Width int`), so the meaning lives in the code, not in an assumed convention. `(*previewImageEntry, int, int, bool)` is the worst case: long *and* two nameless same-type `int`s stranded in the middle.
- **Lifetime**: an `Args` struct lives only as long as the call site it is passed at; a `Res` struct only as long as the call site that received it. Both stay local: storing, forwarding to another function, or returning one further up the call chain is out.
- **If a struct crosses more than one call site** it is not an Args/Res struct, rename it to a plain descriptive name (no suffix) and use pointer currency (`*T`) when passing it.
- **Placement**: each Args/Res struct must be declared immediately before the function that accepts or returns it. If one function has both an Args and a Res struct, declare them together in a single `type (...)` block immediately before that function. The top-of-file type block is for unrelated types.
- **Value passing**: at their single call site, Args/Res structs are passed and returned by value (no `*`), whatever their size. They are short-lived and stack-allocated by design.
- **Pointer threshold (plain-named structs)**: a struct that crosses call sites is passed by value or by pointer according to its size, not by habit. Up to **32 bytes** (4 machine words) always by value. **33–64 bytes**: by value, unless it is read by three or more functions in a chain. Over **64 bytes** (one cache line): by pointer; `hardWrapUnfilled` (72 bytes) and `view.CursorScroll` (~80 bytes) are pointers for this reason, while `edgeState` (16 bytes) is a value. Regardless of size, use a pointer when the callee mutates the struct or its identity matters. Remember that a `&x` that never escapes stays on the stack: the cost of a pointer there is the indirection, not an allocation.
- **Field names must stand alone**: a name that worked as a positional function parameter (short, disambiguated by position and the surrounding call) does not automatically work as a named struct field read on its own at a call site. Spell out abbreviations that aren't immediately decodable without reading the function body: `st` → `style`, `w` → `width`, `e` → `editor`, `v` → `view`. Two fields of the same type still need names that disambiguate them beyond position; `parent Id` next to `id Id` is exactly the same-type-adjacency hazard above; use `parent Id` / `viewID Id`.
- **Literal formatting**: if a struct literal fits on one line, leave it on one line. If it wraps, use exactly one field per line.

```go
// Good, declared immediately before its function, used only at one call site
type renderPaneArgs struct {
    doc     *view.Document
    view    *view.View
    buf     *tui.Buffer
    yOffset int
    focused bool
}

func (r *renderPass) renderPane(args renderPaneArgs) { ... }

// Good, fits on one line, stays on one line
r.renderStatus(renderStatusArgs{doc: doc, view: v, buf: buf})

// Good, wraps, so one field per line
r.renderStatus(renderStatusArgs{
    doc:     doc,
    view:    v,
    buf:     buf,
    at:      geom.Point{X: a.X, Y: yOffset + a.Y + contentH},
    width:   a.Width,
    focused: focused,
})

// Bad, wrapped but multiple fields share a line
r.renderStatus(renderStatusArgs{
    doc: doc, view: v, buf: buf,
    at: geom.Point{X: a.X, Y: yOffset + a.Y + contentH},
    width: a.Width, focused: focused,
})

// Bad, declared in a top-level type block far from its function
type (
    renderPaneArgs struct { ... }  // ← wrong place
    someOtherType  struct { ... }
)

// Bad, forwarded to a second function (no longer a single call site)
func outer(args myArgs) {
    args.x = 0        // mutates
    inner(args)       // forwarded, rename and use *myType instead
}
```

---

## Persistent Data Structures

All core data structures (`Rope`, `ChangeSet`, `Transaction`, `Selection`, `Range`, `History`) are **persistent (immutable)**. Every operation returns a new value and leaves the original intact.

**In-place node mutation is absolutely forbidden.** This applies even inside helper functions. Rotation, rebalancing, and any structural change allocates and returns new nodes, leaving fields on existing ones untouched. Shared nodes are the norm (structural sharing is how persistence stays efficient), so a mutated node corrupts every data structure that references it.

```go
// Bad, mutates a shared node
func rotateRight(n *node) *node {
    p := n.left
    n.left = p.right  // mutates n (may be shared)
    p.right = n       // mutates p (definitely shared)
    refresh(p)
    return p
}

// Good, allocates new nodes, originals untouched
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
point downward, toward more stable semantics, `core`'s editing semantics
change far less often than `term/ui`'s rendering details.

**Rules:**

1. **One authoritative owner per concept.** Every concept (selections,
   diagnostics, diffs, completions) has exactly one package that owns its
   state and invariants. Other packages consume it through that owner rather
   than reimplementing or shadowing it.
2. **Dependencies point toward stability.** `core` depends on nothing else in
   the editor. `view` depends only on `core`. Commands and UI depend on
   `view`, never the reverse.
3. **Boundaries follow authority and reasons to change, not file/line count.**
   Split a package because two parts change for different reasons and are
   owned by different concerns, not because a file got long (see rule 16).
4. **State stays with the module that preserves its invariants.** See
   Configuration Boundaries above for the config-specific version of this
   rule; it applies equally to runtime state, caches, and lifecycle state.
5. **`view.Editor` holds capability seams, not module implementation state.**
   `Editor` may hold a `VersionControl`, `LanguageServerController`, or
   similar interface value (see `SetVersionControl`/`SetLanguageServerController`
   in `internal/view`). Fields belonging to `lsp` or `vcs` internals (client
   transports, provider state, differs) stay in those packages.
6. **Interfaces are consumer-defined and minimal.** `view.VersionControl` and
   `view.LanguageServerController` are declared in `view` because `view` is
   the consumer; they expose only what `view`/commands/UI need, not the full
   surface of `vcs.Provider` or the LSP protocol.
7. **Interfaces earn their place; a package boundary alone doesn't call for
   one.** A concrete type passed and used directly is fine. Introduce an
   interface only when there is a real substitutable implementation or the
   consumer needs to decouple from a concrete lifecycle.
8. **Boundary values speak the receiving package's language.** `vcs.Session`
   returns `view.DiffHunk`/`view.FileChange`; `lsp` results are normalized
   into `view.CompletionItem`, `view.Location`, `view.Symbol`, etc. before
   crossing into `view`. Provider/protocol-shaped types (raw LSP structs, git
   plumbing types) stay inside their owning package.
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
    handler calls `view/action`, `view`, `lsp`, or `vcs` APIs; substantial
    editing, rendering, LSP protocol, VCS diffing, and persistence logic live
    in the owning package. If a handler is doing real work, move that work
    where it belongs.
12. **Reusable editing lives in `view/action`; pure text semantics live in
    `core`.** `core` never depends on `view` or terminal packages. Anything
    that needs a `Document`/`Editor`/`View` but is reusable across commands
    and UI belongs in `view/action`, not duplicated per command module.
13. **Calls request; observers/events announce.** `view.DocumentObserver`
    methods (`DocumentOpened`, `DocumentChanged`, `DocumentSaved`,
    `DocumentClosed`) report facts that already happened; implementations
    react to the fact and leave further mutation of that document to a later
    call. A direct method call (`SetVersionControl`,
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
17. **Move code to fix an actual dependency problem, not to satisfy a
    layering ideal.** Preserve cohesion.
    Moving a function to a "more correct" layer that adds indirection
    (forwarding wrappers, an interface with one implementation) without
    changing an actual dependency problem is a net loss.

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
  `view/action`, `lsp`, and `vcs`, it is where commands bridge to services.
- `cmd/toe/internal/app.go` may import any concrete module; it is the only
  place allowed to wire everything together.

**Before moving code or proposing a package, answer:**

- Who owns this concept today, and why is that owner wrong?
- What concept does the new home own, and which invariants does it preserve?
- What may it import, and what may import it?
- What independent reason to change justifies the move?
- Does the move reduce the number of packages a caller must understand?
- Will the move introduce forwarding wrappers, dependency inversion with no
  substitutable implementation, or a generic helper package?
- Can the boundary be enforced through imports alone or a narrow
  consumer-owned interface, without new indirection?

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

// Bad - verbose names for tight scope
for rangeIndex, currentRange := range ranges {
    if exists := currentRange.Valid(); !exists {  // Use 'ok', not 'exists'
        continue
    }
}
```

Name a local for its semantic subject rather than restating its type:

```go
// Good
rope := doc.Text()
sel := doc.Selection(vid)
tx := core.NewTransaction(rope)
entry, ok := p.spanCache[id]

// Bad
ropeValue := doc.Text()
selectionValue := doc.Selection(vid)
transactionValue := core.NewTransaction(ropeValue)
spanCacheEntry, ok := pickerState.spanCache[documentID]
```

**Longer names for wider scope**: struct fields, exported APIs, and tests where the variable itself is the subject under test.

```go
// Good - clear at API boundaries
func (e *Editor) OpenFile(path string) (*View, error)

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

`Default` prefix for defaults. `Max`/`Min` for limits. Name every magic number, group related constants in one block, and use typed constants when the type carries meaning:

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

Name the condition. Reach for `Is`/`Has` when the name needs it, not because the type is `bool`.

- Fields and locals: plain state name. `open`, `ready`, `enabled`, `ok`. Prefix only when the bare word reads as a value (`hasBOM`, `isBase`) or collides with a sibling (`hasStaged` beside `staged core.FileChangeKind`).
- `IsX` for a state, adjective, or classification: `IsDir`, `IsAbs`, `IsZero`, `IsEmpty`.
- Verb alone when the verb is the predicate: `HasPrefix`, `Contains`, `Exists`, `Equal`. Never `IsContains`.
- Bare property name when it reads as neither command nor getter. `fileWatchOp(ev)` is a getter, so `isFileWatchOp`; `ignoreCase(opts)` is an order, so `ignoresCase`.
- Imperatives stay operations. `Open`/`Close`/`Start`/`Stop`/`Read`/`Write` act; `IsOpen()` asks. A command may return a bool saying whether it acted.
- Standard library names keep their signature and meaning: `Read`, `Write`, `Close`, `Flush`, `String`.

Unsure? Copy the analogous stdlib API.

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

Markdown files expect to be soft-wrapped: keep paragraphs as readable logical lines and let the editor wrap them, since the code line-width limit applies to code, not prose. Preserve deliberate line breaks in lists, tables, code fences, quoted text, and other Markdown structures where the newline carries meaning.

### User Documentation

`README.md` and user-facing pages under `docs/content` should help competent users operate toe:

- Include what a feature does, how to use it, and choices or consequences that affect a user's workflow.
- State the general rule first, followed by meaningful exceptions.
- Keep command, keybinding, and configuration references complete and accurate.
- Use kebab-case command names in user documentation; underscore names are internal identifiers.
- Omit implementation details, internal mechanics, incidental behavior, tuning constants, and obvious facts.
- Document a change when knowing it materially helps someone use toe; observability alone is not the bar.
- State the main behavior outright, in plain terms readers can act on.
- Keep prose concise, and let a table or another appropriate page carry what it already states.

The architecture page is developer-facing and may describe internals, as long as its details serve architectural understanding rather than catalog implementation trivia.

### Line Width

Maximum 80 characters per line (tabs count as 4 spaces). This applies to code _and_ comments, not Markdown prose. Keep short argument lists on a single line when they fit; only break lines when the 80-character limit would be exceeded. When wrapping function signatures or call arguments, pack as many arguments per line as will fit under the limit before wrapping again, one per line only when a line would still exceed the limit. When you must wrap, break after the opening paren:

```go
func NewChangeSetFromChanges(
	doc Rope, changes []Change,
) (ChangeSet, error) {

func renderPreviewDocInto(
	buf *tui.Buffer, x0, y0 int, args *previewDocRender,
) {
```

```go
c, err := client.NewClient("embedded://", client.WithEmbedded(tr))
```

### Multi-line Calls with \*testing.T

When a function call wraps and the first argument is the test instance (`t`), keep `t` on the first line and break immediately after it, so `t` always shares a line with the call.

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
// Bad, type declared inside a function
func process() {
    type work struct{ id int }   // FORBIDDEN
    const limit = 100            // FORBIDDEN
    const (                      // FORBIDDEN
        kindA = iota
        kindB
    )
}

// Good, all at package level
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
4. Exported constructors (`New...`)
5. Exported methods
6. Exported functions
7. Unexported methods
8. Unexported functions

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

func NewHistory() History { ... }                        // constructor

func (h *History) CurrentRevision() int { ... }          // exported method
func (h *History) CommitRevision(...) error { ... }      // exported method

func UndoSteps(n int) UndoKind { ... }                   // exported function

func (h *History) jumpTo(to int) []Transaction { ... }   // unexported method
func indentWidth(s string, tabWidth int) int { ... }     // unexported helper
```

### Method Ordering

Within the order above, group exported methods by functionality and keep related methods together, ordered by call chain or first use. Unexported methods follow the exported ones they support, and pure helper functions (non-methods) sit at the bottom of the file.

### Concern Grouping

Within a package, organize files around real concerns, not arbitrary helper categories. Prefer concern-oriented grouping when that matches the code's behavior:

- `picker.go`: `picker-component.go`, `picker-render.go`
- `picker-files.go`: `picker-search.go`, `picker-commands.go`
- `render-document.go`: `render-status.go`
- `model-action.go`: `model-types.go`

Every file owns real behavior; forwarding calls to another package or renaming errors is not enough to earn one.

## Struct Literals

**NEVER construct a struct using positional field order.** Always use named fields. Positional literals are fragile: a field reorder or insertion silently compiles and corrupts data.

```go
// Good
Separator{Layout: LayoutVertical, X: a.X + a.Width, Y: c.area.Y, W: 1, H: c.area.Height}

// Bad, positional, breaks silently on field reorder
Separator{LayoutVertical, a.X + a.Width, c.area.Y, 1, c.area.Height}
```

The only exception is single-field structs where the field name adds no information (e.g. `Point{3}` when `Point` wraps a single `int`).

## Control Flow

### Early Returns

Use guard clauses to reject invalid preconditions before substantial main logic. No else when an early return avoids nesting:

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

Reserve guard clauses for real preconditions; simple linear functions read fine without them. When a value and success indicator are used only by a short success path, always declare them in the `if` initializer, keep the value scoped to that branch, and put the fallback afterward. Use this form when the condition is only the success indicator, or the success indicator plus one simple boolean, nil, equality, or predicate check. If the condition needs more than two checks, becomes multi-line, or the success branch contains substantial logic, use guard clauses instead.

```go
// Good
func lookup(name string) (Value, bool) {
	if value, ok := values[name]; ok && value.Available() {
		return value, true
	}
	return Value{}, false
}

// Bad, value and ok escape the only branch that uses them
func lookup(name string) (Value, bool) {
	value, ok := values[name]
	if !ok || !value.Available() {
		return Value{}, false
	}
	return value, true
}
```

### Multi-Assignment

Never assign multiple variables in one statement from independent sources, `a, b := x, y` where `x` and `y` are separate expressions. This is unreadable: the reader must visually pair each name on the left with its value on the right instead of reading top to bottom. The only exception is routing a single call's multiple return values, where the pairing is already fixed by the function's signature.

```go
// Good, routing one call's multi-return
value, ok := lookup(name)

// Good, independent sources, one per line
style := menuStyle
match := matchStyle

// Bad, independent sources crammed into one statement
style, match := menuStyle, matchStyle
```

This applies to plain assignment (`=`) as well as `:=`.

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

**`t.Run()` descriptions must be short and concise, never more than ~40 characters.** They label the scenario, not document it. Drop the subject (it's in the function name), drop "with", drop "without", drop the function name itself. Think: what's different here?

```go
// Good, concise, fits in one glance
t.Run("undoes and redoes edits", ...)
t.Run("navigates by steps", ...)
t.Run("empty selection returns error", ...)
t.Run("clips long lines", ...)

// Bad, too long, restates the subject
t.Run("History undoes and redoes edits correctly", ...)
t.Run("MoveRight with empty selection returns error", ...)
t.Run("PickerPreview clips long lines to width", ...)
```

```go
// Good - short function name holding the subtests above
func TestHistory(t *testing.T) { ... }

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

// Bad - message arguments
assert.NoError(t, err, "should not error")
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

Once the source has been split cleanly, split the test files to match.

### Running Tests

Run the fast suite while working, the full suite once at the end:

```sh
go test ./... -short   # while iterating
go test ./...          # before declaring the work done
```

Tests that spawn a pty, watch the real filesystem, shell out per subtest, or
wait on a real timer skip under `-short`:

```go
if testing.Short() {
    t.Skip("slow: spawns a real pty per subtest")
}
```

Put the skip on the slow subtest, not the whole parent, when only one subtest
is expensive. Never report work as passing on a `-short` run alone.

## Comments

### Godoc

**Exported** funcs, methods, types, consts, and vars always need godoc, but no more than 3 lines. Describe what it does, not how. If it takes more than 3 lines to say what something does, it isn't coded well:

```go
// History stores committed document revisions and supports undo/redo
// navigation by step count or time period
type History struct {

// NewHistory returns an empty history positioned at the root revision
func NewHistory() History {
```

Two exceptions need no godoc: sentinel error vars, where the message is the documentation, and the members of an enum const block, which their type and block name already document. Document the block itself when the set needs explaining, not each member.

**Unexported** funcs and methods get no godoc by default. Only add one, capped at 2 lines, when the behavior is genuinely non-trivial and needs explanation:

```go
// unexported, self-explanatory, no comment
func clampSelection(sel core.Selection, maxChars int) core.Selection {

// unexported, but the "why" isn't obvious from the signature, 2 lines max
// diffDebounce: single async debounce; the gutter trails a keystroke by this
const diffDebounce = 50 * time.Millisecond
```

Godoc rule: end the last sentence of a comment without a period.

### Inline Comments

Comment what the code cannot say for itself: a warranted comment explains
WHY, capped at 2 lines. Self-describing code stands on its own:

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
// Bad, mutable global state
var idCounter atomic.Int64

// Good, state lives on the owning struct
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
, `view.ErrNoLanguageServer`, not a same-named local copy that happens to
equal it.

```go
// Bad, lsp package re-exports view's sentinel under its own name
var ErrNoLanguageServer = view.ErrNoLanguageServer
...
return ErrNoLanguageServer

// Good, call sites use the owning package's identifier directly
return view.ErrNoLanguageServer
```

This applies to sentinel errors, constants, and any other exported value:
if `view` owns it, every package that needs it imports `view` and writes
`view.X`, under the one name the owner gave it.

## Interface Compliance

Compile-time interface checks:

```go
var _ CharMatcher = (*RuneMatcher)(nil)
```

## Error Handling

- **Always return errors** - never panic
- **Typed errors only** - All production code must use package-level vars with `Err` prefix
- **Pattern: `%w: context`**: wrapped error first, then context variable
- Plain error messages acceptable only in examples/documentation
- Handle errors immediately, early return

**Production Code - Always Use Typed Errors:**

```go
var (
	ErrEmptySelection       = errors.New("empty selection")
	ErrPrimaryIndexNotFound = errors.New("primary index not found")
)

// Good - %w: %d pattern with typed error
if primaryIndex >= len(ranges) {
    return fmt.Errorf("%w: %d", ErrPrimaryIndexNotFound, primaryIndex)
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
```

**Testing - Use errors.Is() to Check Typed Errors:**

Tests check for specific error types with `errors.Is()`, which keeps the check robust where `strings.Contains()` stays brittle:

```go
// Good - use errors.Is for typed errors
_, err := core.NewSelection(nil, 0)
assert.True(t, errors.Is(err, core.ErrEmptySelection))

// Bad - fragile string matching
assert.True(t, strings.Contains(err.Error(), "empty selection"))
```

**Examples/Documentation Only - Plain Messages OK:**

```go
// Only acceptable in README examples, not in engine code
return fmt.Errorf("invalid configuration: %s", reason)
```

---

# UI Library Policy

**Prefer Bubbletea and the `tui` buffer layer over home-grown alternatives.** Before writing custom terminal UI code, check whether an existing primitive already handles it:

- Use `tui.Style` for color and text modifiers, and the `tui.Border*` glyphs (`internal/tui/border.go`) for box drawing, not manual ANSI or ad-hoc `strings.Repeat` frames.
- Use `wrapText` (`internal/term/ui/wrap.go`) for word wrapping. Always wrap plain text first, then apply styling per cell, never feed styled/ANSI text to the wrapper; by then it is too late to wrap well.
- Use `runewidth.StringWidth` / `runewidth.Truncate` for cell widths and single-row clipping. `runewidth` is a direct dependency for exactly this: it walks runes and slices the input, so it allocates nothing, while the `ansi` equivalents scan for escape sequences and build through a `strings.Builder` (measured: 2–3 allocations and ~28% slower per call). Reach for `ansi.StringWidth` / `ansi.Truncate` only when the string genuinely carries escape sequences, which in practice means terminal-pane content, everywhere else toe styles per cell, so the text is plain by the time it is measured or clipped.
- For overlay panels, implement `BufferOverlayComponent` and draw into the `tui.Buffer` directly. The buffer-native path skips the ANSI round-trip and is fast for complex overlays.
- Use the `popup` struct (`internal/term/ui/popup.go`) for any bordered popup window, it fills the box with the content style and draws the border in one pass, so callers write per-cell content without worrying about ANSI background resets.
- Use `tea.View.Cursor` for cursor shape and position instead of raw DECSCUSR escapes in content strings.

The only valid reason to roll your own is when the library genuinely has no equivalent (e.g. tab expansion, custom fuzzy-match highlight, per-character cursor/selection coloring in the editor viewport).

---

# i18n Policy

Any user-facing English prose, status messages, prompts, hints shown during
an interactive mode, must go through `internal/i18n`, not a hardcoded Go
string constant. Add a `Key` in `internal/i18n/keys.go` and a translation
entry in each locale file (`en.json`, `de.json`, `fr.json`, `it.json`) under
`internal/i18n/translations/`, then reference it with `i18n.Text(key, ...)`.

`internal/i18n/translations/common.json` is reserved for values shared
identically across all locales (e.g. the `:` command prompt), not a catch-all
or a place to skip translating a new message into the other languages.

A message whose wording depends on a count takes plural forms instead of one
key per form: give the message an object value keyed by `zero`, `one`, or
`other`, and pass the number as the `count` variable.

```json
"status.yanked": {
  "one": "yanked {count} selection to register {register}",
  "other": "yanked {count} selections to register {register}"
}
```

A missing category falls back to `other`, so a language only writes the forms
it needs, French supplies `zero` where English does not. `other` is required:
a plural message without it fails to load. A message without a `count` variable
stays a plain string.

The one exception is a hint that echoes a literal keystroke sequence back at
the user (`"ms"`, `"r"`, `"^r"`), that's not language, so it
stays a plain Go string. A hint that also contains descriptive prose (e.g.
`"h/j/k/l or ←/↓/↑/→ resize, esc/enter exits"`) is not exempt and must be
translated.

---

# Tools

Use the **Serena MCP** for all code navigation and editing tasks: symbol lookup, find references, rename, go-to-definition, diagnostics. Prefer it over shell commands (`grep`, `find`, `sed`) for anything code-structural.

---

# CRITICAL: Rendering Performance

**ALWAYS benchmark when changing editor or preview rendering.** The render path runs on every keystroke and every frame; a per-render regression (re-parsing config, re-allocating per character, building off-screen content) makes the editor lag and back up input events. Measure, rather than reasoning about performance from first principles.

- Before and after any change to the editor content renderer or the picker preview renderer, run a Go benchmark with `-benchmem` and compare `ns/op`, `B/op`, and `allocs/op`. Profile with `-cpuprofile` / `-memprofile` and `go tool pprof` to locate the actual hotspot.
- Benchmark the realistic worst case: a long single line, the cursor scrolled far right, a split layout. See `BenchmarkRenderLongLine` (`internal/term/ui/bench_test.go`) and `BenchmarkVisualColumn` (`internal/view/bench_internal_test.go`).
- Rendering must do work **once** and re-do it only when its inputs change. Anything parsed, decoded, or loaded (config TOML, language definitions, themes, syntax) is cached once and invalidated on change, never re-parsed per render. Per-character work in the row loop stays allocation-free (use the ASCII fast paths) and runs only for on-screen columns.

---

# CRITICAL: Git Commits and Staging

**NEVER COMMIT. EVER.** Do not use `git commit` under any circumstances unless explicitly and directly instructed by the user in that exact session. Do not ask permission. Do not commit. Period.

The only exception is if the user explicitly says "commit" or "create a commit" in their current message.

**NEVER STAGE OR UNSTAGE.** Do not use `git add`, `git reset`, `git restore --staged`, or `git stash` unless explicitly and directly instructed in that exact session. The staged set is the user's own record of what he has reviewed; git keeps no index history, so unstaging destroys it permanently.

When told to stage a specific set, stage exactly that set and touch nothing else, an instruction to stage some files never implies unstaging others.
