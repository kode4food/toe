package ui

import (
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	rowRender struct {
		lineStr             string
		lgStyles            *lipglossStyles
		tuiStyles           *tuiStyles
		hlStyle             func(string) tui.Style
		format              *language.TextFormat
		ws                  view.Whitespace
		ig                  view.IndentGuides
		hlSpans             []highlight.Span
		searchMatches       []matchSpan
		selSpans            []selectionSpan
		primaryCursorCols   map[int]bool
		secondaryCursorCols map[int]bool
		cursor              int
		cursorLine          int
		lineNum             int
		lineStart           int
		lineEnd             int
		// indentCol is the visual column where the line's indentation ends,
		// pre-computed by the caller so rows() never needs to re-scan lineStr
		// from position 0 when lineStr has been sliced to the visible window
		indentCol      int
		colOffset      int // visual column where lineStr starts (0 unless windowed)
		softWrap       bool
		cursorlinePrim bool
		cursorlineSec  bool
		cursorIsBlock  bool
		hStart         int
		hWidth         int
		maxRows        int
	}

	selectionSpan struct {
		from, to, cur int
		primary       bool
	}

	documentGlyph uint8
)

const (
	documentGlyphNone documentGlyph = iota
	documentGlyphWhitespace
	documentGlyphGuide
)

const asciiTable = "" +
	"\x00\x01\x02\x03\x04\x05\x06\x07" +
	"\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17" +
	"\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
	" !\"#$%&'" +
	"()*+,-./" +
	"01234567" +
	"89:;<=>?" +
	"@ABCDEFG" +
	"HIJKLMNO" +
	"PQRSTUVW" +
	"XYZ[\\]^_" +
	"`abcdefg" +
	"hijklmno" +
	"pqrstuvw" +
	"xyz{|}~\x7f"

func (r *rowRender) rows() []renderedRow {
	tabW := r.format.TabWidth
	indentCol := r.indentCol
	endGuide := indentCol / tabW
	startGuide := r.ig.GetSkipLevels()

	// A visual row holds at most ViewportWidth cells (one per column), capped
	// by the line's byte length. Pre-sizing cells avoids the geometric regrowth
	// of appending grapheme-by-grapheme from nil — the dominant per-frame alloc
	cellCap := min(len(r.lineStr)+1, r.format.ViewportWidth+1)

	var row renderedRow
	row.cells = make([]renderedCell, 0, cellCap)
	col := r.colOffset
	pos := r.lineStart

	breaks := r.softWrapBreaks(tabW)
	breakIdx := 0
	maxRows := max(r.maxRows, 1)
	rows := make([]renderedRow, 0, min(len(breaks)+1, maxRows))
	rowStart := 0
	flushRow := func(nextStart int) {
		row.offset = rowStart
		rows = append(rows, row)
		row = renderedRow{cells: make([]renderedCell, 0, cellCap)}
		rowStart = nextStart
	}
	writeRendered := func(rendered string, width int, style tui.Style) {
		if r.softWrap && len(rows) >= maxRows {
			return
		}
		row.write(rendered, width, style)
	}

	windowed := !r.softWrap && r.hWidth > 0
	hEnd := r.hStart + r.hWidth
	if windowed {
		row.colStart = r.colOffset
	}

	wsRender := r.ws.Render
	wsChars := r.ws.Characters
	for _, ch := range r.lineStr {
		if r.softWrap && breakIdx < len(breaks) &&
			pos-r.lineStart == breaks[breakIdx] {
			flushRow(breaks[breakIdx])
			breakIdx++
			if len(rows) >= maxRows {
				break
			}
		}
		if windowed && col >= hEnd {
			break
		}
		colBefore := col
		rendered, width, glyph := r.renderGrapheme(rowGraphemeArgs{
			ch: ch, col: col, indentCol: indentCol,
			startGuide: startGuide, endGuide: endGuide,
		})
		col += width
		selAt := r.selectionAt(pos)
		ts := r.tuiStyles
		switch {
		case selAt.cursor && selAt.primary && r.cursorIsBlock:
			writeRendered(rendered, width, ts.cursorPrim)
		case selAt.cursor && !selAt.primary:
			writeRendered(rendered, width, ts.cursor)
		case selAt.selected && selAt.selPrimary:
			writeRendered(rendered, width, overlaySelStyle(
				r.baseStyleAt(pos, glyph), ts.selectionPrim,
			))
		case selAt.selected:
			writeRendered(rendered, width, overlaySelStyle(
				r.baseStyleAt(pos, glyph), ts.selection,
			))
		case r.searchMatch(pos):
			writeRendered(rendered, width, ts.searchMatch)
		case glyph == documentGlyphGuide:
			writeRendered(rendered, width, ts.indentGuide)
		case glyph == documentGlyphWhitespace:
			writeRendered(rendered, width, ts.whitespace)
		case r.cursorlinePrim:
			writeRendered(rendered, width, ts.cursorlinePrim)
		case r.cursorlineSec:
			writeRendered(rendered, width, ts.cursorlineSec)
		case r.primaryCursorCols[colBefore]:
			writeRendered(rendered, width, ts.cursorcolumnPrim)
		case r.secondaryCursorCols[colBefore]:
			writeRendered(rendered, width, ts.cursorcolumnSec)
		case r.hlSpans != nil:
			if scope, ok := highlight.SpanAt(r.hlSpans, pos); ok {
				writeRendered(rendered, width, r.hlStyle(scope))
			} else {
				writeRendered(rendered, width, ts.text)
			}
		default:
			writeRendered(rendered, width, ts.text)
		}
		pos++
	}

	if wsRender.NewlineRender() == view.WhitespaceRenderAll &&
		r.cursor != r.lineEnd {
		writeRendered(string(wsChars.NewlineRune()), 1,
			r.tuiStyles.whitespace)
	}
	if r.cursorIsBlock && r.cursor == r.lineEnd && r.lineNum == r.cursorLine {
		writeRendered(" ", 1, r.tuiStyles.cursorPrim)
	}
	if r.softWrap {
		if (!row.empty() || len(rows) == 0) && len(rows) < maxRows {
			flushRow(0)
		}
		return rows
	}
	return []renderedRow{row}
}

