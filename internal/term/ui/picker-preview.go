package ui

import (
	"path/filepath"
	"strings"

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
		wrap   int
		hlFrom int
		hlTo   int

		theme  *theme.Theme
		styles *styles
	}

	previewDocRender struct {
		text   core.Rope
		spans  []highlight.Span
		format *language.TextFormat
		opts   *view.Options

		area    geom.Area
		vScroll int
		hScroll int

		hlFrom    int
		hlTo      int
		diffLines map[int]diffGutterKind

		theme  *theme.Theme
		styles *styles
	}
)

func (p *previewCtx) renderInto(buf *tui.Buffer, at geom.Point) {
	if p.item.DiffPreview {
		p.renderDiffInto(buf, at)
		return
	}
	switch {
	case p.item.Location.Target.ID != view.InvalidDocumentId:
		doc := p.editor.Document(p.item.Location.Target.ID)
		if doc == nil {
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
		if doc := openDocumentPreview(p.editor, path); doc != nil {
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
	if fv := p.editor.FocusedView(); fv != nil && fv.DocID() == doc.ID() {
		return doc.SelectionFor(fv.ID())
	}
	return doc.Selection()
}

func (p *previewCtx) renderDocInto(
	buf *tui.Buffer, at geom.Point, doc *view.Document,
) {
	entry := p.picker.preview.cache.doc(p.syntax, doc)
	format := doc.TextFormatForConfig(p.wrap, p.editor.Options())
	r := &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, opts: p.editor.Options(),
		theme: p.theme, area: geom.Area{Point: at, Size: p.size},
		hlFrom: p.hlFrom, hlTo: p.hlTo,
		diffLines: p.itemDiffLines(entry.rope),
		vScroll:   p.picker.preview.vScroll,
		hScroll:   p.picker.preview.hScroll,
		styles:    p.styles,
	}
	renderPreviewDocInto(buf, r)
	p.picker.preview.vScroll = r.vScroll
	p.picker.preview.hScroll = r.hScroll
}

func (p *previewCtx) itemDiffLines(text core.Rope) map[int]diffGutterKind {
	return diffGutterLines(p.item.DiffHunks(), text.LenLines())
}

func (p *previewCtx) renderDiffInto(buf *tui.Buffer, at geom.Point) {
	vc := p.editor.VersionControl()
	if vc == nil {
		p.blitPlaceholderInto(buf, at, "<No version control>")
		return
	}
	staged := p.item.Location.Target.Variant == changedFileStaged
	base := p.picker.diffBaseFor(vc, p.item.BasePath, staged)
	work := p.workingPreview(vc, staged)
	opts := p.editor.Options()
	r := &diffPreviewRender{
		working: work.rope, base: base, spans: work.spans,
		lines: buildDiffPreviewLines(buildDiffPreviewLinesArgs{
			kind:    p.item.DiffKind,
			working: work.rope,
			base:    base,
			hunks:   p.item.DiffHunks(),
		}),
		format: language.TextFormatForConfig(
			language.LoadLanguage(work.lang), opts.TextWidth, opts.SoftWrap,
			p.wrap,
		),
		opts: opts, theme: p.theme,
		area:    geom.Area{Point: at, Size: p.size},
		vScroll: p.picker.preview.vScroll,
		hScroll: p.picker.preview.hScroll,
		styles:  p.styles,
	}
	renderDiffPreviewInto(buf, r)
	p.picker.preview.vScroll = r.vScroll
	p.picker.preview.hScroll = r.hScroll
}

// a staged row shows what a commit would record, so its working side is the
// index text rather than the file on disk
func (p *previewCtx) workingPreview(
	vc view.VersionControl, staged bool,
) previewDocEntry {
	if staged {
		return *p.picker.preview.cache.indexText(
			p.syntax, vc, p.item.Location.Target.Path,
		)
	}
	if p.item.Location.Target.ID != view.InvalidDocumentId {
		if doc := p.editor.Document(p.item.Location.Target.ID); doc != nil {
			return *p.picker.preview.cache.doc(p.syntax, doc)
		}
	}
	path := p.item.Location.Target.Path
	if doc := openDocumentPreview(p.editor, path); doc != nil {
		return *p.picker.preview.cache.doc(p.syntax, doc)
	}
	e, ok := p.picker.preview.cache.path(p.syntax, path).(*previewDocEntry)
	if ok {
		return *e
	}
	return previewDocEntry{rope: core.NewRope(""), lang: view.DefaultLanguage}
}

func (p *previewCtx) renderFileInto(
	buf *tui.Buffer, at geom.Point, path string,
) {
	p.picker.preview.cache.path(p.syntax, path).renderInto(p, buf, at)
}

func (p *previewDocEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	opts := ctx.editor.Options()
	format := language.TextFormatForConfig(
		language.LoadLanguage(p.lang), opts.TextWidth, opts.SoftWrap,
		ctx.wrap,
	)
	r := &previewDocRender{
		text: p.rope, spans: p.spans,
		format: format, opts: ctx.editor.Options(),
		theme: ctx.theme, area: geom.Area{Point: at, Size: ctx.size},
		hlFrom: ctx.hlFrom, hlTo: ctx.hlTo,
		diffLines: ctx.itemDiffLines(p.rope),
		vScroll:   ctx.picker.preview.vScroll,
		hScroll:   ctx.picker.preview.hScroll,
		styles:    ctx.styles,
	}
	renderPreviewDocInto(buf, r)
	ctx.picker.preview.vScroll = r.vScroll
	ctx.picker.preview.hScroll = r.hScroll
}

func (p *previewDirEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	popupBg := ctx.theme.Get("ui.popup").BgColor()
	fillTUI := ctx.theme.Get("ui.popup")
	dirTUI := tui.Style{}.
		Fg(ctx.theme.Get("ui.text.directory").FgColor()).
		Bg(popupBg)
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
		textAt := rowAt
		width := ctx.size.Width
		if ctx.editor.Options().NerdFonts {
			icon := pickerDefaultDirectoryIcon
			if !entry.dir {
				path := filepath.Join(p.path, entry.name)
				icon = ctx.picker.fileIcon(path, nil)
			}
			iconTUI := pickerFileIconStyle(ctx.theme, fillTUI, icon.color)
			buf.SetString(rowAt, icon.glyph, iconTUI)
			iconWidth := runewidth.StringWidth(icon.glyph) + 1
			textAt.X += iconWidth
			width = max(width-iconWidth, 0)
		}
		buf.SetString(
			textAt,
			runewidth.Truncate(entry.name, width, ""),
			st,
		)
	}
}

func (p noPreviewEntry) renderInto(
	ctx *previewCtx, buf *tui.Buffer, at geom.Point,
) {
	style := tui.Style{}.Bg(ctx.theme.Get("ui.popup").BgColor())
	renderCenteredMessage(
		buf, geom.Area{Point: at, Size: ctx.size}, string(p), style,
	)
}

func (p *previewCtx) blitPlaceholderInto(
	buf *tui.Buffer, at geom.Point, text string,
) {
	fillTUI := tui.Style{}.Bg(p.theme.Get("ui.popup").BgColor())
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
			s := runewidth.Truncate(plain, area.Width, "")
			buf.SetString(at, s, fillStyle)
		}
	}
}

