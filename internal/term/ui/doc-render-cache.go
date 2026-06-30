package ui

import (
	"cmp"
	"regexp"
	"slices"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	renderCache struct {
		// docCaches holds per-document raw-text, highlight, and search-match
		// caches so multiple panes showing different documents do not evict
		// each other's tokenization every frame
		docCaches map[view.DocumentId]*docRenderCache

		// rebuilt only when theme or mode changes between frames
		stylesKey  string
		lgStyles   *lipglossStyles
		tuiStyles  *tuiStyles
		hlFn       func(string) lipgloss.Style
		hlTUICache map[string]tui.Style

		viewRowMaps map[view.Id][]viewRowEntry
	}

	viewRowEntry struct {
		logLine int
		offset  int
		prefixW int
	}

	// docRenderCache memoizes a single document's derived render state, keyed
	// internally by revision so it is recomputed only when the document changes
	docRenderCache struct {
		rawTextRev    int
		rawTextCached string

		hlRev   int
		hlLang  string
		hlSpans []highlight.Span

		smRev   int
		smPat   string
		smSpans []matchSpan
	}

	matchSpan struct{ from, to int }

	colorSpan struct {
		from, to int
		style    tui.Style
	}

	diagnosticSpan struct {
		from, to int
		severity view.DiagnosticSeverity
		style    tui.Style
	}

	inlineAnnotation struct {
		pos   int
		text  string
		style tui.Style
	}
)

func newRenderCache() *renderCache {
	return &renderCache{
		docCaches:   map[view.DocumentId]*docRenderCache{},
		viewRowMaps: map[view.Id][]viewRowEntry{},
	}
}

func (dc *docRenderCache) ensureRawText(rev int, text core.Rope) string {
	if dc.rawTextRev != rev || dc.rawTextCached == "" {
		dc.rawTextRev = rev
		dc.rawTextCached = text.String()
	}
	return dc.rawTextCached
}

func (dc *docRenderCache) ensureHL(
	rev int, lang, rawText string,
) []highlight.Span {
	if lang != "text" && (dc.hlRev != rev || dc.hlLang != lang) {
		dc.hlRev = rev
		dc.hlLang = lang
		dc.hlSpans = syntax.Tokenize(
			highlight.NormalizeNewlines(rawText), lang,
		)
	}
	if lang == "text" {
		return nil
	}
	return dc.hlSpans
}

func (dc *docRenderCache) ensureSearchSpans(rev int, pat, rawText string) {
	if dc.smRev == rev && dc.smPat == pat {
		return
	}
	dc.smRev = rev
	dc.smPat = pat
	dc.smSpans = nil
	if pat == "" {
		return
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return
	}
	locs := re.FindAllStringIndex(rawText, -1)
	if len(locs) == 0 {
		return
	}
	b2r := make([]int, len(rawText)+1)
	ri := 0
	for bi := range rawText {
		b2r[bi] = ri
		ri++
	}
	b2r[len(rawText)] = ri
	for _, loc := range locs {
		from, to := b2r[loc[0]], b2r[loc[1]]
		if to > from {
			dc.smSpans = append(dc.smSpans, matchSpan{from, to})
		}
	}
}

func documentHighlightSpans(
	highlights []view.DocumentHighlight,
) []matchSpan {
	if len(highlights) == 0 {
		return nil
	}
	out := make([]matchSpan, 0, len(highlights))
	for _, h := range highlights {
		if h.From < h.To {
			out = append(out, matchSpan{from: h.From, to: h.To})
		}
	}
	return out
}

func documentLinkSpans(links []view.DocumentLink) []matchSpan {
	if len(links) == 0 {
		return nil
	}
	out := make([]matchSpan, 0, len(links))
	for _, link := range links {
		if link.From < link.To {
			out = append(out, matchSpan{from: link.From, to: link.To})
		}
	}
	return out
}

func documentColorSpans(colors []view.DocumentColor) []colorSpan {
	if len(colors) == 0 {
		return nil
	}
	out := make([]colorSpan, 0, len(colors))
	for _, color := range colors {
		if color.From < color.To {
			out = append(out, colorSpan{
				from:  color.From,
				to:    color.To,
				style: documentColorStyle(color),
			})
		}
	}
	return out
}

