package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	diffPreviewRender struct {
		working core.Rope
		base    core.Rope
		spans   []highlight.Span
		lines   []diffPreviewLine

		format *language.TextFormat
		opts   *view.Options

		area    geom.Area
		vScroll int
		hScroll int

		theme  *theme.Theme
		styles *styles
	}

	diffPreviewLine struct {
		kind diffLineKind
		line int
	}

	tintColors struct {
		base   tui.Color
		accent tui.Color
	}

	diffLineKind uint8

	// diffBaseKey identifies a row's base text, the head for a staged row and
	// the index for an unstaged one, so a path caches both apart
	diffBaseKey struct {
		path   string
		staged bool
	}
)

const (
	diffLineContext diffLineKind = iota
	diffLineAdded
	diffLineRemoved
)

const (
	diffGutterW     = 2
	diffPreviewLead = 3
	diffTintAmount  = 0.2
)

// the base of a staged row is the head, of an unstaged row the index, so both
// rows of a file edited in both places diff against the right side
func (p *Picker) diffBaseFor(
	vc view.VersionControl, path string, staged bool,
) core.Rope {
	key := diffBaseKey{path: path, staged: staged}
	if rope, ok := p.preview.diffBaseCache[key]; ok {
		return rope
	}
	text := vc.IndexText(path)
	if staged {
		text = vc.HeadText(path)
	}
	rope := core.NewRope(text)
	p.preview.diffBaseCache[key] = rope
	return rope
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
					kind: diffLineContext, line: l,
				})
			}
			for l := h.BaseFrom; l < h.BaseTo; l++ {
				out = append(out, diffPreviewLine{
					kind: diffLineRemoved, line: l,
				})
			}
			for l := h.From; l < h.To && l < nWork; l++ {
				out = append(out, diffPreviewLine{
					kind: diffLineAdded, line: l,
				})
			}
			prev = h.To
		}
		for l := prev; l < nWork; l++ {
			out = append(out, diffPreviewLine{
				kind: diffLineContext, line: l,
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
	hl := previewHighlighter(args.theme)
	ws := args.opts.Whitespace
	ig := args.opts.IndentGuides
	fillTUI := tui.Style{}.Bg(args.theme.Get("ui.popup").BgColor())
	popupBg := fillTUI.BgColor()
	addedBg := tintToward(&tintColors{
		base:   popupBg,
		accent: args.theme.Get("diff.plus").FgColor(),
	})
	removedBg := tintToward(&tintColors{
		base:   popupBg,
		accent: args.theme.Get("diff.minus").FgColor(),
	})

	contentX := args.area.X + diffGutterW
	contentW := args.area.Width - diffGutterW

	anchor := max(0, firstChangedLine(args.lines)-diffPreviewLead)
	maxStart := max(0, len(args.lines)-args.area.Height)
	start := max(0, min(anchor+args.vScroll, maxStart))
	args.vScroll = start - anchor
	hOff := clampDiffHScroll(clampDiffHScrollArgs{
		render:       args,
		startRow:     start,
		contentWidth: contentW,
	})
	args.hScroll = hOff

	for row := range args.area.Height {
		idx := start + row
		at := geom.Point{X: contentX, Y: args.area.Y + row}
		signAt := geom.Point{X: args.area.X, Y: at.Y}
		buf.FillRange(signAt, args.area.Width, fillTUI)
		buf.PatchBgRange(signAt, args.area.Width, popupBg)
		if idx >= len(args.lines) {
			continue
		}
		dl := args.lines[idx]
		src, spans := args.working, args.spans
		if dl.kind == diffLineRemoved {
			src, spans = args.base, nil
		}
		lineStart, err := src.LineToChar(dl.line)
		if err != nil {
			continue
		}
		lineEnd, err := src.LineEndCharIndex(dl.line)
		if err != nil {
			continue
		}
		rr := rowRender{
			lineText: lineString(src, core.Span{
				From: lineStart,
				To:   lineEnd,
			}),
			styles:     args.styles,
			hlStyle:    hl,
			format:     args.format,
			whitespace: ws,
			indents:    ig,
			hlSpans:    spans,
			cursor:     -1,
			cursorLine: -1,
			lineNum:    dl.line,
			lineStart:  lineStart,
			lineEnd:    lineEnd,
			colStart:   hOff,
			colWidth:   contentW,
			maxRows:    1,
		}
		rendered := rr.rows()
		rendered[0].writeToBuffer(rowWriteArgs{
			buf: buf, at: at, fillStyle: fillTUI, width: contentW,
			startCol: hOff,
		})
		buf.PatchBgRange(at, contentW, popupBg)

		sign, signStyle := " ", fillTUI
		switch dl.kind {
		case diffLineAdded:
			buf.PatchBgRange(at, contentW, addedBg)
			sign, signStyle = "+", args.styles.diffAdded.Bg(popupBg)
		case diffLineRemoved:
			buf.PatchBgRange(at, contentW, removedBg)
			sign, signStyle = "-", args.styles.diffRemoved.Bg(popupBg)
		case diffLineContext:
			// no-op
		}
		buf.SetString(signAt, sign, signStyle)
	}
	applyPreviewRulers(buf, args.opts.Rulers, geom.Area{
		X:      contentX,
		Y:      args.area.Y,
		Width:  contentW,
		Height: args.area.Height,
	}, args.styles.rulerBg)
}

func tintToward(colors *tintColors) tui.Color {
	base := rgb8(colors.base)
	accent := rgb8(colors.accent)
	mix := func(from, to uint8) uint8 {
		return uint8(float64(from) + (float64(to)-float64(from))*diffTintAmount)
	}
	return tui.ColorRGB(
		mix(base.red, accent.red),
		mix(base.green, accent.green),
		mix(base.blue, accent.blue),
	)
}

type rgb8Res struct {
	red   uint8
	green uint8
	blue  uint8
}

func rgb8(c tui.Color) rgb8Res {
	r, g, b, _ := c.RGBA()
	return rgb8Res{
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
		widest = max(widest, lineDisplayWidth(src, dl.line))
	}
	return min(args.render.hScroll, max(widest-args.contentWidth, 0))
}
