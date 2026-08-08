package core

type (
	// Range is a selection span with an immovable anchor and movable head
	Range struct {
		Anchor int
		Head   int
	}

	// Span is a half-open interval of character offsets, [From, To)
	Span struct {
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

// PointRange returns an empty range at head
func PointRange(head int) Range {
	return Range{Anchor: head, Head: head}
}

// From is the lower of anchor and head
func (r Range) From() int {
	if r.Anchor < r.Head {
		return r.Anchor
	}
	return r.Head
}

// To is the higher of anchor and head
func (r Range) To() int {
	if r.Anchor > r.Head {
		return r.Anchor
	}
	return r.Head
}

// Len is the character count the range covers
func (r Range) Len() int {
	return r.To() - r.From()
}

// Span is the character interval the range covers, ignoring direction
func (r Range) Span() Span {
	return Span{From: r.From(), To: r.To()}
}

// Len is the character count the span covers
func (s Span) Len() int {
	return s.To - s.From
}

// Empty reports whether the span covers no characters
func (s Span) Empty() bool {
	return s.From == s.To
}

// Empty reports whether anchor and head coincide
func (r Range) Empty() bool {
	return r.Anchor == r.Head
}

// Direction reports which side of the range the head sits on
func (r Range) Direction() Direction {
	if r.Head < r.Anchor {
		return DirectionBackward
	}
	return DirectionForward
}

// Flip swaps anchor and head, reversing direction
func (r Range) Flip() Range {
	return Range{Anchor: r.Head, Head: r.Anchor}
}

// WithDirection flips the range only when it faces the other way
func (r Range) WithDirection(dir Direction) Range {
	if r.Direction() == dir {
		return r
	}
	return r.Flip()
}

// Overlaps reports whether the two ranges share any character
func (r Range) Overlaps(q Range) bool {
	return r.From() == q.From() || (r.To() > q.From() && q.To() > r.From())
}

// ContainsRange reports whether q falls entirely inside this range
func (r Range) ContainsRange(q Range) bool {
	return r.From() <= q.From() && r.To() >= q.To()
}

// Contains reports whether pos falls inside this range
func (r Range) Contains(pos int) bool {
	return r.From() <= pos && pos < r.To()
}

// LineSpan returns the inclusive line span the range touches. An empty range
// covers one line; a non-empty one excludes an end on a line start
func (r Range) LineSpan(text Rope) (Span, error) {
	from := r.From()
	to := r.To()
	if !r.Empty() {
		to--
	}
	start, err := text.CharToLine(from)
	if err != nil {
		return Span{}, err
	}
	end, err := text.CharToLine(to)
	if err != nil {
		return Span{}, err
	}
	return Span{From: start, To: end}, nil
}

// Extend grows the range to also cover the span, keeping its direction
func (r Range) Extend(s Span) Range {
	if r.Anchor <= r.Head {
		return Range{Anchor: min(r.Anchor, s.From), Head: max(r.Head, s.To)}
	}
	return Range{Anchor: max(r.Anchor, s.To), Head: min(r.Head, s.From)}
}

// Slice returns the rope sub-range covered by this range
func (r Range) Slice(doc Rope) (Rope, error) {
	return doc.Slice(r.Span())
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
		return Range{Anchor: pos, Head: pos}
	case r.Anchor < r.Head:
		return Range{
			Anchor: EnsureGraphemeBoundaryPrev(doc, r.Anchor),
			Head:   EnsureGraphemeBoundaryNext(doc, r.Head),
		}
	default:
		return Range{
			Anchor: EnsureGraphemeBoundaryNext(doc, r.Anchor),
			Head:   EnsureGraphemeBoundaryPrev(doc, r.Head),
		}
	}
}

// MinWidth1 ensures the range covers at least one grapheme by advancing the
// head forward if the range is empty
func (r Range) MinWidth1(doc Rope) Range {
	if r.Anchor != r.Head {
		return r
	}
	return Range{Anchor: r.Anchor, Head: NextGraphemeBoundary(doc, r.Head)}
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
		return Range{Anchor: anchor, Head: NextGraphemeBoundary(doc, charIdx)}
	}
	return Range{Anchor: anchor, Head: charIdx}
}

// CursorLine returns the line number that the block cursor is on
func (r Range) CursorLine(doc Rope) (int, error) {
	return doc.CharToLine(r.Cursor(doc))
}

// Merge returns the range spanning both, backward only when both are
func (r Range) Merge(q Range) Range {
	if r.Anchor > r.Head && q.Anchor > q.Head {
		return Range{Anchor: max(r.Anchor, q.Anchor), Head: min(r.Head, q.Head)}
	}
	return Range{Anchor: min(r.From(), q.From()), Head: max(r.To(), q.To())}
}
