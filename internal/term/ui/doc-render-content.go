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

	// --- selection / cursor state ---
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

	// --- gutter ---
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

	// --- cache: raw text, highlight, search ---
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

	pat, hasPat := r.cx.Editor.Registers().First('/')
	if !hasPat {
		pat = ""
	}
	dc.ensureSearchSpans(rev, pat, rawText)
	searchMatches := dc.smSpans
	docDiagnostics := doc.Diagnostics()
	docHighlights := documentHighlightSpans(doc.DocumentHighlights(v.ID()))
	docLinks := documentLinkSpans(doc.DocumentLinks())
	docColors := documentColorSpans(doc.DocumentColors())

	// --- styles (rebuilt only on theme/mode change) ---
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

	// --- options ---
	cursorKind := opts.CursorShapeForMode(r.cx.Editor.Mode().String())
	cursorIsBlock := cursorKind == view.CursorKindBlock && r.ec.focused &&
		viewFocused
	cursorlineEnabled := opts.Cursorline
	ws := opts.Whitespace
	ig := opts.IndentGuides
	rulers := opts.Rulers
	relativeLineNumbers := opts.LineNumber == view.LineNumberRelative
	insertMode := r.cx.Editor.Mode() == view.ModeInsert

	// --- format / scroll ---
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
	v.EnsureCursorVisibleHorizontal(
		text, sel, hWidth, format.TabWidth, opts.ScrollOff,
	)
	hOff := v.Offset().HorizontalOffset

	// --- pre-converted TUI styles and layout constants ---
	lineTUI := lipglossToTUIStyle(lgStyles.line)
	lineSelTUI := lipglossToTUIStyle(lgStyles.lineSelected)
	rulerTUI := lipglossToTUIStyle(lgStyles.ruler)
	fillTUI := lipglossToTUIStyle(lgStyles.text)
	contentX := x + gutterW
	gutter := newGutterSpec(
		text, gutterLayout, gutterLineNumberW, lineTUI, lineSelTUI,
		tuiStyles, docDiagnostics,
	)

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
		hStart:        hOff,
		hWidth:        format.ViewportWidth,
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
			rowMap = append(rowMap, viewRowEntry{logLine: max(nLines-1, 0)})
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
			row.writeToBuffer(rowWriteArgs{
				buf: buf, x: contentX, y: bufRow, fillStyle: fillTUI,
				width: format.ViewportWidth, startCol: hOff,
			})
			rowMap = append(rowMap, viewRowEntry{logLine: lineNum})
			bufRow++
			continue
		}

		_, isAnyCursorLine := cursorLines[lineNum]
		isPrimaryCursorLine := cursorlineEnabled && lineNum == cursorLine
		isSecondaryCursorLine := cursorlineEnabled &&
			!isPrimaryCursorLine && isAnyCursorLine

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

		lineStart, err := text.LineToChar(lineNum)
		if err != nil {
			bufRow++
			continue
		}
		lineContentEnd, err := text.LineEndCharIndex(lineNum)
		if err != nil {
			bufRow++
			continue
		}

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
			rowIndentCol, rowLineStart, rowColOffset = scanLinePrefix(
				text, lineStart, lineContentEnd, tabW, hOff,
			)
			lStr = lineString(text, rowLineStart, renderEnd)
		} else {
			rowLineStart = lineStart
			lStr = lineString(text, lineStart, renderEnd)
			rowIndentCol = indentWidth(lStr, tabW)
		}

		var primaryCursorCols, secondaryCursorCols map[int]bool
		if opts.Cursorcolumn {
			primaryCursorCols, secondaryCursorCols = cursorCols(
				selSpans, lStr, rowLineStart, lineContentEnd,
				tabW, rowColOffset,
			)
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
		rr.primaryCursorCols = primaryCursorCols
		rr.secondaryCursorCols = secondaryCursorCols
		rr.lineNum = lineNum
		rr.lineStart = rowLineStart
		rr.lineEnd = lineContentEnd
		rr.annotations = lineAnnotations(
			annotations, rowLineStart, lineContentEnd,
		)
		rr.diagnostics = lineDiagnosticSpans(
			diagnostics, rowLineStart, lineContentEnd,
		)
		rr.cursorlinePrim = isPrimaryCursorLine
		rr.cursorlineSec = isSecondaryCursorLine
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
					lineNum, cr.offset, rowPrefixW,
				})
				bufRow++
			}
		} else {
			contentRows[0].writeToBuffer(rowWriteArgs{
				buf: buf, x: contentX, y: bufRow, fillStyle: fillTUI,
				width: format.ViewportWidth, startCol: hOff,
			})
			rowMap = append(rowMap, viewRowEntry{lineNum, 0, 0})
			bufRow++
		}
	}

	// Rulers are a background overlay drawn once over the whole content area,
	// after all rows, so they sit behind text without altering its foreground
	if len(rulers) > 0 {
		applyRulers(
			buf, contentX, y, format.ViewportWidth, height, hOff,
			rulers, rulerTUI.BgColor(),
		)
	}

	c.viewRowMaps[v.ID()] = rowMap
}
