package ui

import (
	"fmt"
	"image/color"
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
	act "github.com/kode4food/toe/internal/view/action"
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
		// hStart/hWidth define the visible horizontal window in content columns
		// for non-wrapped lines. Graphemes outside it are not built, so a long
		// line only allocates cells for the visible slice. hWidth <= 0 disables
		// windowing (the whole line is built, e.g. under soft-wrap)
		hStart int
		hWidth int
		// maxRows bounds how many soft-wrapped visual rows to build for the
		// line, so it never builds more rows than the viewport can show
		maxRows int
	}

	renderedRow struct {
		cells  []renderedCell
		width  int
		offset int
		// colStart is the absolute content column of the first cell, non-zero
		// only when the build was horizontally windowed
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

func (r *renderPass) screenCharPos(
	doc *view.Document, v *view.View, x, contentY int,
) (int, bool) {
	a := v.Area()
	localY := contentY - a.Y
	if localY < 0 {
		return 0, false
	}
	rowMap := r.ec.cache.viewRowMaps[v.ID()]
	if len(rowMap) == 0 {
		return 0, false
	}
	if localY >= len(rowMap) {
		localY = len(rowMap) - 1
	}
	entry := rowMap[localY]

	text := doc.Text()
	g2 := r.cx.Editor.Options().Gutters
	gutterW := 0
	if g2.HasGutterType(view.GutterTypeLineNumbers) {
		gutterW = max(lineNumberDigits(text), g2.LineNumberMinWidth()) + 1
	}
	// Add the horizontal scroll offset: screen column 0 of the content maps to
	// content column hOff. The gutter is fixed and excluded from the offset
	contentX := max(x-a.X-gutterW-entry.prefixW, 0) +
		v.Offset().HorizontalOffset
	return charPosInLineSeg(charPosInLineSegArgs{
		text: text, docLine: entry.logLine, charOff: entry.offset,
		targetX: contentX, tabW: doc.TabWidth(),
	})
}

// contentViewAt returns the view whose editor content area contains screen
// point (x, y), excluding the pane's status row. Cursor positioning uses this
// so a click on the status or command line is ignored
func (r *renderPass) contentViewAt(x, y int) (*view.View, bool) {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	contentY := y - yOff
	if contentY < 0 {
		return nil, false
	}
	for _, vs := range r.cx.Editor.Tree().Views() {
		a := vs.View.Area()
		contentH := max(a.Height-1, 0)
		if x >= a.X && x < a.X+a.Width &&
			contentY >= a.Y && contentY < a.Y+contentH {
			return vs.View, true
		}
	}
	return nil, false
}

func (r *renderPass) handleMouseClick(x, y int, mod tea.KeyMod) {
	// A click outside any editor content area (status line, command line, or a
	// gap) must not move the cursor
	res, ok := r.resolveClickPos(x, y)
	if !ok {
		return
	}

	text := res.doc.Text()
	prevSel := res.doc.SelectionFor(res.v.ID())
	r.ec.mouseDownRange = new(prevSel.Primary())

	var newSel core.Selection
	switch {
	case mod&tea.ModAlt != 0:
		newSel = prevSel.Push(core.PointRange(res.pos))
	case r.cx.Editor.Mode() == view.ModeSelect:
		// In select mode a click extends the primary selection rather than
		// collapsing it, discarding any secondary selections
		primary := prevSel.Primary().PutCursor(text, res.pos, true)
		if s, err := core.NewSelection([]core.Range{primary}, 0); err == nil {
			newSel = s
		} else {
			newSel = core.PointSelection(res.pos)
		}
	default:
		newSel = core.PointSelection(res.pos)
	}
	tx := core.NewTransaction(text).WithSelection(newSel)
	_ = r.cx.Editor.Apply(tx)
}

func (r *renderPass) handleMouseDrag(x, y int) {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	contentY := y - yOff
	if contentY < 0 {
		return
	}

	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
		return
	}

	pos, ok := r.screenCharPos(doc, v, x, contentY)
	if !ok {
		return
	}

	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary().PutCursor(text, pos, true)
	newSel, err := sel.Replace(sel.PrimaryIndex(), primary)
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithSelection(newSel)
	_ = r.cx.Editor.Apply(tx)
}

