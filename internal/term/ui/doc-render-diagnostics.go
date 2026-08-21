package ui

import (
	"cmp"
	"regexp"
	"slices"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

// a path ending in an exported identifier, how servers name a foreign type
var qualifiedNamePattern = regexp.MustCompile(
	`[\w.\-]+(?:/[\w.\-]+)+\.[A-Z]\w*`,
)

var diagnosticPopupScopes = [...]string{
	view.DiagnosticSeverityHint:    "diagnostic.hint",
	view.DiagnosticSeverityInfo:    "diagnostic.info",
	view.DiagnosticSeverityWarning: "diagnostic.warning",
	view.DiagnosticSeverityError:   "diagnostic.error",
}

func (r *renderPass) renderDiagnosticPopup(buf *tui.Buffer) {
	doc := r.context.Editor.FocusedDocument()
	if doc == nil {
		return
	}
	v := r.context.Editor.FocusedView()
	if v == nil {
		return
	}
	diag, ok := diagnosticAtCursor(doc, v)
	if !ok {
		return
	}
	text := diagnosticPopupText(diag)
	if text == "" {
		return
	}
	r.drawDiagnosticPopup(buf, text, diag.Severity)
}

func (r *renderPass) drawDiagnosticPopup(
	buf *tui.Buffer, text string, severity view.DiagnosticSeverity,
) {
	maxW := min(buf.Width, 60)
	lines := diagnosticPopupLines(diagnosticPopupLinesArgs{
		text:     text,
		width:    max(maxW-4, 1),
		maxLines: 4,
	})
	if len(lines) == 0 {
		return
	}
	bodyW := 0
	for _, line := range lines {
		bodyW = max(bodyW, runewidth.StringWidth(line))
	}
	st := diagnosticPopupStyle(r.context, severity)
	pop := popup{
		borderStyle:  st,
		contentStyle: st,
		padX:         1,
	}
	w := min(bodyW+2+2*pop.padX, maxW)
	h := len(lines) + 2
	x := max(buf.Width-w, 0)
	y := 0
	if bufferlineVisible(r.context) {
		y = 1
	}
	if y+h > buf.Height {
		y = max(buf.Height-h, 0)
	}
	area := pop.drawInto(buf, geom.Area{
		X: x, Y: y, Width: w, Height: h,
	})
	for i, line := range lines {
		buf.SetString(area.Point.Add(geom.Point{Y: i}), line, st)
	}
}

func currentDiagnosticPopupKey(cx *Context) diagPopupKey {
	doc := cx.Editor.FocusedDocument()
	if doc == nil {
		return diagPopupKey{}
	}
	v := cx.Editor.FocusedView()
	if v == nil {
		return diagPopupKey{}
	}
	diag, ok := diagnosticAtCursor(doc, v)
	if !ok {
		return diagPopupKey{}
	}
	text := diagnosticPopupText(diag)
	if text == "" {
		return diagPopupKey{}
	}
	return diagPopupKey{severity: diag.Severity, text: text}
}

func diagnosticPopupStyle(
	cx *Context, severity view.DiagnosticSeverity,
) tui.Style {
	bg := cx.Theme().Get("ui.popup").BgColor()
	if severity <= 0 || int(severity) >= len(diagnosticPopupScopes) {
		return cx.Theme().Get("ui.popup")
	}
	scope := diagnosticPopupScopes[severity]
	st := cx.Theme().Get(scope)
	fg := st.FgColor()
	if fg.IsReset() {
		fg = st.UnderlineColor()
	}
	return tui.Style{}.Fg(fg).Bg(bg)
}

func diagnosticAtCursor(
	doc *view.Document, v *view.View,
) (view.Diagnostic, bool) {
	cursor := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	var best view.Diagnostic
	ok := false
	for _, diag := range doc.Diagnostics() {
		if diag.Message == "" {
			continue
		}
		bounds := diagnosticRangeBounds(diag)
		if cursor < bounds.From() || cursor >= bounds.To() {
			continue
		}
		if !ok || diag.Severity > best.Severity {
			best = diag
			ok = true
		}
	}
	return best, ok
}

func diagnosticRangeBounds(diag view.Diagnostic) core.Range {
	from := diag.Range.From
	to := diag.Range.To
	if from > to {
		from, to = to, from
	}
	if from == to {
		to++
	}
	return core.Range{Anchor: from, Head: to}
}

func diagnosticPopupText(diag view.Diagnostic) string {
	msg := DiagnosticMessageText(diag.Message)
	if diag.Source == "" {
		return msg
	}
	return diag.Source + ": " + msg
}

// DiagnosticMessageText flattens a server's message onto one line and
// shortens the qualified names in it
func DiagnosticMessageText(message string) string {
	lines := strings.FieldsFunc(message, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	return shortenQualifiedNames(strings.Join(lines, "  "))
}

func shortenQualifiedNames(message string) string {
	return qualifiedNamePattern.ReplaceAllStringFunc(message,
		func(name string) string {
			return name[strings.LastIndexByte(name, '/')+1:]
		},
	)
}

func diagnosticSpans(
	diags []view.Diagnostic, styles *styles,
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
	diags []diagnosticSpan, s core.Span,
) []diagnosticSpan {
	return filterLineItems(diags, lineItemBounds[diagnosticSpan]{
		before: func(d diagnosticSpan) bool { return d.to <= s.From },
		after:  func(d diagnosticSpan) bool { return d.from > s.To },
	})
}

func diagnosticStyle(
	severity view.DiagnosticSeverity, styles *styles,
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

type diagnosticPopupLinesArgs struct {
	text     string
	width    int
	maxLines int
}

func diagnosticPopupLines(args diagnosticPopupLinesArgs) []string {
	text := strings.TrimSpace(args.text)
	if text == "" || args.width <= 0 || args.maxLines <= 0 {
		return nil
	}
	var lines []string
	wrapped := core.ReflowHardWrap(text, args.width)
	for line := range strings.SplitSeq(wrapped, "\n") {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) == args.maxLines {
			break
		}
	}
	return lines
}
