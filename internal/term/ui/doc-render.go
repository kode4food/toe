package ui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

func (r *renderPass) renderBufferline(buf *tui.Buffer, y int) {
	th := r.activeTheme()
	bgTUI := lipglossToTUIStyle(th.Get("ui.bufferline.background"))
	activeTUI := lipglossToTUIStyle(th.Get("ui.bufferline.active"))
	inactiveTUI := lipglossToTUIStyle(th.Get("ui.bufferline"))

	buf.SetString(0, y, strings.Repeat(" ", r.w), bgTUI)

	focusedDoc, _ := r.cx.Editor.FocusedDocument()
	docs := r.cx.Editor.AllDocuments()
	slices.SortFunc(docs, func(a, b *view.Document) int {
		return int(a.ID() - b.ID())
	})

	x := 0
	for _, doc := range docs {
		name := doc.DisplayName()
		if name == "" {
			name = "[scratch]"
		}
		mod := ""
		if doc.Modified() {
			mod = "[+]"
		}
		label := " " + name + mod + " "
		style := inactiveTUI
		if focusedDoc != nil && doc.ID() == focusedDoc.ID() {
			style = activeTUI
		}
		buf.SetString(x, y, label, style)
		x += ansi.StringWidth(label)
	}
}

func (r *renderPass) editorCursor() (tea.Cursor, bool) {
	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return tea.Cursor{}, false
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
		return tea.Cursor{}, false
	}
	opts := r.cx.Editor.Options()
	kind := opts.CursorShapeForMode(r.cx.Editor.Mode().String())
	switch kind {
	case view.CursorKindHidden:
		return tea.Cursor{}, false
	case view.CursorKindBlock:
		if r.ec.focused {
			// block cursor drawn manually in content; terminal cursor hidden
			return tea.Cursor{}, false
		}
		// terminal lost focus: use underline so position is still visible
		kind = view.CursorKindUnderline
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	cursor := sel.Primary().Cursor(text)
	g0 := opts.Gutters
	gutterW := max(lineNumberDigits(text), g0.LineNumberMinWidth()) + 1
	area := v.Area()
	if !g0.HasGutterType(view.GutterTypeLineNumbers) {
		gutterW = 0
	}
	xOff := area.X
	yOff := area.Y
	if bufferlineVisible(r.cx) {
		yOff++
	}

	rowMap := r.ec.cache.viewRowMaps[v.ID()]
	visualY, visualX := cursorScreenPos(cursorScreenPosArgs{
		text: text, cursor: cursor, gutterW: gutterW,
		rowMap: rowMap, tabW: doc.TabWidth(),
		hOff: v.Offset().HorizontalOffset,
	})
	return tea.Cursor{
		Position: tea.Position{
			X: xOff + visualX,
			Y: yOff + visualY,
		},
		Shape: cursorKindToShape(kind),
		Blink: false,
	}, true
}

type renderPaneArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	y0      int
	focused bool
}

func (r *renderPass) renderPane(args renderPaneArgs) {
	doc := args.doc
	v := args.view
	a := v.Area()
	opts := r.cx.Editor.Options()
	scrolloff := opts.ScrollOff
	contentH := max(a.Height-1, 0)
	editorX := a.X
	editorW := a.Width

	// Build the soft-wrap layout so vertical visibility is measured in visual
	// rows; nil keeps the text-line fallback when soft-wrap is off
	text := doc.Text()
	gutterW := gutterWidthFor(text, opts.Gutters)
	format := doc.TextFormatForConfig(editorW-gutterW, r.cx.Editor.Options())
	var vf *core.VisualMoveFormat
	if format.SoftWrap && gutterW < editorW {
		vf = &core.VisualMoveFormat{
			ViewportWidth:    format.ViewportWidth,
			TabWidth:         format.TabWidth,
			MaxWrap:          format.MaxWrap,
			MaxIndentRetain:  format.MaxIndentRetain,
			WrapIndicatorLen: ansi.StringWidth(format.WrapIndicator),
		}
	}
	v.EnsureCursorVisible(
		text, doc.SelectionFor(v.ID()), contentH, scrolloff, vf,
	)
	r.renderContent(renderContentArgs{
		doc:     doc,
		view:    v,
		buf:     args.buf,
		x:       editorX,
		y:       args.y0 + a.Y,
		width:   editorW,
		height:  contentH,
		focused: args.focused,
	})
	r.renderStatus(renderStatusArgs{
		doc:     doc,
		view:    v,
		buf:     args.buf,
		x:       a.X,
		y:       args.y0 + a.Y + contentH,
		width:   a.Width,
		focused: args.focused,
	})
}

