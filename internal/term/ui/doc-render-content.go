package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view/language"
)

type contentRenderState struct {
	args renderContentArgs

	text   core.Rope
	sel    core.Selection
	cursor int

	cursorLines map[int]struct{}
	cursorLine  int
	anchorLine  int
	vOff        int

	nLines        int
	trailingEmpty bool

	dc      *docRenderCache
	rev     int
	rawText string
	lineIdx []lineIndexEntry
	rowMap  []viewRowEntry

	tuiStyles *tuiStyles

	diagnostics []diagnosticSpan
	annotations []inlineAnnotation

	cursorIsBlock       bool
	cursorLineEnabled   bool
	relativeLineNumbers bool
	insertMode          bool

	format   *language.TextFormat
	softWrap bool

	hOff int

	fillTUI         tui.Style
	cursorLinePriBg tui.Color
	cursorLineSecBg tui.Color
	contentX        int

	cursorColumnEnabled bool
	cursorColumnBg      tui.Color
	rulers              []int
	rulerBg             tui.Color

	gutter gutterSpec
	rr     rowRender
}

func (r *renderPass) renderContent(args renderContentArgs) {
	st := r.prepareContentRender(args)
	r.paintContentOverlays(st)
	r.renderContentRows(st)
	r.ec.cache.viewRowMaps[args.view.ID()] = st.rowMap
}

// cursor column paints first so rulers render over it
func (r *renderPass) paintContentOverlays(st *contentRenderState) {
	args := st.args
	buf := args.buf
	contentX := st.contentX
	format := st.format

	if st.cursorColumnEnabled && st.cursorLine < len(st.lineIdx)-1 {
		entry := st.lineIdx[st.cursorLine]
		next := st.lineIdx[st.cursorLine+1]
		end := next.byteStart - entry.endingLen
		cursorLStr := st.rawText[entry.byteStart:end]
		col := st.cursor - entry.charStart
		vcol := visualColOf(cursorLStr, col, format.TabWidth)
		rel := vcol - st.hOff
		if rel >= 0 && rel < format.ViewportWidth {
			sx := contentX + rel
			for row := args.area.Y; row < args.area.Y+args.area.Height; row++ {
				buf.PatchBg(geom.Point{X: sx, Y: row}, st.cursorColumnBg)
			}
		}
	}
	if len(st.rulers) > 0 {
		applyRulers(applyRulersArgs{
			buf: buf,
			at:  geom.Point{X: contentX, Y: args.area.Y},
			size: geom.Size{
				Width:  format.ViewportWidth,
				Height: args.area.Height,
			},
			horizontalOffset: st.hOff,
			rulers:           st.rulers,
			rulerBackground:  st.rulerBg,
		})
	}
}

