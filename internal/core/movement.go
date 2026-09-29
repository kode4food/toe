package core

import "unicode"

type (
	// VerticalMove describes one vertical cursor motion. GoalColumn is the
	// column to aim for across a run of moves, in the units of the mover that
	// produced it; a zero value aims at the cursor's own column
	VerticalMove struct {
		Dir        Direction
		Count      int
		Movement   Movement
		GoalColumn int
	}

	// Movement controls whether a motion extends the selection or moves it
	Movement int

	// WordMotionTarget identifies the destination of a word motion
	WordMotionTarget int

	// VisualMoveFormat carries the display parameters needed to compute visual
	// row positions when soft-wrap is active. A zero-value (ViewportWidth == 0)
	// causes MoveVerticallyVisual to fall back to text-line movement
	VisualMoveFormat struct {
		ViewportWidth      int
		TabWidth           int
		MaxWrap            int
		MaxIndentRetain    int
		WrapIndicatorWidth int
	}

	// charStep is the character pair straddling a candidate word boundary
	charStep struct {
		prev rune
		next rune
	}

	visualMover struct {
		format *VisualMoveFormat
	}

	visualLine struct {
		runes       []rune
		format      *VisualMoveFormat
		rowStarts   []int
		prefixWidth int
	}

	charIter struct {
		runes []rune
		pos   int
		rev   bool
	}
)

const (
	MovementMove Movement = iota + 1
	MovementExtend
)

const (
	WordMotionNextWordStart WordMotionTarget = iota + 1
	WordMotionNextWordEnd
	WordMotionPrevWordStart
	WordMotionPrevWordEnd
	WordMotionNextLongWordStart
	WordMotionNextLongWordEnd
	WordMotionPrevLongWordStart
	WordMotionPrevLongWordEnd
	WordMotionNextSubWordStart
	WordMotionNextSubWordEnd
	WordMotionPrevSubWordStart
	WordMotionPrevSubWordEnd
)

// MoveHorizontally moves range by count grapheme clusters in dir
func (r Range) MoveHorizontally(
	doc Rope, dir Direction, count int, move Movement,
) Range {
	pos := r.Cursor(doc)
	var newPos int
	if dir == DirectionForward {
		newPos = NthNextGraphemeBoundary(doc, GraphemeStep{
			From:  pos,
			Count: count,
		})
	} else {
		newPos = NthPrevGraphemeBoundary(doc, GraphemeStep{
			From:  pos,
			Count: count,
		})
	}
	return r.PutCursor(doc, newPos, move == MovementExtend)
}

// MoveNextWordStart moves count words forward to the start of the next word
func MoveNextWordStart(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionNextWordStart)
}

// MoveNextWordEnd moves count words forward to the end of the next word
func MoveNextWordEnd(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionNextWordEnd)
}

// MovePrevWordStart moves count words backward to the start of the previous
// word
func MovePrevWordStart(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionPrevWordStart)
}

// MovePrevWordEnd moves count words backward to the end of the previous word
func MovePrevWordEnd(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionPrevWordEnd)
}

// MoveNextLongWordStart moves count WORDS forward to the start of the next
// WORD
func MoveNextLongWordStart(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionNextLongWordStart)
}

// MoveNextLongWordEnd moves count WORDS forward to the end of the next WORD
func MoveNextLongWordEnd(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionNextLongWordEnd)
}

// MovePrevLongWordStart moves count WORDS backward to the start of the
// previous WORD
func MovePrevLongWordStart(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionPrevLongWordStart)
}

// MovePrevLongWordEnd moves count WORDS backward to the end of the previous
// WORD
func MovePrevLongWordEnd(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionPrevLongWordEnd)
}

// MoveNextSubWordStart moves count sub-words forward to the start of the next
// sub-word
func MoveNextSubWordStart(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionNextSubWordStart)
}

// MoveNextSubWordEnd moves count sub-words forward to the end of the next
// sub-word
func MoveNextSubWordEnd(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionNextSubWordEnd)
}

