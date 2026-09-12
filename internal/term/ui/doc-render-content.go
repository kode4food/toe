package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type (
	contentRenderState struct {
		buf  *tui.Buffer
		area geom.Area

		stopAtEnd bool
		trackRows bool
		afterRow  func(geom.Point, viewRowEntry, bool)

		cursorLines map[int]struct{}
		anchorLine  int
		vOff        int

		lineCount     int
		trailingEmpty bool

		docCache *docRenderCache
		rawText  string
		lineIdx  []lineIndexEntry
		rowMap   []viewRowEntry

		diagnostics []diagnosticSpan
		annotations []inlineAnnotation

		cursorLineEnabled   bool
		relativeLineNumbers bool
		insertMode          bool

		fillTUI         tui.Style
		cursorLinePriBg tui.Color
		cursorLineSecBg tui.Color
		contentX        int

		cursorColumnEnabled bool
		cursorColumnBg      tui.Color
		rulers              []int
		rulerBg             tui.Color

		gutter gutterSpec
		row    rowRender
	}

	lineItemBounds[T any] struct {
		before func(T) bool
		after  func(T) bool
	}
)

func (r *renderPass) renderContent(target *contentRenderTarget) {
	st := r.prepareContentRender(target)
	r.paintContentOverlays(st)
	renderContentRows(st)
	r.editor.cache.viewRowMaps[target.view.ID()] = st.rowMap
}

func (r *renderPass) paintContentOverlays(st *contentRenderState) {
	buf := st.buf
	contentX := st.contentX
	format := st.row.format

	if st.cursorColumnEnabled && st.row.cursorLine < len(st.lineIdx)-1 {
		entry := st.lineIdx[st.row.cursorLine]
		next := st.lineIdx[st.row.cursorLine+1]
		end := next.byteStart - entry.endingWidth
		cursorLStr := st.rawText[entry.byteStart:end]
		col := st.row.cursor - entry.charStart
		vcol := visualColOf(visualColOfArgs{
			line:     cursorLStr,
			charOff:  col,
			tabWidth: format.TabWidth,
		})
		rel := vcol - st.row.colStart
		if rel >= 0 && rel < format.ViewportWidth {
			sx := contentX + rel
			for row := st.area.Y; row < st.area.Y+st.area.Height; row++ {
				buf.PatchBg(geom.Point{X: sx, Y: row}, st.cursorColumnBg)
			}
		}
	}
	if len(st.rulers) > 0 {
		applyRulers(applyRulersArgs{
			buf: buf,
			at:  geom.Point{X: contentX, Y: st.area.Y},
			size: geom.Size{
				Width:  format.ViewportWidth,
				Height: st.area.Height,
			},
			horzOff: st.row.colStart,
			rulers:  st.rulers,
			rulerBg: st.rulerBg,
		})
	}
}

