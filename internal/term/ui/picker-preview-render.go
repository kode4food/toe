package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type previewLineCtx struct {
	format      *language.TextFormat
	lgStyles    *lipglossStyles
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

func renderPreviewDocInto(buf *tui.Buffer, x, y int, args *previewDocRender) {
	lgStyles := new(buildLipglossStyles(args.th, view.ModeNormal))
	tuiStyles := buildTUIStyles(lgStyles)
	hlLipgloss := previewHlStyleFn(hlStyleFnFor(args.th))
	hlCache := make(map[string]tui.Style, 32)
	hlStyleFn := func(scope string) tui.Style {
		if st, ok := hlCache[scope]; ok {
			return st
		}
		st := lipglossToTUIStyle(hlLipgloss(scope))
		hlCache[scope] = st
		return st
	}
	ws := args.opts.Whitespace
	ig := args.opts.IndentGuides
	rulers := args.opts.Rulers
	rulerBg := lipglossToTUIStyle(lgStyles.ruler).BgColor()
	// syntax spans have stripped backgrounds; patch popup bg onto every row
	// so the pane provides it uniformly rather than showing terminal default
	fillTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Background(
			args.th.Get("ui.popup").GetBackground(),
		),
	)
	popupBg := fillTUI.BgColor()

	markerW := 0
	if len(args.diffLines) > 0 {
		markerW = 1
	}
	contentX := x + markerW
	contentW := args.w - markerW

	softWrap := args.format.SoftWrap && args.format.ViewportWidth > 0
	anchorLine, vOff := previewAnchor(
		args.text, args.format, softWrap, args.hlFrom, args.hlTo, args.h,
	)
	nLines := args.text.LenLines()
	// Apply wheel scroll past the anchor, then write the applied delta back
	// so the stored scroll stays bounded. The upper bound pins the last line
	// to the bottom of the pane rather than letting content scroll off top.
	// The clamp counts logical lines; a tall soft-wrapped final line can still
	// leave a gap. Use visual-row pinning if that becomes observable
	if args.scroll != 0 {
		base := anchorLine
		anchorLine = max(0, min(base+args.scroll, max(0, nLines-args.h)))
		args.scroll = anchorLine - base
		vOff = 0 // moving off the anchor line starts at its first visual row
	}
	var hlBg tui.Color
	if args.hlFrom >= 0 {
		hlBg = lipglossToTUIStyle(args.th.Get("ui.highlight")).BgColor()
	}

	bufRow := 0
	logLine := anchorLine
	for bufRow < args.h && logLine < nLines {
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
			lineStr: lStr, lgStyles: lgStyles, tuiStyles: tuiStyles,
			hlStyle: hlStyleFn, format: args.format, ws: ws, ig: ig,
			hlSpans: args.spans, cursor: -1, cursorLine: -1,
			lineNum: lineNum, lineStart: start, lineEnd: end,
			softWrap: softWrap, hStart: 0, hWidth: contentW,
			maxRows: args.h - bufRow + rowSkip,
		}
		rendered := rr.rows()
		highlighted := args.hlFrom >= 0 &&
			lineNum >= args.hlFrom && lineNum <= args.hlTo

		lineCtx := previewLineCtx{
			format: args.format, lgStyles: lgStyles,
			fillTUI: fillTUI, popupBg: popupBg,
			hlBg: hlBg, w: contentW,
			rowSkip: rowSkip, maxH: args.h - bufRow,
			softWrap: softWrap, lStr: lStr,
			highlighted: highlighted, markerW: markerW,
		}
		if kind, ok := args.diffLines[lineNum]; ok {
			lineCtx.marker, lineCtx.markerStyle =
				previewDiffMarker(kind, tuiStyles)
		}
		bufRow += emitPreviewLine(buf, contentX, y+bufRow, rendered, lineCtx)
	}
	if len(rulers) > 0 {
		applyRulers(buf, x, y, args.w, args.h, 0, rulers, rulerBg)
	}
}

func emitPreviewLine(
	buf *tui.Buffer, x, y int,
	rendered []renderedRow, ctx previewLineCtx,
) int {
	n := 0
	if ctx.softWrap {
		indent := indentWidth(ctx.lStr, ctx.format.TabWidth)
		prefixRow := softWrapContinuationRow(ctx.format, indent, ctx.lgStyles)
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
			row.writeToBuffer(rowWriteArgs{
				buf: buf, x: x, y: y + n,
				fillStyle: ctx.fillTUI, width: ctx.w,
			})
			buf.PatchBgRange(x, y+n, ctx.w, ctx.popupBg)
			if ctx.highlighted {
				buf.PatchBgRange(x, y+n, ctx.w, ctx.hlBg)
			}
			ctx.emitMarker(buf, x, y+n, n == 0)
			n++
		}
	} else {
		rendered[0].writeToBuffer(rowWriteArgs{
			buf: buf, x: x, y: y,
			fillStyle: ctx.fillTUI, width: ctx.w,
		})
		buf.PatchBgRange(x, y, ctx.w, ctx.popupBg)
		if ctx.highlighted {
			buf.PatchBgRange(x, y, ctx.w, ctx.hlBg)
		}
		ctx.emitMarker(buf, x, y, true)
		n = 1
	}
	return n
}

func (c previewLineCtx) emitMarker(buf *tui.Buffer, x, y int, first bool) {
	if c.markerW == 0 {
		return
	}
	mx := x - c.markerW
	buf.FillRange(mx, y, c.markerW, c.fillTUI)
	buf.PatchBgRange(mx, y, c.markerW, c.popupBg)
	if c.marker != "" && first {
		st := c.markerStyle.Bg(c.popupBg)
		buf.SetString(mx, y, c.marker, st)
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