// MovePrevSubWordStart moves count sub-words backward to the start of the
// previous sub-word
func MovePrevSubWordStart(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionPrevSubWordStart)
}

// MovePrevSubWordEnd moves count sub-words backward to the end of the previous
// sub-word
func MovePrevSubWordEnd(doc Rope, r Range, count int) Range {
	return wordMove(doc, r, count, WordMotionPrevSubWordEnd)
}

// MoveVertically moves the cursor by mv.Count lines, keeping the column the
// caller last moved to horizontally. It answers the goal column to carry into
// the next move, which a short line clamps the cursor to but does not forget
func (r Range) MoveVertically(doc Rope, mv VerticalMove) (Range, int) {
	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r, mv.GoalColumn
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return r, mv.GoalColumn
	}
	col := max(mv.GoalColumn, cursor-lineStart)

	var target int
	if mv.Dir == DirectionForward {
		target = min(line+mv.Count, doc.LenLines()-1)
	} else {
		target = max(line-mv.Count, 0)
	}

	targetStart, err := doc.LineToChar(target)
	if err != nil {
		return r, col
	}
	extend := mv.Movement == MovementExtend
	lineEnd, err := doc.LineEndCharIndex(target)
	if err != nil {
		return r.PutCursor(doc, targetStart, extend), col
	}
	lineLen := lineEnd - targetStart
	newPos := targetStart + min(col, max(lineLen-1, 0))
	if lineLen == 0 {
		newPos = targetStart
	}
	return r.PutCursor(doc, newPos, extend), col
}

// MoveVerticallyVisual moves r up or down by mv.Count visual rows when
// soft-wrap is active, falling back to text-line movement when it is not. It
// answers the goal column to carry into the next move
func (vf *VisualMoveFormat) MoveVerticallyVisual(
	doc Rope, r Range, mv VerticalMove,
) (Range, int) {
	return visualMover{format: vf}.moveVertically(doc, r, mv)
}

func isWordBoundary(step charStep) bool {
	return CategorizeChar(step.prev) != CategorizeChar(step.next)
}

func isLongWordBoundary(step charStep) bool {
	ca := CategorizeChar(step.prev)
	cb := CategorizeChar(step.next)
	switch {
	case ca == CharCategoryWord && cb == CharCategoryPunctuation:
		return false
	case ca == CharCategoryPunctuation && cb == CharCategoryWord:
		return false
	default:
		return ca != cb
	}
}

func isSubWordBoundary(step charStep, dir Direction) bool {
	a := step.prev
	b := step.next
	ca := CategorizeChar(a)
	cb := CategorizeChar(b)
	if ca == CharCategoryWord && cb == CharCategoryWord {
		if (a == '_') != (b == '_') {
			return true
		}
		if dir == DirectionForward {
			return unicode.IsLower(a) && unicode.IsUpper(b)
		}
		return unicode.IsUpper(a) && unicode.IsLower(b)
	}
	return ca != cb
}

func isWhitespaceChar(ch rune) bool {
	return CharIsWhitespace(ch) || CharIsLineEnding(ch)
}

func isPrevWordMotion(t WordMotionTarget) bool {
	switch t {
	case WordMotionPrevWordStart, WordMotionPrevLongWordStart,
		WordMotionPrevSubWordStart, WordMotionPrevWordEnd,
		WordMotionPrevLongWordEnd, WordMotionPrevSubWordEnd:
		return true
	default:
		return false
	}
}

func atWordStartPos(step charStep) bool {
	return CharIsLineEnding(step.next) || !isWhitespaceChar(step.next)
}

func atWordEndPos(step charStep) bool {
	return !isWhitespaceChar(step.prev) || CharIsLineEnding(step.next)
}

func atSubWordStartPos(step charStep) bool {
	return CharIsLineEnding(step.next) || !isSubWordStop(step.next)
}

func atSubWordEndPos(step charStep) bool {
	return !isSubWordStop(step.prev) || CharIsLineEnding(step.next)
}

func isSubWordStop(ch rune) bool {
	return isWhitespaceChar(ch) || ch == '_'
}
