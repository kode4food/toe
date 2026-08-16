---
title: "Scripting"
weight: 45
---

# Scripting

toe is scripted with [Ale](https://github.com/kode4food/ale), a small Lisp. Configuration lives in `$XDG_CONFIG_HOME/toe/init.ale`, evaluated at startup; a trusted workspace may add `.toe/init.ale`. The user file is evaluated first, so its bindings win conflicts. Restart toe after changing an `init.ale` file.

## Bindings

Bind keys with `toe/bind`. Every command in the command reference is available in the `toe` namespace under its primary kebab-case name:

```clojure
(toe/bind :modes :normal :keys "C-s" :doc "Write"
  (toe/write))

(toe/bind :modes :normal :keys "spc w"
  (toe/format)
  (toe/write))
```

A binding takes zero or more `:keyword value` options followed by the action, one or more expressions that run when the key is pressed. The action needs no `lambda` wrapper and need not return anything; call commands in list form, `(toe/write)`.

`:modes` accepts one mode or a vector, such as `[:normal :select]`. `:keys` accepts one key sequence or a vector, such as `["C-s" "spc w"]`. The supported modes are `:normal`, `:insert`, `:select`, `:terminal`, and `:image`. The optional `:doc` text appears in pending-key menus; bindings without it remain hidden.

Separate keys in a sequence with spaces. Printable characters name themselves; use `spc`, `ret`, `bksp`, `del`, `esc`, `tab`, `up`, `down`, `left`, `right`, `home`, `end`, `pgup`, and `pgdn` for special keys, with `C-`, `A-`, and `S-` modifiers.

Command arguments are strings:

```clojure
(toe/bind :modes :normal :keys "spc l"
  (toe/set-language "go"))
```

Binding an occupied key sequence is an error.

## Reading Editor State

Every binding action receives `ctx`, a read-only view of the editor, so an action can branch on where the cursor is, which language is active, or what is selected. Commands remain the only way to change state.

Properties are keyword-accessed, and a missing property returns nothing unless you supply a default:

```clojure
(toe/bind :modes :normal :keys "spc t"
  (if (eq "go" (:language (:document (:pane ctx)) ""))
    (toe/set-language "text")
    (toe/set-language "go")))
```

To gate a whole binding on state instead of branching inside it, use `:when` (below).

The context root:

| Key | Value |
|-----|-------|
| `:cwd` | Working directory |
| `:pane` | The focused pane |

The pane:

| Key | Value |
|-----|-------|
| `:kind` | `:view`, `:terminal`, or `:image` |
| `:mode` | `:normal`, `:insert`, `:select`, `:terminal`, or `:image` |
| `:path` | Pane path, when available |
| `:document` | The document, for a view pane |
| `:selection` | The selection, for a view pane |

The document:

| Key | Value |
|-----|-------|
| `:name` | Display name |
| `:path` | Backing path, when present |
| `:language` | Language identifier |
| `:modified` | Whether there are unsaved changes |
| `:read-only` | Whether the document is read-only |

The selection:

| Key | Value |
|-----|-------|
| `:primary` | Zero-based primary range index |
| `:ranges` | Vector of range objects |

Each range has zero-based character offsets. `:anchor` and `:head` preserve direction; `:from`, `:to`, and `:cursor` expose the derived range:

```clojure
{:anchor 10 :head 15 :from 10 :to 15 :cursor 14}
```

## Conditional Bindings

An optional `:when` expression makes a binding available only when it evaluates to true. Unlike branching inside the action, `:when` also hides the key from the pending-key menu while it is unavailable. The expression is wrapped like the action body, so `ctx` is in scope, no `lambda` needed:

```clojure
(toe/bind :modes :normal :keys "spc F" :doc "Format Go"
  :when (eq "go" (:language (:document (:pane ctx)) ""))
  (toe/format))
```

Negate it to bind a key only when the current document is *not* already Go:

```clojure
(toe/bind :modes [:normal] :keys ["spc l"] :doc "Set language to Go"
  :when (not (eq "go" (:language (:document (:pane ctx)) "")))
  (toe/set-language "go"))
```

When the expression evaluates to false, or raises an error, the key is treated as unbound. `:when` does not permit two bindings to share a key sequence; conflicts remain an error.