func openDocumentPreview(e *view.Editor, path string) *view.Document {
	for _, doc := range e.AllDocuments() {
		if doc.Path() == path {
			return doc
		}
	}
	return nil
}

type previewSpansArgs struct {
	cache *syntax.Cache
	text  string
	lang  string
}

func previewSpans(args previewSpansArgs) []highlight.Span {
	if args.lang == view.DefaultLanguage {
		return nil
	}
	return args.cache.Tokenize(core.Source{Text: args.text, Lang: args.lang})
}

// previewHighlighter: the background is stripped so the pane can patch its own
// onto every row
func previewHighlighter(th *theme.Theme) func(string) tui.Style {
	fn := highlighterFor(th)
	cache := make(map[string]tui.Style, 32)
	return func(scope string) tui.Style {
		if st, ok := cache[scope]; ok {
			return st
		}
		st := clearStyleBackground(fn(scope))
		cache[scope] = st
		return st
	}
}

type clampPreviewHScrollArgs struct {
	hScroll      int
	text         core.Rope
	lines        core.Span
	contentWidth int
}

func clampPreviewHScroll(args clampPreviewHScrollArgs) int {
	if args.hScroll <= 0 {
		return 0
	}
	widest := 0
	for line := args.lines.From; line <= args.lines.To; line++ {
		widest = max(widest, lineDisplayWidth(args.text, line))
	}
	return min(args.hScroll, max(widest-args.contentWidth, 0))
}

func lineDisplayWidth(text core.Rope, line int) int {
	start, err := text.LineToChar(line)
	if err != nil {
		return 0
	}
	end, err := text.LineEndCharIndex(line)
	if err != nil {
		return 0
	}
	return runewidth.StringWidth(lineString(text, core.Span{
		From: start,
		To:   end,
	}))
}
