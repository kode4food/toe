package action

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// MoveLeft moves all cursors one grapheme to the left
func MoveLeft(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionBackward, n, core.MovementMove,
		)
	})
}

// MoveRight moves all cursors one grapheme to the right
func MoveRight(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.MoveHorizontally(
			doc, core.DirectionForward, n, core.MovementMove,
		)
	})
}

// MoveUp moves all cursors up one visual line, respecting soft-wrap
func MoveUp(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.MoveVerticallyVisual(doc, r, core.DirectionBackward, n)
	})
}

// MoveDown moves all cursors down one visual line, respecting soft-wrap
func MoveDown(e *view.Editor) {
	n := countOrOne(e)
	vf := visualMoveFormat(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return vf.MoveVerticallyVisual(doc, r, core.DirectionForward, n)
	})
}

// MoveWordForward moves all cursors to the start of the next word
func MoveWordForward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextWordStart(doc, r, n)
	})
}

// MoveWordBackward moves all cursors to the start of the previous word
func MoveWordBackward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevWordStart(doc, r, n)
	})
}

// MoveWordEnd moves all cursors to the end of the current or next word
func MoveWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextWordEnd(doc, r, n)
	})
}

// MoveLongWordForward moves all cursors to the start of the next WORD
func MoveLongWordForward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextLongWordStart(doc, r, n)
	})
}

// MoveLongWordBackward moves all cursors to the start of the previous WORD
func MoveLongWordBackward(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevLongWordStart(doc, r, n)
	})
}

// MoveLongWordEnd moves all cursors to the end of the current or next WORD
func MoveLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextLongWordEnd(doc, r, n)
	})
}

// MovePrevWordEnd moves all cursors to the end of the previous word
func MovePrevWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevWordEnd(doc, r, n)
	})
}

// MovePrevLongWordEnd moves all cursors to the end of the previous WORD
func MovePrevLongWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevLongWordEnd(doc, r, n)
	})
}

// MoveNextSubWordStart moves to the start of the next sub-word
func MoveNextSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextSubWordStart(doc, r, n)
	})
}

// MovePrevSubWordStart moves to the start of the previous sub-word
func MovePrevSubWordStart(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevSubWordStart(doc, r, n)
	})
}

// MoveNextSubWordEnd moves to the end of the next sub-word
func MoveNextSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MoveNextSubWordEnd(doc, r, n)
	})
}

// MovePrevSubWordEnd moves to the end of the previous sub-word
func MovePrevSubWordEnd(e *view.Editor) {
	n := countOrOne(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return core.MovePrevSubWordEnd(doc, r, n)
	})
}

// MoveLineStart moves all cursors to the start of their current line
func MoveLineStart(e *view.Editor) {
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
		return r.PutCursor(doc, start, false)
	})
}

// MoveLineEnd moves all cursors to the last non-newline character of
// their current line
func MoveLineEnd(e *view.Editor) {
	moveLineEnd(e, false)
}

// MoveLineNonWhitespace moves all cursors to the first non-whitespace
// character of their current line
func MoveLineNonWhitespace(e *view.Editor) {
	moveToNonWhitespace(e, e.Mode() == view.ModeSelect)
}

// ExtendToNonWhitespace extends the selection to the first non-whitespace
// character on the current line
func ExtendToNonWhitespace(e *view.Editor) {
	moveToNonWhitespace(e, true)
}

// MoveFileStart moves all cursors to the start of the document
func MoveFileStart(e *view.Editor) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		return r.PutCursor(doc, 0, false)
	})
}

// MoveFileEnd moves all cursors to the start of the last non-blank line
func MoveFileEnd(e *view.Editor) {
	moveFileEnd(e, false)
}

// GotoLine moves (or extends in select mode) the cursor to line n (1-based)
// If n is 0 the command is a no-op. Clamps to the last non-empty line
func GotoLine(e *view.Editor, n int) {
	if n <= 0 {
		return
	}
	SaveSelection(e)
	extend := e.Mode() == view.ModeSelect
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		nLines := doc.LenLines()
		maxLine := nLines - 1
		// If the last line is blank, don't jump to it
		if maxLine > 0 {
			lastLineStart, err := doc.LineToChar(maxLine)
			if err == nil && lastLineStart >= doc.LenChars() {
				maxLine--
			}
		}
		line := min(n-1, maxLine)
		pos, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func moveLineEnd(e *view.Editor, extend bool) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		pos := max(lineEnd-1, 0)
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		if pos < start {
			pos = start
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func moveToNonWhitespace(e *view.Editor, extend bool) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		start, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		pos := start
		for pos < lineEnd {
			ch, err := doc.CharAt(pos)
			if err != nil {
				break
			}
			if ch != ' ' && ch != '\t' {
				break
			}
			pos++
		}
		return r.PutCursor(doc, pos, extend)
	})
}

func moveFileEnd(e *view.Editor, extend bool) {
	SaveSelection(e)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		nLines := doc.LenLines()
		lineIdx := nLines - 1
		if lineIdx > 0 {
			lastStart, err := doc.LineToChar(lineIdx)
			if err == nil && lastStart >= doc.LenChars() {
				lineIdx--
			}
		}
		pos, err := doc.LineToChar(lineIdx)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
	})
}

// countOrOne returns the pending count or 1 if none is set
func countOrOne(e *view.Editor) int {
	if n := e.Count(); n > 0 {
		return n
	}
	return 1
}

// visualMoveFormat builds a VisualMoveFormat for the focused document if
// soft-wrap is active, returning a zero value otherwise
func visualMoveFormat(e *view.Editor) *core.VisualMoveFormat {
	w := e.ViewContentWidth()
	if w <= 0 {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	format := doc.TextFormatForConfig(w, e.Options())
	if !format.SoftWrap {
		return nil
	}
	return &core.VisualMoveFormat{
		ViewportWidth:    format.ViewportWidth,
		TabWidth:         format.TabWidth,
		MaxWrap:          format.MaxWrap,
		MaxIndentRetain:  format.MaxIndentRetain,
		WrapIndicatorLen: ansi.StringWidth(format.WrapIndicator),
	}
}
