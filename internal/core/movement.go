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

// MoveVertically moves r up or down by count lines, preserving the horizontal
// column position as closely as possible
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

func (v visualMover) moveVertically(
	doc Rope, r Range, dir Direction, count int,
) Range {
	vf := v.format
	move := v.movement
	if vf == nil || vf.ViewportWidth <= 0 {
		return r.MoveVertically(doc, dir, count, move)
	}

	cursor := r.Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return r
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return r
	}

	vl := newVisualLine(doc, line, vf)
	// curCol is the absolute visual column (including any continuation prefix),
	// so vertical moves keep the cursor under the same on-screen column
	curRow, curCol := vl.posOf(cursor - lineStart)

	if dir == DirectionForward {
		total := vl.rowCount()
		remaining := count
		rowsBelow := total - 1 - curRow
		if remaining <= rowsBelow {
			off := vl.charAtPos(curRow+remaining, curCol)
			return r.PutCursor(doc, lineStart+off, move == MovementExtend)
		}
		remaining -= rowsBelow + 1
		nextLine := line + 1
		nLines := doc.LenLines()
		for remaining > 0 && nextLine < nLines-1 {
			tStart, err := doc.LineToChar(nextLine)
			if err != nil {
				break
			}
			tl := newVisualLine(doc, nextLine, vf)
			tRows := tl.rowCount()
			if remaining < tRows {
				off := tl.charAtPos(remaining, curCol)
				return r.PutCursor(doc, tStart+off, move == MovementExtend)
			}
			remaining -= tRows
			nextLine++
		}
		tLine := min(nextLine, nLines-1)
		tStart, err := doc.LineToChar(tLine)
		if err != nil {
			return r
		}
		tl := newVisualLine(doc, tLine, vf)
		off := tl.charAtPos(0, curCol)
		return r.PutCursor(doc, tStart+off, move == MovementExtend)
	}

	// DirectionBackward
	remaining := count
	if remaining <= curRow {
		off := vl.charAtPos(curRow-remaining, curCol)
		return r.PutCursor(doc, lineStart+off, move == MovementExtend)
	}
	remaining -= curRow + 1
	prevLine := line - 1
	for remaining > 0 && prevLine > 0 {
		tStart, err := doc.LineToChar(prevLine)
		if err != nil {
			break
		}
		tl := newVisualLine(doc, prevLine, vf)
		tRows := tl.rowCount()
		if remaining < tRows {
			off := tl.charAtPos(tRows-1-remaining, curCol)
			return r.PutCursor(doc, tStart+off, move == MovementExtend)
		}
		remaining -= tRows
		prevLine--
	}
	tLine := max(prevLine, 0)
	tStart, err := doc.LineToChar(tLine)
	if err != nil {
		return r
	}
	tl := newVisualLine(doc, tLine, vf)
	tRows := tl.rowCount()
	off := tl.charAtPos(tRows-1, curCol)
	return r.PutCursor(doc, tStart+off, move == MovementExtend)
}

// SkipWhile returns the first position at or after pos where f returns false,
// and false if all remaining characters satisfy f
func SkipWhile(doc Rope, pos int, f func(rune) bool) (int, bool) {
	runes := []rune(doc.String())
	for i := pos; i < len(runes); i++ {
		if !f(runes[i]) {
			return i, true
		}
	}
	return 0, false
}

// BackwardsSkipWhile returns the first position at or before pos where f
// returns false, and false if all characters toward the start satisfy f
func BackwardsSkipWhile(doc Rope, pos int, f func(rune) bool) (int, bool) {
	runes := []rune(doc.String())
	for i := pos - 1; i >= 0; i-- {
		if !f(runes[i]) {
			return i + 1, true
		}
	}
	return 0, false
}

