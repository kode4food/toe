package core

import "unicode"

type (
	// Movement controls whether a motion extends the selection or moves it
	Movement int

	// WordMotionTarget identifies the destination of a word motion
	WordMotionTarget int

	// VisualMoveFormat carries the display parameters needed to compute visual
	// row positions when soft-wrap is active. A zero-value (ViewportWidth == 0)
	// causes MoveVerticallyVisual to fall back to text-line movement
	VisualMoveFormat struct {
		ViewportWidth    int
		TabWidth         int
		MaxWrap          int
		MaxIndentRetain  int
		WrapIndicatorLen int
	}

	visualMover struct {
		format   *VisualMoveFormat
		movement Movement
	}

	visualLine struct {
		runes  []rune
		format *VisualMoveFormat
		// rowStarts holds the char offset (from line start) where each visual
		// row after the first begins; empty for a line that fits one row
		rowStarts []int
		prefixW   int
	}

	// charIter is a bidirectional rune iterator over a Rope's characters
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
		newPos = NthNextGraphemeBoundary(doc, pos, count)
	} else {
		newPos = NthPrevGraphemeBoundary(doc, pos, count)
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

func (r Range) MoveVertically(
	doc Rope, dir Direction, count int, move Movement,
) Range {
	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return r
	}
	col := cursor - lineStart

	var target int
	if dir == DirectionForward {
		target = min(line+count, doc.LenLines()-1)
	} else {
		target = max(line-count, 0)
	}

	targetStart, err := doc.LineToChar(target)
	if err != nil {
		return r
	}
	lineEnd, err := doc.LineEndCharIndex(target)
	if err != nil {
		return r.PutCursor(doc, targetStart, move == MovementExtend)
	}
	lineLen := lineEnd - targetStart
	newPos := targetStart + min(col, max(lineLen-1, 0))
	if lineLen == 0 {
		newPos = targetStart
	}
	return r.PutCursor(doc, newPos, move == MovementExtend)
}

// MoveVerticallyVisual moves r up or down by count visual rows when soft-wrap
// is active. Falls back to text-line movement when soft-wrap is disabled
func (vf *VisualMoveFormat) MoveVerticallyVisual(
	doc Rope, r Range, dir Direction, count int,
) Range {
	return visualMover{
		format:   vf,
		movement: MovementMove,
	}.moveVertically(doc, r, dir, count)
}

// ExtendVerticallyVisual extends r up or down by count visual rows when
// soft-wrap is active. Falls back to text-line movement when soft-wrap is
// disabled
func (vf *VisualMoveFormat) ExtendVerticallyVisual(
	doc Rope, r Range, dir Direction, count int,
) Range {
	return visualMover{
		format:   vf,
		movement: MovementExtend,
	}.moveVertically(doc, r, dir, count)
}

func isWordBoundary(a, b rune) bool {
	return CategorizeChar(a) != CategorizeChar(b)
}

func isLongWordBoundary(a, b rune) bool {
	ca, cb := CategorizeChar(a), CategorizeChar(b)
	switch {
	case ca == CharCategoryWord && cb == CharCategoryPunctuation:
		return false
	case ca == CharCategoryPunctuation && cb == CharCategoryWord:
		return false
	default:
		return ca != cb
	}
}

func isSubWordBoundary(a, b rune, dir Direction) bool {
	ca, cb := CategorizeChar(a), CategorizeChar(b)
	if ca == CharCategoryWord && cb == CharCategoryWord {
		if (a == '_') != (b == '_') {
			return true
		}
		if dir == DirectionForward {
			return isLower(a) && isUpper(b)
		}
		return isUpper(a) && isLower(b)
	}
	return ca != cb
}

func isWhitespaceChar(ch rune) bool {
	return CharIsWhitespace(ch) || CharIsLineEnding(ch)
}

func isUpper(ch rune) bool {
	return unicode.IsUpper(ch)
}

func isLower(ch rune) bool {
	return unicode.IsLower(ch)
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

func atWordStartPos(next rune) bool {
	return CharIsLineEnding(next) || !isWhitespaceChar(next)
}

func atWordEndPos(prev, next rune) bool {
	return !isWhitespaceChar(prev) || CharIsLineEnding(next)
}

func atSubWordStartPos(next rune) bool {
	return CharIsLineEnding(next) || !isSubWordStop(next)
}

func atSubWordEndPos(prev, next rune) bool {
	return !isSubWordStop(prev) || CharIsLineEnding(next)
}

func isSubWordStop(ch rune) bool {
	return isWhitespaceChar(ch) || ch == '_'
}
