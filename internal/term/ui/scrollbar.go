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

	scrollbarGeom struct {
		rows   int
		maxTop int
	}

	scrollMark struct {
		kind     scrollMarkKind
		lastLine int
		count    int
	}

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
	scrollbarMidThin    = "\u2500" // '─' - box drawings light horizontal
	scrollbarLowerThin  = "\u2581" // '▁' - lower one eighth block
)

const (
	scrollbarThumbTint = 0.40
	scrollbarMarkSlots = 3
)

func newScrollbar(
	g scrollbarGeom, at geom.Point, styles *styles, topLine int,
) *scrollbar {
	return &scrollbar{
		styles:  styles,
		marks:   make([]scrollMark, max(g.rows, 0)*scrollbarMarkSlots),
		at:      at,
		geom:    g,
		topLine: topLine,
	}
}

func (s *scrollbar) addMark(line int, kind scrollMarkKind) {
	g := s.geom
	if line < 0 || line >= g.scrollLines() || g.rows <= 0 {
		return
	}
	half := min(line*len(s.marks)/g.scrollLines(), len(s.marks)-1)
	mark := &s.marks[half]
	if kind > mark.kind {
		mark.kind = kind
	}
	if mark.count == 0 || mark.lastLine != line {
		mark.lastLine = line
		mark.count++
	}
}

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
	thumbTop := g.thumbTop(s.topLine)
	thumbEnd := g.thumbEnd(s.topLine)
	for row := range g.rows {
		glyph, st := s.cell(row)
		if row >= thumbTop && row < thumbEnd {
			st = s.tint(st)
		}
		buf.SetString(s.at.Add(geom.Point{Y: row}), glyph, st)
	}
}

func (s *scrollbar) tint(st tui.Style) tui.Style {
	accent := s.styles.scrollThumb.BgColor()
	shift := func(c tui.Color) tui.Color {
		return tintToward(&tintColors{
			base:   c,
			accent: accent,
			amount: scrollbarThumbTint,
		})
	}
	return st.Fg(shift(st.FgColor())).Bg(shift(st.BgColor()))
}

func (s *scrollbar) cell(row int) (string, tui.Style) {
	slots := s.marks[row*scrollbarMarkSlots:][:scrollbarMarkSlots]
	kind := scrollMarkNone
	args := scrollbarGlyphArgs{}
	for i, mark := range slots {
		if mark.kind == scrollMarkNone {
			continue
		}
		kind = max(kind, mark.kind)
		args.slotsFilled++
		args.lineCount += mark.count
		args.slot = i
	}
	base := s.styles.scrollTrack
	if args.slotsFilled == 0 {
		return " ", base
	}
	return scrollbarGlyph(args), base.Fg(s.markColor(kind))
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

func (s scrollbarGeom) thumbTop(topLine int) int {
	if s.maxTop <= 0 {
		return 0
	}
	return min(topLine*s.rows/s.scrollLines(), s.rows-1)
}

func (s scrollbarGeom) thumbEnd(topLine int) int {
	if s.maxTop <= 0 {
		return s.rows
	}
	lines := s.scrollLines()
	end := ((topLine+s.rows)*s.rows + lines - 1) / lines
	return min(max(end, s.thumbTop(topLine)+1), s.rows)
}

func (s scrollbarGeom) topLineAt(row int) int {
	if s.maxTop <= 0 {
		return 0
	}
	target := max(row-s.thumbLen()/2, 0)
	top := (target*s.scrollLines() + s.rows - 1) / s.rows
	return min(top, s.maxTop)
}

func (s scrollbarGeom) thumbLen() int {
	if s.maxTop <= 0 {
		return s.rows
	}
	return max(s.rows*s.rows/s.scrollLines(), 1)
}

func (s scrollbarGeom) scrollLines() int {
	return s.maxTop + s.rows
}

type scrollbarGlyphArgs struct {
	slotsFilled int
	lineCount   int
	slot        int
}

func scrollbarGlyph(args scrollbarGlyphArgs) string {
	if args.slotsFilled > 1 || args.lineCount > 1 {
		switch {
		case args.slotsFilled > 1:
			return scrollbarFullGlyph
		case args.slot == 0:
			return scrollbarUpperGlyph
		case args.slot == scrollbarMarkSlots-1:
			return scrollbarLowerGlyph
		}
		return scrollbarFullGlyph
	}
	switch args.slot {
	case 0:
		return scrollbarUpperThin
	case scrollbarMarkSlots - 1:
		return scrollbarLowerThin
	}
	return scrollbarMidThin
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
