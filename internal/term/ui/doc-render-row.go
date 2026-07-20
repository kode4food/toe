package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	rowRender struct {
		lineStr       string
		lgStyles      *lipglossStyles
		tuiStyles     *tuiStyles
		hlStyle       func(string) tui.Style
		format        *language.TextFormat
		ws            view.Whitespace
		ig            view.IndentGuides
		hlSpans       []highlight.Span
		searchMatches []matchSpan
		docHighlights []matchSpan
		docLinks      []matchSpan
		docColors     []colorSpan
		diagnostics   []diagnosticSpan
		annotations   []inlineAnnotation
		selSpans      []selectionSpan
		cursor        int
		cursorLine    int
		lineNum       int
		lineStart     int
		lineEnd       int
		// indentCol is the visual column where the line's indentation ends,
		// pre-computed by the caller so rows() never needs to re-scan lineStr
		// from position 0 when lineStr has been sliced to the visible window
		indentCol int
		// visual column where lineStr starts (0 unless windowed)
		colOffset     int
		softWrap      bool
		cursorIsBlock bool
		mode          view.Mode
		hStart        int
		hWidth        int
		maxRows       int
		// reused across rows() calls when not soft-wrapping; the returned
		// row must be consumed before the next call
		cellScratch []renderedCell
		rowScratch  []renderedRow
		// index of the current highlight span; pos only moves forward
		hlIdx int
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
	if r.softWrap {
		row.cells = make([]renderedCell, 0, cellCap)
	} else {
		if cap(r.cellScratch) < cellCap {
			r.cellScratch = make([]renderedCell, 0, cellCap)
		}
		row.cells = r.cellScratch[:0]
	}
	col := r.colOffset
	pos := r.lineStart
	if r.hlSpans != nil {
		r.hlIdx = spanLowerBound(r.hlSpans, pos)
	}

	breaks := r.softWrapBreaks(tabW)
	breakIdx := 0
	maxRows := max(r.maxRows, 1)
	var rows []renderedRow
	if r.softWrap {
		rows = make([]renderedRow, 0, min(len(breaks)+1, maxRows))
	}
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
	ts := r.tuiStyles
	annIdx := 0
	writeAnnotations := func(pos int) {
		for annIdx < len(r.annotations) && r.annotations[annIdx].pos == pos {
			ann := r.annotations[annIdx]
			writeRendered(
				ann.text, runewidth.StringWidth(ann.text), ann.style,
			)
			annIdx++
		}
	}
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
		if r.annotations != nil {
			writeAnnotations(pos)
		}
		rendered, width, glyph := r.renderGrapheme(rowGraphemeArgs{
			char:       ch,
			col:        col,
			indentCol:  indentCol,
			startGuide: startGuide,
			endGuide:   endGuide,
		})
		col += width
		selAt := r.selectionAt(pos)
		var colorStyle tui.Style
		colorOK := false
		if r.docColors != nil {
			colorStyle, colorOK = r.colorAt(pos)
		}
		var diagStyle tui.Style
		diagOK := false
		if r.diagnostics != nil {
			diagStyle, diagOK = r.diagnosticAt(pos)
		}
		switch {
		case selAt.cursor && selAt.primary && r.cursorIsBlock:
			writeRendered(rendered, width, ts.cursorPrim)
		case selAt.cursor && selAt.primary && r.mode != view.ModeInsert:
			writeRendered(rendered, width, overlaySelStyle(
				r.baseStyleAt(pos, glyph), ts.selection,
			))
		case selAt.cursor && !selAt.primary:
			writeRendered(rendered, width, ts.cursor)
		case selAt.selected:
			writeRendered(rendered, width, overlaySelStyle(
				r.baseStyleAt(pos, glyph), ts.selection,
			))
		case r.mode == view.ModeSelect:
			writeRendered(rendered, width, r.baseStyleAt(pos, glyph))
		case rangeMatch(r.docHighlights, pos):
			writeRendered(rendered, width, overlaySelStyle(
				r.baseStyleAt(pos, glyph), ts.documentHighlight,
			))
		case rangeMatch(r.docLinks, pos):
			writeRendered(rendered, width, overlaySelStyle(
				r.baseStyleAt(pos, glyph), ts.documentLink,
			))
		case colorOK:
			writeRendered(rendered, width, colorStyle)
		case rangeMatch(r.searchMatches, pos):
			writeRendered(rendered, width, overlayBgStyle(
				r.baseStyleAt(pos, glyph), ts.searchMatch,
			))
		case diagOK:
			writeRendered(rendered, width, overlayDiagnosticStyle(
				r.baseStyleAt(pos, glyph), diagStyle,
			))
		case glyph == documentGlyphGuide:
			writeRendered(rendered, width, ts.indentGuide)
		case glyph == documentGlyphWhitespace:
			writeRendered(rendered, width, ts.whitespace)
		case r.hlSpans != nil:
			if scope, ok := r.hlScopeAt(pos); ok {
				writeRendered(rendered, width, r.hlStyle(scope))
			} else {
				writeRendered(rendered, width, ts.text)
			}
		default:
			writeRendered(rendered, width, ts.text)
		}
		pos++
	}
	if r.annotations != nil {
		writeAnnotations(pos)
	}

	selEnd := r.selectionAt(r.lineEnd)
	nlWhitespace := wsRender.NewlineRender() == view.WhitespaceRenderAll
	drawEnd := selEnd.selected || nlWhitespace ||
		(selEnd.cursor && (r.mode != view.ModeInsert || !selEnd.primary))
	if drawEnd && !(windowed && col >= hEnd) {
		glyph := documentGlyphNone
		if nlWhitespace {
			glyph = documentGlyphWhitespace
		}
		switch {
		case selEnd.cursor && selEnd.primary && r.cursorIsBlock:
			writeRendered(" ", 1, ts.cursorPrim)
		case selEnd.cursor && selEnd.primary && r.mode != view.ModeInsert:
			writeRendered(" ", 1, overlaySelStyle(
				r.baseStyleAt(r.lineEnd, glyph), ts.selection,
			))
		case selEnd.cursor && !selEnd.primary:
			writeRendered(" ", 1, ts.cursor)
		case selEnd.selected:
			writeRendered(" ", 1, overlaySelStyle(
				r.baseStyleAt(r.lineEnd, glyph), ts.selection,
			))
		default:
			writeRendered(string(wsChars.NewlineRune()), 1, ts.whitespace)
		}
	}
	if r.softWrap {
		if (!row.empty() || len(rows) == 0) && len(rows) < maxRows {
			flushRow(0)
		}
		return rows
	}
	r.cellScratch = row.cells[:0]
	r.rowScratch = append(r.rowScratch[:0], row)
	return r.rowScratch
}
