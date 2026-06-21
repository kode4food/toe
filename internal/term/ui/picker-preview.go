package ui

import (
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
			if ansi.StringWidth(s) > w {
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
			WrapIndicatorLen: ansi.StringWidth(format.WrapIndicator),
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

func renderPreviewDocInto(buf *tui.Buffer, x0, y0 int, args *previewDocRender) {
	lgStyles := new(buildLipglossStyles(args.th, view.ModeNormal))
	lgStyles.clearBackground()
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
			softWrap: softWrap, hStart: 0, hWidth: args.w,
			maxRows: args.h - bufRow + rowSkip,
		}
		rendered := rr.rows()
		highlighted := args.hlFrom >= 0 &&
			lineNum >= args.hlFrom && lineNum <= args.hlTo

		bufRow += emitPreviewLine(
			buf, x0, y0+bufRow, rendered,
			previewLineCtx{
				format: args.format, lgStyles: lgStyles,
				fillTUI: fillTUI, popupBg: popupBg,
				hlBg: hlBg, w: args.w,
				rowSkip: rowSkip, maxH: args.h - bufRow,
				softWrap: softWrap, lStr: lStr,
				highlighted: highlighted,
			},
		)
	}
	if len(rulers) > 0 {
		applyRulers(buf, x0, y0, args.w, args.h, 0, rulers, rulerBg)
	}
}

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
		n = 1
	}
	return n
}
