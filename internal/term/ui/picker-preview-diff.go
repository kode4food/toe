package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	diffPreviewRender struct {
		highlight func(string) tui.Style
		working   *previewDocEntry
		base      *previewDocEntry
		lines     []diffPreviewLine

		format *language.TextFormat
		opts   *view.Options

		area    geom.Area
		vScroll int
		hScroll int

		anchorLine int
		lineCount  int

		theme  *theme.Theme
		styles *styles
	}

	diffPreviewLine struct {
		kind diffLineKind
		line int
	}

	diffLineKind uint8

	diffBaseKey struct {
		path   string
		staged bool
	}

	tintColors struct {
		base      tui.Color
		accent    tui.Color
		amount    float64
		trueColor bool
	}

	rgbColor struct {
		red   uint8
		green uint8
		blue  uint8
	}
)

const (
	diffLineContext diffLineKind = iota
	diffLineAdded
	diffLineRemoved

	diffGutterW     = 2
	diffPreviewLead = 3
	tintAmount      = 0.2
)

func (p *Picker) diffBaseFor(
	vc view.VersionControl, path string, staged bool,
) *previewDocEntry {
	key := diffBaseKey{path: path, staged: staged}
	if entry, ok := p.preview.diffBaseCache[key]; ok {
		return entry
	}
	text := vc.IndexText(path)
	if staged {
		text = vc.HeadText(path)
	}
	entry := &previewDocEntry{rope: core.NewRope(text)}
	p.preview.diffBaseCache[key] = entry
	return entry
}

type buildDiffPreviewLinesArgs struct {
	kind    view.FileChangeKind
	working core.Rope
	base    core.Rope
	hunks   []view.DiffHunk
}

func buildDiffPreviewLines(args buildDiffPreviewLinesArgs) []diffPreviewLine {
	switch args.kind {
	case view.FileChangeAdded, view.FileChangeUntracked:
		return allLines(args.working, diffLineAdded)
	case view.FileChangeDeleted:
		return allLines(args.base, diffLineRemoved)
	default:
		var out []diffPreviewLine
		nWork := args.working.LenLines()
		prev := 0
		for _, h := range args.hunks {
			for l := prev; l < h.From && l < nWork; l++ {
				out = append(out, diffPreviewLine{
					kind: diffLineContext,
					line: l,
				})
			}
			for l := h.BaseFrom; l < h.BaseTo; l++ {
				out = append(out, diffPreviewLine{
					kind: diffLineRemoved,
					line: l,
				})
			}
			for l := h.From; l < h.To && l < nWork; l++ {
				out = append(out, diffPreviewLine{
					kind: diffLineAdded,
					line: l,
				})
			}
			prev = h.To
		}
		for l := prev; l < nWork; l++ {
			out = append(out, diffPreviewLine{
				kind: diffLineContext,
				line: l,
			})
		}
		return out
	}
}

func allLines(text core.Rope, kind diffLineKind) []diffPreviewLine {
	n := text.LenLines()
	out := make([]diffPreviewLine, 0, n)
	for l := range n {
		out = append(out, diffPreviewLine{kind: kind, line: l})
	}
	return out
}

