package core

import "unicode"

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
	col := 0
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
			atEdge := col+wordWidth >= viewport
			tooWide := atEdge && wordWidth > maxWrap
			overflowed := tooWide && col+wordWidth > viewport
			if overflowed {
				wordWidth -= lastW
				i--
			}
			if tooWide {
				break
			}
			if atEdge {
				starts = append(starts, wordStart)
				col = indentCarry()
				wordWidth += wrapInd
			}
			if i >= len(runes) {
				break
			}
			ch := runes[i]
			if indent < 0 && !CharIsWhitespace(ch) {
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

func visualRuneW(ch rune, col, tabW int) int {
	if ch == '\t' {
		return TabWidthAt(col, tabW)
	}
	return graphemeWidth(string(ch))
}
