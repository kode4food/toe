package ui

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	gutterSpec struct {
		layout          []view.GutterType
		lineNumberW     int
		width           int
		lineStyle       tui.Style
		lineSelected    tui.Style
		diagLines       map[int]view.DiagnosticSeverity
		diffLines       map[int]diffGutterKind
		severityHint    tui.Style
		severityInfo    tui.Style
		severityWarning tui.Style
		severityError   tui.Style
		diffAdded       tui.Style
		diffModified    tui.Style
		diffRemoved     tui.Style
	}

	diffGutterKind byte
)

const (
	diffGutterAdded diffGutterKind = iota
	diffGutterModified
	diffGutterRemoved
)

// diagnosticGutterMark is the dot drawn in the diagnostics gutter column
const diagnosticGutterMark = "●"

func (g gutterSpec) renderBlank(buf *tui.Buffer, at geom.Point) {
	buf.FillRange(at, g.width, g.lineStyle)
}

func (g gutterSpec) renderTilde(
	buf *tui.Buffer, at geom.Point, selected bool,
) {
	col := at.X
	st := g.lineStyleFor(selected)
	buf.FillRange(at, g.width, st)
	for _, gt := range g.layout {
		w := g.gutterTypeWidth(gt)
		if gt == view.GutterTypeLineNumbers && g.lineNumberW > 0 {
			buf.SetString(geom.Point{
				X: col + g.lineNumberW - 1,
				Y: at.Y,
			}, "~", g.lineStyleFor(selected))
		}
		col += w
	}
}

func (g gutterSpec) renderLine(
	buf *tui.Buffer, at geom.Point, lineNum, num int, selected bool,
) {
	col := at.X
	st := g.lineStyleFor(selected)
	buf.FillRange(at, g.width, st)
	for _, gt := range g.layout {
		w := g.gutterTypeWidth(gt)
		switch gt {
		case view.GutterTypeDiagnostics:
			if sev, ok := g.diagLines[lineNum]; ok {
				st := overlayDiagnosticStyle(
					g.lineStyleFor(selected), g.diagnosticStyle(sev),
				)
				buf.SetString(geom.Point{X: col, Y: at.Y}, diagnosticGutterMark, st)
			}
		case view.GutterTypeLineNumbers:
			if g.lineNumberW > 0 {
				buf.SetRightAlignedInt(
					geom.Point{X: col, Y: at.Y}, g.lineNumberW, num,
					g.lineStyleFor(selected),
				)
			}
		case view.GutterTypeDiff:
			if kind, ok := g.diffLines[lineNum]; ok {
				icon, st := g.diffMarker(kind)
				buf.SetString(
					geom.Point{X: col, Y: at.Y}, icon, overlayDiagnosticStyle(
						g.lineStyleFor(selected), st,
					),
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

func (g gutterSpec) diffMarker(kind diffGutterKind) (string, tui.Style) {
	switch kind {
	case diffGutterAdded:
		return "▍", g.diffAdded
	case diffGutterRemoved:
		return "▔", g.diffRemoved
	default:
		return "▍", g.diffModified
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

// diffGutterLines maps lines to markers; pure removals mark the line where
// they sit, clamped at end of file
func diffGutterLines(
	hunks []view.DiffHunk, nLines int,
) map[int]diffGutterKind {
	if len(hunks) == 0 {
		return nil
	}
	out := map[int]diffGutterKind{}
	for _, h := range hunks {
		if h.PureRemoval() {
			line := min(h.From, nLines-1)
			if _, ok := out[line]; !ok {
				out[line] = diffGutterRemoved
			}
			continue
		}
		kind := diffGutterModified
		if h.PureInsertion() {
			kind = diffGutterAdded
		}
		for line := h.From; line < h.To && line < nLines; line++ {
			out[line] = kind
		}
	}
	return out
}

func documentDiffLines(
	e *view.Editor, doc *view.Document, nLines int,
) map[int]diffGutterKind {
	vc := e.VersionControl()
	if vc == nil {
		return nil
	}
	return diffGutterLines(vc.DiffHunks(doc), nLines)
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
