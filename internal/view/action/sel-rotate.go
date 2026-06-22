package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// RotateSelectionsForward rotates the primary selection index forward by
// count steps (wrapping around)
func RotateSelectionsForward(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	n := len(sel.Ranges())
	if n == 0 {
		return
	}
	count := max(e.Count(), 1)
	newSel, err := core.NewSelection(sel.Ranges(), (sel.PrimaryIndex()+count)%n)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// RotateSelectionsBackward rotates the primary selection index backward by
// count steps (wrapping around)
func RotateSelectionsBackward(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	n := len(sel.Ranges())
	if n == 0 {
		return
	}
	count := max(e.Count(), 1)
	prev := (sel.PrimaryIndex() + n - count%n) % n
	newSel, err := core.NewSelection(sel.Ranges(), prev)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// RotateContentsForward rotates the text content of each selection
// range forward by count steps
func RotateContentsForward(e *view.Editor) {
	rotateSelectionContents(e, true)
}

// RotateContentsBackward rotates the text content of each selection
// range backward by count steps
func RotateContentsBackward(e *view.Editor) {
	rotateSelectionContents(e, false)
}

func rotateSelectionContents(e *view.Editor, forward bool) {
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
	n := len(ranges)
	if n == 0 {
		return
	}
	count := max(e.Count(), 1)
	steps := min(count, n)
	texts := make([]string, n)
	for i, r := range ranges {
		slice, err := text.Slice(r.From(), r.To())
		if err != nil {
			return
		}
		texts[i] = slice.String()
	}
	rotated := make([]string, n)
	var newPrimary int
	p := sel.PrimaryIndex()
	if forward {
		for i := range n {
			rotated[i] = texts[(i-steps+n)%n]
		}
		newPrimary = (p + steps) % n
	} else {
		for i := range n {
			rotated[i] = texts[(i+steps)%n]
		}
		newPrimary = (p + n - steps) % n
	}
	changes := make([]core.Change, n)
	for i, r := range ranges {
		changes[i] = core.TextChange(r.From(), r.To(), rotated[i])
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newRanges := rangesAfterReplace(ranges, rotated)
	newSel, err := core.NewSelection(newRanges, newPrimary)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

func rangesAfterReplace(
	ranges []core.Range, replacements []string,
) []core.Range {
	out := make([]core.Range, len(ranges))
	delta := 0
	for i, r := range ranges {
		newFrom := r.From() + delta
		newLen := len([]rune(replacements[i]))
		out[i] = core.NewRange(newFrom, newFrom+newLen)
		delta += newLen - (r.To() - r.From())
	}
	return out
}
