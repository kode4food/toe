package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

func (r *rowRender) isGuideAt(col, indentCol, startGuide, endGuide int) bool {
	if !r.ig.Render || col >= indentCol {
		return false
	}
	tabW := r.format.TabWidth
	level := col / tabW
	return col%tabW == 0 && level >= startGuide && level < endGuide
}

type rowGraphemeArgs struct {
	ch         rune
	col        int
	indentCol  int
	startGuide int
	endGuide   int
}

func (r *rowRender) renderGrapheme(
	args rowGraphemeArgs,
) (string, int, documentGlyph) {
	ch := args.ch
	col := args.col
	if ch >= 0x21 && ch < 0x7F {
		return asciiTable[ch : ch+1], 1, documentGlyphNone
	}
	tabW := r.format.TabWidth
	wsRender := r.ws.Render
	wsChars := r.ws.Characters
	guide := r.isGuideAt(col, args.indentCol, args.startGuide, args.endGuide)
	switch ch {
	case '\t':
		width := tabW - col%tabW
		if guide {
			rendered := string(r.ig.CharRune()) +
				strings.Repeat(string(wsChars.TabpadRune()), width-1)
			return rendered, width, documentGlyphGuide
		}
		if wsRender.TabRender() == view.WhitespaceRenderAll {
			tabpad := strings.Repeat(string(wsChars.TabpadRune()), width-1)
			return string(wsChars.TabRune()) + tabpad,
				width, documentGlyphWhitespace
		}
		return strings.Repeat(" ", width), width, documentGlyphNone
	case ' ':
		if guide {
			return string(r.ig.CharRune()), 1, documentGlyphGuide
		}
		if wsRender.SpaceRender() == view.WhitespaceRenderAll {
			return string(wsChars.SpaceRune()), 1, documentGlyphWhitespace
		}
		return " ", 1, documentGlyphNone
	case '\xa0':
		if wsRender.NbspRender() == view.WhitespaceRenderAll {
			return string(wsChars.NbspRune()), 1, documentGlyphWhitespace
		}
		return string(ch), 1, documentGlyphNone
	case '\u202f':
		if wsRender.NnbspRender() == view.WhitespaceRenderAll {
			return string(wsChars.NnbspRune()), 1, documentGlyphWhitespace
		}
		return string(ch), 1, documentGlyphNone
	default:
		return string(ch), runewidth.RuneWidth(ch), documentGlyphNone
	}
}

func (r *rowRender) softWrapBreaks(tabW int) []int {
	if !r.softWrap {
		return nil
	}
	w := 0
	for _, ch := range r.lineStr {
		w += view.RuneWidth(ch, w, tabW)
	}
	if w <= r.format.ViewportWidth {
		return nil
	}
	vf := &core.VisualMoveFormat{
		ViewportWidth:    r.format.ViewportWidth,
		TabWidth:         r.format.TabWidth,
		MaxWrap:          r.format.MaxWrap,
		MaxIndentRetain:  r.format.MaxIndentRetain,
		WrapIndicatorLen: runewidth.StringWidth(r.format.WrapIndicator),
	}
	return vf.VisualRowStarts([]rune(r.lineStr))
}

// scanLinePrefix walks the rope from lineStart, returning indentCol (the visual
// column where indentation ends), windowPos (the char offset of the first char
// at or after column hStart), and windowCol (that char's visual column, which
// may be < hStart when a tab straddles the window boundary).
//
// For printable ASCII the inner loop uses a direct width-1 assignment instead
// of calling view.RuneWidth, so common code files need no per-char function
// call overhead in the prefix
func scanLinePrefix(
	text core.Rope, lineStart, lineEnd, tabW, hStart int,
) (indentCol, windowPos, windowCol int) {
	pos := lineStart
	col := 0
	indentDone := false
	found := false
	text.ForEachSegment(lineStart, lineEnd, func(seg string) {
		if found || col >= hStart {
			return
		}
		for _, ch := range seg {
			if !indentDone {
				switch ch {
				case '\t', ' ', '\xa0', ' ':
				default:
					indentDone = true
					indentCol = col
				}
			}
			var w int
			if uint32(ch)-0x20 < 0x5f {
				w = 1
			} else {
				w = view.RuneWidth(ch, col, tabW)
			}
			if col+w > hStart {
				found = true
				return
			}
			col += w
			pos++
		}
	})
	if !indentDone {
		indentCol = col
	}
	windowPos = pos
	windowCol = col
	return
}

func cursorCols(
	selSpans []selectionSpan, lStr string,
	lineStart, lineEnd, tabW, colStart int,
) (primary, secondary map[int]bool) {
	for _, sp := range selSpans {
		if sp.cur < lineStart || sp.cur > lineEnd {
			continue
		}
		vcol := colStart
		offset := sp.cur - lineStart
		charIdx := 0
		for _, ch := range lStr {
			if charIdx >= offset {
				break
			}
			charIdx++
			if ch == '\t' {
				vcol += tabW - vcol%tabW
			} else {
				vcol++
			}
		}
		if sp.primary {
			if primary == nil {
				primary = make(map[int]bool)
			}
			primary[vcol] = true
		} else {
			if secondary == nil {
				secondary = make(map[int]bool)
			}
			secondary[vcol] = true
		}
	}
	return
}

func indentWidth(lineStr string, tabW int) int {
	col := 0
	for _, ch := range lineStr {
		switch ch {
		case '\t':
			col += tabW - col%tabW
		case ' ', '\xa0', '\u202f':
			col++
		default:
			return col
		}
	}
	return col
}

func lineString(text core.Rope, from, to int) string {
	if from >= to {
		return ""
	}
	s, err := text.SliceString(from, to)
	if err != nil {
		return ""
	}
	return s
}

func softWrapContinuationRow(
	format *language.TextFormat, indent int, lipglossStyles *lipglossStyles,
) renderedRow {
	prefix := softWrapPrefix(format, indent)
	indentW := max(runewidth.StringWidth(prefix)-
		runewidth.StringWidth(format.WrapIndicator), 0)
	wrapW := runewidth.StringWidth(format.WrapIndicator)
	row := renderedRow{}
	if indentW > 0 {
		row.write(strings.Repeat(" ", indentW), indentW,
			lipglossToTUIStyle(lipglossStyles.text))
	}
	if wrapW > 0 {
		row.write(format.WrapIndicator, wrapW,
			lipglossToTUIStyle(lipglossStyles.whitespace))
	}
	return row
}
