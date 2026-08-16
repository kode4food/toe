package core

import (
	"unicode"

	"github.com/kode4food/toe/internal/geom"
)

// VisualRowStarts returns the char offsets at which each soft-wrapped visual
// row after the first begins; empty if the line fits on one row. Wraps at
// word boundaries, breaking mid-word only past MaxWrap
func (vf *VisualMoveFormat) VisualRowStarts(runes []rune) []int {
	viewport := vf.ViewportWidth
	if viewport <= 0 || len(runes) == 0 {
		return nil
	}
	maxWrap := max(vf.MaxWrap, 0)
	tabW := vf.TabWidth
	wrapInd := vf.WrapIndicatorWidth

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
			lastW = visualRuneW(ch, TabStop{
				Column:   col + wordWidth,
				TabWidth: tabW,
			})
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

func newVisualLine(doc Rope, line int, format *VisualMoveFormat) visualLine {
	runes := visualLineRunes(doc, line)
	v := visualLine{runes: runes, format: format}
	if format != nil {
		v.prefixWidth = visualPrefixWidth(runes, format)
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
	return v.prefixWidth
}

func (v visualLine) rowCount() int {
	return len(v.rowStarts) + 1
}

func (v visualLine) posOf(charOff int) geom.Point {
	var at geom.Point
	for at.Y < len(v.rowStarts) && v.rowStarts[at.Y] <= charOff {
		at.Y++
	}
	at.X = v.rowStartCol(at.Y)
	tabW := v.format.TabWidth
	for i := v.rowStartOffset(at.Y); i < charOff && i < len(v.runes); i++ {
		at.X += visualRuneW(v.runes[i], TabStop{
			Column:   at.X,
			TabWidth: tabW,
		})
	}
	return at
}

func (v visualLine) charAtPos(at geom.Point) int {
	start := v.rowStartOffset(at.Y)
	end := v.rowStartOffset(at.Y + 1)
	col := v.rowStartCol(at.Y)
	tabW := v.format.TabWidth
	best := start
	for i := start; i < end && i < len(v.runes); i++ {
		if col > at.X {
			break
		}
		best = i
		col += visualRuneW(v.runes[i], TabStop{
			Column:   col,
			TabWidth: tabW,
		})
	}
	return best
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
	if sl, err := doc.Slice(Span{From: lineStart, To: lineEnd}); err == nil {
		return []rune(sl.String())
	}
	return nil
}

func visualPrefixWidth(runes []rune, vf *VisualMoveFormat) int {
	indent := visualLineIndentW(runes, vf.TabWidth)
	if indent > vf.MaxIndentRetain {
		indent = 0
	}
	return indent + vf.WrapIndicatorWidth
}

func visualLineIndentW(runes []rune, tabW int) int {
	col := 0
	for _, ch := range runes {
		switch ch {
		case '\t':
			col += TabWidthAt(TabStop{Column: col, TabWidth: tabW})
		case ' ':
			col++
		default:
			return col
		}
	}
	return col
}

func charIsWord(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsNumber(ch)
}

func visualRuneW(ch rune, at TabStop) int {
	if ch == '\t' {
		return TabWidthAt(at)
	}
	return graphemeWidth(string(ch))
}