func renderDiffPreviewInto(buf *tui.Buffer, args *diffPreviewRender) {
	fillTUI := tui.Style{}.Bg(args.theme.Get("ui.popup").BgColor())
	popupBg := fillTUI.BgColor()
	addedBg := tintToward(tintColors{
		base:      popupBg,
		accent:    args.theme.Get("diff.plus").FgColor(),
		amount:    tintAmount,
		trueColor: args.styles.trueColor,
	})
	removedBg := tintToward(tintColors{
		base:      popupBg,
		accent:    args.theme.Get("diff.minus").FgColor(),
		amount:    tintAmount,
		trueColor: args.styles.trueColor,
	})

	barW := 0
	if args.opts.Scrollbar {
		barW = 1
	}
	paneW := args.area.Width - barW
	contentX := args.area.X + diffGutterW
	contentW := paneW - diffGutterW

	anchor := max(0, firstChangedLine(args.lines)-diffPreviewLead)
	maxStart := max(0, len(args.lines)-args.area.Height)
	start := max(0, min(anchor+args.vScroll, maxStart))
	args.vScroll = start - anchor
	args.anchorLine = anchor
	args.lineCount = len(args.lines)
	hOff := clampDiffHScroll(clampDiffHScrollArgs{
		render:       args,
		startRow:     start,
		contentWidth: contentW,
	})
	args.hScroll = hOff

	working := args.working.ensureRenderCache()
	base := args.base.ensureRenderCache()
	rr := rowRender{
		styles:     args.styles,
		hlStyle:    args.highlight,
		format:     args.format,
		whitespace: args.opts.Whitespace,
		indents:    args.opts.IndentGuides,
		cursor:     -1,
		cursorLine: -1,
		colStart:   hOff,
		colWidth:   contentW,
		maxRows:    1,
	}
	for row := range args.area.Height {
		idx := start + row
		at := geom.Point{X: contentX, Y: args.area.Y + row}
		signAt := geom.Point{X: args.area.X, Y: at.Y}
		buf.FillRange(signAt, paneW, fillTUI)
		buf.PatchBgRange(signAt, paneW, popupBg)
		if idx >= len(args.lines) {
			continue
		}
		dl := args.lines[idx]
		src := working
		rr.hlSpans = args.working.spans
		if dl.kind == diffLineRemoved {
			src = base
			rr.hlSpans = nil
		}
		if !rr.prepareLine(src, dl.line) {
			continue
		}
		rendered := rr.rows()
		rendered[0].writeToBuffer(rowWriteArgs{
			buf:       buf,
			at:        at,
			fillStyle: fillTUI,
			width:     contentW,
			startCol:  hOff,
		})
		buf.PatchBgRange(at, contentW, popupBg)

		sign := " "
		signStyle := fillTUI
		switch dl.kind {
		case diffLineAdded:
			buf.PatchBgRange(at, contentW, addedBg)
			sign = "+"
			signStyle = args.styles.diffAdded.Bg(popupBg)
		case diffLineRemoved:
			buf.PatchBgRange(at, contentW, removedBg)
			sign = "-"
			signStyle = args.styles.diffRemoved.Bg(popupBg)
		}
		buf.SetString(signAt, sign, signStyle)
	}
	if barW > 0 {
		drawDiffPreviewScrollbar(args, buf, start)
	}
	applyRulers(applyRulersArgs{
		buf:     buf,
		at:      geom.Point{X: contentX, Y: args.area.Y},
		size:    geom.Size{Width: contentW, Height: args.area.Height},
		rulers:  args.opts.Rulers,
		rulerBg: args.styles.rulerBg,
	})
}

func drawDiffPreviewScrollbar(
	args *diffPreviewRender, buf *tui.Buffer, start int,
) {
	bar := newScrollbar(
		scrollbarGeom{
			rows:   args.area.Height,
			maxTop: max(len(args.lines)-args.area.Height, 0),
		},
		geom.Point{
			X: args.area.X + args.area.Width - 1,
			Y: args.area.Y,
		},
		args.styles, start,
	)
	for row, dl := range args.lines {
		switch dl.kind {
		case diffLineAdded:
			bar.addMark(row, scrollMarkDiffAdded)
		case diffLineRemoved:
			bar.addMark(row, scrollMarkDiffRemoved)
		}
	}
	bar.draw(buf)
}

func tintToward(args tintColors) tui.Color {
	base := rgb8(args.base)
	accent := rgb8(args.accent)
	amount := args.amount
	mixed := tui.ColorRGB(
		uint8(float64(base.red)+
			(float64(accent.red)-float64(base.red))*amount),
		uint8(float64(base.green)+
			(float64(accent.green)-float64(base.green))*amount),
		uint8(float64(base.blue)+
			(float64(accent.blue)-float64(base.blue))*amount),
	)
	if !args.trueColor {
		return mixed.Quantized()
	}
	return mixed
}

func rgb8(c tui.Color) rgbColor {
	r, g, b, _ := c.RGBA()
	return rgbColor{
		red:   uint8(r >> 8),
		green: uint8(g >> 8),
		blue:  uint8(b >> 8),
	}
}

func firstChangedLine(lines []diffPreviewLine) int {
	for i, dl := range lines {
		if dl.kind != diffLineContext {
			return i
		}
	}
	return 0
}

type clampDiffHScrollArgs struct {
	render       *diffPreviewRender
	startRow     int
	contentWidth int
}

func clampDiffHScroll(args clampDiffHScrollArgs) int {
	if args.render.hScroll <= 0 {
		return 0
	}
	widest := 0
	for row := range args.render.area.Height {
		idx := args.startRow + row
		if idx >= len(args.render.lines) {
			break
		}
		dl := args.render.lines[idx]
		src := args.render.working
		if dl.kind == diffLineRemoved {
			src = args.render.base
		}
		widest = max(widest, lineDisplayWidth(src.rope, dl.line))
	}
	return min(args.render.hScroll, max(widest-args.contentWidth, 0))
}
