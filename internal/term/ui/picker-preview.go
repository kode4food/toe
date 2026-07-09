package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
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
		w, h   int
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
		w, h      int
		hlFrom    int
		hlTo      int
		diffLines map[int]diffGutterKind
		scroll    int
	}
)

func (p *previewCtx) renderInto(buf *tui.Buffer, x, y int) {
	switch {
	case p.item.Location.Target.ID != view.InvalidDocumentId:
		doc, ok := p.editor.Document(p.item.Location.Target.ID)
		if !ok {
			p.blitPlaceholderInto(buf, x, y, "<Invalid file location>")
			return
		}
		if p.hlFrom < 0 {
			sel := doc.Selection()
			if l, err := doc.Text().CharToLine(
				sel.Primary().Cursor(doc.Text()),
			); err == nil {
				p.hlFrom, p.hlTo = l, l
			}
		}
		p.renderDocInto(buf, x, y, doc)
	case p.item.Location.Target.Path != "":
		path := p.item.Location.Target.Path
		if doc := openDocumentPreview(path, p.editor); doc != nil {
			p.renderDocInto(buf, x, y, doc)
			return
		}
		p.renderFileInto(buf, x, y, path)
	case p.item.Preview != nil:
		text := p.item.Preview(p.w, p.h)
		p.blitPlaceholderInto(buf, x, y, text)
	}
}

func (p *previewCtx) renderDocInto(
	buf *tui.Buffer, x, y int, doc *view.Document,
) {
	entry := p.picker.previewCache.doc(p.syntax, doc)
	format := doc.TextFormatForConfig(p.w, p.editor.Options())
	r := &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, opts: p.editor.Options(),
		th: p.th, w: p.w, h: p.h,
		hlFrom: p.hlFrom, hlTo: p.hlTo,
		diffLines: p.itemDiffLines(entry.rope),
		scroll:    p.picker.previewScroll,
	}
	renderPreviewDocInto(buf, x, y, r)
	p.picker.previewScroll = r.scroll
}

func (p *previewCtx) itemDiffLines(text core.Rope) map[int]diffGutterKind {
	return diffGutterLines(p.item.DiffHunks, text.LenLines())
}

func (p *previewCtx) renderFileInto(buf *tui.Buffer, x, y int, path string) {
	p.picker.previewCache.path(p.syntax, path).renderInto(p, buf, x, y)
}

func (p *previewDocEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, x, y int,
) {
	opts := ctx.editor.Options()
	format := language.TextFormatForConfig(
		language.LoadLanguage(p.lang), opts.TextWidth, opts.SoftWrap, ctx.w,
	)
	r := &previewDocRender{
		text: p.rope, spans: p.spans,
		format: format, opts: ctx.editor.Options(),
		th: ctx.th, w: ctx.w, h: ctx.h,
		hlFrom: ctx.hlFrom, hlTo: ctx.hlTo,
		diffLines: ctx.itemDiffLines(p.rope),
		scroll:    ctx.picker.previewScroll,
	}
	renderPreviewDocInto(buf, x, y, r)
	ctx.picker.previewScroll = r.scroll
}

func (p *previewDirEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, x, y int,
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
		if i >= ctx.h {
			return
		}
		st := fillTUI
		if entry.dir {
			st = dirTUI
		}
		buf.FillRange(x, y+i, ctx.w, fillTUI)
		buf.SetString(x, y+i, ansi.Truncate(entry.name, ctx.w, ""), st)
	}
}

func (p noPreviewEntry) renderInto(ctx *previewCtx, buf *tui.Buffer, x, y int) {
	ctx.blitPlaceholderInto(buf, x, y, string(p))
}

// ANSI codes in callback preview strings are stripped so the popup style
// applies
func (p *previewCtx) blitPlaceholderInto(
	buf *tui.Buffer, x, y int, text string,
) {
	fillTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Background(
			p.th.Get("ui.popup").GetBackground(),
		),
	)
	blitTextInto(buf, x, y, p.w, p.h, text, fillTUI)
}

func blitTextInto(
	buf *tui.Buffer, x, y, w, h int, text string, fillStyle tui.Style,
) {
	lines := strings.SplitN(text, "\n", h+1)
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, line := range lines {
		plain := ansi.Strip(line)
		buf.FillRange(x, y+i, w, fillStyle)
		if w > 0 && plain != "" {
			s := plain
			if runewidth.StringWidth(s) > w {
				s = ansi.Truncate(s, w, "")
			}
			buf.SetString(x, y+i, s, fillStyle)
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
	if lang == "text" {
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

func previewAnchor(
	text core.Rope, format *language.TextFormat, softWrap bool, from, to, h int,
) (int, int) {
	if from < 0 {
		return 0, 0
	}
	var vf *core.VisualMoveFormat
	if softWrap {
		vf = &core.VisualMoveFormat{
			ViewportWidth:    format.ViewportWidth,
			TabWidth:         format.TabWidth,
			MaxWrap:          format.MaxWrap,
			MaxIndentRetain:  format.MaxIndentRetain,
			WrapIndicatorLen: runewidth.StringWidth(format.WrapIndicator),
		}
	} else {
		vf = &core.VisualMoveFormat{}
	}
	if to-from >= h {
		return from, 0
	}
	middle := from + (to-from)/2
	anchorLine, vOff := vf.VisualScrollUp(text, middle, 0, h/2)
	if from < anchorLine {
		return from, 0
	}
	return anchorLine, vOff
}
