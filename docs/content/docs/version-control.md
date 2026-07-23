---
title: "Version Control"
weight: 55
---

# Version Control

toe has built-in Git support for changed files and per-line diffs.

## Diff Gutters

When a file is tracked by git, toe shows line changes in the gutter:

| Marker | Meaning |
|--------|---------|
| `▍` | Added or modified lines |
| `▔` | Removed lines |

## Change Navigation

| Key | Command | Action |
|-----|---------|--------|
| `]g` | `goto-next-change` | Jump to the next changed hunk |
| `[g` | `goto-prev-change` | Jump to the previous changed hunk |
| `]G` | `goto-last-change` | Jump to the last changed hunk |
| `[G` | `goto-first-change` | Jump to the first changed hunk |

## Changed Files

Use `Space+g` or `:changed-file-picker` to list changed workspace files and preview their diffs.

## Resetting Changes

Use `:reset-diff-change` to reset the changed hunk under each selection back to the git base version.

Aliases:

```text
:reset-diff-change
:diff-reset
```

## Statusline

The `version-control` statusline element displays the current git head or branch name when available.
