package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type previewLineCtx struct {
	format      *language.TextFormat
	styles      *tuiStyles
	fillTUI     tui.Style
	popupBg     tui.Color
	hlBg        tui.Color
	w           int
	rowSkip     int
	maxH        int
	softWrap    bool
	lStr        string
	highlighted bool
	markerW     int
	marker      string
	markerStyle tui.Style
}

func renderPreviewDocInto(buf *tui.Buffer, args *previewDocRender) {
	tuiStyles := buildTUIStyles(args.th, view.ModeNormal)
	hlStyle := previewHlStyleFn(hlStyleFnFor(args.th))
	hlCache := make(map[string]tui.Style, 32)
	hlStyleFn := func(scope string) tui.Style {
		if st, ok := hlCache[scope]; ok {
			return st
		}
		st := hlStyle(scope)
		hlCache[scope] = st
		return st
	}
	ws := args.opts.Whitespace
	ig := args.opts.IndentGuides
	// syntax spans have stripped backgrounds; patch popup bg onto every row
	// so the pane provides it uniformly rather than showing terminal default
	fillTUI := tui.Style{}.Bg(args.th.Get("ui.popup").BgColor())
	popupBg := fillTUI.BgColor()

	markerW := 0
	if len(args.diffLines) > 0 {
		markerW = 1
	}
	contentX := args.area.X + markerW
	contentW := args.area.Width - markerW

	softWrap := args.format.SoftWrap && args.format.ViewportWidth > 0
	anchor := previewAnchor(previewAnchorArgs{
		text:     args.text,
		format:   args.format,
		softWrap: softWrap,
		from:     args.hlFrom,
		to:       args.hlTo,
		height:   args.area.Height,
	})
	anchorLine, vOff := anchor.line, anchor.verticalOffset
	nLines := args.text.LenLines()
	// clamp scroll to keep the last line pinned to the pane bottom, then
	// write the applied delta back so stored scroll stays bounded
	if args.scroll != 0 {
		base := anchorLine
		anchorLine = max(0, min(
			base+args.scroll, max(0, nLines-args.area.Height),
		))
		args.scroll = anchorLine - base
		vOff = 0 // moving off the anchor line starts at its first visual row
	}
	var hlBg tui.Color
	if args.hlFrom >= 0 {
		hlBg = args.th.Get("ui.highlight").BgColor()
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
		lStr := lineString(args.text, start, end)
		rowSkip := 0
		if softWrap && lineNum == anchorLine {
			rowSkip = vOff
		}
		rr := rowRender{
			lineStr:    lStr,
			tuiStyles:  tuiStyles,
			hlStyle:    hlStyleFn,
			format:     args.format,
			ws:         ws,
			ig:         ig,
			hlSpans:    args.spans,
			cursor:     -1,
			cursorLine: -1,
			lineNum:    lineNum,
			lineStart:  start,
			lineEnd:    end,
			softWrap:   softWrap,
			hStart:     0,
			hWidth:     contentW,
			maxRows:    args.area.Height - bufRow + rowSkip,
		}
		rendered := rr.rows()
		highlighted := args.hlFrom >= 0 &&
			lineNum >= args.hlFrom && lineNum <= args.hlTo

		lineCtx := previewLineCtx{
			format:      args.format,
			styles:      tuiStyles,
			fillTUI:     fillTUI,
			popupBg:     popupBg,
			hlBg:        hlBg,
			w:           contentW,
			rowSkip:     rowSkip,
			maxH:        args.area.Height - bufRow,
			softWrap:    softWrap,
			lStr:        lStr,
			highlighted: highlighted,
			markerW:     markerW,
		}
		if kind, ok := args.diffLines[lineNum]; ok {
			lineCtx.marker, lineCtx.markerStyle =
				previewDiffMarker(kind, tuiStyles)
		}
		bufRow += emitPreviewLine(
			buf, geom.Point{X: contentX, Y: args.area.Y + bufRow}, rendered,
			lineCtx,
		)
	}
}

func emitPreviewLine(
	buf *tui.Buffer, at geom.Point, rendered []renderedRow, ctx previewLineCtx,
) int {
	n := 0
	if ctx.softWrap {
		indent := indentWidth(ctx.lStr, ctx.format.TabWidth)
		prefixRow := softWrapContinuationRow(ctx.format, indent, ctx.styles)
		for i, cr := range rendered {
			if i < ctx.rowSkip {
				continue
			}
			if n >= ctx.maxH {
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
				fillStyle: ctx.fillTUI,
				width:     ctx.w,
			})
			buf.PatchBgRange(rowAt, ctx.w, ctx.popupBg)
			if ctx.highlighted {
				buf.PatchBgRange(rowAt, ctx.w, ctx.hlBg)
			}
			ctx.emitMarker(buf, rowAt, n == 0)
			n++
		}
	} else {
		rendered[0].writeToBuffer(rowWriteArgs{
			buf:       buf,
			at:        at,
			fillStyle: ctx.fillTUI,
			width:     ctx.w,
		})
		buf.PatchBgRange(at, ctx.w, ctx.popupBg)
		if ctx.highlighted {
			buf.PatchBgRange(at, ctx.w, ctx.hlBg)
		}
		ctx.emitMarker(buf, at, true)
		n = 1
	}
	return n
}

func (c previewLineCtx) emitMarker(buf *tui.Buffer, at geom.Point, first bool) {
	if c.markerW == 0 {
		return
	}
	mAt := at.Sub(geom.Point{X: c.markerW})
	buf.FillRange(mAt, c.markerW, c.fillTUI)
	buf.PatchBgRange(mAt, c.markerW, c.popupBg)
	if c.marker != "" && first {
		st := c.markerStyle.Bg(c.popupBg)
		buf.SetString(mAt, c.marker, st)
	}
}

func previewDiffMarker(
	kind diffGutterKind, styles *tuiStyles,
) (string, tui.Style) {
	switch kind {
	case diffGutterAdded:
		return "▍", styles.diffAdded
	case diffGutterRemoved:
		return "▔", styles.diffRemoved
	default:
		return "▍", styles.diffModified
	}
}
