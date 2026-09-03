---
title: "Sessions"
weight: 40
---

# Sessions

toe can restore open panes, layout, cursor state, and editor options between runs.

## Session File

Sessions are stored in `.toe/session.json.gz` at your project root (the workspace directory). Each project has its own independent session. An older `.toe/session.json` is still read if no `.gz` file is present, and is removed the next time the session is saved.

Auto-session only saves and restores session files for trusted workspaces. Run `:workspace-trust` in the project before relying on automatic session restore/save.

## Auto-Session

When `auto-session` is enabled (the default) and toe is launched with no file arguments, toe automatically:

- **Restores** the previous session
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
| Binary panes | ✓ |
| Terminal panes | Fresh shells |
| Split layout | ✓ |
| Cursor and view state | ✓ |
| Editor options (from `:set`) | ✓ |

Undo history is **not** saved. Each document starts with a fresh history after restore.