func renderContentRows(st *contentRenderState) int {
	x := st.area.X
	y := st.area.Y
	height := st.area.Height

	rr := &st.row
	gutter := &st.gutter
	format := st.row.format
	styles := st.row.styles
	fillTUI := st.fillTUI
	hOff := st.row.colStart
	contentX := st.contentX
	lineIdx := st.lineIdx
	nLines := st.lineCount
	cursorLine := st.row.cursorLine
	anchorLine := st.anchorLine
	vOff := st.vOff
	trailingEmpty := st.trailingEmpty
	cursorIsBlock := st.row.cursorIsBlock
	cursorLineEnabled := st.cursorLineEnabled
	cursorLinePriBg := st.cursorLinePriBg
	cursorLineSecBg := st.cursorLineSecBg
	relativeLineNumbers := st.relativeLineNumbers
	insertMode := st.insertMode
	softWrap := st.row.softWrap
	cursor := st.row.cursor
	cursorLines := st.cursorLines
	diagnostics := st.diagnostics
	annotations := st.annotations
	rowMap := st.rowMap

	bufRow := y
	logLine := anchorLine
	for bufRow < y+height {
		lineNum := logLine
		logLine++

		if lineNum >= nLines && st.stopAtEnd {
			break
		}
		if lineNum >= nLines {
			if gutter.width > 0 {
				gutter.renderBlank(st.buf, geom.Point{X: x, Y: bufRow})
			}
			var blank renderedRow
			blank.writeToBuffer(rowWriteArgs{
				buf:       st.buf,
				at:        geom.Point{X: contentX, Y: bufRow},
				fillStyle: fillTUI,
				width:     format.ViewportWidth,
				startCol:  hOff,
			})
			rowMap = append(rowMap, viewRowEntry{
				logLine: max(nLines-1, 0),
				filler:  true,
			})
			bufRow++
			continue
		}

		if lineNum == nLines-1 && trailingEmpty {
			if gutter.width > 0 {
				gutter.renderTilde(
					st.buf, geom.Point{X: x, Y: bufRow},
					lineNum == cursorLine,
				)
			}
			var row renderedRow
			if cursorIsBlock && lineNum == cursorLine &&
				cursor == lineIdx[lineNum].charStart {
				row.write(" ", 1, styles.cursorPrim)
			}
			if cursorLineEnabled && lineNum == cursorLine {
				st.buf.PatchBgRange(
					geom.Point{X: contentX, Y: bufRow},
					format.ViewportWidth, cursorLinePriBg,
				)
			}
			row.writeToBuffer(rowWriteArgs{
				buf:       st.buf,
				at:        geom.Point{X: contentX, Y: bufRow},
				fillStyle: fillTUI,
				width:     format.ViewportWidth,
				startCol:  hOff,
			})
			rowMap = append(rowMap, viewRowEntry{logLine: lineNum})
			bufRow++
			continue
		}

		_, isAnyCursorLine := cursorLines[lineNum]
		paintCursorLine := cursorLineEnabled && isAnyCursorLine
		cursorLineBg := cursorLineSecBg
		if lineNum == cursorLine {
			cursorLineBg = cursorLinePriBg
		}

		if gutter.width > 0 {
			var num int
			if isAnyCursorLine {
				num = lineNum + 1
			} else if relativeLineNumbers && !insertMode {
				rel := lineNum - cursorLine
				num = max(rel, -rel)
			} else {
				num = lineNum + 1
			}
			gutter.renderLine(renderLineArgs{
				buffer:        st.buf,
				at:            geom.Point{X: x, Y: bufRow},
				docLine:       lineNum,
				displayNumber: num,
				selected:      isAnyCursorLine,
			})
		}

		rr.prepareLine(st.docCache, lineNum)

		// The anchor line is scrolled by vOff visual rows, so skip those when
		// drawing so a wrapped line taller than the viewport scrolls within
		rowSkip := 0
		if softWrap && lineNum == anchorLine {
			rowSkip = vOff
		}

		lineSpan := core.Span{From: rr.lineStart, To: rr.lineEnd}
		rr.annotations = lineAnnotations(annotations, lineSpan)
		rr.diagnostics = lineDiagnosticSpans(diagnostics, lineSpan)
		rr.maxRows = y + height - bufRow + rowSkip
		contentRows := rr.rows()

		var prefix renderedRow
		if softWrap {
			prefix = softWrapContinuationRow(format, rr.indentCol, styles)
		}
		first := true
		for i, cr := range contentRows {
			if i < rowSkip {
				continue
			}
			if bufRow >= y+height {
				break
			}
			at := geom.Point{X: contentX, Y: bufRow}
			row := cr
			prefixWidth := 0
			if i > 0 {
				row = prefix
				row.append(cr)
				prefixWidth = prefix.width
			}
			if i > 0 && gutter.width > 0 {
				gutter.renderBlank(st.buf, geom.Point{X: x, Y: bufRow})
			}
			if paintCursorLine {
				st.buf.PatchBgRange(at, format.ViewportWidth, cursorLineBg)
			}
			row.writeToBuffer(rowWriteArgs{
				buf:       st.buf,
				at:        at,
				fillStyle: fillTUI,
				width:     rr.colWidth,
				startCol:  hOff,
			})
			entry := viewRowEntry{
				logLine:     lineNum,
				offset:      cr.offset,
				prefixWidth: prefixWidth,
				annotations: rr.annotations,
			}
			if st.trackRows {
				rowMap = append(rowMap, entry)
			}
			if st.afterRow != nil {
				st.afterRow(at, entry, first)
			}
			first = false
			bufRow++
		}
	}

	st.rowMap = rowMap
	return bufRow - y
}

func lineAnnotations(
	annotations []inlineAnnotation, s core.Span,
) []inlineAnnotation {
	return filterLineItems(annotations, lineItemBounds[inlineAnnotation]{
		before: func(a inlineAnnotation) bool { return a.pos < s.From },
		after:  func(a inlineAnnotation) bool { return a.pos > s.To },
	})
}

func filterLineItems[T any](items []T, bounds lineItemBounds[T]) []T {
	if len(items) == 0 {
		return nil
	}
	start := len(items)
	end := start
	for i, item := range items {
		if bounds.before(item) {
			continue
		}
		if bounds.after(item) {
			break
		}
		if start == len(items) {
			start = i
		}
		end = i + 1
	}
	if start == len(items) {
		return nil
	}
	return items[start:end]
}