func (r *renderPass) handleMouseMiddleRelease(x, y int, mod tea.KeyMod) {
	if mod&tea.ModAlt != 0 {
		act.PrimaryClipboardReplace(r.cx.Editor)
		return
	}

	res, ok := r.resolveClickPos(x, y)
	if !ok {
		return
	}
	text := res.doc.Text()
	tx := core.NewTransaction(text).WithSelection(core.PointSelection(res.pos))
	_ = r.cx.Editor.Apply(tx)
	act.PastePrimaryClipboardBefore(r.cx.Editor)
}

type resolveClickPosRes struct {
	doc *view.Document
	v   *view.View
	pos int
}

func (r *renderPass) resolveClickPos(x, y int) (resolveClickPosRes, bool) {
	v, ok := r.contentViewAt(x, y)
	if !ok {
		return resolveClickPosRes{}, false
	}
	r.cx.Editor.FocusView(v.ID())
	doc, ok := r.cx.Editor.Document(v.DocID())
	if !ok {
		return resolveClickPosRes{}, false
	}
	contentY := y
	if bufferlineVisible(r.cx) {
		contentY--
	}
	pos, ok := r.screenCharPos(doc, v, x, contentY)
	if !ok {
		return resolveClickPosRes{}, false
	}
	return resolveClickPosRes{doc: doc, v: v, pos: pos}, true
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

	if dc.rawTextRev != rev || dc.rawTextCached == "" {
		dc.rawTextRev = rev
		dc.rawTextCached = text.String()
	}
	rawText := dc.rawTextCached

	if lang != "text" && (dc.hlRev != rev || dc.hlLang != lang) {
		dc.hlRev = rev
		dc.hlLang = lang
		dc.hlSpans = syntax.Tokenize(
			highlight.NormalizeNewlines(rawText), lang)
	}
	hlSpans := dc.hlSpans
	if lang == "text" {
		hlSpans = nil
	}

	pat, hasPat := r.cx.Editor.Registers().First('/')
	if !hasPat {
		pat = ""
	}
	if dc.smRev != rev || dc.smPat != pat {
		dc.smRev = rev
		dc.smPat = pat
		dc.smSpans = nil
		if pat != "" {
			if re, err := regexp.Compile(pat); err == nil {
				locs := re.FindAllStringIndex(rawText, -1)
				if len(locs) > 0 {
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
							dc.smSpans = append(
								dc.smSpans, matchSpan{from, to},
							)
						}
					}
				}
			}
		}
	}
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
			for _, sp := range selSpans {
				if sp.cur < lineStart || sp.cur > lineContentEnd {
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
					if primaryCursorCols == nil {
						primaryCursorCols = make(map[int]bool)
					}
					primaryCursorCols[vcol] = true
				} else {
					if secondaryCursorCols == nil {
						secondaryCursorCols = make(map[int]bool)
					}
					secondaryCursorCols[vcol] = true
				}
			}
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
				if i == 0 {
					cr.writeToBuffer(rowWriteArgs{
						buf: buf, x: contentX, y: bufRow,
						fillStyle: fillTUI, width: format.ViewportWidth,
						startCol: hOff,
					})
				} else {
					if showLineNumbers {
						buf.SetString(
							x0, bufRow, blankGutter, lineTUI,
						)
					}
					cont := prefixRow
					cont.append(cr)
					cont.writeToBuffer(rowWriteArgs{
						buf: buf, x: contentX, y: bufRow,
						fillStyle: fillTUI, width: format.ViewportWidth,
						startCol: hOff,
					})
					rowPrefixW = prefixW
				}
				rowMap = append(rowMap, viewRowEntry{lineNum, cr.offset, rowPrefixW})
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

