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
	"github.com/kode4food/toe/internal/view/config"
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
		cfg    *config.Config
		th     *theme.Theme
	}

	previewFile struct {
		text string
		lang string
	}

	previewDocRender struct {
		text   core.Rope
		spans  []highlight.Span
		format *config.TextFormat
		cfg    *config.Config
		th     *theme.Theme
		w, h   int
		hlFrom int
		hlTo   int
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
	format := doc.TextFormatForConfig(p.w, p.cfg)
	renderPreviewDocInto(buf, x0, y0, &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, cfg: p.cfg, th: p.th, w: p.w, h: p.h,
		hlFrom: p.hlFrom, hlTo: p.hlTo,
	})
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
	format := config.TextFormatForLanguageWithConfig(entry.lang, p.cfg, p.w)
	renderPreviewDocInto(buf, x0, y0, &previewDocRender{
		text: entry.rope, spans: entry.spans,
		format: format, cfg: p.cfg, th: p.th, w: p.w, h: p.h,
		hlFrom: p.hlFrom, hlTo: p.hlTo,
	})
}

// ANSI codes in callback preview strings are stripped so the popup style applies
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
	text core.Rope, format *config.TextFormat, softWrap bool, from, to, h int,
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
	ws := args.cfg.Whitespace()
	ig := args.cfg.IndentGuides()
	rulers := args.cfg.Rulers()
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
	var hlBg tui.Color
	if args.hlFrom >= 0 {
		hlBg = lipglossToTUIStyle(args.th.Get("ui.highlight")).BgColor()
	}

	nLines := args.text.LenLines()
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

		if softWrap {
			indent := indentWidth(lStr, args.format.TabWidth)
			prefixRow := softWrapContinuationRow(args.format, indent, lgStyles)
			for i, cr := range rendered {
				if i < rowSkip {
					continue
				}
				if bufRow >= args.h {
					break
				}
				row := cr
				if i > 0 {
					row = prefixRow
					row.append(cr)
				}
				row.writeToBuffer(rowWriteArgs{
					buf: buf, x: x0, y: y0 + bufRow,
					fillStyle: fillTUI, width: args.w,
				})
				buf.PatchBgRange(x0, y0+bufRow, args.w, popupBg)
				if highlighted {
					buf.PatchBgRange(x0, y0+bufRow, args.w, hlBg)
				}
				bufRow++
			}
		} else {
			rendered[0].writeToBuffer(rowWriteArgs{
				buf: buf, x: x0, y: y0 + bufRow,
				fillStyle: fillTUI, width: args.w,
			})
			buf.PatchBgRange(x0, y0+bufRow, args.w, popupBg)
			if highlighted {
				buf.PatchBgRange(x0, y0+bufRow, args.w, hlBg)
			}
			bufRow++
		}
	}
	if len(rulers) > 0 {
		applyRulers(buf, x0, y0, args.w, args.h, 0, rulers, rulerBg)
	}
}
