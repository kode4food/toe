package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// SelectAll selects the entire document in the focused view
func SelectAll(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	n := doc.Text().LenChars()
	sel, err := core.NewSelection([]core.Range{core.NewRange(0, n)}, 0)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
}

// CollapseSelection collapses every selection to its cursor position
func CollapseSelection(e *view.Editor) {
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
	ranges := sel.Ranges()
	for i, r := range ranges {
		pos := r.Cursor(text)
		ranges[i] = core.PointRange(pos)
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// FlipSelections swaps anchor and head for every selection range
func FlipSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = r.Flip()
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// KeepPrimarySelection discards all but the primary selection range
func KeepPrimarySelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary()
	newSel, err := core.NewSelection(
		[]core.Range{core.NewRange(primary.Anchor, primary.Head)}, 0,
	)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}
