---
title: "Sessions"
weight: 40
---

# Sessions

toe can save and restore your editing session: open documents, image panes, terminal slots, split layout, cursor positions, view modes, and editor options.

## Session File

Sessions are stored in `.toe/session.toml` at your project root (the workspace directory). Each project has its own independent session.

Auto-session only saves and restores session files for trusted workspaces. Run `:workspace_trust` in the project before relying on automatic session restore/save.

## Auto-Session

When `auto-session` is enabled (the default), toe automatically:

- **Restores** the previous session when launched with no file arguments
- **Saves** the current session when you quit

```toml
# ~/.config/toe/config.toml
[editor]
auto-session = true   # default: true
```

To disable auto-session:

```sh
:set auto-session false
```

## Manual Save and Restore

You can save or restore at any time regardless of the auto-session setting:

```
:save_session      (alias: save-session)
:restore_session   (alias: restore-session)
```

`save_session` writes the current state to `.toe/session.toml`. `restore_session` reads it back, reopening documents, image panes, and terminal slots while restoring split layout.

## What Is Saved

| Item | Saved |
|------|-------|
| Open documents | ✓ |
| Image panes | ✓ |
| Terminal panes | ✓ |
| Split layout (horizontal/vertical) | ✓ |
| Cursor position per view | ✓ |
| Scroll offset per view | ✓ |
| View mode (normal/insert/select/image/terminal) | ✓ |
| Free-scroll flag | ✓ |
| Selection per view | ✓ |
| Editor options (from `:set`) | ✓ |

Undo history is **not** saved. Each document starts with a fresh history after
restore.

Terminal panes restore as fresh shells. toe saves the pane path, using the shell's OSC 7 current-directory report when available.