func (r *renderPass) renderEditorContent(buf *tui.Buffer) {
	th := r.activeTheme()

	bgTUI := lipglossToTUIStyle(th.Get("ui.background"))
	buf.Fill(bgTUI)

	y0 := 0
	if bufferlineVisible(r.cx) {
		r.renderBufferline(buf, 0)
		y0 = 1
	}

	for _, vs := range r.cx.Editor.Tree().Views() {
		v := vs.View
		doc, ok := r.cx.Editor.Document(v.DocID())
		if !ok {
			continue
		}
		r.renderPane(renderPaneArgs{
			doc: doc, view: v, buf: buf, y0: y0, focused: vs.Focused,
		})
	}

	sepTUI := lipglossToTUIStyle(th.Get("ui.border"))
	r.cx.Editor.Tree().WalkSeparators(func(x, y, h int) {
		for row := y; row < y+h; row++ {
			buf.SetString(x, y0+row, "│", sepTUI)
		}
	})

	r.renderCmdline(buf, r.h-1)

	if r.ec.infoTitle != "" || len(r.ec.infoItems) > 0 {
		r.renderInfoOverlay(buf)
	}
}

func (r *renderPass) renderInfoOverlay(buf *tui.Buffer) {
	items := r.ec.infoItems
	title := r.ec.infoTitle
	th := r.activeTheme()

	popupSt := th.Get("ui.popup")
	popupTUI := lipglossToTUIStyle(popupSt)

	keyW := 0
	for _, item := range items {
		if w := ansi.StringWidth(item.Key); w > keyW {
			keyW = w
		}
	}
	rawLines := make([]string, len(items))
	bodyW := 0
	for i, item := range items {
		rawLines[i] = fmt.Sprintf("%-*s  %s", keyW, item.Key, item.Label)
		if w := ansi.StringWidth(rawLines[i]); w > bodyW {
			bodyW = w
		}
	}
	if tw := ansi.StringWidth(title); tw > bodyW {
		bodyW = tw
	}

	pop := popup{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  popupTUI,
		contentStyle: popupTUI,
		padX:         1,
	}
	boxW := bodyW + 2 + 2*pop.padX
	boxH := len(rawLines) + 2
	x := max(r.w-boxW, 0)
	y := max(r.h-boxH-1, 0)

	area := pop.drawInto(buf, x, y, boxW, boxH)

	if title != "" {
		buf.SetString(x+1, y, " "+title+" ", popupTUI)
	}
	for i, raw := range rawLines {
		buf.SetString(area.x, area.y+i, raw, popupTUI)
	}
}

type renderContentArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	x, y    int
	width   int
	height  int
	focused bool
}

func (dc *docRenderCache) ensureRawText(rev int, text core.Rope) string {
	if dc.rawTextRev != rev || dc.rawTextCached == "" {
		dc.rawTextRev = rev
		dc.rawTextCached = text.String()
	}
	return dc.rawTextCached
}

func (dc *docRenderCache) ensureHL(
	rev int, lang, rawText string,
) []highlight.Span {
	if lang != "text" && (dc.hlRev != rev || dc.hlLang != lang) {
		dc.hlRev = rev
		dc.hlLang = lang
		dc.hlSpans = syntax.Tokenize(
			highlight.NormalizeNewlines(rawText), lang,
		)
	}
	if lang == "text" {
		return nil
	}
	return dc.hlSpans
}

func (dc *docRenderCache) ensureSearchSpans(
	rev int, pat, rawText string,
) {
	if dc.smRev == rev && dc.smPat == pat {
		return
	}
	dc.smRev = rev
	dc.smPat = pat
	dc.smSpans = nil
	if pat == "" {
		return
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return
	}
	locs := re.FindAllStringIndex(rawText, -1)
	if len(locs) == 0 {
		return
	}
	b2r := make([]int, len(rawText)+1)
	ri := 0
	for bi := range rawText {
		b2r[bi] = ri
		ri++
	}
	b2r[len(rawText)] = ri
	for _, loc := range locs {
		from, to := b2r[loc[0]], b2r[loc[1]]
		if to > from {
			dc.smSpans = append(dc.smSpans, matchSpan{from, to})
		}
	}
}