func wordMove(doc Rope, r Range, count int, target WordMotionTarget) Range {
	isPrev := target == WordMotionPrevWordStart ||
		target == WordMotionPrevLongWordStart ||
		target == WordMotionPrevSubWordStart ||
		target == WordMotionPrevWordEnd ||
		target == WordMotionPrevLongWordEnd ||
		target == WordMotionPrevSubWordEnd

	n := doc.LenChars()
	if isPrev && r.Head == 0 || !isPrev && r.Head == n {
		return r
	}

	var start Range
	if isPrev {
		if r.Anchor < r.Head {
			start = NewRange(r.Head, PrevGraphemeBoundary(doc, r.Head))
		} else {
			start = NewRange(NextGraphemeBoundary(doc, r.Head), r.Head)
		}
	} else {
		if r.Anchor < r.Head {
			start = NewRange(PrevGraphemeBoundary(doc, r.Head), r.Head)
		} else {
			start = NewRange(r.Head, NextGraphemeBoundary(doc, r.Head))
		}
	}

	cur := start
	for range count {
		next := rangeToTarget(doc, cur, target)
		if next == cur {
			break
		}
		cur = next
	}
	return cur
}

// rangeToTarget extends or moves origin to reach the given word motion target
func rangeToTarget(doc Rope, origin Range, target WordMotionTarget) Range {
	isPrev := target == WordMotionPrevWordStart ||
		target == WordMotionPrevLongWordStart ||
		target == WordMotionPrevSubWordStart ||
		target == WordMotionPrevWordEnd ||
		target == WordMotionPrevLongWordEnd ||
		target == WordMotionPrevSubWordEnd

	it := newCharIterAt(doc, origin.Head)
	if isPrev {
		it.reverse()
	}

	var advance func(int) int
	if isPrev {
		advance = func(idx int) int {
			if idx > 0 {
				return idx - 1
			}
			return idx
		}
	} else {
		advance = func(idx int) int { return idx + 1 }
	}

	anchor := origin.Anchor
	head := origin.Head

	// Peek at the character before head for context
	var prevCh rune
	hasPrev := false
	if ch, ok := it.prev(); ok {
		prevCh = ch
		hasPrev = true
		_, _ = it.next()
	}

	// Skip initial newline characters
	for {
		ch, ok := it.next()
		if !ok {
			break
		}
		if !CharIsLineEnding(ch) {
			_, _ = it.prev()
			break
		}
		prevCh = ch
		hasPrev = true
		head = advance(head)
	}
	if hasPrev && CharIsLineEnding(prevCh) {
		anchor = head
	}

	// Find the target position
	headStart := head
	for {
		nextCh, ok := it.next()
		if !ok {
			break
		}
		if !hasPrev || reachedTarget(target, prevCh, nextCh) {
			if head == headStart {
				anchor = head
			} else {
				break
			}
		}
		prevCh = nextCh
		hasPrev = true
		head = advance(head)
	}

	return NewRange(anchor, head)
}

func newCharIterAt(doc Rope, charIdx int) *charIter {
	return &charIter{runes: []rune(doc.String()), pos: charIdx}
}

func (c *charIter) next() (rune, bool) {
	if c.rev {
		if c.pos == 0 {
			return 0, false
		}
		c.pos--
		return c.runes[c.pos], true
	}
	if c.pos >= len(c.runes) {
		return 0, false
	}
	ch := c.runes[c.pos]
	c.pos++
	return ch, true
}

func (c *charIter) prev() (rune, bool) {
	if c.rev {
		if c.pos >= len(c.runes) {
			return 0, false
		}
		ch := c.runes[c.pos]
		c.pos++
		return ch, true
	}
	if c.pos == 0 {
		return 0, false
	}
	c.pos--
	return c.runes[c.pos], true
}