func (r *rowRender) rows() []renderedRow {
	tabW := r.format.TabWidth
	indentCol := indentWidth(r.lineStr, tabW)
	endGuide := indentCol / tabW
	startGuide := r.ig.GetSkipLevels()

	var row renderedRow
	col := 0
	pos := r.lineStart

	// Soft-wrap break points are computed once per line by the shared visual
	// formatter so rendering wraps at exactly the same offsets that cursor
	// movement and scrolling use. Each break offset is where a new visual row
	// begins. maxRows bounds the build to what the viewport can show
	var breaks []int
	breakIdx := 0
	maxRows := max(r.maxRows, 1)
	if r.softWrap {
		// A line only wraps when its width exceeds the viewport. Sum the width
		// with an alloc-free scan first so the common short line skips the
		// rune-slice conversion and break computation entirely
		w := 0
		for _, ch := range r.lineStr {
			w += view.RuneWidth(ch, w, tabW)
		}
		if w > r.format.ViewportWidth {
			vf := &core.VisualMoveFormat{
				ViewportWidth:    r.format.ViewportWidth,
				TabWidth:         r.format.TabWidth,
				MaxWrap:          r.format.MaxWrap,
				MaxIndentRetain:  r.format.MaxIndentRetain,
				WrapIndicatorLen: ansi.StringWidth(r.format.WrapIndicator),
			}
			breaks = vf.VisualRowStarts([]rune(r.lineStr))
		}
	}
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

	// windowed builds only the visible horizontal slice of a non-wrapped line,
	// skipping the per-grapheme work for off-screen columns so a long line does
	// not allocate a cell per character
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
	guide := r.ig.Render && col < args.indentCol &&
		col%tabW == 0 && col/tabW >= args.startGuide &&
		col/tabW < args.endGuide
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
	case ' ':
		if wsRender.NbspRender() == view.WhitespaceRenderAll {
			return string(wsChars.NbspRune()), 1, documentGlyphWhitespace
		}
		return string(ch), 1, documentGlyphNone
	case ' ':
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
		case ' ', ' ', ' ':
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
	r.cells = append(r.cells, renderedCell{text: text, width: width, style: style})
	r.width += width
}

func (r *renderedRow) append(other renderedRow) {
	r.cells = append(r.cells, other.cells...)
	r.width += other.width
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
func lipglossToTUIStyle(s lipgloss.Style) tui.Style {
	st := tui.Style{}
	st = st.Fg(lipglossColorToTUI(s.GetForeground()))
	st = st.Bg(lipglossColorToTUI(s.GetBackground()))
	st = st.UlColor(lipglossColorToTUI(s.GetUnderlineColor()))
	st = st.UlStyle(lipglossUnderlineToTUI(s.GetUnderlineStyle()))
	var m tui.Modifier
	if s.GetBold() {
		m |= tui.ModifierBold
	}
	if s.GetFaint() {
		m |= tui.ModifierDim
	}
	if s.GetItalic() {
		m |= tui.ModifierItalic
	}
	if s.GetBlink() {
		m |= tui.ModifierSlowBlink
	}
	if s.GetReverse() {
		m |= tui.ModifierReversed
	}
	if s.GetStrikethrough() {
		m |= tui.ModifierCrossedOut
	}
	if m != 0 {
		st = st.Mod(m)
	}
	return st
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

func lipglossColorToTUI(c color.Color) tui.Color {
	if c == nil {
		return tui.ColorReset
	}
	switch v := c.(type) {
	case lipgloss.NoColor:
		return tui.ColorReset
	case ansi.BasicColor:
		return basicTUIColor(uint8(v))
	case ansi.IndexedColor:
		return tui.ColorIndexed(uint8(v))
	default:
		r, g, b, _ := c.RGBA()
		return tui.ColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
}

func basicTUIColor(idx uint8) tui.Color {
	switch idx {
	case 0:
		return tui.ColorBlack
	case 1:
		return tui.ColorRed
	case 2:
		return tui.ColorGreen
	case 3:
		return tui.ColorYellow
	case 4:
		return tui.ColorBlue
	case 5:
		return tui.ColorMagenta
	case 6:
		return tui.ColorCyan
	case 7:
		return tui.ColorLightGray
	case 8:
		return tui.ColorGray
	case 9:
		return tui.ColorLightRed
	case 10:
		return tui.ColorLightGreen
	case 11:
		return tui.ColorLightYellow
	case 12:
		return tui.ColorLightBlue
	case 13:
		return tui.ColorLightMagenta
	case 14:
		return tui.ColorLightCyan
	case 15:
		return tui.ColorWhite
	default:
		return tui.ColorIndexed(idx)
	}
}

func lipglossUnderlineToTUI(u lipgloss.Underline) tui.UnderlineStyle {
	switch u {
	case lipgloss.UnderlineSingle:
		return tui.UnderlineLine
	case lipgloss.UnderlineDouble:
		return tui.UnderlineDoubleLine
	case lipgloss.UnderlineCurly:
		return tui.UnderlineCurl
	case lipgloss.UnderlineDotted:
		return tui.UnderlineDotted
	case lipgloss.UnderlineDashed:
		return tui.UnderlineDashed
	default:
		return tui.UnderlineReset
	}
}

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
