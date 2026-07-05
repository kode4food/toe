package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// InsertChar inserts a character at every cursor position in insert mode
// Auto-pairs are applied when the character matches an opener or closer
func InsertChar(e *view.Editor, ch rune) {
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
	pairs, pairEnabled := autoPairsForDocument(e, doc)

	changes := make([]core.Change, 0, len(ranges))
	staged := make([]core.Range, len(ranges))
	paired := make([]bool, len(ranges))
	seen := map[int]bool{}

	for i, r := range ranges {
		pos := r.Cursor(text)
		staged[i] = r
		if seen[pos] {
			continue
		}
		seen[pos] = true
		if pairEnabled {
			if change, newR, ok := core.HookInsert(text, r, ch, pairs); ok {
				changes = append(changes, change)
				staged[i] = newR
				paired[i] = true
				continue
			}
		}
		changes = append(changes, core.TextChange(pos, pos, string(ch)))
	}

	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}

	newRanges := make([]core.Range, len(ranges))
	for i, r := range staged {
		if paired[i] {
			newRanges[i] = r
			continue
		}
		mapped, err := cs.MapRange(r)
		if err != nil {
			return
		}
		newRanges[i] = mapped
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

func autoPairsForDocument(
	e *view.Editor, doc *view.Document,
) (core.AutoPairs, bool) {
	global, ok := e.Options().AutoPairs()
	if !ok {
		return core.AutoPairs{}, false
	}
	lang := language.LoadLanguage(doc.Lang())
	if pairs, ok := lang.AutoPairs.AutoPairs(); ok {
		return pairs, true
	}
	return global, true
}