func (c *charIter) reverse() {
	c.rev = !c.rev
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

func reachedTarget(target WordMotionTarget, prev, next rune) bool {
	switch target {
	case WordMotionNextWordStart, WordMotionPrevWordEnd:
		return isWordBoundary(prev, next) &&
			(CharIsLineEnding(next) || !isWhitespaceChar(next))
	case WordMotionNextWordEnd, WordMotionPrevWordStart:
		return isWordBoundary(prev, next) &&
			(!isWhitespaceChar(prev) || CharIsLineEnding(next))
	case WordMotionNextLongWordStart, WordMotionPrevLongWordEnd:
		return isLongWordBoundary(prev, next) &&
			(CharIsLineEnding(next) || !isWhitespaceChar(next))
	case WordMotionNextLongWordEnd, WordMotionPrevLongWordStart:
		return isLongWordBoundary(prev, next) &&
			(!isWhitespaceChar(prev) || CharIsLineEnding(next))
	case WordMotionNextSubWordStart:
		return isSubWordBoundary(prev, next, DirectionForward) &&
			(CharIsLineEnding(next) || !(isWhitespaceChar(next) || next == '_'))
	case WordMotionPrevSubWordEnd:
		return isSubWordBoundary(prev, next, DirectionBackward) &&
			(CharIsLineEnding(next) || !(isWhitespaceChar(next) || next == '_'))
	case WordMotionNextSubWordEnd:
		return isSubWordBoundary(prev, next, DirectionForward) &&
			(!(isWhitespaceChar(prev) || prev == '_') || CharIsLineEnding(next))
	case WordMotionPrevSubWordStart:
		return isSubWordBoundary(prev, next, DirectionBackward) &&
			(!(isWhitespaceChar(prev) || prev == '_') || CharIsLineEnding(next))
	default:
		return false
	}
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

func visualLineRunes(doc Rope, lineIdx int) []rune {
	lineStart, err := doc.LineToChar(lineIdx)
	if err != nil {
		return nil
	}
	lineEnd, err := doc.LineEndCharIndex(lineIdx)
	if err != nil {
		return nil
	}
	if lineEnd <= lineStart {
		return nil
	}
	sl, err := doc.Slice(lineStart, lineEnd)
	if err != nil {
		return nil
	}
	return []rune(sl.String())
}

func visualPrefixWidth(
	runes []rune, tabW, maxIndentRetain, wrapIndLen int,
) int {
	indent := visualLineIndentW(runes, tabW)
	if indent > maxIndentRetain {
		indent = 0
	}
	return indent + wrapIndLen
}

func visualLineIndentW(runes []rune, tabW int) int {
	col := 0
	for _, ch := range runes {
		switch ch {
		case '\t':
			col += TabWidthAt(col, tabW)
		case ' ':
			col++
		default:
			return col
		}
	}
	return col
}

// VisualRows returns the number of visual (soft-wrapped) rows the given text
// line occupies. It returns 1 when soft-wrap is inactive
func (vf *VisualMoveFormat) VisualRows(doc Rope, line int) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 1
	}
	return newVisualLine(doc, line, vf).rowCount()
}

// VisualRowOfOffset returns the zero-based visual row within its text line on
// which the character at charOff (relative to line start) is displayed
func (vf *VisualMoveFormat) VisualRowOfOffset(doc Rope, line, charOff int) int {
	if vf == nil || vf.ViewportWidth <= 0 {
		return 0
	}
	row, _ := newVisualLine(doc, line, vf).posOf(charOff)
	return row
}

// VisualScrollUp returns the (line, vertical offset within that line) reached
// by moving up by up visual rows from the visual position (line, row), clamped
// at the start of the document. It is the basis for both keeping the cursor in
// view and aligning a preview range
func (vf *VisualMoveFormat) VisualScrollUp(
	doc Rope, line, row, up int,
) (int, int) {
	for up > 0 {
		if row >= up {
			return line, row - up
		}
		up -= row + 1
		if line == 0 {
			return 0, 0
		}
		line--
		row = vf.VisualRows(doc, line) - 1
	}
	return line, row
}

func newVisualLine(doc Rope, line int, format *VisualMoveFormat) visualLine {
	runes := visualLineRunes(doc, line)
	v := visualLine{runes: runes, format: format}
	if format != nil {
		v.prefixW = visualPrefixWidth(
			runes, format.TabWidth, format.MaxIndentRetain,
			format.WrapIndicatorLen,
		)
		v.rowStarts = format.VisualRowStarts(runes)
	}
	return v
}

