package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// DeleteCharForward deletes the grapheme cluster under each cursor
func DeleteCharForward(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		next := core.NthNextGraphemeBoundary(text, core.GraphemeStep{
			From:  pos,
			Count: 1,
		})
		if next <= pos {
			continue
		}
		changes = append(changes, core.DeleteChange(core.Span{
			From: pos,
			To:   next,
		}))
	}
	applyDeletesAtCursor(e, applyDeletesAtCursorArgs{
		text:    text,
		sel:     sel,
		ranges:  ranges,
		changes: changes,
	})
}

// DeleteWordBackward deletes from the cursor to the start of the previous
// word, for use in insert mode (C-w)
func DeleteWordBackward(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if pos == 0 || seen[pos] {
			continue
		}
		seen[pos] = true
		wordStart := core.MovePrevWordStart(
			text, core.PointRange(pos), 1,
		).From()
		changes = append(changes, core.DeleteChange(core.Span{
			From: wordStart,
			To:   pos,
		}))
	}
	applyDeletesAtCursor(e, applyDeletesAtCursorArgs{
		text:    text,
		sel:     sel,
		ranges:  ranges,
		changes: changes,
	})
}

// DeleteWordForward deletes from the cursor to the end of the next word,
// for use in insert mode (A-d)
func DeleteWordForward(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		wordEnd := core.MoveNextWordEnd(text, core.PointRange(pos), 1).To()
		if wordEnd <= pos {
			continue
		}
		changes = append(changes, core.DeleteChange(core.Span{
			From: pos,
			To:   wordEnd,
		}))
	}
	applyDeletesAtCursor(e, applyDeletesAtCursorArgs{
		text:    text,
		sel:     sel,
		ranges:  ranges,
		changes: changes,
	})
}

type applyDeletionsArgs struct {
	text   core.Rope
	sel    core.Selection
	ranges []core.Range
}

func applyDeletions(e *view.Editor, args applyDeletionsArgs) bool {
	changes := make([]core.Change, 0, len(args.ranges))
	for _, r := range args.ranges {
		eff := r.MinWidth1(args.text)
		changes = append(changes, core.DeleteChange(core.Span{
			From: eff.From(),
			To:   eff.To(),
		}))
	}
	cs, err := core.NewChangeSetFromChanges(args.text, changes)
	if err != nil {
		return false
	}
	newRanges := make([]core.Range, len(args.ranges))
	for i, r := range args.ranges {
		eff := r.MinWidth1(args.text)
		mapped, err := cs.MapRange(eff)
		if err != nil {
			return false
		}
		newRanges[i] = core.PointRange(mapped.From())
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return false
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	return true
}

type applyDeletesAtCursorArgs struct {
	text    core.Rope
	sel     core.Selection
	ranges  []core.Range
	changes []core.Change
}

func applyDeletesAtCursor(e *view.Editor, args applyDeletesAtCursorArgs) {
	if len(args.changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(args.text, args.changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(args.ranges))
	for i, r := range args.ranges {
		if mapped, err := cs.MapRange(r); err == nil {
			newRanges[i] = core.PointRange(mapped.Head)
			continue
		}
		return
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}