type selectionAtRes struct {
	cursor     bool
	primary    bool
	selected   bool
	selPrimary bool
}

func (r *rowRender) selectionAt(pos int) selectionAtRes {
	for _, sp := range r.selSpans {
		if pos == sp.cur {
			return selectionAtRes{cursor: true, primary: sp.primary}
		}
		if pos >= sp.from && pos < sp.to {
			return selectionAtRes{selected: true, selPrimary: sp.primary}
		}
	}
	return selectionAtRes{}
}

func (r *rowRender) searchMatch(pos int) bool {
	lo, hi := 0, len(r.searchMatches)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		sp := r.searchMatches[mid]
		if pos < sp.from {
			hi = mid - 1
		} else if pos >= sp.to {
			lo = mid + 1
		} else {
			return true
		}
	}
	return false
}

// baseStyleAt returns the syntax/glyph style that would apply to pos absent any
// selection or cursor overlay
func (r *rowRender) baseStyleAt(pos int, glyph documentGlyph) tui.Style {
	switch {
	case glyph == documentGlyphGuide:
		return r.tuiStyles.indentGuide
	case glyph == documentGlyphWhitespace:
		return r.tuiStyles.whitespace
	case r.hlSpans != nil:
		if scope, ok := highlight.SpanAt(r.hlSpans, pos); ok {
			return r.hlStyle(scope)
		}
	}
	return r.tuiStyles.text
}

// overlaySelStyle overlays the bg (and explicit fg) of sel onto base,
// preserving the syntax foreground and attributes when sel has none
func overlaySelStyle(base, sel tui.Style) tui.Style {
	if !sel.BgColor().IsReset() {
		base = base.Bg(sel.BgColor())
	}
	if !sel.FgColor().IsReset() {
		base = base.Fg(sel.FgColor())
	}
	return base
}
