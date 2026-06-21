package core

type (
	// Range is a selection span with an immovable anchor and movable head
	Range struct {
		Anchor            int
		Head              int
		OldVisualPosition *Position
	}

	// LineRange is an inclusive range of document lines
	LineRange struct {
		From int
		To   int
	}

	// Direction describes whether a range head is before or after its anchor
	Direction int
)

const (
	DirectionBackward Direction = iota + 1
	DirectionForward
)

func NewRange(anchor, head int) Range {
	return Range{Anchor: anchor, Head: head}
}

func PointRange(head int) Range {
	return NewRange(head, head)
}

func (r Range) From() int {
	if r.Anchor < r.Head {
		return r.Anchor
	}
	return r.Head
}

func (r Range) To() int {
	if r.Anchor > r.Head {
		return r.Anchor
	}
	return r.Head
}

func (r Range) Len() int {
	return r.To() - r.From()
}

func (r Range) Empty() bool {
	return r.Anchor == r.Head
}

func (r Range) Direction() Direction {
	if r.Head < r.Anchor {
		return DirectionBackward
	}
	return DirectionForward
}

func (r Range) Flip() Range {
	return Range{
		Anchor:            r.Head,
		Head:              r.Anchor,
		OldVisualPosition: r.OldVisualPosition,
	}
}

func (r Range) WithDirection(dir Direction) Range {
	if r.Direction() == dir {
		return r
	}
	return r.Flip()
}

func (r Range) Overlaps(q Range) bool {
	return r.From() == q.From() || (r.To() > q.From() && q.To() > r.From())
}

func (r Range) ContainsRange(q Range) bool {
	return r.From() <= q.From() && r.To() >= q.To()
}

func (r Range) Contains(pos int) bool {
	return r.From() <= pos && pos < r.To()
}

func (r Range) LineRange(text Rope) (LineRange, error) {
	from := r.From()
	to := r.To()
	if !r.Empty() {
		to--
	}
	start, err := text.CharToLine(from)
	if err != nil {
		return LineRange{}, err
	}
	end, err := text.CharToLine(to)
	if err != nil {
		return LineRange{}, err
	}
	return LineRange{From: start, To: end}, nil
}

func (r Range) Extend(from, to int) Range {
	if r.Anchor <= r.Head {
		return Range{Anchor: min(r.Anchor, from), Head: max(r.Head, to)}
	}
	return Range{Anchor: max(r.Anchor, to), Head: min(r.Head, from)}
}

// Slice returns the rope sub-range covered by this range
func (r Range) Slice(doc Rope) (Rope, error) {
	return doc.Slice(r.From(), r.To())
}

// Fragment returns the text covered by this range as a string
func (r Range) Fragment(doc Rope) (string, error) {
	s, err := r.Slice(doc)
	if err != nil {
		return "", err
	}
	return s.String(), nil
}

// GraphemeAligned snaps both ends of the range to grapheme cluster boundaries
func (r Range) GraphemeAligned(doc Rope) Range {
	switch {
	case r.Anchor == r.Head:
		pos := EnsureGraphemeBoundaryPrev(doc, r.Anchor)
		return Range{
			Anchor:            pos,
			Head:              pos,
			OldVisualPosition: r.OldVisualPosition,
		}
	case r.Anchor < r.Head:
		a := EnsureGraphemeBoundaryPrev(doc, r.Anchor)
		h := EnsureGraphemeBoundaryNext(doc, r.Head)
		ovp := r.OldVisualPosition
		if a != r.Anchor {
			ovp = nil
		}
		return Range{Anchor: a, Head: h, OldVisualPosition: ovp}
	default:
		a := EnsureGraphemeBoundaryNext(doc, r.Anchor)
		h := EnsureGraphemeBoundaryPrev(doc, r.Head)
		ovp := r.OldVisualPosition
		if a != r.Anchor {
			ovp = nil
		}
		return Range{Anchor: a, Head: h, OldVisualPosition: ovp}
	}
}

// MinWidth1 ensures the range covers at least one grapheme by advancing the
// head forward if the range is empty
func (r Range) MinWidth1(doc Rope) Range {
	if r.Anchor != r.Head {
		return r
	}
	return Range{
		Anchor:            r.Anchor,
		Head:              NextGraphemeBoundary(doc, r.Head),
		OldVisualPosition: r.OldVisualPosition,
	}
}

// IsSingleGrapheme reports whether this range covers exactly one grapheme
// cluster
func (r Range) IsSingleGrapheme(doc Rope) bool {
	if r.From() >= r.To() {
		return false
	}
	first := NextGraphemeBoundary(doc, r.From())
	return first >= r.To()
}

// Cursor returns the char index of the block-cursor position. For a forward
// range the cursor sits one grapheme before the head; for backward or empty
// ranges it is the head itself
func (r Range) Cursor(doc Rope) int {
	if r.Head > r.Anchor {
		return PrevGraphemeBoundary(doc, r.Head)
	}
	return r.Head
}

// PutCursor moves the block cursor to charIdx, optionally extending the
// selection anchor using 1-width block cursor semantics
func (r Range) PutCursor(doc Rope, charIdx int, extend bool) Range {
	if !extend {
		return PointRange(charIdx)
	}
	anchor := r.Anchor
	if r.Head >= r.Anchor && charIdx < r.Anchor {
		anchor = NextGraphemeBoundary(doc, r.Anchor)
	} else if r.Head < r.Anchor && charIdx >= r.Anchor {
		anchor = PrevGraphemeBoundary(doc, r.Anchor)
	}
	if anchor <= charIdx {
		return NewRange(anchor, NextGraphemeBoundary(doc, charIdx))
	}
	return NewRange(anchor, charIdx)
}

// CursorLine returns the line number that the block cursor is on
func (r Range) CursorLine(doc Rope) (int, error) {
	return doc.CharToLine(r.Cursor(doc))
}

func (r Range) Merge(q Range) Range {
	if r.Anchor > r.Head && q.Anchor > q.Head {
		return Range{Anchor: max(r.Anchor, q.Anchor), Head: min(r.Head, q.Head)}
	}
	return Range{Anchor: min(r.From(), q.From()), Head: max(r.To(), q.To())}
}
