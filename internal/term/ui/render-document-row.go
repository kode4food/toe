package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/core"
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
		softWrap            bool
		cursorlinePrim      bool
		cursorlineSec       bool
		cursorIsBlock       bool
		hStart              int
		hWidth              int
		maxRows             int
	}

	renderedRow struct {
		cells    []renderedCell
		width    int
		offset   int
		colStart int
	}

	// renderedCell stores plain text + tui.Style rather than pre-rendered ANSI,
	// keeping lipgloss.Style.Render() out of the per-rune character loop
	renderedCell struct {
		text  string
		width int
		style tui.Style
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

func cursorCols(
	selSpans []selectionSpan, lStr string, lineStart, lineEnd, tabW int,
) (primary, secondary map[int]bool) {
	for _, sp := range selSpans {
		if sp.cur < lineStart || sp.cur > lineEnd {
			continue
		}
		vcol := 0
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

func (r *rowRender) rows() []renderedRow {
	tabW := r.format.TabWidth
	indentCol := indentWidth(r.lineStr, tabW)
	endGuide := indentCol / tabW
	startGuide := r.ig.GetSkipLevels()

	var row renderedRow
	col := 0
	pos := r.lineStart

	breaks := r.softWrapBreaks(tabW)
	breakIdx := 0
	maxRows := max(r.maxRows, 1)
	rows := make([]renderedRow, 0, min(len(breaks)+1, maxRows))
	rowStart := 0
	flushRow := func(nextStart int) {
		row.offset = rowStart
		rows = append(rows, row)
		row = renderedRow{}
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
		row.colStart = r.hStart
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
		if windowed {
			w := view.RuneWidth(ch, col, tabW)
			if col+w <= r.hStart {
				col += w
				pos++
				continue
			}
			if col >= hEnd {
				break
			}
			if len(row.cells) == 0 {
				row.colStart = col
			}
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

func (r *rowRender) isGuideAt(
	col, indentCol, startGuide, endGuide int,
) bool {
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
		return string(ch), ansi.StringWidth(string(ch)), documentGlyphNone
	}
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
		WrapIndicatorLen: ansi.StringWidth(r.format.WrapIndicator),
	}
	return vf.VisualRowStarts([]rune(r.lineStr))
}

func softWrapContinuationRow(
	format *language.TextFormat, indent int, lipglossStyles *lipglossStyles,
) renderedRow {
	prefix := softWrapPrefix(format, indent)
	indentW := max(ansi.StringWidth(prefix)-
		ansi.StringWidth(format.WrapIndicator), 0)
	wrapW := ansi.StringWidth(format.WrapIndicator)
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

// writeCellsWindowed draws the visual-column window [startCol, startCol+width)
// of cells at screen [x, x+width), returning the screen x just past the last
// drawn column. Cells fully outside the window are skipped; a multi-width cell
// (tab/padding/wide rune) straddling either edge is drawn partially. startCol
// is the view's horizontal scroll offset (0 when not horizontally scrolled);
// the caller has already placed x past the fixed gutter, which never shifts
func writeCellsWindowed(
	buf *tui.Buffer, cells []renderedCell, x, y, width, startCol, cellsCol int,
) int {
	col := cellsCol
	end := startCol + width
	cx := x
	for _, c := range cells {
		if col >= end {
			break
		}
		cutOff := startCol - col
		switch {
		case cutOff <= 0 && col+c.width <= end:
			sx := x + col - startCol
			buf.SetString(sx, y, c.text, c.style)
			cx = sx + c.width
		case cutOff > 0 && cutOff < c.width:
			// straddles the left edge: the visible remainder of a tab or wide
			// rune is drawn as styled blanks
			visW := c.width - cutOff
			buf.FillRange(x, y, visW, c.style)
			cx = x + visW
		}
		// else: fully off-screen, or straddles the right edge — drawn as
		// nothing, leaving the column for the trailing fill
		col += c.width
	}
	return cx
}

type rowWriteArgs struct {
	buf       *tui.Buffer
	x, y      int
	fillStyle tui.Style
	width     int
	// startCol is the horizontal scroll offset in content columns; 0 unless the
	// view is horizontally scrolled (always 0 for soft-wrapped views)
	startCol int
}

// writeToBuffer draws the row's cells into the buffer and pads the remainder of
// the row with the fill style. Rulers are applied separately as a background
// overlay once the whole pane is drawn (see applyRulers)
func (r *renderedRow) writeToBuffer(args rowWriteArgs) {
	cx := writeCellsWindowed(
		args.buf, r.cells, args.x, args.y, args.width, args.startCol,
		r.colStart,
	)
	r.writeFillToBuffer(rowFillArgs{
		buf: args.buf, x: cx, y: args.y,
		width: max(args.x+args.width-cx, 0), style: args.fillStyle,
	})
}

// applyRulers overlays the configured ruler columns as a background highlight
// across the rows [y0, y0+height) of the content area, leaving each cell's
// glyph and foreground untouched. rulers are 1-based content columns; hOff is
// the horizontal scroll offset
func applyRulers(
	buf *tui.Buffer, contentX, y0, width, height, hOff int,
	rulers []int, rulerBg tui.Color,
) {
	for _, ruler := range rulers {
		rel := ruler - 1 - hOff
		if rel < 0 || rel >= width {
			continue
		}
		sx := contentX + rel
		for y := y0; y < y0+height; y++ {
			buf.PatchBg(sx, y, rulerBg)
		}
	}
}

type rowFillArgs struct {
	buf   *tui.Buffer
	x, y  int
	width int
	style tui.Style
}

func (r *renderedRow) writeFillToBuffer(args rowFillArgs) {
	if args.width <= 0 {
		return
	}
	args.buf.FillRange(args.x, args.y, args.width, args.style)
}

func (r *renderedRow) empty() bool {
	return len(r.cells) == 0
}

func (r *renderedRow) write(text string, width int, style tui.Style) {
	if text == "" || width <= 0 {
		return
	}
	r.cells = append(r.cells, renderedCell{
		text: text, width: width, style: style,
	})
	r.width += width
}

func (r *renderedRow) append(other renderedRow) {
	r.cells = append(r.cells, other.cells...)
	r.width += other.width
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
