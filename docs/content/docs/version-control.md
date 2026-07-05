---
title: "Version Control"
weight: 55
---

# Version Control

toe has built-in git support for changed files and per-line diffs. The version-control provider is detected from the current workspace; git is the only provider today.

## Diff Gutters

When a file is tracked by git, toe shows line changes in the gutter:

| Marker | Meaning |
|--------|---------|
| `▍` | Added or modified lines |
| `▔` | Removed lines |

The gutter updates as buffers change and after files are saved or reloaded.

## Change Navigation

| Key | Command | Action |
|-----|---------|--------|
| `]g` | `goto_next_change` | Jump to the next changed hunk |
| `[g` | `goto_prev_change` | Jump to the previous changed hunk |
| `]G` | `goto_last_change` | Jump to the last changed hunk |
| `[G` | `goto_first_change` | Jump to the first changed hunk |

## Changed Files

Use `Space+g` or `:changed_file_picker` to open the changed-file picker. It lists changed workspace files and previews the first diff hunk for the selected file.

## Resetting Changes

Use `:reset_diff_change` to reset the changed hunk under each selection back to the git base version.

Aliases:

```text
:reset-diff-change
:diff-reset
```

## Statusline

The `version-control` statusline element displays the current git head or branch name when available.
