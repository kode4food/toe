package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	contentRenderState struct {
		args renderContentArgs

		text   core.Rope
		sel    core.Selection
		cursor int

		cursorLines map[int]struct{}
		cursorLine  int
		anchorLine  int
		vOff        int

		lineCount     int
		trailingEmpty bool

		docCache *docRenderCache
		rev      int
		rawText  string
		lineIdx  []lineIndexEntry
		rowMap   []viewRowEntry

		styles *styles

		diagnostics []diagnosticSpan
		annotations []inlineAnnotation

		cursorIsBlock       bool
		cursorLineEnabled   bool
		relativeLineNumbers bool
		insertMode          bool

		format   *language.TextFormat
		softWrap bool

		horzOff int

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

	// lineItemBounds reports whether an item falls before the line's start or
	// after its end
	lineItemBounds[T any] struct {
		before func(T) bool
		after  func(T) bool
	}
)

func (r *renderPass) renderContent(args renderContentArgs) {
	st := r.prepareContentRender(args)
	r.paintContentOverlays(st)
	r.renderContentRows(st)
	r.editor.cache.viewRowMaps[args.view.ID()] = st.rowMap
}

func (r *renderPass) paintContentOverlays(st *contentRenderState) {
	args := st.args
	buf := args.buf
	contentX := st.contentX
	format := st.format

	if st.cursorColumnEnabled && st.cursorLine < len(st.lineIdx)-1 {
		entry := st.lineIdx[st.cursorLine]
		next := st.lineIdx[st.cursorLine+1]
		end := next.byteStart - entry.endingWidth
		cursorLStr := st.rawText[entry.byteStart:end]
		col := st.cursor - entry.charStart
		vcol := visualColOf(visualColOfArgs{
			line:     cursorLStr,
			charOff:  col,
			tabWidth: format.TabWidth,
		})
		rel := vcol - st.horzOff
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
			horzOff: st.horzOff,
			rulers:  st.rulers,
			rulerBg: st.rulerBg,
		})
	}
}

func (r *renderPass) renderContentRows(st *contentRenderState) {
	args := st.args
	x := args.area.X
	y := args.area.Y
	height := args.area.Height

	rr := st.row
	gutter := st.gutter
	format := st.format
	styles := st.styles
	fillTUI := st.fillTUI
	hOff := st.horzOff
	contentX := st.contentX
	text := st.text
	rawText := st.rawText
	lineIdx := st.lineIdx
	dc := st.docCache
	rev := st.rev
	nLines := st.lineCount
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
				gutter.renderBlank(args.buf, geom.Point{X: x, Y: bufRow})
			}
			var blank renderedRow
			blank.writeToBuffer(rowWriteArgs{
				buf:       args.buf,
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
					args.buf, geom.Point{X: x, Y: bufRow},
					lineNum == cursorLine,
				)
			}
			var row renderedRow
			if cursorIsBlock && lineNum == cursorLine {
				lineStart, err := text.LineToChar(lineNum)
				if err == nil && cursor == lineStart {
					row.write(
						" ", 1,
						styles.cursorPrim,
					)
				}
			}
			if cursorLineEnabled && lineNum == cursorLine {
				args.buf.PatchBgRange(
					geom.Point{X: contentX, Y: bufRow},
					format.ViewportWidth, cursorLinePriBg,
				)
			}
			row.writeToBuffer(rowWriteArgs{
				buf:       args.buf,
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
			gutter.renderLine(renderLineArgs{
				buffer:        args.buf,
				at:            geom.Point{X: x, Y: bufRow},
				docLine:       lineNum,
				displayNumber: num,
				selected:      isAnyCursorLine,
			})
		}

		entry, next := lineIdx[lineNum], lineIdx[lineNum+1]
		lineStart := entry.charStart
		lineContentEnd := next.charStart - entry.endingWidth

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
		var rowIndentCol, rowLineStart, rowColOff int
		if !softWrap && hOff > 0 {
			prefix := dc.ensureLinePrefix(linePrefixArgs{
				rev:       rev,
				lineNum:   lineNum,
				lineStart: lineStart,
				lineEnd:   lineContentEnd,
				tabWidth:  tabW,
				horzOff:   hOff,
				text:      text,
			})
			rowIndentCol = prefix.indentCol
			rowLineStart = prefix.windowPos
			rowColOff = prefix.windowCol
			lStr = lineString(text, core.Span{
				From: rowLineStart,
				To:   renderEnd,
			})
		} else {
			rowLineStart = lineStart
			if renderEnd == lineContentEnd {
				st := entry.byteStart
				et := next.byteStart - entry.endingWidth
				lStr = rawText[st:et]
			} else {
				lStr = lineString(text, core.Span{
					From: lineStart,
					To:   renderEnd,
				})
			}
			rowIndentCol = indentWidth(lStr, tabW)
		}

		// The anchor line is scrolled by vOff visual rows, so skip those when
		// drawing so a wrapped line taller than the viewport scrolls within
		rowSkip := 0
		if softWrap && lineNum == anchorLine {
			rowSkip = vOff
		}

		rr.lineText = lStr
		rr.indentCol = rowIndentCol
		rr.colOff = rowColOff
		rr.lineNum = lineNum
		rr.lineStart = rowLineStart
		rr.lineEnd = lineContentEnd
		lineSpan := core.Span{From: rowLineStart, To: lineContentEnd}
		rr.annotations = lineAnnotations(annotations, lineSpan)
		rr.diagnostics = lineDiagnosticSpans(diagnostics, lineSpan)
		rr.maxRows = y + height - bufRow + rowSkip
		contentRows := rr.rows()

		if softWrap {
			indent := indentWidth(lStr, tabW)
			prefixRow := softWrapContinuationRow(format, indent, styles)
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
					gutter.renderBlank(args.buf, geom.Point{X: x, Y: bufRow})
				}
				if paintCursorLine {
					args.buf.PatchBgRange(geom.Point{X: contentX, Y: bufRow},
						format.ViewportWidth, cursorLineBg)
				}
				if i == 0 {
					cr.writeToBuffer(rowWriteArgs{
						buf:       args.buf,
						at:        geom.Point{X: contentX, Y: bufRow},
						fillStyle: fillTUI,
						width:     format.ViewportWidth,
						startCol:  hOff,
					})
				} else {
					cont := prefixRow
					cont.append(cr)
					cont.writeToBuffer(rowWriteArgs{
						buf:       args.buf,
						at:        geom.Point{X: contentX, Y: bufRow},
						fillStyle: fillTUI,
						width:     format.ViewportWidth,
						startCol:  hOff,
					})
					rowPrefixW = prefixW
				}
				rowMap = append(rowMap, viewRowEntry{
					logLine:     lineNum,
					offset:      cr.offset,
					prefixWidth: rowPrefixW,
				})
				bufRow++
			}
		} else {
			if paintCursorLine {
				args.buf.PatchBgRange(geom.Point{X: contentX, Y: bufRow},
					format.ViewportWidth, cursorLineBg)
			}
			contentRows[0].writeToBuffer(rowWriteArgs{
				buf:       args.buf,
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
