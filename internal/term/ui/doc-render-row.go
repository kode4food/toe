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
		lineText   string
		styles     *styles
		hlStyle    func(string) tui.Style
		format     *language.TextFormat
		whitespace view.Whitespace
		indents    view.IndentGuides

		hlSpans       []highlight.Span
		searchMatches []matchSpan
		docHighlights []matchSpan
		docLinks      []matchSpan
		docColors     []colorSpan
		diagnostics   []diagnosticSpan
		annotations   []inlineAnnotation
		selSpans      []selectionSpan

		cursor     int
		cursorLine int
		lineNum    int
		lineStart  int
		lineEnd    int
		indentCol  int
		colOff     int

		softWrap      bool
		cursorIsBlock bool
		mode          view.Mode

		colStart int
		colWidth int
		maxRows  int

		cellScratch []renderedCell
		rowScratch  []renderedRow
		hlIdx       int
	}

	selectionSpan struct {
		from, to, cursor int
		primary          bool
	}

	documentGlyph uint8
)

const (
	documentGlyphNone documentGlyph = iota
	documentGlyphWhitespace
	documentGlyphGuide
)

func (r *rowRender) rows() []renderedRow {
	tabW := r.format.TabWidth
	indentCol := r.indentCol
	guides := indentGuides{
		indentCol: indentCol,
		start:     r.indents.GetSkipLevels(),
		end:       indentCol / tabW,
	}

	// A visual row holds at most ViewportWidth cells (one per column), capped
	// by the line's byte length. Pre-sizing cells avoids the geometric regrowth
	// of appending grapheme-by-grapheme from nil, the dominant per-frame alloc
	cellCap := min(len(r.lineText)+1, r.format.ViewportWidth+1)

	var row renderedRow
	if r.softWrap {
		row.cells = make([]renderedCell, 0, cellCap)
	} else {
		if cap(r.cellScratch) < cellCap {
			r.cellScratch = make([]renderedCell, 0, cellCap)
		}
		row.cells = r.cellScratch[:0]
	}
	col := r.colOff
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

	windowed := !r.softWrap && r.colWidth > 0
	hEnd := r.colStart + r.colWidth
	if windowed {
		row.colStart = r.colOff
	}

	wsRender := r.whitespace.Render
	wsChars := r.whitespace.Characters
	ts := r.styles
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
	for _, ch := range r.lineText {
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
		res := r.renderGrapheme(ch, col, guides)
		rendered := res.text
		glyph := res.glyph
		col += res.width
		selAt := r.selectionAt(pos)
		var colorStyle tui.Style
		colorOK := false
		if r.docColors != nil {
			colorStyle, colorOK = r.colorAt(pos)
		}
		var diag diagnosticSpan
		diagOK := false
		if r.diagnostics != nil {
			diag, diagOK = r.diagnosticAt(pos)
		}
		var style tui.Style
		switch {
		case selAt.cursor && selAt.primary && r.cursorIsBlock:
			style = ts.cursorPrim
		case selAt.cursor && selAt.primary && r.mode != view.ModeInsert:
			style = overlaySelStyle(styleOverlay{
				base:    r.baseStyleAt(pos, glyph),
				overlay: ts.selection,
			})
		case selAt.cursor && !selAt.primary:
			style = ts.cursor
		case selAt.selected:
			style = overlaySelStyle(styleOverlay{
				base:    r.baseStyleAt(pos, glyph),
				overlay: ts.selection,
			})
		case r.mode == view.ModeSelect:
			style = r.baseStyleAt(pos, glyph)
		case rangeMatch(r.docHighlights, pos):
			style = overlaySelStyle(styleOverlay{
				base:    r.baseStyleAt(pos, glyph),
				overlay: ts.documentHighlight,
			})

		case rangeMatch(r.docLinks, pos):
			style = overlaySelStyle(styleOverlay{
				base:    r.baseStyleAt(pos, glyph),
				overlay: ts.documentLink,
			})
		case colorOK:
			style = colorStyle
		case rangeMatch(r.searchMatches, pos):
			style = overlayBgStyle(styleOverlay{
				base:    r.baseStyleAt(pos, glyph),
				overlay: ts.searchMatch,
			})
		case glyph == documentGlyphGuide:
			style = ts.indentGuide
		case glyph == documentGlyphWhitespace:
			style = ts.whitespace
		default:
			style = r.baseStyleAt(pos, glyph)
		}
		if diagOK {
			style = overlayDiagnosticStyle(styleOverlay{
				base:    style,
				overlay: diag.style,
			})
		}
		writeRendered(rendered, res.width, style)
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
			writeRendered(" ", 1, overlaySelStyle(styleOverlay{
				base:    r.baseStyleAt(r.lineEnd, glyph),
				overlay: ts.selection,
			}))
		case selEnd.cursor && !selEnd.primary:
			writeRendered(" ", 1, ts.cursor)
		case selEnd.selected:
			writeRendered(" ", 1, overlaySelStyle(styleOverlay{
				base:    r.baseStyleAt(r.lineEnd, glyph),
				overlay: ts.selection,
			}))
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
