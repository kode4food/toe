package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// DeleteCharBackward deletes the character before each cursor in insert mode
// Dedents when the cursor is at the end of leading whitespace; otherwise
// deletes an auto-pair if applicable, or one grapheme backward
func DeleteCharBackward(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	tabWidth := doc.TabWidth()
	indentWidth := doc.IndentStyle().IndentWidth(tabWidth)

	entries := make([]insertEntry, 0, len(ranges))
	seen := map[int]bool{}
	pairs, pairEnabled := autoPairsForDocument(e, doc)

	for _, r := range ranges {
		pos := r.Cursor(text)
		if pos == 0 || seen[pos] {
			entries = append(entries, insertEntry{})
			continue
		}
		seen[pos] = true
		// Dedent: if everything from line start to cursor is whitespace,
		// delete one indent unit
		if del, ok := dedentDelete(text, r, tabWidth, indentWidth); ok {
			entries = append(entries, insertEntry{del: del})
			continue
		}
		if pairEnabled {
			del, newR, ok := core.HookDelete(text, r, pairs)
			if ok {
				entries = append(entries, insertEntry{
					del:  del,
					newR: newR,
					pair: true,
				})
				continue
			}
		}
		prev := core.NthPrevGraphemeBoundary(text, pos, 1)
		entries = append(entries, insertEntry{
			del: core.Deletion{From: prev, To: pos},
		})
	}

	changes := make([]core.Change, 0, len(entries))
	for _, en := range entries {
		if en.del.From == en.del.To {
			continue
		}
		changes = append(changes, core.DeleteChange(en.del.From, en.del.To))
	}
	if len(changes) == 0 {
		return
	}

	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}

	newRanges := make([]core.Range, len(ranges))
	for i, r := range ranges {
		en := entries[i]
		if en.pair && (en.del.From != en.del.To) {
			mapped, err := cs.MapRange(en.newR)
			if err != nil {
				return
			}
			newRanges[i] = mapped
		} else {
			mapped, err := cs.MapRange(r)
			if err != nil {
				return
			}
			newRanges[i] = core.PointRange(mapped.Anchor)
		}
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

func dedentDelete(
	text core.Rope, r core.Range, tabWidth, indentWidth int,
) (core.Deletion, bool) {
	pos := r.Cursor(text)
	line, err := text.CharToLine(pos)
	if err != nil {
		return core.Deletion{}, false
	}
	lineStart, err := text.LineToChar(line)
	if err != nil {
		return core.Deletion{}, false
	}
	if pos == lineStart {
		return core.Deletion{}, false
	}
	// Verify the slice [lineStart, pos) is all whitespace
	width := 0
	for i := lineStart; i < pos; i++ {
		ch, err := text.CharAt(i)
		if err != nil || (ch != ' ' && ch != '\t') {
			return core.Deletion{}, false
		}
		if ch == '\t' {
			width += tabWidth
		} else {
			width++
		}
	}
	// If last char is a tab, delete one tab
	prevCh, err := text.CharAt(pos - 1)
	if err != nil {
		return core.Deletion{}, false
	}
	if prevCh == '\t' {
		return core.Deletion{From: pos - 1, To: pos}, true
	}
	// Otherwise delete enough spaces to reach the previous indent stop
	drop := width % indentWidth
	if drop == 0 {
		drop = indentWidth
	}
	start := pos
	for i := 0; i < drop; i++ {
		ch, err := text.CharAt(start - 1)
		if err != nil || ch != ' ' {
			break
		}
		start--
	}
	return core.Deletion{From: start, To: pos}, true
}
