package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// ExtendToLineBounds extends each selection range to cover complete lines
// (from line start to next-line start), preserving direction
func ExtendToLineBounds(e *view.Editor) {
	applyMove(e, func(text core.Rope, r core.Range) core.Range {
		lr, err := r.LineRange(text)
		if err != nil {
			return r
		}
		start, err := text.LineToChar(lr.From)
		if err != nil {
			return r
		}
		nLines := text.LenLines()
		endLine := lr.To + 1
		var end int
		if endLine >= nLines {
			end = text.LenChars()
		} else {
			end, err = text.LineToChar(endLine)
			if err != nil {
				return r
			}
		}
		return core.NewRange(start, end).WithDirection(r.Direction())
	})
}

// ShrinkToLineBounds shrinks each multi-line selection so that it no longer
// includes leading/trailing line endings. Single-line selections are unchanged
func ShrinkToLineBounds(e *view.Editor) {
	applyMove(e, func(text core.Rope, r core.Range) core.Range {
		lr, err := r.LineRange(text)
		if err != nil {
			return r
		}
		if lr.From == lr.To {
			return r
		}
		nLines := text.LenLines()
		start, err := text.LineToChar(lr.From)
		if err != nil {
			return r
		}
		endLine := lr.To + 1
		var end int
		if endLine >= nLines {
			end = text.LenChars()
		} else {
			end, err = text.LineToChar(endLine)
			if err != nil {
				return r
			}
		}
		if start != r.From() {
			nextLine := lr.From + 1
			if nextLine < nLines {
				start, err = text.LineToChar(nextLine)
				if err != nil {
					return r
				}
			}
		}
		if end != r.To() {
			end, err = text.LineToChar(lr.To)
			if err != nil {
				return r
			}
		}
		return core.NewRange(start, end).WithDirection(r.Direction())
	})
}
