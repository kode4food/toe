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
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	staged := make([]core.Range, len(ranges))
	kinds := make([]rangeKind, len(ranges))
	seen := map[int]bool{}
	pairs, pairEnabled := autoPairsForDocument(e, doc)

	for i, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			staged[i] = r
			kinds[i] = kindDup
			continue
		}
		seen[pos] = true
		if pairEnabled {
			change, newR, ok := core.HookInsert(text, r, ch, pairs)
			if ok {
				changes = append(changes, change)
				staged[i] = newR
				kinds[i] = kindAutoPair
				continue
			}
		}
		changes = append(changes, core.TextChange(pos, pos, string(ch)))
		staged[i] = r
		kinds[i] = kindNormal
	}

	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}

	newRanges := make([]core.Range, len(ranges))
	for i, r := range staged {
		switch kinds[i] {
		case kindAutoPair:
			newRanges[i] = r
		default:
			mapped, err := cs.MapRange(r)
			if err != nil {
				return
			}
			newRanges[i] = core.PointRange(mapped.Head)
		}
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
