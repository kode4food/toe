# Toe Scripting API

## Goal

Expose toe's live editor state to Ale without eagerly constructing a large object, leaking Go implementation values, or creating a speculative event framework.

Commands remain the mutation interface. The context API is read-only.

## Context

Add a zero-argument procedure:

```ale
(toe/context)
```

It returns a lightweight `data.Mapped` implementation backed by the editor. The value is a live view, not a snapshot: every property lookup reads the current authoritative state.

The root context initially supports:

| Key | Value |
|-----|-------|
| `:cwd` | Toe's working directory |
| `:pane` | The currently focused pane |

Unavailable properties are absent. `Get` returns `(data.Null, false)` for a missing or unknown key, allowing callers to choose a fallback:

```ale
(:pane (toe/context) false)
```

Do not insert `false`, `null`, empty strings, or placeholder objects to represent missing properties.

## Lazy Values

The context and its nested values implement `data.Mapped`. A lookup constructs only the requested branch:

- `:pane` queries the focused pane
- `:document` queries the document displayed by a view pane
- `:selection` queries that view's current selection
- `:ranges` converts selection ranges to Ale values

Do not eagerly build document or selection data when a script only asks for `:cwd`, `:kind`, or `:mode`.

Mapped values use identity equality. They should implement `fmt.Stringer`; explicit stringification may materialize their available properties because the caller requested a printable representation.

These values are property providers, not Ale objects. Keyword access and `mapped?` work, but object enumeration and merging are not part of the contract. Add an explicit snapshot operation only if scripts later need a frozen, enumerable object.

## Pane

The focused pane value initially supports:

| Key | Value |
|-----|-------|
| `:kind` | `:view`, `:terminal`, or `:image` |
| `:mode` | `:normal`, `:insert`, `:select`, `:terminal`, or `:image` |
| `:path` | The pane path, when available |
| `:document` | Document details for a view pane |
| `:selection` | The selection for the focused view |

Derive the current pane kind from the existing focused view and pane mode. Do not add another pane-kind abstraction solely for scripting.

## Document

The document value initially supports:

| Key | Value |
|-----|-------|
| `:name` | Display name |
| `:path` | Backing path, when present |
| `:language` | Active language identifier |
| `:modified` | Whether the document has unsaved changes |
| `:read-only` | Whether the document is read-only |

Do not expose the complete document text, internal IDs, revisions, history, services, or mutable Go values. Add purpose-built text queries when a concrete script needs them.

## Selection

The selection value initially supports:

| Key | Value |
|-----|-------|
| `:primary` | Zero-based primary range index |
| `:ranges` | Vector of range objects |

Each range is a small concrete Ale object:

```ale
{:anchor 10
 :head 15
 :from 10
 :to 15
 :cursor 14}
```

Positions are zero-based character offsets. `:anchor` and `:head` preserve direction; `:from`, `:to`, and `:cursor` expose toe's derived range semantics.

Do not include selected text in every range. Add an explicit range-text operation if needed so large selections are copied only on request.

## Conditional Bindings

Context queries inside a bound action allow the action to branch, but they do not hide an unavailable action from pending-key menus. Treat conditional availability as a separate change.

When needed, extend `toe/bind` with an optional `:when` procedure:

```ale
(toe/bind
  :modes :normal
  :keys "spc g"
  :doc "Go Action"
  :when (lambda (context)
          (let [pane (:pane context false)
                doc  (and pane (:document pane false))]
            (and doc (eq "go" (:language doc)))))
  action)
```

The predicate receives a context value and uses Ale truthiness. It controls both key dispatch and pending-key hints. Predicate errors make the binding unavailable and surface through the normal status-message path.

Binding conflicts remain static. Conditional predicates do not permit multiple bindings for the same key sequence.

The command package must know only an availability function over `*view.Editor`; it must not import or understand Ale.

## Events

Do not implement listeners until there is a concrete event use case.

The eventual public shape is:

```ale
(toe/on :document-saved
  (lambda (event)
    ...))
```

Unlike the live context, an event is a concrete immutable Ale object captured when the event occurs:

```ale
{:type :document-saved
 :context {...}
 :document {...}}
```

Event requirements:

- `App` owns the Ale runtime for the application lifetime
- Ale procedures execute serially on the Bubble Tea event loop
- observers and timer goroutines enqueue events instead of calling Ale directly
- callbacks run after the editor operation that announced the event
- each callback invocation owns its own `command.Result`
- runtime errors surface through the normal status-message path
- startup evaluation errors remain fatal
- listeners run in registration order

Start with one concrete event and its existing authoritative owner. Add focus, mode, selection, timer, and other events individually as real scripting needs appear. Do not create a universal event bus in advance.

An unsubscribe API is unnecessary while listener registration occurs only during startup. Add one when scripts can reload or register listeners dynamically.

## Implementation Order

1. Implement the live mapped context and pane properties.
2. Add document properties.
3. Add lazy selection and range properties.
4. Add black-box Ale tests covering missing keys, live updates, each pane kind, scratch documents, modes, and multiple selections.
5. Document the scripting API and its live-value semantics.
6. Add conditional bindings only when contextual menu visibility is required.
7. Add event dispatch only with the first concrete listener use case.

## Acceptance Criteria

- Reading a shallow property does not construct deeper values
- Missing keys support Ale's mapped default lookup
- Context reads current editor state after focus, mode, document, or selection changes
- No scripting value exposes mutable editor internals
- Existing commands and bindings behave unchanged
- Tests remain black-box and the full pre-commit suite passes
