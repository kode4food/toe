package ui

import (
	"cmp"
	"slices"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type renderContentArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	area    geom.Area
	focused bool
}

func (r *renderPass) prepareContentRender(
	args renderContentArgs,
) *contentRenderState {
	doc := args.doc
	v := args.view

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

	pat, hasPat := r.cx.Editor.FirstRegister('/')
	if !hasPat || !doc.SearchHighlightsActive(v.ID()) {
		pat = ""
	}
	dc.ensureSearchSpans(rev, pat, rawText)
	searchMatches := dc.smSpans
	docDiagnostics := doc.Diagnostics()
	var docHighlights []matchSpan
	if r.cx.Editor.Mode() != view.ModeSelect && r.ec.mouseDownRange == nil {
		docHighlights = documentHighlightSpans(doc.DocumentHighlights(v.ID()))
	}
	var docLinks []matchSpan
	var docColors []colorSpan
	if r.cx.Editor.Mode() == view.ModeNormal {
		docLinks = documentLinkSpans(doc.DocumentLinks())
		docColors = documentColorSpans(doc.DocumentColors())
	}

	// styles rebuilt only when theme or mode changes
	th := r.activeTheme()
	key := styleKey{theme: th.Name(), mode: r.cx.Editor.Mode()}
	if c.stylesKey != key {
		c.stylesKey = key
		c.tuiStyles = buildTUIStyles(th, r.cx.Editor.Mode())
		c.hlFn = hlStyleFnFor(th)
		c.hlTUICache = make(map[string]tui.Style, 64)
	}
	tuiStyles := c.tuiStyles
	hlLipgloss := c.hlFn
	hlTUICache := c.hlTUICache
	hlStyleFn := func(scope string) tui.Style {
		if st, ok := hlTUICache[scope]; ok {
			return st
		}
		st := styleToTUI(hlLipgloss(scope))
		hlTUICache[scope] = st
		return st
	}

	diagnostics := diagnosticSpans(docDiagnostics, tuiStyles)
	var annotations []inlineAnnotation
	if r.cx.Editor.Mode() == view.ModeNormal {
		annotations = inlayHintAnnotations(doc.InlayHints(v.ID()), tuiStyles)
		annotations = append(
			annotations, documentColorAnnotations(doc.DocumentColors())...,
		)
		slices.SortStableFunc(annotations, func(a, b inlineAnnotation) int {
			return cmp.Compare(a.pos, b.pos)
		})
	}

	cursorKind := opts.CursorShapeForMode(r.cx.Editor.Mode().String())
	cursorIsBlock := cursorKind == view.CursorKindBlock && r.ec.focused &&
		args.focused
	cursorLineEnabled := opts.CursorLine
	ws := opts.Whitespace
	ig := opts.IndentGuides
	relativeLineNumbers := opts.LineNumber == view.LineNumberRelative
	insertMode := r.cx.Editor.Mode() == view.ModeInsert

	format := doc.TextFormatForConfig(
		args.area.Width-gutterW, r.cx.Editor.Options(),
	)
	softWrap := format.SoftWrap && gutterW < args.area.Width
	contentW := args.area.Width - gutterW
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

	lineTUI := tuiStyles.line
	lineSelTUI := tuiStyles.lineSelected
	rulerTUI := tuiStyles.ruler
	fillTUI := tuiStyles.text
	cursorLinePriBg := tuiStyles.cursorLinePrim.BgColor()
	cursorLineSecBg := tuiStyles.cursorLineSec.BgColor()
	contentX := args.area.X + gutterW
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

	return &contentRenderState{
		args: args,

		text:   text,
		sel:    sel,
		cursor: cursor,

		cursorLines: cursorLines,
		cursorLine:  cursorLine,
		anchorLine:  anchorLine,
		vOff:        vOff,

		nLines:        nLines,
		trailingEmpty: trailingEmpty,

		dc:      dc,
		rev:     rev,
		rawText: rawText,
		lineIdx: lineIdx,
		rowMap:  rowMap,

		tuiStyles: tuiStyles,

		diagnostics: diagnostics,
		annotations: annotations,

		cursorIsBlock:       cursorIsBlock,
		cursorLineEnabled:   cursorLineEnabled,
		relativeLineNumbers: relativeLineNumbers,
		insertMode:          insertMode,

		format:   format,
		softWrap: softWrap,

		hOff: hOff,

		fillTUI:         fillTUI,
		cursorLinePriBg: cursorLinePriBg,
		cursorLineSecBg: cursorLineSecBg,
		contentX:        contentX,

		cursorColumnEnabled: opts.CursorColumn,
		cursorColumnBg:      tuiStyles.cursorColumn.BgColor(),
		rulers:              opts.Rulers,
		rulerBg:             rulerTUI.BgColor(),

		gutter: gutter,
		rr:     rr,
	}
}

func documentHighlightSpans(highlights []view.DocumentHighlight) []matchSpan {
	if len(highlights) == 0 {
		return nil
	}
	out := make([]matchSpan, 0, len(highlights))
	for _, h := range highlights {
		if h.From < h.To {
			out = append(out, matchSpan{from: h.From, to: h.To})
		}
	}
	return out
}

func documentLinkSpans(links []view.DocumentLink) []matchSpan {
	if len(links) == 0 {
		return nil
	}
	out := make([]matchSpan, 0, len(links))
	for _, link := range links {
		if link.From < link.To {
			out = append(out, matchSpan{from: link.From, to: link.To})
		}
	}
	return out
}

func inlayHintAnnotations(
	hints []view.InlayHint, styles *tuiStyles,
) []inlineAnnotation {
	if len(hints) == 0 {
		return nil
	}
	out := make([]inlineAnnotation, 0, len(hints)*3)
	for _, hint := range hints {
		if hint.Label == "" {
			continue
		}
		st := inlayHintStyle(hint.Kind, styles)
		if hint.PaddingLeft {
			out = append(out, inlineAnnotation{
				pos: hint.Pos, text: " ", style: st,
			})
		}
		out = append(out, inlineAnnotation{
			pos: hint.Pos, text: hint.Label, style: st,
		})
		if hint.PaddingRight {
			out = append(out, inlineAnnotation{
				pos: hint.Pos, text: " ", style: st,
			})
		}
	}
	return out
}

func inlayHintStyle(kind string, styles *tuiStyles) tui.Style {
	switch kind {
	case "type":
		return styles.inlayHintType
	case "parameter":
		return styles.inlayHintParam
	default:
		return styles.inlayHint
	}
}
