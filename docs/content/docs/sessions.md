---
title: "Sessions"
weight: 40
---

# Sessions

toe can save and restore your editing session: the set of open documents,
split layout, cursor positions, view modes, and editor options.

## Session File

Sessions are stored in `.toe/session.toml` at your project root (the workspace
directory). Each project has its own independent session.

## Auto-Session

When `editor.auto-session` is enabled (the default), toe automatically:

- **Restores** the previous session when launched with no file arguments
- **Saves** the current session when you quit

```toml
# ~/.config/toe/config.toml
[editor]
auto-session = true   # default: true
```

To disable auto-session:

```sh
:set editor.auto-session false
```

## Manual Save and Restore

You can save or restore at any time regardless of the auto-session setting:

```
:save_session      (alias: save-session)
:restore_session   (alias: restore-session)
```

`save_session` writes the current state to `.toe/session.toml`. `restore_session`
reads it back, reopening all documents and restoring split layout.

## What Is Saved

| Item | Saved |
|------|-------|
| Open documents | ✓ |
| Split layout (horizontal/vertical) | ✓ |
| Cursor position per view | ✓ |
| Scroll offset per view | ✓ |
| View mode (normal/insert/select) | ✓ |
| Free-scroll flag | ✓ |
| Selection per view | ✓ |
| Editor options (from `:set`) | ✓ |

Undo history is **not** saved. Each document starts with a fresh history after
restore.