func (r *renderPass) renderContentRows(st *contentRenderState) {
	args := st.args
	buf := args.buf
	x := args.area.X
	y := args.area.Y
	height := args.area.Height

	rr := st.rr
	gutter := st.gutter
	format := st.format
	tuiStyles := st.tuiStyles
	fillTUI := st.fillTUI
	hOff := st.hOff
	contentX := st.contentX
	text := st.text
	rawText := st.rawText
	lineIdx := st.lineIdx
	dc := st.dc
	rev := st.rev
	nLines := st.nLines
	cursorLine := st.cursorLine
	anchorLine := st.anchorLine
	vOff := st.vOff
	trailingEmpty := st.trailingEmpty
	cursorIsBlock := st.cursorIsBlock
	cursorLineEnabled := st.cursorLineEnabled
	cursorLinePriBg := st.cursorLinePriBg
	cursorLineSecBg := st.cursorLineSecBg
	relativeLineNumbers := st.relativeLineNumbers
	insertMode := st.insertMode
	softWrap := st.softWrap
	cursor := st.cursor
	cursorLines := st.cursorLines
	diagnostics := st.diagnostics
	annotations := st.annotations
	rowMap := st.rowMap

	bufRow := y
	logLine := anchorLine
	for bufRow < y+height {
		lineNum := logLine
		logLine++

		if lineNum >= nLines {
			if gutter.width > 0 {
				gutter.renderBlank(buf, geom.Point{X: x, Y: bufRow})
			}
			var blank renderedRow
			blank.writeToBuffer(rowWriteArgs{
				buf:       buf,
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
					buf, geom.Point{X: x, Y: bufRow}, lineNum == cursorLine,
				)
			}
			var row renderedRow
			if cursorIsBlock && lineNum == cursorLine {
				lineStart, err := text.LineToChar(lineNum)
				if err == nil && cursor == lineStart {
					row.write(
						" ", 1,
						tuiStyles.cursorPrim,
					)
				}
			}
			if cursorLineEnabled && lineNum == cursorLine {
				buf.PatchBgRange(
					geom.Point{X: contentX, Y: bufRow},
					format.ViewportWidth, cursorLinePriBg,
				)
			}
			row.writeToBuffer(rowWriteArgs{
				buf:       buf,
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
				if rel < 0 {
					rel = -rel
				}
				num = rel
			} else {
				num = lineNum + 1
			}
			gutter.renderLine(
				buf, geom.Point{X: x, Y: bufRow}, lineNum, num,
				isAnyCursorLine,
			)
		}

		entry, next := lineIdx[lineNum], lineIdx[lineNum+1]
		lineStart := entry.charStart
		lineContentEnd := next.charStart - entry.endingLen

		tabW := format.TabWidth

		// scan the invisible prefix without materializing it when scrolled
		renderEnd := lineContentEnd
		if !softWrap {
			bound := lineStart + hOff + format.ViewportWidth
			if bound < renderEnd {
				renderEnd = bound
			}
		}
		var lStr string
		var rowIndentCol, rowLineStart, rowColOffset int
		if !softWrap && hOff > 0 {
			prefix := dc.ensureLinePrefix(linePrefixArgs{
				rev:              rev,
				lineNum:          lineNum,
				lineStart:        lineStart,
				lineEnd:          lineContentEnd,
				tabWidth:         tabW,
				horizontalOffset: hOff,
				text:             text,
			})
			rowIndentCol = prefix.indentCol
			rowLineStart = prefix.windowPos
			rowColOffset = prefix.windowCol
			lStr = lineString(text, rowLineStart, renderEnd)
		} else {
			rowLineStart = lineStart
			if renderEnd == lineContentEnd {
				lStr = rawText[entry.byteStart : next.byteStart-entry.endingLen]
			} else {
				lStr = lineString(text, lineStart, renderEnd)
			}
			rowIndentCol = indentWidth(lStr, tabW)
		}

		// The anchor line is scrolled by vOff visual rows; skip those rows when
		// drawing so a wrapped line taller than the viewport scrolls within
		rowSkip := 0
		if softWrap && lineNum == anchorLine {
			rowSkip = vOff
		}

		rr.lineStr = lStr
		rr.indentCol = rowIndentCol
		rr.colOffset = rowColOffset
		rr.lineNum = lineNum
		rr.lineStart = rowLineStart
		rr.lineEnd = lineContentEnd
		rr.annotations = lineAnnotations(
			annotations, rowLineStart, lineContentEnd,
		)
		rr.diagnostics = lineDiagnosticSpans(
			diagnostics, rowLineStart, lineContentEnd,
		)
		rr.maxRows = y + height - bufRow + rowSkip
		contentRows := rr.rows()

		if softWrap {
			indent := indentWidth(lStr, tabW)
			prefixRow := softWrapContinuationRow(format, indent, tuiStyles)
			prefixW := runewidth.StringWidth(softWrapPrefix(format, indent))
			for i, cr := range contentRows {
				if i < rowSkip {
					continue
				}
				if bufRow >= y+height {
					break
				}
				rowPrefixW := 0
				if i > 0 && gutter.width > 0 {
					gutter.renderBlank(buf, geom.Point{X: x, Y: bufRow})
				}
				if paintCursorLine {
					buf.PatchBgRange(geom.Point{X: contentX, Y: bufRow},
						format.ViewportWidth, cursorLineBg)
				}
				if i == 0 {
					cr.writeToBuffer(rowWriteArgs{
						buf:       buf,
						at:        geom.Point{X: contentX, Y: bufRow},
						fillStyle: fillTUI,
						width:     format.ViewportWidth,
						startCol:  hOff,
					})
				} else {
					cont := prefixRow
					cont.append(cr)
					cont.writeToBuffer(rowWriteArgs{
						buf:       buf,
						at:        geom.Point{X: contentX, Y: bufRow},
						fillStyle: fillTUI,
						width:     format.ViewportWidth,
						startCol:  hOff,
					})
					rowPrefixW = prefixW
				}
				rowMap = append(rowMap, viewRowEntry{
					logLine: lineNum, offset: cr.offset, prefixW: rowPrefixW,
				})
				bufRow++
			}
		} else {
			if paintCursorLine {
				buf.PatchBgRange(geom.Point{X: contentX, Y: bufRow},
					format.ViewportWidth, cursorLineBg)
			}
			contentRows[0].writeToBuffer(rowWriteArgs{
				buf:       buf,
				at:        geom.Point{X: contentX, Y: bufRow},
				fillStyle: fillTUI,
				width:     format.ViewportWidth,
				startCol:  hOff,
			})
			rowMap = append(rowMap, viewRowEntry{logLine: lineNum})
			bufRow++
		}
	}

	st.rowMap = rowMap
}

func lineAnnotations(
	annotations []inlineAnnotation, from, to int,
) []inlineAnnotation {
	return filterLineItems(annotations,
		func(a inlineAnnotation) bool { return a.pos < from },
		func(a inlineAnnotation) bool { return a.pos > to },
	)
}

func filterLineItems[T any](items []T, before, after func(T) bool) []T {
	if len(items) == 0 {
		return nil
	}
	start := len(items)
	end := start
	for i, item := range items {
		if before(item) {
			continue
		}
		if after(item) {
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
