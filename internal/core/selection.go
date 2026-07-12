package core

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

// Selection is a non-empty ordered set of ranges with one primary range
type Selection struct {
	ranges       []Range
	primaryIndex int
}

var (
	ErrEmptySelection       = errors.New("empty selection")
	ErrPrimaryIndexNotFound = errors.New("primary index not found")
	ErrLastRangeRemoval     = errors.New("last range removal")
	ErrRangeIndexNotFound   = errors.New("range index not found")
)

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

func (s Selection) Primary() Range {
	return s.ranges[s.primaryIndex]
}

func (s Selection) Ranges() []Range {
	return slices.Clone(s.ranges)
}

func (s Selection) PrimaryIndex() int {
	return s.primaryIndex
}

// Equal reports whether two selections have identical ranges and the same
// primary index
func (s Selection) Equal(other Selection) bool {
	return s.primaryIndex == other.primaryIndex &&
		slices.Equal(s.ranges, other.ranges)
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
	s.ranges = slices.Delete(s.Ranges(), idx, idx+1)
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
	if idx := slices.Index(ranges, target); idx >= 0 {
		return idx
	}
	return 0
}