func (r *renderPass) renderContent(args renderContentArgs) {
	doc := args.doc
	v := args.view
	buf := args.buf
	x0 := args.x
	y0 := args.y
	width := args.width
	height := args.height
	viewFocused := args.focused
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
	g3 := r.cx.Editor.Options().Gutters
	gutterMinDigits := g3.LineNumberMinWidth()
	showLineNumbers := g3.HasGutterType(view.GutterTypeLineNumbers)
	gutterW := 0
	if showLineNumbers {
		gutterW = max(lineNumberDigits(text), gutterMinDigits) + 1
	}

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
	hlSpans := dc.ensureHL(rev, lang, rawText)

	pat, hasPat := r.cx.Editor.Registers().First('/')
	if !hasPat {
		pat = ""
	}
	dc.ensureSearchSpans(rev, pat, rawText)
	searchMatches := dc.smSpans
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
	opts := r.cx.Editor.Options()
	cursorKind := opts.CursorShapeForMode(r.cx.Editor.Mode().String())
	cursorIsBlock := cursorKind == view.CursorKindBlock && r.ec.focused &&
		viewFocused
	cursorlineEnabled := opts.Cursorline
	ws := opts.Whitespace
	ig := opts.IndentGuides
	rulers := opts.Rulers

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

	lineTUI := lipglossToTUIStyle(lgStyles.line)
	lineSelTUI := lipglossToTUIStyle(lgStyles.lineSelected)
	rulerTUI := lipglossToTUIStyle(lgStyles.ruler)
	fillTUI := lipglossToTUIStyle(lgStyles.text)
	blankGutter := strings.Repeat(" ", gutterW)
	contentX := x0 + gutterW

	bufRow := y0
	logLine := anchorLine
	for bufRow < y0+height {
		lineNum := logLine
		logLine++

		if lineNum >= nLines {
			if showLineNumbers {
				buf.SetString(x0, bufRow, blankGutter, lineTUI)
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
			if showLineNumbers {
				buf.FillRange(x0, bufRow, gutterW, lineTUI)
				buf.SetString(x0+gutterW-2, bufRow, "~", lineTUI)
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

		if showLineNumbers {
			relative := opts.LineNumber == view.LineNumberRelative
			insertMode := r.cx.Editor.Mode() == view.ModeInsert
			var num int
			var gutterTUI tui.Style
			if isAnyCursorLine {
				num = lineNum + 1
				gutterTUI = lineSelTUI
			} else if relative && !insertMode {
				rel := lineNum - cursorLine
				if rel < 0 {
					rel = -rel
				}
				num = rel
				gutterTUI = lineTUI
			} else {
				num = lineNum + 1
				gutterTUI = lineTUI
			}
			buf.SetRightAlignedInt(x0, bufRow, gutterW-1, num, gutterTUI)
			buf.FillRange(x0+gutterW-1, bufRow, 1, gutterTUI)
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

		// Only the chars covering the visible window need to be materialized.
		// Each char occupies at least one visual column, so columns
		// [0, hOff+ViewportWidth) require at most that many chars; the row
		// builder stops at the window's right edge regardless
		renderEnd := lineContentEnd
		if !softWrap {
			if bound := lineStart + hOff + format.ViewportWidth; bound < renderEnd {
				renderEnd = bound
			}
		}
		lStr := lineString(text, lineStart, renderEnd)
		tabW := format.TabWidth

		var primaryCursorCols, secondaryCursorCols map[int]bool
		if opts.Cursorcolumn {
			primaryCursorCols, secondaryCursorCols = cursorCols(
				selSpans, lStr, lineStart, lineContentEnd, tabW,
			)
		}

		// The anchor line is scrolled by vOff visual rows; skip those rows when
		// drawing so a wrapped line taller than the viewport scrolls within
		rowSkip := 0
		if softWrap && lineNum == anchorLine {
			rowSkip = vOff
		}

		rr := rowRender{
			lineStr:             lStr,
			lgStyles:            lgStyles,
			tuiStyles:           tuiStyles,
			hlStyle:             hlStyleFn,
			format:              format,
			ws:                  ws,
			ig:                  ig,
			hlSpans:             hlSpans,
			searchMatches:       searchMatches,
			selSpans:            selSpans,
			primaryCursorCols:   primaryCursorCols,
			secondaryCursorCols: secondaryCursorCols,
			cursor:              cursor,
			cursorLine:          cursorLine,
			lineNum:             lineNum,
			lineStart:           lineStart,
			lineEnd:             lineContentEnd,
			softWrap:            softWrap,
			cursorlinePrim:      isPrimaryCursorLine,
			cursorlineSec:       isSecondaryCursorLine,
			cursorIsBlock:       cursorIsBlock,
			hStart:              hOff,
			hWidth:              format.ViewportWidth,
			maxRows:             y0 + height - bufRow + rowSkip,
		}
		contentRows := rr.rows()

		if softWrap {
			indent := indentWidth(lStr, tabW)
			prefixRow := softWrapContinuationRow(format, indent, lgStyles)
			prefixW := ansi.StringWidth(softWrapPrefix(format, indent))
			for i, cr := range contentRows {
				if i < rowSkip {
					continue
				}
				if bufRow >= y0+height {
					break
				}
				rowPrefixW := 0
				if i > 0 && showLineNumbers {
					buf.SetString(x0, bufRow, blankGutter, lineTUI)
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
			buf, contentX, y0, format.ViewportWidth, height, hOff,
			rulers, rulerTUI.BgColor(),
		)
	}

	c.viewRowMaps[v.ID()] = rowMap
}

func digitCount(n int) int {
	if n <= 0 {
		return 1
	}
	d := 0
	for n > 0 {
		d++
		n /= 10
	}
	return d
}

// lineNumberDigits returns the digit width for the largest line number that
// will be drawn. A trailing empty line produced by a final newline is not
// counted
func lineNumberDigits(text core.Rope) int {
	nLines := text.LenLines()
	lastDrawn := nLines
	if nLines > 0 {
		if last, err := text.Line(nLines - 1); err == nil &&
			last.LenChars() == 0 {
			lastDrawn = nLines - 1
		}
	}
	return digitCount(lastDrawn)
}

// gutterWidthFor returns the gutter width for the given line-number config, or
// 0 when line numbers are not shown
func gutterWidthFor(text core.Rope, g view.Gutter) int {
	if !g.HasGutterType(view.GutterTypeLineNumbers) {
		return 0
	}
	return max(lineNumberDigits(text), g.LineNumberMinWidth()) + 1
}

// lipglossToTUIStyle converts a lipgloss style at the per-cell boundary
type cursorScreenPosArgs struct {
	text    core.Rope
	cursor  int
	gutterW int
	rowMap  []viewRowEntry
	tabW    int
	// hOff is the view's horizontal scroll offset in content columns; the
	// cursor's content column is shifted left by it, the gutter is not
	hOff int
}

func cursorScreenPos(args cursorScreenPosArgs) (visualY, visualX int) {
	text := args.text
	cursor := args.cursor
	gutterW := args.gutterW
	cursorLine, err := text.CharToLine(cursor)
	if err != nil {
		return 0, gutterW
	}
	lineStart, err := text.LineToChar(cursorLine)
	if err != nil {
		return 0, gutterW
	}
	cursorOff := cursor - lineStart

	segY := -1
	segStart := 0
	segPrefixW := 0
	for i, e := range args.rowMap {
		if e.logLine != cursorLine {
			if segY >= 0 {
				break
			}
			continue
		}
		if cursorOff < e.offset {
			break
		}
		segY = i
		segStart = e.offset
		segPrefixW = e.prefixW
	}
	if segY < 0 {
		return 0, gutterW
	}

	lineEnd, err := text.LineEndCharIndex(cursorLine)
	if err != nil {
		return segY, gutterW + segPrefixW
	}
	col := 0
	runeIdx := 0
	for _, ch := range lineString(text, lineStart, lineEnd) {
		if runeIdx >= cursorOff {
			break
		}
		if runeIdx >= segStart {
			col += view.RuneWidth(ch, col, args.tabW)
		}
		runeIdx++
	}
	return segY, gutterW + segPrefixW + col - args.hOff
}

type charPosInLineSegArgs struct {
	text    core.Rope
	docLine int
	charOff int
	targetX int
	tabW    int
}

func charPosInLineSeg(args charPosInLineSegArgs) (int, bool) {
	text := args.text
	docLine := args.docLine
	charOff := args.charOff
	lineStart, err := text.LineToChar(docLine)
	if err != nil {
		return 0, false
	}
	lineEnd, err := text.LineEndCharIndex(docLine)
	if err != nil {
		return 0, false
	}
	col := 0
	charPos := lineStart + charOff
	runeIdx := 0
	for _, ch := range lineString(text, lineStart, lineEnd) {
		if runeIdx < charOff {
			runeIdx++
			continue
		}
		var w int
		if ch == '\t' {
			w = args.tabW - col%args.tabW
		} else {
			w = ansi.StringWidth(string(ch))
		}
		if col+w > args.targetX {
			break
		}
		col += w
		charPos++
		runeIdx++
	}
	return charPos, true
}
