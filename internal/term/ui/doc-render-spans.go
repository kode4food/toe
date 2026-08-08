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

	opts := r.context.Editor.Options()
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
			cursor:  rng.Cursor(text),
			primary: i == primaryIdx,
		})
	}

	cursorLines := make(map[int]struct{}, len(allRanges))
	for _, sp := range selSpans {
		if l, err := text.CharToLine(sp.cursor); err == nil {
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

	c := r.editor.cache
	rowMap := c.viewRowMaps[v.ID()][:0]

	dc := c.docCaches[docID]
	if dc == nil {
		dc = &docRenderCache{}
		c.docCaches[docID] = dc
	}

	rawText := dc.ensureRawText(rev, text)
	hlSpans := dc.ensureHightlight(ensureHighlightArgs{
		cache:   r.context.Syntax,
		rev:     rev,
		lang:    lang,
		rawText: rawText,
	})
	lineIdx := dc.ensureLineIndex(rev, rawText)

	pat, hasPat := r.context.Editor.FirstRegister('/')
	if !hasPat || !doc.SearchHighlightsActive(v.ID()) {
		pat = ""
	}
	dc.ensureSearchSpans(ensureSearchSpansArgs{
		rev:     rev,
		pattern: pat,
		rawText: rawText,
	})
	searchMatches := dc.searchSpans
	docDiagnostics := doc.Diagnostics()
	var docHighlights []matchSpan
	if r.context.Editor.Mode() != view.ModeSelect &&
		r.editor.mouse.downRange == nil {
		docHighlights = documentHighlightSpans(doc.DocumentHighlights(v.ID()))
	}
	var docLinks []matchSpan
	var docColors []colorSpan
	if r.context.Editor.Mode() == view.ModeNormal {
		docLinks = documentLinkSpans(doc.DocumentLinks())
		docColors = documentColorSpans(doc.DocumentColors())
	}

	// styles rebuilt only when theme or mode changes
	th := r.context.Theme()
	mode := r.context.Editor.Mode()
	key := styleKey{theme: th.Name(), mode: mode}
	if c.stylesKey != key {
		c.stylesKey = key
		c.styles = newDocStyleSet(th, mode)
		c.stylesDim = newDocStyleSet(r.context.ThemeFor(false), mode)
	}
	set := c.styles
	if !args.focused {
		set = c.stylesDim
	}
	styles := set.styles
	highlight := func(scope string) tui.Style {
		if st, ok := set.hlCache[scope]; ok {
			return st
		}
		st := set.highlight(scope)
		set.hlCache[scope] = st
		return st
	}

	diagnostics := diagnosticSpans(docDiagnostics, styles)
	var annotations []inlineAnnotation
	if r.context.Editor.Mode() == view.ModeNormal {
		annotations = inlayHintAnnotations(doc.InlayHints(v.ID()), styles)
		annotations = append(
			annotations, documentColorAnnotations(doc.DocumentColors())...,
		)
		slices.SortStableFunc(annotations, func(a, b inlineAnnotation) int {
			return cmp.Compare(a.pos, b.pos)
		})
	}

	cursorKind := opts.CursorShapeForMode(r.context.Editor.Mode())
	cursorIsBlock := cursorKind == view.CursorKindBlock && r.editor.focused &&
		args.focused
	cursorLineEnabled := opts.CursorLine
	ws := opts.Whitespace
	ig := opts.IndentGuides
	relativeLineNumbers := opts.LineNumber == view.LineNumberRelative
	insertMode := r.context.Editor.Mode() == view.ModeInsert

	format := doc.TextFormatForConfig(
		args.area.Width-gutterW, r.context.Editor.Options(),
	)
	softWrap := format.SoftWrap && gutterW < args.area.Width
	contentW := args.area.Width - gutterW
	r.context.Editor.SetViewContentWidth(contentW)

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
		v.EnsureCursorVisibleHorizontal(&view.CursorScroll{
			Doc:       text,
			Selection: sel,
			Width:     hWidth,
			TabWidth:  format.TabWidth,
			ScrollOff: opts.ScrollOff,
		})
	}
	hOff := v.Offset().HorizontalOffset

	lineTUI := styles.line
	lineSelTUI := styles.lineSelected
	fillTUI := styles.text
	cursorLinePriBg := styles.cursorLinePrim.BgColor()
	cursorLineSecBg := styles.cursorLineSec.BgColor()
	contentX := args.area.X + gutterW
	gutter := gutterSpec{
		layout:       gutterLayout,
		lineNumWidth: gutterLineNumberW,
		width:        gutterLayoutWidth(gutterLayout, gutterLineNumberW),
		lineStyle:    lineTUI,
		lineSelected: lineSelTUI,
		diagLines:    diagnosticGutterLines(text, docDiagnostics),
		diffLines: documentDiffLines(
			r.context.Editor, doc, text.LenLines(),
		),
		severityHint:    styles.severityHint,
		severityInfo:    styles.severityInfo,
		severityWarning: styles.severityWarning,
		severityError:   styles.severityError,
		diffAdded:       styles.diffAdded,
		diffModified:    styles.diffModified,
		diffRemoved:     styles.diffRemoved,
	}

	rr := rowRender{
		styles:        styles,
		hlStyle:       highlight,
		format:        format,
		whitespace:    ws,
		indents:       ig,
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
		mode:          r.context.Editor.Mode(),
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

		lineCount:     nLines,
		trailingEmpty: trailingEmpty,

		docCache: dc,
		rev:      rev,
		rawText:  rawText,
		lineIdx:  lineIdx,
		rowMap:   rowMap,

		styles: styles,

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
		cursorColumnBg:      styles.cursorColumn.BgColor(),
		rulers:              opts.Rulers,
		rulerBg:             styles.rulerBg,

		gutter: gutter,
		row:    rr,
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
	hints []view.InlayHint, styles *styles,
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

func inlayHintStyle(kind string, styles *styles) tui.Style {
	switch kind {
	case "type":
		return styles.inlayHintType
	case "parameter":
		return styles.inlayHintParam
	default:
		return styles.inlayHint
	}
}
