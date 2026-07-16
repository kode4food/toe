package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// ExtendLineBelow selects the current line(s) inclusive of the trailing
// newline. If the range already spans the full line, extends one more line
func ExtendLineBelow(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		b, ok := resolveLineBounds(doc, r)
		if !ok {
			return r
		}
		if r.From() == b.start && r.To() == b.end {
			var nextEnd int
			var err error
			if b.endLine+2 >= doc.LenLines() {
				nextEnd = doc.LenChars()
			} else {
				nextEnd, err = doc.LineToChar(b.endLine + 2)
				if err != nil {
					return r
				}
			}
			return core.NewRange(b.start, nextEnd)
		}
		return core.NewRange(b.start, b.end)
	})
}

// SelectLineBelow selects current line(s) extending downward, direction-aware:
// forward selections grow at the head; backward selections shrink at the head
func SelectLineBelow(e *view.Editor) {
	selectLineImpl(e, false)
}

// SelectLineAbove selects current line(s) extending upward, direction-aware:
// backward selections grow at the head; forward selections shrink at the head
func SelectLineAbove(e *view.Editor) {
	selectLineImpl(e, true)
}

type resolveLineBoundsRes struct {
	startLine, endLine int
	start, end         int
}

func resolveLineBounds(
	doc core.Rope, r core.Range,
) (resolveLineBoundsRes, bool) {
	lr, err := r.LineRange(doc)
	if err != nil {
		return resolveLineBoundsRes{}, false
	}
	startLine, endLine := lr.From, lr.To
	start, err := doc.LineToChar(startLine)
	if err != nil {
		return resolveLineBoundsRes{}, false
	}
	var end int
	if endLine+1 >= doc.LenLines() {
		end = doc.LenChars()
	} else {
		end, err = doc.LineToChar(endLine + 1)
		if err != nil {
			return resolveLineBoundsRes{}, false
		}
	}
	return resolveLineBoundsRes{
		startLine: startLine,
		endLine:   endLine,
		start:     start,
		end:       end,
	}, true
}

func selectLineImpl(e *view.Editor, above bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	nLines := text.LenLines()
	sat := func(line int) int { return min(line, nLines) }
	lineChar := func(line int) int {
		pos, _ := text.LineToChar(line)
		return pos
	}
	count := countOrOne(e)
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		startLine, endLine := lr.From, lr.To
		start := lineChar(startLine)
		end := lineChar(sat(endLine + 1))

		// Snapping to line bounds counts as one step
		cnt := count
		if r.From() != start || r.To() != end {
			cnt = max(cnt-1, 0)
		}

		var anchorLine, headLine int
		dir := r.Direction()
		if above {
			switch dir {
			case core.DirectionForward:
				anchorLine = startLine
				headLine = max(endLine-cnt, 0)
			default:
				anchorLine = endLine
				headLine = max(startLine-cnt, 0)
			}
		} else {
			switch dir {
			case core.DirectionForward:
				anchorLine = startLine
				headLine = sat(endLine + cnt)
			default:
				anchorLine = endLine
				headLine = sat(startLine + cnt)
			}
		}

		var anchor, head int
		switch {
		case anchorLine < headLine:
			anchor = lineChar(anchorLine)
			head = lineChar(sat(headLine + 1))
		case anchorLine == headLine:
			if above {
				anchor = lineChar(sat(anchorLine + 1))
				head = lineChar(headLine)
			} else {
				anchor = lineChar(headLine)
				head = lineChar(sat(anchorLine + 1))
			}
		default:
			anchor = lineChar(sat(anchorLine + 1))
			head = lineChar(headLine)
		}
		ranges[i] = core.NewRange(anchor, head)
	}
	e.ResetCount()
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}
