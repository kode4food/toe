package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	previewCtx struct {
		picker *Picker
		item   *PickerItem
		editor *view.Editor
		syntax *syntax.Cache
		images *imageRegistry
		size   geom.Size
		// hlFrom < 0 means full preview, no highlight
		hlFrom int
		hlTo   int
		th     *theme.Theme
	}

	previewDocRender struct {
		text      core.Rope
		spans     []highlight.Span
		format    *language.TextFormat
		opts      *view.Options
		th        *theme.Theme
		area      geom.Area
		hlFrom    int
		hlTo      int
		diffLines map[int]diffGutterKind
		scroll    int
	}
)

func (p *previewCtx) renderInto(buf *tui.Buffer, at geom.Point) {
	switch {
	case p.item.Location.Target.ID != view.InvalidDocumentId:
		doc, ok := p.editor.Document(p.item.Location.Target.ID)
		if !ok {
			p.blitPlaceholderInto(buf, at, "<Invalid file location>")
			return
		}
		if p.hlFrom < 0 {
			sel := p.previewSelection(doc)
			if l, err := doc.Text().CharToLine(
				sel.Primary().Cursor(doc.Text()),
			); err == nil {
				p.hlFrom, p.hlTo = l, l
			}
		}
		p.renderDocInto(buf, at, doc)
	case p.item.Location.Target.Path != "":
		path := p.item.Location.Target.Path
		if doc := openDocumentPreview(path, p.editor); doc != nil {
			p.renderDocInto(buf, at, doc)
			return
		}
		p.renderFileInto(buf, at, path)
	case p.item.Preview != nil:
		text := p.item.Preview(p.size)
		p.blitPlaceholderInto(buf, at, text)
	}
}

func (p *previewCtx) previewSelection(doc *view.Document) core.Selection {
	if fv, ok := p.editor.FocusedView(); ok && fv.DocID() == doc.ID() {
		return doc.SelectionFor(fv.ID())
	}
	return doc.Selection()
}

func (p *previewCtx) renderDocInto(
	buf *tui.Buffer, at geom.Point, doc *view.Document,
) {
	entry := p.picker.previewCache.doc(p.syntax, doc)
	format := doc.TextFormatForConfig(p.size.Width, p.editor.Options())
	r := &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, opts: p.editor.Options(),
		th: p.th, area: geom.Area{Point: at, Size: p.size},
		hlFrom: p.hlFrom, hlTo: p.hlTo,
		diffLines: p.itemDiffLines(entry.rope),
		scroll:    p.picker.previewScroll,
	}
	renderPreviewDocInto(buf, r)
	p.picker.previewScroll = r.scroll
}

func (p *previewCtx) itemDiffLines(text core.Rope) map[int]diffGutterKind {
	return diffGutterLines(p.item.DiffHunks, text.LenLines())
}

func (p *previewCtx) renderFileInto(
	buf *tui.Buffer, at geom.Point, path string,
) {
	p.picker.previewCache.path(p.syntax, path).renderInto(p, buf, at)
}

func (p *previewDocEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	opts := ctx.editor.Options()
	format := language.TextFormatForConfig(
		language.LoadLanguage(p.lang), opts.TextWidth, opts.SoftWrap,
		ctx.size.Width,
	)
	r := &previewDocRender{
		text: p.rope, spans: p.spans,
		format: format, opts: ctx.editor.Options(),
		th: ctx.th, area: geom.Area{Point: at, Size: ctx.size},
		hlFrom: ctx.hlFrom, hlTo: ctx.hlTo,
		diffLines: ctx.itemDiffLines(p.rope),
		scroll:    ctx.picker.previewScroll,
	}
	renderPreviewDocInto(buf, r)
	ctx.picker.previewScroll = r.scroll
}

func (p *previewDirEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	fillTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Background(
			ctx.th.Get("ui.popup").GetBackground(),
		),
	)
	dirTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Foreground(
			ctx.th.Get("ui.text.directory").GetForeground(),
		).Background(ctx.th.Get("ui.popup").GetBackground()),
	)
	for i, entry := range p.rows {
		if i >= ctx.size.Height {
			return
		}
		st := fillTUI
		if entry.dir {
			st = dirTUI
		}
		rowAt := at.Add(geom.Point{Y: i})
		buf.FillRange(rowAt, ctx.size.Width, fillTUI)
		buf.SetString(
			rowAt,
			ansi.Truncate(entry.name, ctx.size.Width, ""),
			st,
		)
	}
}

func (p noPreviewEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	ctx.blitPlaceholderInto(buf, at, string(p))
}

// ANSI codes in callback preview strings are stripped so the popup style
// applies
func (p *previewCtx) blitPlaceholderInto(
	buf *tui.Buffer, at geom.Point, text string,
) {
	fillTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Background(
			p.th.Get("ui.popup").GetBackground(),
		),
	)
	blitTextInto(buf, geom.Area{Point: at, Size: p.size}, text, fillTUI)
}

func blitTextInto(
	buf *tui.Buffer, area geom.Area, text string, fillStyle tui.Style,
) {
	lines := strings.SplitN(text, "\n", area.Height+1)
	if len(lines) > area.Height {
		lines = lines[:area.Height]
	}
	for i, line := range lines {
		plain := ansi.Strip(line)
		at := area.Point.Add(geom.Point{Y: i})
		buf.FillRange(at, area.Width, fillStyle)
		if area.Width > 0 && plain != "" {
			s := plain
			if runewidth.StringWidth(s) > area.Width {
				s = ansi.Truncate(s, area.Width, "")
			}
			buf.SetString(at, s, fillStyle)
		}
	}
}

func openDocumentPreview(path string, editor *view.Editor) *view.Document {
	for _, doc := range editor.AllDocuments() {
		if doc.Path() == path {
			return doc
		}
	}
	return nil
}

func previewSpans(sc *syntax.Cache, text, lang string) []highlight.Span {
	if lang == view.DefaultLanguage {
		return nil
	}
	return sc.Tokenize(text, lang)
}

// span backgrounds are stripped so the pane provides the background uniformly
func previewHlStyleFn(
	fn func(string) lipgloss.Style,
) func(string) lipgloss.Style {
	return func(scope string) lipgloss.Style {
		return clearStyleBackground(fn(scope))
	}
}

type (
	previewAnchorArgs struct {
		text     core.Rope
		format   *language.TextFormat
		softWrap bool
		from     int
		to       int
		height   int
	}
	previewAnchorRes struct {
		line           int
		verticalOffset int
	}
)

func previewAnchor(args previewAnchorArgs) previewAnchorRes {
	if args.from < 0 {
		return previewAnchorRes{}
	}
	var vf *core.VisualMoveFormat
	if args.softWrap {
		vf = &core.VisualMoveFormat{
			ViewportWidth:    args.format.ViewportWidth,
			TabWidth:         args.format.TabWidth,
			MaxWrap:          args.format.MaxWrap,
			MaxIndentRetain:  args.format.MaxIndentRetain,
			WrapIndicatorLen: runewidth.StringWidth(args.format.WrapIndicator),
		}
	} else {
		vf = &core.VisualMoveFormat{}
	}
	if args.to-args.from >= args.height {
		return previewAnchorRes{line: args.from}
	}
	middle := args.from + (args.to-args.from)/2
	anchor := vf.VisualScrollUp(core.VisualScrollUpArgs{
		Doc:  args.text,
		Line: middle,
		Up:   args.height / 2,
	})
	line, vOff := anchor.Line, anchor.Row
	if args.from < line {
		return previewAnchorRes{line: args.from}
	}
	return previewAnchorRes{line: line, verticalOffset: vOff}
}
