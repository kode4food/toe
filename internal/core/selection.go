package core

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

type (
	// Direction describes whether a range head is before or after its anchor
	Direction int

	// Range is a selection span with an immovable anchor and movable head
	Range struct {
		Anchor            int
		Head              int
		OldVisualPosition *Position
	}

	// Selection is a non-empty ordered set of ranges with one primary range
	Selection struct {
		ranges       []Range
		primaryIndex int
	}

	// LineRange is an inclusive range of document lines
	LineRange struct {
		From int
		To   int
	}
)

const (
	DirectionBackward Direction = iota + 1
	DirectionForward
)

var (
	ErrEmptySelection       = errors.New("empty selection")
	ErrPrimaryIndexNotFound = errors.New("primary index not found")
	ErrLastRangeRemoval     = errors.New("last range removal")
	ErrRangeIndexNotFound   = errors.New("range index not found")
)

func NewRange(anchor, head int) Range {
	return Range{Anchor: anchor, Head: head}
}

func PointRange(head int) Range {
	return NewRange(head, head)
}

func NewSelection(ranges []Range, primaryIndex int) (Selection, error) {
	if len(ranges) == 0 {
		return Selection{}, ErrEmptySelection
	}
	if primaryIndex < 0 || primaryIndex >= len(ranges) {
		return Selection{}, fmt.Errorf("%w: %d", ErrPrimaryIndexNotFound,
			primaryIndex)
	}
	s := Selection{
		ranges:       slices.Clone(ranges),
		primaryIndex: primaryIndex,
	}
	return s.normalize(), nil
}

func SingleSelection(anchor, head int) Selection {
	return Selection{
		ranges:       []Range{NewRange(anchor, head)},
		primaryIndex: 0,
	}
}

func PointSelection(pos int) Selection {
	return SingleSelection(pos, pos)
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

func (s Selection) Primary() Range {
	return s.ranges[s.primaryIndex]
}

func (s Selection) Ranges() []Range {
	return slices.Clone(s.ranges)
}

func (s Selection) PrimaryIndex() int {
	return s.primaryIndex
}

func (s Selection) SetPrimaryIndex(idx int) (Selection, error) {
	if idx < 0 || idx >= len(s.ranges) {
		return Selection{}, fmt.Errorf("%w: %d", ErrPrimaryIndexNotFound,
			idx)
	}
	s.primaryIndex = idx
	return s, nil
}

func (s Selection) IntoSingle() Selection {
	if len(s.ranges) == 1 {
		return s
	}
	return Selection{ranges: []Range{s.Primary()}, primaryIndex: 0}
}

func (s Selection) Push(r Range) Selection {
	s.ranges = append(s.Ranges(), r)
	s.primaryIndex = len(s.ranges) - 1
	return s.normalize()
}

func (s Selection) Remove(idx int) (Selection, error) {
	if len(s.ranges) == 1 {
		return Selection{}, ErrLastRangeRemoval
	}
	if idx < 0 || idx >= len(s.ranges) {
		return Selection{}, fmt.Errorf("%w: %d", ErrRangeIndexNotFound, idx)
	}
	s.ranges = append(s.Ranges()[:idx], s.Ranges()[idx+1:]...)
	if idx < s.primaryIndex || s.primaryIndex == len(s.ranges) {
		s.primaryIndex--
	}
	return s, nil
}

func (s Selection) Replace(idx int, r Range) (Selection, error) {
	if idx < 0 || idx >= len(s.ranges) {
		return Selection{}, fmt.Errorf("%w: %d", ErrRangeIndexNotFound, idx)
	}
	s.ranges = s.Ranges()
	s.ranges[idx] = r
	return s.normalize(), nil
}

func (s Selection) MergeRanges() Selection {
	first := s.ranges[0]
	last := s.ranges[len(s.ranges)-1]
	return Selection{ranges: []Range{first.Merge(last)}, primaryIndex: 0}
}

func (s Selection) MergeConsecutiveRanges() Selection {
	sel := s.normalize()
	if len(sel.ranges) < 2 {
		return sel
	}
	primary := sel.Primary()
	out := []Range{sel.ranges[0]}
	for _, r := range sel.ranges[1:] {
		prev := out[len(out)-1]
		if prev.To() != r.From() {
			out = append(out, r)
			continue
		}
		merged := prev.Merge(r)
		if prev == primary || r == primary {
			primary = merged
		}
		out[len(out)-1] = merged
	}
	sel.ranges = out
	sel.primaryIndex = indexRange(out, primary)
	return sel
}

func (s Selection) Transform(f func(Range) Range) Selection {
	ranges := s.Ranges()
	for i, r := range ranges {
		ranges[i] = f(r)
	}
	s.ranges = ranges
	return s.normalize()
}

func (s Selection) Map(cs ChangeSet) (Selection, error) {
	ranges := s.Ranges()
	for i, r := range ranges {
		mapped, err := cs.MapRange(r)
		if err != nil {
			return Selection{}, err
		}
		ranges[i] = mapped
	}
	s.ranges = ranges
	return s.normalize(), nil
}

func (s Selection) LineRanges(text Rope) ([]LineRange, error) {
	out := make([]LineRange, 0, len(s.ranges))
	for _, r := range s.ranges {
		lr, err := r.LineRange(text)
		if err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].To+1 < lr.From {
			out = append(out, lr)
			continue
		}
		if lr.To > out[len(out)-1].To {
			out[len(out)-1].To = lr.To
		}
	}
	return out, nil
}

func (s Selection) normalize() Selection {
	if len(s.ranges) < 2 {
		return s
	}
	primary := s.Primary()
	slices.SortFunc(s.ranges, func(a, b Range) int {
		return cmp.Compare(a.From(), b.From())
	})

	out := []Range{s.ranges[0]}
	for _, r := range s.ranges[1:] {
		prev := out[len(out)-1]
		if !prev.Overlaps(r) {
			out = append(out, r)
			continue
		}
		merged := prev.Merge(r)
		if prev == primary || r == primary {
			primary = merged
		}
		out[len(out)-1] = merged
	}
	s.ranges = out
	s.primaryIndex = indexRange(out, primary)
	return s
}

func indexRange(ranges []Range, target Range) int {
	for i, r := range ranges {
		if r == target {
			return i
		}
	}
	return 0
}
