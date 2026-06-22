package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// ExtendCharLeft extends the selection one grapheme to the left
func ExtendCharLeft(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionBackward, n, core.MovementExtend,
		)
	})
}

// ExtendCharRight extends the selection one grapheme to the right
func ExtendCharRight(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionForward, n, core.MovementExtend,
		)
	})
}

// ExtendLineUp extends the selection up one visual line, respecting soft-wrap
func ExtendLineUp(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.ExtendVerticallyVisual(doc, r, core.DirectionBackward, n)
	})
}

// ExtendLineDown extends the selection down one visual line, respecting
// soft-wrap
func ExtendLineDown(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.ExtendVerticallyVisual(doc, r, core.DirectionForward, n)
	})
}

// ExtendNextWordStart extends the selection to the start of the next word
func ExtendNextWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevWordStart extends the selection to the start of the previous word
func ExtendPrevWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextWordEnd extends the selection to the end of the next word
func ExtendNextWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextLongWordStart extends the selection to the start of the next WORD
func ExtendNextLongWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextLongWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevLongWordStart extends to the start of the previous WORD
func ExtendPrevLongWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevLongWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextLongWordEnd extends the selection to the end of the next WORD
func ExtendNextLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextLongWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevWordEnd extends the selection to the end of the previous word
func ExtendPrevWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevLongWordEnd extends the selection to the end of the previous WORD
func ExtendPrevLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevLongWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextSubWordStart extends to the start of the next sub-word
func ExtendNextSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextSubWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevSubWordStart extends to the start of the previous sub-word
func ExtendPrevSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevSubWordStart(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendNextSubWordEnd extends to the end of the next sub-word
func ExtendNextSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MoveNextSubWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendPrevSubWordEnd extends to the end of the previous sub-word
func ExtendPrevSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		word := core.MovePrevSubWordEnd(doc, r, n)
		return r.PutCursor(doc, word.Cursor(doc), true)
	})
}

// ExtendToLineStart extends the selection to the start of the current line
func ExtendToLineStart(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, start, true)
	})
}

// ExtendToLineEnd extends the selection to the end of the current line
func ExtendToLineEnd(e *view.Editor) {
	moveLineEnd(e, true)
}

// ExtendToFileStart extends the selection to the beginning of the document
func ExtendToFileStart(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, 0, true)
	})
}

// ExtendToLastLine extends to the start of the last non-blank line
func ExtendToLastLine(e *view.Editor) {
	moveFileEnd(e, true)
}

// ExtendToFileEnd extends all selections to the absolute end of the document
func ExtendToFileEnd(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, doc.LenChars(), true)
	})
}
