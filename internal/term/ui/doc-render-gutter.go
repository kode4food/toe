package ui

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type gutterSpec struct {
	layout          []view.GutterType
	lineNumberW     int
	width           int
	lineStyle       tui.Style
	lineSelected    tui.Style
	diagLines       map[int]view.DiagnosticSeverity
	severityHint    tui.Style
	severityInfo    tui.Style
	severityWarning tui.Style
	severityError   tui.Style
}

func (g gutterSpec) renderBlank(buf *tui.Buffer, x, y int) {
	buf.FillRange(x, y, g.width, g.lineStyle)
}

func (g gutterSpec) renderTilde(buf *tui.Buffer, x, y int, selected bool) {
	col := x
	st := g.lineStyleFor(selected)
	buf.FillRange(x, y, g.width, st)
	for _, gt := range g.layout {
		w := g.gutterTypeWidth(gt)
		if gt == view.GutterTypeLineNumbers && g.lineNumberW > 0 {
			buf.SetString(col+g.lineNumberW-1, y, "~", g.lineStyleFor(selected))
		}
		col += w
	}
}

func (g gutterSpec) renderLine(
	buf *tui.Buffer, x, y, lineNum, num int, selected bool,
) {
	col := x
	st := g.lineStyleFor(selected)
	buf.FillRange(x, y, g.width, st)
	for _, gt := range g.layout {
		w := g.gutterTypeWidth(gt)
		switch gt {
		case view.GutterTypeDiagnostics:
			if sev, ok := g.diagLines[lineNum]; ok {
				st := overlayDiagnosticStyle(
					g.lineStyleFor(selected), g.diagnosticStyle(sev),
				)
				buf.SetString(col, y, "\u25cf", st) // ●
			}
		case view.GutterTypeLineNumbers:
			if g.lineNumberW > 0 {
				buf.SetRightAlignedInt(
					col, y, g.lineNumberW, num, g.lineStyleFor(selected),
				)
			}
		}
		col += w
	}
}

func (g gutterSpec) lineStyleFor(selected bool) tui.Style {
	if selected {
		return g.lineSelected
	}
	return g.lineStyle
}

func (g gutterSpec) gutterTypeWidth(gt view.GutterType) int {
	if gt == view.GutterTypeLineNumbers {
		return g.lineNumberW
	}
	return 1
}

func (g gutterSpec) diagnosticStyle(
	severity view.DiagnosticSeverity,
) tui.Style {
	switch severity {
	case view.DiagnosticSeverityError:
		return g.severityError
	case view.DiagnosticSeverityWarning:
		return g.severityWarning
	case view.DiagnosticSeverityInfo:
		return g.severityInfo
	default:
		return g.severityHint
	}
}

func gutterLineNumberWidth(
	text core.Rope, g view.Gutter, layout []view.GutterType,
) int {
	if !gutterLayoutHas(layout, view.GutterTypeLineNumbers) {
		return 0
	}
	return max(lineNumberDigits(text), g.LineNumberMinWidth())
}

func gutterLayoutHas(layout []view.GutterType, gt view.GutterType) bool {
	return slices.Contains(layout, gt)
}

func gutterLayoutWidth(layout []view.GutterType, lineNumberW int) int {
	w := 0
	for _, gt := range layout {
		if gt == view.GutterTypeLineNumbers {
			w += lineNumberW
		} else {
			w++
		}
	}
	return w
}

func diagnosticGutterLines(
	text core.Rope, diags []view.Diagnostic,
) map[int]view.DiagnosticSeverity {
	if len(diags) == 0 {
		return nil
	}
	out := map[int]view.DiagnosticSeverity{}
	for _, diag := range diags {
		from := min(diag.Range.To, diag.Range.From)
		line, err := text.CharToLine(from)
		if err != nil {
			continue
		}
		if diag.Severity > out[line] {
			out[line] = diag.Severity
		}
	}
	return out
}
