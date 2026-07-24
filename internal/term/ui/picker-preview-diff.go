package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"

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
		spans   []highlight.Span // working-text spans; nil for removed lines
		lines   []diffPreviewLine
		format  *language.TextFormat
		opts    *view.Options
		th      *theme.Theme
		area    geom.Area
		scroll  int
	}

	diffPreviewLine struct {
		kind diffLineKind
		line int // index into working (context/added) or base (removed) rope
	}

	diffLineKind uint8
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

// buildDiffPreviewLines produces the ordered unified-diff line list (context,
// removed base, added working) for a change of the given kind
func buildDiffPreviewLines(
	kind view.FileChangeKind, working, base core.Rope, hunks []view.DiffHunk,
) []diffPreviewLine {
	switch kind {
	case view.FileChangeAdded, view.FileChangeUntracked:
		return allLines(working, diffLineAdded)
	case view.FileChangeDeleted:
		return allLines(base, diffLineRemoved)
	default:
		var out []diffPreviewLine
		nWork := working.LenLines()
		prev := 0
		for _, h := range hunks {
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

// ponytail: shell-out per distinct path, cached; fine for a picker's lifetime
func (p *Picker) diffBaseFor(vc view.VersionControl, path string) core.Rope {
	if rope, ok := p.preview.diffBaseCache[path]; ok {
		return rope
	}
	rope := core.NewRope(vc.DiffBaseForPath(path))
	p.preview.diffBaseCache[path] = rope
	return rope
}

func renderDiffPreviewInto(buf *tui.Buffer, args *diffPreviewRender) {
	tuiStyles := buildTUIStyles(args.th, view.ModeNormal)
	hlLipgloss := previewHlStyleFn(hlStyleFnFor(args.th))
	hlCache := make(map[string]tui.Style, 32)
	hlStyleFn := func(scope string) tui.Style {
		if st, ok := hlCache[scope]; ok {
			return st
		}
		st := styleToTUI(hlLipgloss(scope))
		hlCache[scope] = st
		return st
	}
	ws := args.opts.Whitespace
	ig := args.opts.IndentGuides
	fillTUI := styleToTUI(
		lipgloss.NewStyle().Background(
			args.th.Get("ui.popup").GetBackground(),
		),
	)
	popupBg := fillTUI.BgColor()
	popupLg := args.th.Get("ui.popup").GetBackground()
	addedBg := tintToward(popupLg, args.th.Get("diff.plus").GetForeground())
	removedBg := tintToward(popupLg, args.th.Get("diff.minus").GetForeground())

	contentX := args.area.X + diffGutterW
	contentW := args.area.Width - diffGutterW

	anchor := max(0, firstChangedLine(args.lines)-diffPreviewLead)
	maxStart := max(0, len(args.lines)-args.area.Height)
	start := max(0, min(anchor+args.scroll, maxStart))
	args.scroll = start - anchor

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
			lineStr:    lineString(src, lineStart, lineEnd),
			tuiStyles:  tuiStyles,
			hlStyle:    hlStyleFn,
			format:     args.format,
			ws:         ws,
			ig:         ig,
			hlSpans:    spans,
			cursor:     -1,
			cursorLine: -1,
			lineNum:    dl.line,
			lineStart:  lineStart,
			lineEnd:    lineEnd,
			hStart:     0,
			hWidth:     contentW,
			maxRows:    1,
		}
		rendered := rr.rows()
		rendered[0].writeToBuffer(rowWriteArgs{
			buf: buf, at: at, fillStyle: fillTUI, width: contentW,
		})
		buf.PatchBgRange(at, contentW, popupBg)

		sign, signStyle := " ", fillTUI
		switch dl.kind {
		case diffLineAdded:
			buf.PatchBgRange(at, contentW, addedBg)
			sign, signStyle = "+", tuiStyles.diffAdded.Bg(popupBg)
		case diffLineRemoved:
			buf.PatchBgRange(at, contentW, removedBg)
			sign, signStyle = "-", tuiStyles.diffRemoved.Bg(popupBg)
		case diffLineContext:
			// no-op
		}
		buf.SetString(signAt, sign, signStyle)
	}
}

func tintToward(base, accent color.Color) tui.Color {
	br, bg, bb := rgb8(base)
	ar, ag, ab := rgb8(accent)
	mix := func(from, to uint8) uint8 {
		return uint8(float64(from) + (float64(to)-float64(from))*diffTintAmount)
	}
	return tui.ColorRGB(mix(br, ar), mix(bg, ag), mix(bb, ab))
}

func rgb8(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

func firstChangedLine(lines []diffPreviewLine) int {
	for i, dl := range lines {
		if dl.kind != diffLineContext {
			return i
		}
	}
	return 0
}
