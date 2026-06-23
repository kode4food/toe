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
	if ch >= view.RuneFirstPrintableASCII && ch <= view.RuneLastPrintableASCII {
		return asciiTable[ch : ch+1], 1, documentGlyphNone
	}
	tabW := r.format.TabWidth
	wsRender := r.ws.Render
	wsChars := r.ws.Characters
	guide := r.isGuideAt(col, args.indentCol, args.startGuide, args.endGuide)
	switch ch {
	case view.RuneTab:
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
	case view.RuneSpace:
		if guide {
			return string(r.ig.CharRune()), 1, documentGlyphGuide
		}
		if wsRender.SpaceRender() == view.WhitespaceRenderAll {
			return string(wsChars.SpaceRune()), 1, documentGlyphWhitespace
		}
		return " ", 1, documentGlyphNone
	case view.RuneNbsp:
		if wsRender.NbspRender() == view.WhitespaceRenderAll {
			return string(wsChars.NbspRune()), 1, documentGlyphWhitespace
		}
		return string(ch), 1, documentGlyphNone
	case view.RuneNnbsp:
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
