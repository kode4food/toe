package ui

import (
	"cmp"
	"slices"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type renderContentArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	x, y    int
	width   int
	height  int
	focused bool
}

func (r *renderPass) renderContent(args renderContentArgs) {
	doc := args.doc
	v := args.view
	buf := args.buf
	x := args.x
	y := args.y
	width := args.width
	height := args.height
	viewFocused := args.focused

	opts := r.cx.Editor.Options()
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary()
	cursor := primary.Cursor(text)

	allRanges := sel.Ranges()
	primaryIdx := sel.PrimaryIndex()
	selSpans := make([]selectionSpan, 0, len(allRanges))
	for i, rng := range allRanges {
		selSpans = append(selSpans, selectionSpan{
			from:    rng.From(),
			to:      rng.To(),
			cur:     rng.Cursor(text),
			primary: i == primaryIdx,
		})
	}

	cursorLines := make(map[int]struct{}, len(allRanges))
	for _, sp := range selSpans {
		if l, err := text.CharToLine(sp.cur); err == nil {
			cursorLines[l] = struct{}{}
		}
	}

	cursorLine := 0
	if l, err := text.CharToLine(cursor); err == nil {
		cursorLine = l
	}

	anchorLine := 0
	if anchor := v.Offset().Anchor; anchor > 0 {
		if l, err := text.CharToLine(anchor); err == nil {
			anchorLine = l
		}
	}
	// vOff is the number of visual rows scrolled into the anchor line itself,
	// so a soft-wrapped line taller than the viewport can be scrolled within
	vOff := max(v.Offset().VerticalOffset, 0)

	nLines := text.LenLines()
	g3 := opts.Gutters
	gutterLayout := g3.GutterLayout()
	gutterLineNumberW := gutterLineNumberWidth(text, g3, gutterLayout)
	gutterW := gutterLayoutWidth(gutterLayout, gutterLineNumberW)

	trailingEmpty := false
	if nLines > 0 {
		if lastLine, err := text.Line(nLines - 1); err == nil {
			trailingEmpty = lastLine.LenChars() == 0
		}
	}

	lang := doc.Lang()
	docID := doc.ID()
	rev := doc.Revision()

	c := r.ec.cache
	rowMap := c.viewRowMaps[v.ID()][:0]

	dc := c.docCaches[docID]
	if dc == nil {
		dc = &docRenderCache{}
		c.docCaches[docID] = dc
	}

	rawText := dc.ensureRawText(rev, text)
	hlSpans := dc.ensureHL(r.cx.Syntax, rev, lang, rawText)
	lineIdx := dc.ensureLineIndex(rev, rawText)

	pat, hasPat := r.cx.Editor.Registers().First('/')
	if !hasPat || !doc.SearchHighlightsActive(v.ID()) {
		pat = ""
	}
	dc.ensureSearchSpans(rev, pat, rawText)
	searchMatches := dc.smSpans
	docDiagnostics := doc.Diagnostics()
	docHighlights := documentHighlightSpans(doc.DocumentHighlights(v.ID()))
	docLinks := documentLinkSpans(doc.DocumentLinks())
	docColors := documentColorSpans(doc.DocumentColors())

	// styles rebuilt only when theme or mode changes
	th := r.activeTheme()
	stylesKey := th.Name() + "\x00" + r.cx.Editor.Mode().String()
	if c.stylesKey != stylesKey {
		c.stylesKey = stylesKey
		c.lgStyles = new(buildLipglossStyles(th, r.cx.Editor.Mode()))
		c.tuiStyles = buildTUIStyles(c.lgStyles)
		c.hlFn = hlStyleFnFor(th)
		c.hlTUICache = make(map[string]tui.Style, 64)
	}
	lgStyles := c.lgStyles
	tuiStyles := c.tuiStyles
	hlLipgloss := c.hlFn
	hlTUICache := c.hlTUICache
	hlStyleFn := func(scope string) tui.Style {
		if st, ok := hlTUICache[scope]; ok {
			return st
		}
		st := lipglossToTUIStyle(hlLipgloss(scope))
		hlTUICache[scope] = st
		return st
	}

	diagnostics := diagnosticSpans(docDiagnostics, tuiStyles)
	annotations := inlayHintAnnotations(doc.InlayHints(v.ID()), tuiStyles)
	colorAnnotations := documentColorAnnotations(doc.DocumentColors())
	annotations = append(annotations, colorAnnotations...)
	slices.SortStableFunc(annotations, func(a, b inlineAnnotation) int {
		return cmp.Compare(a.pos, b.pos)
	})

	cursorKind := opts.CursorShapeForMode(r.cx.Editor.Mode().String())
	cursorIsBlock := cursorKind == view.CursorKindBlock && r.ec.focused &&
		viewFocused
	cursorLineEnabled := opts.CursorLine
	ws := opts.Whitespace
	ig := opts.IndentGuides
	rulers := opts.Rulers
	relativeLineNumbers := opts.LineNumber == view.LineNumberRelative
	insertMode := r.cx.Editor.Mode() == view.ModeInsert

	format := doc.TextFormatForConfig(
		width-gutterW, r.cx.Editor.Options(),
	)
	softWrap := format.SoftWrap && gutterW < width
	contentW := width - gutterW
	r.cx.Editor.SetViewContentWidth(contentW)

	// Horizontal scrolling keeps the cursor visible when lines run past the
	// content area. Disabled (offset reset to 0) under soft-wrap by passing a
	// non-positive width. The gutter is fixed and never shifts
	hWidth := 0
	if !softWrap {
		hWidth = contentW
	}
	// Free scroll decouples the horizontal offset from the cursor too, but
	// soft-wrap must still reset the offset to 0
	if !v.FreeScroll() || softWrap {
		v.EnsureCursorVisibleHorizontal(
			text, sel, hWidth, format.TabWidth, opts.ScrollOff,
		)
	}
	hOff := v.Offset().HorizontalOffset

	lineTUI := lipglossToTUIStyle(lgStyles.line)
	lineSelTUI := lipglossToTUIStyle(lgStyles.lineSelected)
	rulerTUI := lipglossToTUIStyle(lgStyles.ruler)
	fillTUI := lipglossToTUIStyle(lgStyles.text)
	cursorLinePriBg := tuiStyles.cursorLinePrim.BgColor()
	cursorLineSecBg := tuiStyles.cursorLineSec.BgColor()
	contentX := x + gutterW
	gutter := gutterSpec{
		layout:          gutterLayout,
		lineNumberW:     gutterLineNumberW,
		width:           gutterLayoutWidth(gutterLayout, gutterLineNumberW),
		lineStyle:       lineTUI,
		lineSelected:    lineSelTUI,
		diagLines:       diagnosticGutterLines(text, docDiagnostics),
		diffLines:       documentDiffLines(r.cx.Editor, doc, text.LenLines()),
		severityHint:    tuiStyles.severityHint,
		severityInfo:    tuiStyles.severityInfo,
		severityWarning: tuiStyles.severityWarning,
		severityError:   tuiStyles.severityError,
		diffAdded:       tuiStyles.diffAdded,
		diffModified:    tuiStyles.diffModified,
		diffRemoved:     tuiStyles.diffRemoved,
	}

	rr := rowRender{
		lgStyles:      lgStyles,
		tuiStyles:     tuiStyles,
		hlStyle:       hlStyleFn,
		format:        format,
		ws:            ws,
		ig:            ig,
		hlSpans:       hlSpans,
		searchMatches: searchMatches,
		docHighlights: docHighlights,
		docLinks:      docLinks,
		docColors:     docColors,
		diagnostics:   diagnostics,
		selSpans:      selSpans,
		cursor:        cursor,
		cursorLine:    cursorLine,
		softWrap:      softWrap,
		cursorIsBlock: cursorIsBlock,
		mode:          r.cx.Editor.Mode(),
		hStart:        hOff,
		hWidth:        format.ViewportWidth,
	}

	// Overlay pre-passes: paint the background layers before any rows. Text is
	// drawn preserving these backgrounds, so they show through the glyphs;
	// selection and the cursor carry their own background and overwrite them.
	// CursorColumn is painted first so rulers render over it
	if opts.CursorColumn && cursorLine < len(lineIdx)-1 {
		entry := lineIdx[cursorLine]
		next := lineIdx[cursorLine+1]
		cursorLStr := rawText[entry.byteStart : next.byteStart-entry.endingLen]
		vcol := visualColOf(cursorLStr, cursor-entry.charStart, format.TabWidth)
		rel := vcol - hOff
		if rel >= 0 && rel < format.ViewportWidth {
			sx := contentX + rel
			ccBg := tuiStyles.cursorColumn.BgColor()
			for row := y; row < y+height; row++ {
				buf.PatchBg(sx, row, ccBg)
			}
		}
	}
	if len(rulers) > 0 {
		applyRulers(
			buf, contentX, y, format.ViewportWidth, height, hOff,
			rulers, rulerTUI.BgColor(),
		)
	}

	bufRow := y
	logLine := anchorLine
	for bufRow < y+height {
		lineNum := logLine
		logLine++

		if lineNum >= nLines {
			if gutter.width > 0 {
				gutter.renderBlank(buf, x, bufRow)
			}
			var blank renderedRow
			blank.writeToBuffer(rowWriteArgs{
				buf: buf, x: contentX, y: bufRow, fillStyle: fillTUI,
				width: format.ViewportWidth, startCol: hOff,
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
					buf, x, bufRow, lineNum == cursorLine,
				)
			}
			var row renderedRow
			if cursorIsBlock && lineNum == cursorLine {
				lineStart, err := text.LineToChar(lineNum)
				if err == nil && cursor == lineStart {
					row.write(
						" ", 1,
						lipglossToTUIStyle(lgStyles.cursorPrim),
					)
				}
			}
			if cursorLineEnabled && lineNum == cursorLine {
				buf.PatchBgRange(
					contentX, bufRow, format.ViewportWidth, cursorLinePriBg,
				)
			}
			row.writeToBuffer(rowWriteArgs{
				buf: buf, x: contentX, y: bufRow, fillStyle: fillTUI,
				width: format.ViewportWidth, startCol: hOff,
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
				buf, x, bufRow, lineNum, num, isAnyCursorLine,
			)
		}

		entry, next := lineIdx[lineNum], lineIdx[lineNum+1]
		lineStart := entry.charStart
		lineContentEnd := next.charStart - entry.endingLen

		tabW := format.TabWidth

		// For a horizontally scrolled (windowed) view, scan the invisible
		// prefix once without materializing it: this computes indentCol
		// (needed for indent guides) and finds the first visible char
		// position so lineStr covers only the visible window
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
				rev: rev, lineNum: lineNum, lineStart: lineStart,
				lineEnd: lineContentEnd, tabW: tabW, hOff: hOff,
				text: text,
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
			prefixRow := softWrapContinuationRow(format, indent, lgStyles)
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
					gutter.renderBlank(buf, x, bufRow)
				}
				if paintCursorLine {
					buf.PatchBgRange(
						contentX, bufRow, format.ViewportWidth, cursorLineBg,
					)
				}
				if i == 0 {
					cr.writeToBuffer(rowWriteArgs{
						buf: buf, x: contentX, y: bufRow,
						fillStyle: fillTUI, width: format.ViewportWidth,
						startCol: hOff,
					})
				} else {
					cont := prefixRow
					cont.append(cr)
					cont.writeToBuffer(rowWriteArgs{
						buf: buf, x: contentX, y: bufRow,
						fillStyle: fillTUI, width: format.ViewportWidth,
						startCol: hOff,
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
				buf.PatchBgRange(
					contentX, bufRow, format.ViewportWidth, cursorLineBg,
				)
			}
			contentRows[0].writeToBuffer(rowWriteArgs{
				buf: buf, x: contentX, y: bufRow, fillStyle: fillTUI,
				width: format.ViewportWidth, startCol: hOff,
			})
			rowMap = append(rowMap, viewRowEntry{logLine: lineNum})
			bufRow++
		}
	}

	c.viewRowMaps[v.ID()] = rowMap
}
