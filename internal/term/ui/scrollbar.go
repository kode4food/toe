package ui

import (
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	scrollbar struct {
		styles  *styles
		marks   []scrollMark
		at      geom.Point
		geom    scrollbarGeom
		topLine int
	}

	// scrollbarGeom maps between bar rows and document lines
	scrollbarGeom struct {
		rows  int
		lines int
	}

	// count is how many document lines the half row marks, and line the last
	// one counted, so a line marked twice still counts once
	scrollMark struct {
		kind  scrollMarkKind
		line  int
		count int
	}

	// ordered by priority, the highest kind on a half row is the one drawn
	scrollMarkKind uint8
)

const (
	scrollMarkNone scrollMarkKind = iota
	scrollMarkSearch
	scrollMarkDiffAdded
	scrollMarkDiffModified
	scrollMarkDiffRemoved
	scrollMarkHint
	scrollMarkInfo
	scrollMarkWarning
	scrollMarkError
	scrollMarkCursor
)

const (
	scrollbarFullGlyph  = "\u2588" // '█' - full block
	scrollbarUpperGlyph = "\u2580" // '▀' - upper half block
	scrollbarLowerGlyph = "\u2584" // '▄' - lower half block
	scrollbarUpperThin  = "\u2594" // '▔' - upper one eighth block
	scrollbarLowerThin  = "\u2581" // '▁' - lower one eighth block
)

func newScrollbar(
	g scrollbarGeom, at geom.Point, styles *styles, topLine int,
) *scrollbar {
	// two marks per row, so a half block can place one precisely
	marks := make([]scrollMark, max(g.rows, 0)*2)
	for i := range marks {
		marks[i].line = -1
	}
	return &scrollbar{
		styles:  styles,
		marks:   marks,
		at:      at,
		geom:    g,
		topLine: topLine,
	}
}

func (s *scrollbar) addMark(line int, kind scrollMarkKind) {
	g := s.geom
	if line < 0 || line >= g.lines || g.rows <= 0 {
		return
	}
	half := line * 2
	if g.lines > g.rows {
		half = min(line*len(s.marks)/g.lines, len(s.marks)-1)
	}
	mark := &s.marks[half]
	if kind > mark.kind {
		mark.kind = kind
	}
	if mark.line != line {
		mark.line = line
		mark.count++
	}
}

// the matches are sorted, so their lines are walked forward rather than
// searched for one by one
func (s *scrollbar) addSearchMarks(
	matches []matchSpan, lineIdx []lineIndexEntry,
) {
	line := 0
	for _, m := range matches {
		for line+1 < len(lineIdx) && lineIdx[line+1].charStart <= m.from {
			line++
		}
		s.addMark(line, scrollMarkSearch)
	}
}

func (s *scrollbar) draw(buf *tui.Buffer) {
	g := s.geom
	if g.rows <= 0 {
		return
	}
	thumbLen := g.thumbLen()
	thumbTop := 0
	if g.lines > g.rows {
		thumbTop = min(
			s.topLine*(g.rows-thumbLen)/(g.lines-g.rows), g.rows-thumbLen,
		)
	}
	for row := range g.rows {
		base := s.styles.scrollTrack
		if row >= thumbTop && row < thumbTop+thumbLen {
			base = s.styles.scrollThumb
		}
		glyph, st := s.cell(base, row)
		buf.SetString(s.at.Add(geom.Point{Y: row}), glyph, st)
	}
}

// one cell carries both marks of its row, so neighboring changes stay apart
// and a half standing for a single line draws as a rule, not a block
func (s *scrollbar) cell(base tui.Style, row int) (string, tui.Style) {
	upper := s.marks[row*2]
	lower := s.marks[row*2+1]
	switch {
	case upper.kind == scrollMarkNone && lower.kind == scrollMarkNone:
		return " ", base
	case lower.kind == scrollMarkNone:
		return scrollbarHalfGlyph(upper, true), base.Fg(s.markColor(upper.kind))
	case upper.kind == scrollMarkNone:
		return scrollbarHalfGlyph(lower, false),
			base.Fg(s.markColor(lower.kind))
	case upper.kind == lower.kind:
		return scrollbarFullGlyph, base.Fg(s.markColor(upper.kind))
	default:
		return scrollbarUpperGlyph,
			base.Fg(s.markColor(upper.kind)).Bg(s.markColor(lower.kind))
	}
}

func (s *scrollbar) markColor(kind scrollMarkKind) tui.Color {
	st := s.styles
	switch kind {
	case scrollMarkCursor:
		return st.scrollCursor
	case scrollMarkError:
		return st.severityError.FgColor()
	case scrollMarkWarning:
		return st.severityWarning.FgColor()
	case scrollMarkInfo:
		return st.severityInfo.FgColor()
	case scrollMarkHint:
		return st.severityHint.FgColor()
	case scrollMarkDiffRemoved:
		return st.diffRemoved.FgColor()
	case scrollMarkDiffModified:
		return st.diffModified.FgColor()
	case scrollMarkDiffAdded:
		return st.diffAdded.FgColor()
	default:
		return st.scrollSearch
	}
}

// topLine inverts the thumb placement in draw, so the thumb lands where the
// bar was clicked
func (s scrollbarGeom) topLine(row int) int {
	if s.lines <= s.rows {
		return 0
	}
	span := s.rows - s.thumbLen()
	if span <= 0 {
		return 0
	}
	return min(row*(s.lines-s.rows)/span, s.lines-s.rows)
}

func (s scrollbarGeom) thumbLen() int {
	if s.lines <= s.rows {
		return s.rows
	}
	return max(s.rows*s.rows/s.lines, 1)
}

func scrollbarHalfGlyph(mark scrollMark, upper bool) string {
	switch {
	case upper && mark.count > 1:
		return scrollbarUpperGlyph
	case upper:
		return scrollbarUpperThin
	case mark.count > 1:
		return scrollbarLowerGlyph
	default:
		return scrollbarLowerThin
	}
}

func diffMarkKind(kind diffGutterKind) scrollMarkKind {
	switch kind {
	case diffGutterAdded:
		return scrollMarkDiffAdded
	case diffGutterRemoved:
		return scrollMarkDiffRemoved
	default:
		return scrollMarkDiffModified
	}
}

func severityMarkKind(sev view.DiagnosticSeverity) scrollMarkKind {
	switch sev {
	case view.DiagnosticSeverityError:
		return scrollMarkError
	case view.DiagnosticSeverityWarning:
		return scrollMarkWarning
	case view.DiagnosticSeverityInfo:
		return scrollMarkInfo
	default:
		return scrollMarkHint
	}
}