// VisualRowStarts returns the char offsets (relative to line start) at which
// each soft-wrapped visual row after the first begins for the given line runes.
// The slice is empty when the line fits on a single row. Wrapping happens at
// word boundaries: a word that does not fit is moved to the next row whole,
// unless it is wider than MaxWrap, in which case it is broken at the viewport
// edge. The leading indent is carried onto continuation rows up to
// MaxIndentRetain
func (vf *VisualMoveFormat) VisualRowStarts(runes []rune) []int {
	viewport := vf.ViewportWidth
	if viewport <= 0 || len(runes) == 0 {
		return nil
	}
	maxWrap := max(vf.MaxWrap, 0)
	tabW := vf.TabWidth
	wrapInd := vf.WrapIndicatorLen

	var starts []int
	col := 0 // visual column at which the current word begins
	indent := -1
	indentCarry := func() int {
		if indent < 0 {
			indent = 0
			return 0
		}
		if indent <= vf.MaxIndentRetain {
			return indent
		}
		return 0
	}

	i := 0
	for i < len(runes) {
		wordWidth := 0
		wordStart := i
		lastW := 0
		for {
			if col+wordWidth >= viewport {
				if wordWidth > maxWrap {
					// word too wide to move whole. If a grapheme overflowed the
					// edge it becomes the next word's start; otherwise the word
					// ends exactly at the edge and stays on this row
					if col+wordWidth > viewport {
						wordWidth -= lastW
						i--
					}
					break
				}
				// move the accumulated word to the next row
				starts = append(starts, wordStart)
				col = indentCarry()
				wordWidth += wrapInd
			}
			if i >= len(runes) {
				break
			}
			ch := runes[i]
			if indent < 0 && !charIsWrapWhitespace(ch) {
				indent = col
			}
			lastW = visualRuneW(ch, col+wordWidth, tabW)
			wordWidth += lastW
			i++
			if !charIsWord(ch) {
				break
			}
		}
		col += wordWidth
	}
	return starts
}

func (v visualLine) rowStartOffset(row int) int {
	if row <= 0 {
		return 0
	}
	if row-1 < len(v.rowStarts) {
		return v.rowStarts[row-1]
	}
	return len(v.runes)
}

func (v visualLine) rowStartCol(row int) int {
	if row <= 0 {
		return 0
	}
	return v.prefixW
}

func (v visualLine) rowCount() int {
	return len(v.rowStarts) + 1
}

// posOf returns the (row, col) visual position of charOff within the line. col
// is the absolute visual column, including the continuation prefix on wrapped
// rows
func (v visualLine) posOf(charOff int) (row, col int) {
	for row < len(v.rowStarts) && v.rowStarts[row] <= charOff {
		row++
	}
	col = v.rowStartCol(row)
	tabW := v.format.TabWidth
	for i := v.rowStartOffset(row); i < charOff && i < len(v.runes); i++ {
		col += visualRuneW(v.runes[i], col, tabW)
	}
	return
}

// charAtPos returns the char offset (relative to line start) at the absolute
// visual column targetCol on targetRow. When targetCol falls inside the
// continuation prefix or beyond the row content, the nearest char in that row
// is returned
func (v visualLine) charAtPos(targetRow, targetCol int) int {
	start := v.rowStartOffset(targetRow)
	end := v.rowStartOffset(targetRow + 1)
	col := v.rowStartCol(targetRow)
	tabW := v.format.TabWidth
	best := start
	for i := start; i < end && i < len(v.runes); i++ {
		if col > targetCol {
			break
		}
		best = i
		col += visualRuneW(v.runes[i], col, tabW)
	}
	return best
}

// charIsWord reports whether ch counts as part of a word when deciding
// soft-wrap break points. Wrapping prefers to break between words
func charIsWord(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsNumber(ch)
}

// charIsWrapWhitespace reports whether ch is whitespace for the purpose of
// detecting a line's leading indent (carried onto continuation rows)
func charIsWrapWhitespace(ch rune) bool {
	switch ch {
	case '\t', '\u0020', '\u00A0', '\u180E', '\u202F',
		'\u205F', '\u3000', '\uFEFF':
		return true
	}
	return ch >= '\u2000' && ch <= '\u200B'
}

func visualRuneW(ch rune, col, tabW int) int {
	if ch == '\t' {
		return TabWidthAt(col, tabW)
	}
	return graphemeWidth(string(ch))
}
