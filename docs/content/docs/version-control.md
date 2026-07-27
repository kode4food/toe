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
| <span class="gutter-mark vcs-added">▍</span> | Added lines |
| <span class="gutter-mark vcs-modified">▍</span> | Modified lines |
| <span class="gutter-mark vcs-removed">▔</span> | Removed lines |

Colors follow the active theme; the swatches above use the default `mocha`.

## Change Navigation

| Key | Command | Action |
|-----|---------|--------|
| `]g` | `goto-next-change` | Jump to the next changed hunk |
| `[g` | `goto-prev-change` | Jump to the previous changed hunk |
| `]G` | `goto-last-change` | Jump to the last changed hunk |
| `[G` | `goto-first-change` | Jump to the first changed hunk |

## Changed Files

Use `Space+g` or `:changed-file-picker` to list changed workspace files and preview their diffs.

Each entry carries its change status. With `nerd-fonts` disabled, the short label in the second column is shown instead of the glyph:

| Status | Plain | Meaning |
|--------|-------|---------|
| {{< glyph "diff-added-16" "vcs-added" >}} | `A` | Added file |
| {{< glyph "diff-modified-16" "vcs-modified" >}} | `M` | Modified file |
| {{< glyph "diff-removed-16" "vcs-removed" >}} | `D` | Deleted file |
| {{< glyph "diff-renamed-16" "vcs-renamed" >}} | `R` | Renamed file |
| {{< glyph "question-16" "vcs-untracked" >}} | `?` | Untracked file |
| {{< glyph "alert-16" "vcs-conflict" >}} | `!` | Conflicted file |

## Resetting Changes

Use `:reset-diff-change` to reset the changed hunk under each selection back to the git base version.

Aliases:

```text
:reset-diff-change
:diff-reset
```

## Statusline

When configured, the `version-control` statusline element displays the current git head or branch name.
