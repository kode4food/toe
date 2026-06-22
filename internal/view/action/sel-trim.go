package action

import (
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// TrimSelections trims leading and trailing whitespace from each selection
// range. Empty or all-whitespace ranges are dropped. When all ranges are
// dropped the selection falls back to a single cursor at the primary position
func TrimSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	oldPrimary := sel.Primary()
	out := make([]core.Range, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		// drop empty or all-whitespace ranges entirely
		if from == to {
			continue
		}
		allSpace := true
		for i := from; i < to; i++ {
			ch, err := text.CharAt(i)
			if err != nil || !unicode.IsSpace(ch) {
				allSpace = false
				break
			}
		}
		if allSpace {
			continue
		}
		for from < to {
			ch, _ := text.CharAt(from)
			if !unicode.IsSpace(ch) {
				break
			}
			from++
		}
		for to > from {
			ch, _ := text.CharAt(to - 1)
			if !unicode.IsSpace(ch) {
				break
			}
			to--
		}
		out = append(out, core.NewRange(from, to).WithDirection(r.Direction()))
	}
	if len(out) == 0 {
		// all ranges were empty/whitespace: collapse to primary cursor
		cursor := oldPrimary.Cursor(text)
		newSel, err := core.NewSelection(
			[]core.Range{core.NewRange(cursor, cursor)}, 0)
		if err != nil {
			return
		}
		doc.SetSelectionFor(v.ID(), newSel)
		return
	}
	// set primary to first surviving range that overlaps old primary, else last
	primary := len(out) - 1
	for i, r := range out {
		if r.Overlaps(oldPrimary) {
			primary = i
			break
		}
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}
