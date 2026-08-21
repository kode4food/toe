package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view/language"
)

type previewLineCtx struct {
	format   *language.TextFormat
	styles   *styles
	softWrap bool

	fillStyle   tui.Style
	popupBg     tui.Color
	highlightBg tui.Color

	width     int
	maxHeight int
	rowSkip   int
	horzOff   int

	lineText    string
	highlighted bool

	marker      string
	markerWidth int
	markerStyle tui.Style
}

func (c previewLineCtx) emitMarker(buf *tui.Buffer, at geom.Point, first bool) {
	if c.markerWidth == 0 {
		return
	}
	mAt := at.Sub(geom.Point{X: c.markerWidth})
	buf.FillRange(mAt, c.markerWidth, c.fillStyle)
	buf.PatchBgRange(mAt, c.markerWidth, c.popupBg)
	if c.marker != "" && first {
		st := c.markerStyle.Bg(c.popupBg)
		buf.SetString(mAt, c.marker, st)
	}
}

func renderPreviewDocInto(buf *tui.Buffer, args *previewDocRender) {
	highlight := previewHighlighter(args.theme)
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
	anchorLine, vOff := anchor.line, anchor.offset
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
	hOff := 0
	if !softWrap {
		hOff = clampPreviewHScroll(clampPreviewHScrollArgs{
			hScroll: args.hScroll,
			text:    args.text,
			lines: core.Span{
				From: anchorLine,
				To:   min(anchorLine+args.area.Height-1, nLines-1),
			},
			contentWidth: contentW,
		})
	}
	args.hScroll = hOff
	var hlBg tui.Color
	if args.hlFrom >= 0 {
		hlBg = args.theme.Get("ui.highlight").BgColor()
	}

	bufRow := 0
	logLine := anchorLine
	for bufRow < args.area.Height && logLine < nLines {
		lineNum := logLine
		logLine++
		start, err := args.text.LineToChar(lineNum)
		if err != nil {
			continue
		}
		end, err := args.text.LineEndCharIndex(lineNum)
		if err != nil {
			continue
		}
		lStr := lineString(args.text, core.Span{From: start, To: end})
		rowSkip := 0
		if softWrap && lineNum == anchorLine {
			rowSkip = vOff
		}
		rr := rowRender{
			lineText:   lStr,
			styles:     args.styles,
			hlStyle:    highlight,
			format:     args.format,
			whitespace: ws,
			indents:    ig,
			hlSpans:    args.spans,
			cursor:     -1,
			cursorLine: -1,
			lineNum:    lineNum,
			lineStart:  start,
			lineEnd:    end,
			softWrap:   softWrap,
			colStart:   hOff,
			colWidth:   contentW,
			maxRows:    args.area.Height - bufRow + rowSkip,
		}
		rendered := rr.rows()
		highlighted := args.hlFrom >= 0 &&
			lineNum >= args.hlFrom && lineNum <= args.hlTo

		lineCtx := previewLineCtx{
			format:      args.format,
			styles:      args.styles,
			fillStyle:   fillTUI,
			popupBg:     popupBg,
			highlightBg: hlBg,
			width:       contentW,
			rowSkip:     rowSkip,
			horzOff:     hOff,
			maxHeight:   args.area.Height - bufRow,
			softWrap:    softWrap,
			lineText:    lStr,
			highlighted: highlighted,
			markerWidth: markerW,
		}
		if kind, ok := args.diffLines[lineNum]; ok {
			lineCtx.marker, lineCtx.markerStyle =
				previewDiffMarker(kind, args.styles)
		}
		bufRow += emitPreviewLine(
			buf, geom.Point{X: contentX, Y: args.area.Y + bufRow}, rendered,
			lineCtx,
		)
	}
	applyPreviewRulers(buf, args.opts.Rulers, geom.Area{
		X:      contentX,
		Y:      args.area.Y,
		Width:  contentW,
		Height: bufRow,
	}, args.styles.rulerBg)
}

func applyPreviewRulers(
	buf *tui.Buffer, rulers []int, content geom.Area, rulerBg tui.Color,
) {
	if len(rulers) == 0 {
		return
	}
	applyRulers(applyRulersArgs{
		buf:     buf,
		at:      content.Point,
		size:    content.Size,
		rulers:  rulers,
		rulerBg: rulerBg,
	})
}

func emitPreviewLine(
	buf *tui.Buffer, at geom.Point, rendered []renderedRow, ctx previewLineCtx,
) int {
	n := 0
	if ctx.softWrap {
		indent := indentWidth(ctx.lineText, ctx.format.TabWidth)
		prefixRow := softWrapContinuationRow(ctx.format, indent, ctx.styles)
		for i, cr := range rendered {
			if i < ctx.rowSkip {
				continue
			}
			if n >= ctx.maxHeight {
				break
			}
			row := cr
			if i > 0 {
				row = prefixRow
				row.append(cr)
			}
			rowAt := at.Add(geom.Point{Y: n})
			row.writeToBuffer(rowWriteArgs{
				buf:       buf,
				at:        rowAt,
				fillStyle: ctx.fillStyle,
				width:     ctx.width,
				startCol:  ctx.horzOff,
			})
			buf.PatchBgRange(rowAt, ctx.width, ctx.popupBg)
			if ctx.highlighted {
				buf.PatchBgRange(rowAt, ctx.width, ctx.highlightBg)
			}
			ctx.emitMarker(buf, rowAt, n == 0)
			n++
		}
	} else {
		rendered[0].writeToBuffer(rowWriteArgs{
			buf:       buf,
			at:        at,
			fillStyle: ctx.fillStyle,
			width:     ctx.width,
			startCol:  ctx.horzOff,
		})
		buf.PatchBgRange(at, ctx.width, ctx.popupBg)
		if ctx.highlighted {
			buf.PatchBgRange(at, ctx.width, ctx.highlightBg)
		}
		ctx.emitMarker(buf, at, true)
		n = 1
	}
	return n
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
