package ui

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	scrollbar struct {
		styles  *styles
		marks   []scrollMarkKind
		at      geom.Point
		geom    scrollbarGeom
		topLine int
	}

	scrollbarGeom struct {
		rows   int
		maxTop int
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

	scrollbarFullGlyph  = "\u2588" // '█' - full block
	scrollbarUpperGlyph = "\u2580" // '▀' - upper half block
	scrollbarLowerGlyph = "\u2584" // '▄' - lower half block
	scrollbarUpperThin  = "\u2594" // '▔' - upper one eighth block
	scrollbarLowerThin  = "\u2581" // '▁' - lower one eighth block

	scrollbarThumbTint = 0.40
	// the image draws a pixel row per slot, the glyph bar quantizes them
	scrollbarSlotsPerCell = 48
)

func (s *scrollbar) reset(
	g scrollbarGeom, at geom.Point, styles *styles, topLine int,
) {
	n := max(g.rows, 0) * scrollbarSlotsPerCell
	s.marks = slices.Grow(s.marks, max(n-len(s.marks), 0))[:n]
	clear(s.marks)
	s.styles = styles
	s.at = at
	s.geom = g
	s.topLine = topLine
}

func (s *scrollbar) addMark(line int, kind scrollMarkKind) {
	g := s.geom
	if line < 0 || line >= g.scrollLines() || g.rows <= 0 {
		return
	}
	from := g.slotAt(line)
	to := from + 1
	if kind.isDiff() {
		to = max(to, g.slotEnd(line))
	}
	for slot := from; slot < to; slot++ {
		s.marks[slot] = max(s.marks[slot], kind)
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

func (s *scrollbar) addDocumentMarks(doc *docRenderCache) {
	s.addSearchMarks(doc.searchSpans, doc.lineIndex)
	for line, kind := range doc.diffLines {
		s.addMark(line, diffMarkKind(kind))
	}
	for line, severity := range doc.diagLines {
		s.addMark(line, severityMarkKind(severity))
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
		glyph, kind := s.cell(row)
		st := s.styles.scrollTrack[kind]
		if row >= thumbTop && row < thumbEnd {
			st = s.styles.scrollThumb[kind]
		}
		buf.Set(s.at.Add(geom.Point{Y: row}), tui.Cell{
			Symbol: glyph,
			Style:  st,
		})
	}
}

func (s *scrollbar) cell(row int) (string, scrollMarkKind) {
	slots := s.marks[row*scrollbarSlotsPerCell:][:scrollbarSlotsPerCell]
	kind := scrollMarkNone
	args := scrollbarGlyphArgs{firstSlot: -1, lastSlot: -1}
	for i, mark := range slots {
		if mark == scrollMarkNone {
			continue
		}
		kind = max(kind, mark)
		if args.firstSlot < 0 {
			args.firstSlot = i
		}
		args.lastSlot = i
	}
	if args.firstSlot < 0 {
		return " ", kind
	}
	return scrollbarGlyph(args), kind
}

func (s scrollbarGeom) thumbTop(topLine int) int {
	if s.maxTop <= 0 {
		return 0
	}
	return s.thumbEnd(topLine) - s.thumbLen()
}

func (s scrollbarGeom) thumbEnd(topLine int) int {
	if s.maxTop <= 0 {
		return s.rows
	}
	end := s.slotAt(topLine+s.rows-1)/scrollbarSlotsPerCell + 1
	return min(max(end, s.thumbLen()), s.rows)
}

func (s scrollbarGeom) topLineAt(row int) int {
	if s.maxTop <= 0 {
		return 0
	}
	wanted := row - s.thumbLen()/2
	if wanted <= 0 {
		return 0
	}
	end := min(wanted+s.thumbLen(), s.rows)
	last := ((end-1)*s.scrollLines() + s.rows - 1) / s.rows
	return min(max(last-s.rows+1, 0), s.maxTop)
}

// grown to the cell bar's length without favoring either end
func (s scrollbarGeom) thumbSpan(topLine int) core.Span {
	length := s.thumbLen() * scrollbarSlotsPerCell
	if s.maxTop <= 0 {
		return core.Span{From: 0, To: length}
	}
	from := s.slotAt(topLine)
	if grow := length - (s.slotEnd(topLine+s.rows-1) - from); grow > 0 {
		from -= grow / 2
	}
	from = min(max(from, 0), s.slots()-length)
	return core.Span{From: from, To: from + length}
}

func (s scrollbarGeom) thumbLen() int {
	if s.maxTop <= 0 {
		return s.rows
	}
	lines := s.scrollLines()
	return min((s.rows*(s.rows-1)+lines-1)/lines+1, s.rows)
}

func (s scrollbarGeom) slotAt(line int) int {
	slots := s.slots()
	return min(line*slots/s.scrollLines(), slots-1)
}

func (s scrollbarGeom) slotEnd(line int) int {
	slots := s.slots()
	return min((line+1)*slots/s.scrollLines(), slots)
}

func (s scrollbarGeom) slots() int {
	return s.rows * scrollbarSlotsPerCell
}

func (s scrollbarGeom) scrollLines() int {
	return s.maxTop + s.rows
}

func (k scrollMarkKind) isDiff() bool {
	return k >= scrollMarkDiffAdded && k <= scrollMarkDiffRemoved
}

type scrollbarGlyphArgs struct {
	firstSlot int
	lastSlot  int
}

func scrollbarGlyph(args scrollbarGlyphArgs) string {
	last := scrollbarSlotsPerCell - 1
	// a small mark says only where it is, so its weight must not wander
	if (args.lastSlot-args.firstSlot+1)*2 < scrollbarSlotsPerCell {
		if args.firstSlot+args.lastSlot >= last {
			return scrollbarLowerThin
		}
		return scrollbarUpperThin
	}
	switch {
	case args.firstSlot == 0 && args.lastSlot == last:
		return scrollbarFullGlyph
	case args.lastSlot == last:
		return scrollbarLowerGlyph
	case args.firstSlot == 0:
		return scrollbarUpperGlyph
	}
	return scrollbarFullGlyph
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
