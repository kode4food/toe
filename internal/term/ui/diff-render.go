package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// renderDiffPane draws every file's diff appended into one scroll, each file
// under a path header and highlighted in its own language
func (r *renderPass) renderDiffPane(
	buf *tui.Buffer, pane *DiffPane, y0 int, focused bool,
) {
	if len(pane.rows) == 0 {
		return
	}
	a := pane.Area()
	th := r.context.ThemeFor(focused)
	opts := r.context.Editor.Options()
	styles := buildStyles(th, view.ModeDiff)
	popupBg := th.Get("ui.popup").BgColor()
	fill := tui.Style{}.Bg(popupBg)
	addedBg := tintToward(tintColors{
		base:      popupBg,
		accent:    th.Get("diff.plus").FgColor(),
		amount:    tintAmount,
		trueColor: styles.trueColor,
	})
	removedBg := tintToward(tintColors{
		base:      popupBg,
		accent:    th.Get("diff.minus").FgColor(),
		amount:    tintAmount,
		trueColor: styles.trueColor,
	})
	headStyle := th.Get("ui.linenr.selected").Bg(popupBg)
	contentX := a.X + diffGutterW
	contentW := a.Width - diffGutterW
	start := min(max(pane.vScroll, 0), pane.maxScroll())
	pane.vScroll = start

	rr := rowRender{
		styles:     styles,
		hlStyle:    previewHighlighter(th),
		whitespace: opts.Whitespace,
		indents:    opts.IndentGuides,
		cursor:     -1,
		cursorLine: -1,
		colStart:   0,
		colWidth:   contentW,
		maxRows:    1,
	}
	fmtLang := "\x00" // sentinel so the first line always builds a format
	for row := range a.Height {
		y := y0 + a.Y + row
		at := geom.Point{X: contentX, Y: y}
		signAt := geom.Point{X: a.X, Y: y}
		buf.FillRange(signAt, a.Width, fill)
		buf.PatchBgRange(signAt, a.Width, popupBg)
		idx := start + row
		if idx >= len(pane.rows) {
			continue
		}
		dr := pane.rows[idx]
		if dr.header {
			label := runewidth.Truncate(dr.entry.path, a.Width, "…")
			buf.SetString(signAt, label, headStyle)
			continue
		}
		e := dr.entry
		if e.lang != fmtLang {
			rr.format = language.TextFormatForConfig(
				language.LoadLanguage(e.lang),
				opts.TextWidth, opts.SoftWrap, max(contentW, 1),
			)
			fmtLang = e.lang
		}
		src := e.work.ensureRenderCache()
		rr.hlSpans = e.work.spans
		if dr.dl.kind == diffLineRemoved {
			src = e.base.ensureRenderCache()
			rr.hlSpans = nil
		}
		if !rr.prepareLine(src, dr.dl.line) {
			continue
		}
		rr.rows()[0].writeToBuffer(rowWriteArgs{
			buf:       buf,
			at:        at,
			fillStyle: fill,
			width:     contentW,
			startCol:  0,
		})
		buf.PatchBgRange(at, contentW, popupBg)
		sign := " "
		signStyle := fill
		switch dr.dl.kind {
		case diffLineAdded:
			buf.PatchBgRange(at, contentW, addedBg)
			sign = "+"
			signStyle = styles.diffAdded.Bg(popupBg)
		case diffLineRemoved:
			buf.PatchBgRange(at, contentW, removedBg)
			sign = "-"
			signStyle = styles.diffRemoved.Bg(popupBg)
		}
		buf.SetString(signAt, sign, signStyle)
	}
}
