package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

const (
	runeSpace rune = ' '      // U+0020 space
	runeTab   rune = '\t'     // U+0009 horizontal tab
	runeNbsp  rune = '\u00a0' // U+00A0 no-break space
	runeNnbsp rune = '\u202f' // U+202F narrow no-break space

	runeFirstPrintableASCII rune = 0x21 // '!' - first printable non-space ASCII
	runeLastPrintableASCII  rune = 0x7e // '~' - last printable ASCII
)

// indentGuides describes where indent guides are drawn on a row: guides run
// from level Start up to (not including) End, within the indent columns
type indentGuides struct {
	indentCol int
	start     int
	end       int
}

func (r *rowRender) isGuideAt(col int, guides indentGuides) bool {
	if !r.indents.Render || col >= guides.indentCol {
		return false
	}
	tabW := r.format.TabWidth
	level := col / tabW
	return col%tabW == 0 && level >= guides.start && level < guides.end
}

type renderGraphemeRes struct {
	text  string
	width int
	glyph documentGlyph
}

func (r *rowRender) renderGrapheme(
	ch rune, col int, guides indentGuides,
) renderGraphemeRes {
	if ch >= runeFirstPrintableASCII && ch <= runeLastPrintableASCII {
		return renderGraphemeRes{
			text:  tui.ASCIIString(ch),
			width: 1,
			glyph: documentGlyphNone,
		}
	}
	tabW := r.format.TabWidth
	wsRender := r.whitespace.Render
	wsChars := r.whitespace.Characters
	guide := r.isGuideAt(col, guides)
	switch ch {
	case runeTab:
		width := tabW - col%tabW
		if guide {
			rendered := string(r.indents.CharRune()) +
				strings.Repeat(string(wsChars.TabpadRune()), width-1)
			return renderGraphemeRes{
				text:  rendered,
				width: width,
				glyph: documentGlyphGuide,
			}
		}
		if wsRender.TabRender() == view.WhitespaceRenderAll {
			tabpad := strings.Repeat(string(wsChars.TabpadRune()), width-1)
			return renderGraphemeRes{
				text:  string(wsChars.TabRune()) + tabpad,
				width: width,
				glyph: documentGlyphWhitespace,
			}
		}
		return renderGraphemeRes{
			text:  strings.Repeat(" ", width),
			width: width,
			glyph: documentGlyphNone,
		}
	case runeSpace:
		if guide {
			return renderGraphemeRes{
				text:  string(r.indents.CharRune()),
				width: 1,
				glyph: documentGlyphGuide,
			}
		}
		if wsRender.SpaceRender() == view.WhitespaceRenderAll {
			return renderGraphemeRes{
				text:  string(wsChars.SpaceRune()),
				width: 1,
				glyph: documentGlyphWhitespace,
			}
		}
		return renderGraphemeRes{
			text:  " ",
			width: 1,
			glyph: documentGlyphNone,
		}
	case runeNbsp, runeNnbsp:
		if wsRender.NbspRender() == view.WhitespaceRenderAll {
			return renderGraphemeRes{
				text:  string(wsChars.NbspRune()),
				width: 1,
				glyph: documentGlyphWhitespace,
			}
		}
		return renderGraphemeRes{
			text:  string(ch),
			width: 1,
			glyph: documentGlyphNone,
		}
	default:
		return renderGraphemeRes{
			text:  string(ch),
			width: runewidth.RuneWidth(ch),
			glyph: documentGlyphNone,
		}
	}
}

func (r *rowRender) softWrapBreaks(tabW int) []int {
	if !r.softWrap {
		return nil
	}
	w := 0
	for _, ch := range r.lineText {
		w += view.RuneWidth(ch, core.TabStop{Column: w, TabWidth: tabW})
	}
	if w <= r.format.ViewportWidth {
		return nil
	}
	vf := &core.VisualMoveFormat{
		ViewportWidth:   r.format.ViewportWidth,
		TabWidth:        r.format.TabWidth,
		MaxWrap:         r.format.MaxWrap,
		MaxIndentRetain: r.format.MaxIndentRetain,
		WrapIndicatorWidth: runewidth.StringWidth(
			r.format.WrapIndicatorPrefix(),
		),
	}
	return vf.VisualRowStarts([]rune(r.lineText))
}

func softWrapPrefix(format *language.TextFormat, indent int) string {
	if indent > format.MaxIndentRetain {
		indent = 0
	}
	return strings.Repeat(" ", indent) + format.WrapIndicatorPrefix()
}

func softWrapContinuationRow(
	format *language.TextFormat, indent int, styles *styles,
) renderedRow {
	prefix := softWrapPrefix(format, indent)
	wrap := format.WrapIndicatorPrefix()
	wrapW := runewidth.StringWidth(wrap)
	indentW := max(runewidth.StringWidth(prefix)-wrapW, 0)
	row := renderedRow{}
	if indentW > 0 {
		row.write(strings.Repeat(" ", indentW), indentW, styles.text)
	}
	if wrapW > 0 {
		row.write(wrap, wrapW, styles.whitespace)
	}
	return row
}