func diagnosticSpans(
	diags []view.Diagnostic, styles *tuiStyles,
) []diagnosticSpan {
	if len(diags) == 0 {
		return nil
	}
	out := make([]diagnosticSpan, 0, len(diags))
	for _, diag := range diags {
		from := diag.Range.From
		to := diag.Range.To
		if from > to {
			from, to = to, from
		}
		if from == to {
			to++
		}
		out = append(out, diagnosticSpan{
			from:     from,
			to:       to,
			severity: diag.Severity,
			style:    diagnosticStyle(diag.Severity, styles),
		})
	}
	slices.SortStableFunc(out, func(a, b diagnosticSpan) int {
		if n := cmp.Compare(a.from, b.from); n != 0 {
			return n
		}
		return cmp.Compare(a.to, b.to)
	})
	return out
}

func lineDiagnosticSpans(
	diags []diagnosticSpan, from, to int,
) []diagnosticSpan {
	if len(diags) == 0 {
		return nil
	}
	start := len(diags)
	end := start
	for i, diag := range diags {
		if diag.to <= from {
			continue
		}
		if diag.from > to {
			break
		}
		if start == len(diags) {
			start = i
		}
		end = i + 1
	}
	if start == len(diags) {
		return nil
	}
	return diags[start:end]
}

func documentColorAnnotations(colors []view.DocumentColor) []inlineAnnotation {
	if len(colors) == 0 {
		return nil
	}
	out := make([]inlineAnnotation, 0, len(colors))
	for _, color := range colors {
		if color.From < color.To {
			out = append(out, inlineAnnotation{
				pos:   color.From,
				text:  "\u25a0", // ■
				style: documentColorStyle(color),
			})
		}
	}
	return out
}

func inlayHintAnnotations(
	hints []view.InlayHint, styles *tuiStyles,
) []inlineAnnotation {
	if len(hints) == 0 {
		return nil
	}
	out := make([]inlineAnnotation, 0, len(hints)*3)
	for _, hint := range hints {
		if hint.Label == "" {
			continue
		}
		st := inlayHintStyle(hint.Kind, styles)
		if hint.PaddingLeft {
			out = append(out, inlineAnnotation{
				pos: hint.Pos, text: " ", style: st,
			})
		}
		out = append(out, inlineAnnotation{
			pos: hint.Pos, text: hint.Label, style: st,
		})
		if hint.PaddingRight {
			out = append(out, inlineAnnotation{
				pos: hint.Pos, text: " ", style: st,
			})
		}
	}
	return out
}

func inlayHintStyle(kind string, styles *tuiStyles) tui.Style {
	switch kind {
	case "type":
		return styles.inlayHintType
	case "parameter":
		return styles.inlayHintParam
	default:
		return styles.inlayHint
	}
}

func diagnosticStyle(
	severity view.DiagnosticSeverity, styles *tuiStyles,
) tui.Style {
	switch severity {
	case view.DiagnosticSeverityError:
		return styles.diagnosticError
	case view.DiagnosticSeverityWarning:
		return styles.diagnosticWarning
	case view.DiagnosticSeverityInfo:
		return styles.diagnosticInfo
	case view.DiagnosticSeverityHint:
		return styles.diagnosticHint
	default:
		return styles.diagnostic
	}
}

func lineAnnotations(
	annotations []inlineAnnotation, from, to int,
) []inlineAnnotation {
	if len(annotations) == 0 {
		return nil
	}
	start := len(annotations)
	end := start
	for i, ann := range annotations {
		if ann.pos < from {
			continue
		}
		if ann.pos > to {
			break
		}
		if start == len(annotations) {
			start = i
		}
		end = i + 1
	}
	if start == len(annotations) {
		return nil
	}
	return annotations[start:end]
}

func documentColorStyle(color view.DocumentColor) tui.Style {
	bg := tui.ColorRGB(color.Red, color.Green, color.Blue)
	fg := tui.ColorWhite
	if colorLuma(color) > 128000 {
		fg = tui.ColorBlack
	}
	return tui.Style{}.Fg(fg).Bg(bg)
}

func colorLuma(color view.DocumentColor) int {
	return int(color.Red)*299 + int(color.Green)*587 + int(color.Blue)*114
}
