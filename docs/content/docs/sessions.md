---
title: "Sessions"
weight: 40
---

# Sessions

toe can restore open panes, layout, cursor state, and editor options between runs.

## Session File

Sessions are stored in `.toe/session.toml` at your project root (the workspace directory). Each project has its own independent session.

Auto-session only saves and restores session files for trusted workspaces. Run `:workspace-trust` in the project before relying on automatic session restore/save.

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
:save-session
:restore-session
```

## What Is Saved

| Item | Saved |
|------|-------|
| Open documents | ✓ |
| Image panes | ✓ |
| Terminal panes | Fresh shells |
| Split layout | ✓ |
| Cursor and view state | ✓ |
| Editor options (from `:set`) | ✓ |

Undo history is **not** saved. Each document starts with a fresh history after restore.
