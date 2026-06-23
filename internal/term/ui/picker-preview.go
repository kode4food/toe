package ui

import (
	"os"
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
		w, h   int
		// hlFrom < 0 means full preview, no highlight
		hlFrom int
		hlTo   int
		th     *theme.Theme
	}

	previewFile struct {
		text string
		lang string
	}

	previewDocRender struct {
		text   core.Rope
		spans  []highlight.Span
		format *language.TextFormat
		opts   *view.Options
		th     *theme.Theme
		w, h   int
		hlFrom int
		hlTo   int
		// scroll offsets the viewport by logical lines past the match anchor;
		// the renderer clamps it to the document and writes the applied value
		// back so the caller can persist a bounded scroll position
		scroll int
	}

	pickerPreviewKind int
)

const (
	pickerPreviewDirectory pickerPreviewKind = iota
	pickerPreviewBinary
	pickerPreviewLargeFile
	pickerPreviewNotFound
)

func (p *previewCtx) renderInto(buf *tui.Buffer, x0, y0 int) {
	switch {
	case p.item.Location.Target.ID != view.InvalidDocumentId:
		doc, ok := p.editor.Document(p.item.Location.Target.ID)
		if !ok {
			p.blitPlaceholderInto(buf, x0, y0, "<Invalid file location>")
			return
		}
		p.renderDocInto(buf, x0, y0, doc)
	case p.item.Location.Target.Path != "":
		path := p.item.Location.Target.Path
		if doc := openDocumentPreview(path, p.editor); doc != nil {
			p.renderDocInto(buf, x0, y0, doc)
			return
		}
		p.renderFileInto(buf, x0, y0, path)
	case p.item.Preview != nil:
		text := p.item.Preview(p.w, p.h)
		p.blitPlaceholderInto(buf, x0, y0, text)
	}
}

func (p *previewCtx) renderDocInto(
	buf *tui.Buffer, x0, y0 int, doc *view.Document,
) {
	lang := doc.Lang()
	rev := doc.Revision()
	id := doc.ID()
	entry, ok := p.picker.spanCache[id]
	if !ok || entry.rev != rev || entry.lang != lang {
		text := highlight.NormalizeNewlines(doc.Text().String())
		entry = previewSpanEntry{
			rev: rev, lang: lang,
			rope: core.NewRope(text), spans: previewSpans(text, lang),
		}
		p.picker.spanCache[id] = entry
	}
	format := doc.TextFormatForConfig(p.w, p.editor.Options())
	r := &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, opts: p.editor.Options(),
		th: p.th, w: p.w, h: p.h,
		hlFrom: p.hlFrom, hlTo: p.hlTo, scroll: p.picker.previewScroll,
	}
	renderPreviewDocInto(buf, x0, y0, r)
	p.picker.previewScroll = r.scroll
}

func (p *previewCtx) renderFileInto(
	buf *tui.Buffer, x0, y0 int, path string,
) {
	entry, ok := p.picker.fileCache[path]
	if !ok {
		f, ph := loadPreviewFile(path)
		if f == nil {
			p.blitPlaceholderInto(buf, x0, y0, pickerPlaceholderText(ph))
			return
		}
		entry = fileSpanEntry{
			rope:  core.NewRope(f.text),
			spans: previewSpans(f.text, f.lang),
			lang:  f.lang,
		}
		p.picker.fileCache[path] = entry
	}
	opts := p.editor.Options()
	format := language.TextFormatForLanguageWithConfig(
		entry.lang, opts.TextWidth, opts.SoftWrap, p.w,
	)
	r := &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, opts: p.editor.Options(),
		th: p.th, w: p.w, h: p.h,
		hlFrom: p.hlFrom, hlTo: p.hlTo, scroll: p.picker.previewScroll,
	}
	renderPreviewDocInto(buf, x0, y0, r)
	p.picker.previewScroll = r.scroll
}

// ANSI codes in callback preview strings are stripped so the popup style
// applies
func (p *previewCtx) blitPlaceholderInto(
	buf *tui.Buffer, x0, y0 int, text string,
) {
	fillTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Background(
			p.th.Get("ui.popup").GetBackground(),
		),
	)
	blitTextInto(buf, x0, y0, p.w, p.h, text, fillTUI)
}

func blitTextInto(
	buf *tui.Buffer, x0, y0, w, h int, text string, fillStyle tui.Style,
) {
	lines := strings.SplitN(text, "\n", h+1)
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, line := range lines {
		plain := ansi.Strip(line)
		buf.FillRange(x0, y0+i, w, fillStyle)
		if w > 0 && plain != "" {
			s := plain
			if runewidth.StringWidth(s) > w {
				s = ansi.Truncate(s, w, "")
			}
			buf.SetString(x0, y0+i, s, fillStyle)
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

func loadPreviewFile(path string) (*previewFile, pickerPreviewKind) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, pickerPreviewNotFound
	}
	if info.IsDir() {
		return nil, pickerPreviewDirectory
	}
	if info.Size() > pickerMaxPreview {
		return nil, pickerPreviewLargeFile
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, pickerPreviewNotFound
	}
	if looksBinary(data) {
		return nil, pickerPreviewBinary
	}
	text := highlight.NormalizeNewlines(string(data))
	lang := highlight.DetectLanguage(path, text)
	return &previewFile{text: text, lang: lang}, 0
}

func pickerPlaceholderText(kind pickerPreviewKind) string {
	switch kind {
	case pickerPreviewDirectory:
		return "<Invalid directory location>"
	case pickerPreviewBinary:
		return "<Binary file>"
	case pickerPreviewLargeFile:
		return "<File too large to preview>"
	default:
		return "<File not found>"
	}
}

func previewSpans(text, lang string) []highlight.Span {
	if lang == "text" {
		return nil
	}
	return syntax.Tokenize(text, lang)
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
