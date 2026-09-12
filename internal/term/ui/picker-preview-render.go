package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

func renderPreviewDocInto(buf *tui.Buffer, args *previewDocRender) {
	highlight := args.highlight
	ws := args.opts.Whitespace
	ig := args.opts.IndentGuides
	// syntax spans have stripped backgrounds, so patch popup bg onto every
	// row rather than letting the terminal default show through
	fillTUI := tui.Style{}.Bg(args.theme.Get("ui.popup").BgColor())
	popupBg := fillTUI.BgColor()

	markerW := 0
	if len(args.diffLines) > 0 {
		markerW = 1
	}
	contentX := args.area.X + markerW
	contentW := args.area.Width - markerW

	softWrap := args.format.SoftWrap && args.format.ViewportWidth > 0
	anchor := (&selectionViewport{
		text:      args.text,
		format:    args.format,
		from:      args.hlFrom,
		to:        args.hlTo,
		height:    args.area.Height,
		scrolloff: args.opts.ScrollOff,
	}).anchor()
	anchorLine := anchor.line
	vOff := anchor.offset
	nLines := args.text.LenLines()
	// clamp scroll to keep the last line pinned to the pane bottom, then
	// write the applied delta back so stored scroll stays bounded
	if args.vScroll != 0 {
		base := anchorLine
		anchorLine = max(0, min(
			base+args.vScroll, max(0, nLines-args.area.Height),
		))
		args.vScroll = anchorLine - base
		vOff = 0 // moving off the anchor line starts at its first visual row
	}
	visible := core.Span{
		From: anchorLine,
		To:   min(anchorLine+args.area.Height-1, nLines-1),
	}
	hOff := 0
	if !softWrap {
		hOff = clampPreviewHScroll(clampPreviewHScrollArgs{
			hScroll:      args.hScroll,
			text:         args.text,
			lines:        visible,
			contentWidth: contentW,
		})
	}
	args.hScroll = hOff
	var hlBg tui.Color
	if args.hlFrom >= 0 {
		hlBg = args.theme.Get("ui.highlight").BgColor()
	}
	raw := args.cache.rawTextCached
	lineIdx := args.cache.lineIndex
	var annotations []inlineAnnotation
	if args.opts.ColorSwatches {
		annotations = appendColorAnnotations(
			nil, args.cache.visibleHexColors(raw, lineIdx, core.Span{
				From: visible.From,
				To:   visible.To + 1,
			}),
			args.opts.NerdFonts,
		)
	}
	st := &contentRenderState{
		buf:         buf,
		area:        args.area,
		stopAtEnd:   true,
		lineCount:   nLines,
		anchorLine:  anchorLine,
		vOff:        vOff,
		fillTUI:     fillTUI,
		contentX:    contentX,
		docCache:    args.cache,
		rawText:     raw,
		lineIdx:     lineIdx,
		annotations: annotations,
		row: rowRender{
			styles:     args.styles,
			hlStyle:    highlight,
			format:     args.format,
			whitespace: ws,
			indents:    ig,
			hlSpans:    args.spans,
			cursor:     -1,
			cursorLine: -1,
			softWrap:   softWrap,
			colStart:   hOff,
			colWidth:   contentW,
		},
		afterRow: func(at geom.Point, row viewRowEntry, first bool) {
			bg := popupBg
			if args.hlFrom >= 0 && row.logLine >= args.hlFrom &&
				row.logLine <= args.hlTo {
				bg = hlBg
			}
			buf.PatchBgRange(at, contentW, bg)
			if markerW == 0 {
				return
			}
			markerAt := at.Sub(geom.Point{X: markerW})
			buf.FillRange(markerAt, markerW, fillTUI)
			if kind, ok := args.diffLines[row.logLine]; ok && first {
				marker, style := previewDiffMarker(kind, args.styles)
				buf.SetString(markerAt, marker, style.Bg(popupBg))
			}
		},
	}
	bufRow := renderContentRows(st)
	applyRulers(applyRulersArgs{
		buf:     buf,
		at:      geom.Point{X: contentX, Y: args.area.Y},
		size:    geom.Size{Width: contentW, Height: bufRow},
		rulers:  args.opts.Rulers,
		rulerBg: args.styles.rulerBg,
	})
}

func previewDiffMarker(
	kind diffGutterKind, styles *styles,
) (string, tui.Style) {
	switch kind {
	case diffGutterAdded:
		return diffGutterBar, styles.diffAdded
	case diffGutterRemoved:
		return diffGutterTop, styles.diffRemoved
	default:
		return diffGutterBar, styles.diffModified
	}
}
