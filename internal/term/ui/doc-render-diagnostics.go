package ui

import (
	"cmp"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

var diagnosticPopupScopes = [...]string{
	view.DiagnosticSeverityHint:    "diagnostic.hint",
	view.DiagnosticSeverityInfo:    "diagnostic.info",
	view.DiagnosticSeverityWarning: "diagnostic.warning",
	view.DiagnosticSeverityError:   "diagnostic.error",
}

func (r *renderPass) renderDiagnosticPopup(buf *tui.Buffer) {
	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
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
	lines := diagnosticPopupLines(text, max(maxW-4, 1), 4)
	if len(lines) == 0 {
		return
	}
	bodyW := 0
	for _, line := range lines {
		bodyW = max(bodyW, runewidth.StringWidth(line))
	}
	st := diagnosticPopupStyle(r.cx, severity)
	pop := popup{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  st,
		contentStyle: st,
		padX:         1,
	}
	w := min(bodyW+2+2*pop.padX, maxW)
	h := len(lines) + 2
	x := max(buf.Width-w, 0)
	y := 0
	if bufferlineVisible(r.cx) {
		y = 1
	}
	if y+h > buf.Height {
		y = max(buf.Height-h, 0)
	}
	area := pop.drawInto(buf, geom.Area{
		Point: geom.Point{X: x, Y: y},
		Size:  geom.Size{Width: w, Height: h},
	})
	for i, line := range lines {
		buf.SetString(area.Point.Add(geom.Point{Y: i}), line, st)
	}
}

func currentDiagnosticPopupKey(cx *Context) diagPopupKey {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return diagPopupKey{}
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
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
	bg := styleToTUI(cx.Theme().Get("ui.popup")).BgColor()
	if severity <= 0 || int(severity) >= len(diagnosticPopupScopes) {
		return styleToTUI(cx.Theme().Get("ui.popup"))
	}
	scope := diagnosticPopupScopes[severity]
	st := styleToTUI(cx.Theme().Get(scope))
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
		from, to := diagnosticRangeBounds(diag)
		if cursor < from || cursor >= to || diag.Message == "" {
			continue
		}
		if !ok || diag.Severity > best.Severity {
			best = diag
			ok = true
		}
	}
	return best, ok
}

func diagnosticRangeBounds(diag view.Diagnostic) (int, int) {
	from := diag.Range.From
	to := diag.Range.To
	if from > to {
		from, to = to, from
	}
	if from == to {
		to++
	}
	return from, to
}

func diagnosticPopupText(diag view.Diagnostic) string {
	msg := diagnosticMessageText(diag.Message)
	if diag.Source == "" {
		return msg
	}
	return diag.Source + ": " + msg
}

func diagnosticMessageText(message string) string {
	lines := strings.FieldsFunc(message, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	return strings.Join(lines, "  ")
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
	return filterLineItems(diags,
		func(d diagnosticSpan) bool { return d.to <= from },
		func(d diagnosticSpan) bool { return d.from > to },
	)
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

func diagnosticPopupLines(text string, width, maxLines int) []string {
	text = strings.TrimSpace(text)
	if text == "" || width <= 0 || maxLines <= 0 {
		return nil
	}
	var lines []string
	for line := range strings.SplitSeq(lipgloss.Wrap(text, width, ""), "\n") {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) == maxLines {
			break
		}
	}
	return lines
}
