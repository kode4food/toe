package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// CopyOnNextLine duplicates each selection range to the same column
// on the next line (count times)
func CopyOnNextLine(e *view.Editor) {
	copySelectionOnLine(e, true)
}

// CopyOnPrevLine duplicates each selection range to the same column
// on the previous line (count times)
func CopyOnPrevLine(e *view.Editor) {
	copySelectionOnLine(e, false)
}

func copySelectionOnLine(e *view.Editor, forward bool) {
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
	n := max(e.Count(), 1)
	nLines := text.LenLines()

	primary := sel.PrimaryIndex()
	ranges := sel.Ranges()
	out := make([]core.Range, len(ranges))
	copy(out, ranges)
	newPrimary := primary

	for i, r := range ranges {
		anchorLine, err := text.CharToLine(r.From())
		if err != nil {
			continue
		}
		headLine, err := text.CharToLine(r.To())
		if err != nil {
			continue
		}
		// height is the number of lines spanned by the selection (min 1)
		height := headLine - anchorLine + 1

		anchorLineStart, _ := text.LineToChar(anchorLine)
		headLineStart, _ := text.LineToChar(headLine)
		anchorCol := r.From() - anchorLineStart
		headCol := r.To() - headLineStart

		added := 0
		for step := 1; added < n; step++ {
			offset := step * height
			var destAnchorLine, destHeadLine int
			if forward {
				destAnchorLine = anchorLine + offset
				destHeadLine = headLine + offset
			} else {
				destAnchorLine = anchorLine - offset
				destHeadLine = headLine - offset
			}
			if destAnchorLine < 0 || destHeadLine < 0 ||
				destAnchorLine >= nLines || destHeadLine >= nLines {
				break
			}
			destAnchorStart, err := text.LineToChar(destAnchorLine)
			if err != nil {
				break
			}
			destHeadStart, err := text.LineToChar(destHeadLine)
			if err != nil {
				break
			}
			destAnchorLineEnd, _ := text.LineEndCharIndex(destAnchorLine)
			destHeadLineEnd, _ := text.LineEndCharIndex(destHeadLine)
			newAnchor := min(destAnchorStart+anchorCol, destAnchorLineEnd)
			newHead := min(destHeadStart+headCol, destHeadLineEnd)

			newRange := core.NewRange(newAnchor, newHead)
			if hasDuplicateHead(out, newRange) {
				break
			}
			out = append(out, newRange)
			if i == primary {
				newPrimary = len(out) - 1
			}
			added++
		}
	}

	newSel, err := core.NewSelection(out, newPrimary)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

func hasDuplicateHead(ranges []core.Range, r core.Range) bool {
	for _, existing := range ranges {
		if existing.Head == r.Head {
			return true
		}
	}
	return false
}
