package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestRange(t *testing.T) {
	t.Run("reports bounds and length", func(t *testing.T) {
		r := core.NewRange(9, 4)

		assert.Equal(t, 4, r.From())
		assert.Equal(t, 9, r.To())
		assert.Equal(t, 5, r.Len())
		assert.False(t, r.Empty())
	})

	t.Run("reports empty point", func(t *testing.T) {
		r := core.PointRange(3)

		assert.True(t, r.Empty())
		assert.Equal(t, 3, r.From())
		assert.Equal(t, 3, r.To())
	})

	t.Run("reports direction and flips", func(t *testing.T) {
		r := core.NewRange(8, 2)

		assert.Equal(t, core.DirectionBackward, r.Direction())
		assert.Equal(t, core.DirectionForward, r.Flip().Direction())
		assert.Equal(t, core.NewRange(2, 8),
			r.WithDirection(core.DirectionForward),
		)
	})

	t.Run("checks overlap and containment", func(t *testing.T) {
		r := core.NewRange(2, 7)

		assert.True(t, r.Overlaps(core.NewRange(5, 9)))
		assert.False(t, r.Overlaps(core.NewRange(7, 9)))
		assert.True(t, core.PointRange(7).Overlaps(core.NewRange(7, 9)))
		assert.True(t, r.ContainsRange(core.NewRange(3, 6)))
		assert.True(t, r.Contains(4))
		assert.False(t, r.Contains(7))
	})

	t.Run("computes line range", func(t *testing.T) {
		text := core.NewRope("one\ntwo\nthree")
		r := core.NewRange(2, 7)

		lr, err := r.LineRange(text)

		assert.NoError(t, err)
		assert.Equal(t, core.LineRange{From: 0, To: 1}, lr)
	})

	t.Run("extends forward and backward ranges", func(t *testing.T) {
		assert.Equal(t, core.NewRange(2, 9),
			core.NewRange(4, 7).Extend(2, 9),
		)
		assert.Equal(t, core.NewRange(9, 2),
			core.NewRange(7, 4).Extend(2, 9),
		)
	})

	t.Run("merges with direction behavior", func(t *testing.T) {
		assert.Equal(t, core.NewRange(2, 9),
			core.NewRange(2, 5).Merge(core.NewRange(4, 9)),
		)
		assert.Equal(t, core.NewRange(9, 2),
			core.NewRange(7, 2).Merge(core.NewRange(9, 4)),
		)
	})
}

func TestSelection(t *testing.T) {
	t.Run("rejects empty ranges", func(t *testing.T) {
		_, err := core.NewSelection(nil, 0)

		assert.True(t, errors.Is(err, core.ErrEmptySelection))
	})

	t.Run("rejects invalid primary index", func(t *testing.T) {
		_, err := core.NewSelection([]core.Range{core.PointRange(1)}, 2)

		assert.True(t, errors.Is(err, core.ErrPrimaryIndexNotFound))
	})

	t.Run("normalizes and tracks primary range", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(10, 12),
			core.NewRange(2, 6),
			core.NewRange(5, 9),
		}, 1)

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{
			core.NewRange(2, 9),
			core.NewRange(10, 12),
		}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
		assert.Equal(t, core.NewRange(2, 9), s.Primary())
	})

	t.Run("makes a single primary selection", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(1, 2),
			core.NewRange(4, 6),
		}, 1)
		assert.NoError(t, err)

		s = s.IntoSingle()

		assert.Equal(t, []core.Range{core.NewRange(4, 6)}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("pushes range as primary and normalizes", func(t *testing.T) {
		s := core.SingleSelection(1, 3).Push(core.NewRange(2, 5))

		assert.Equal(t, []core.Range{core.NewRange(1, 5)}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("removes range and adjusts primary", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(1, 2),
			core.NewRange(4, 6),
			core.NewRange(8, 9),
		}, 2)
		assert.NoError(t, err)

		s, err = s.Remove(1)

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{
			core.NewRange(1, 2),
			core.NewRange(8, 9),
		}, s.Ranges())
		assert.Equal(t, 1, s.PrimaryIndex())
	})

	t.Run("rejects removing the last range", func(t *testing.T) {
		_, err := core.PointSelection(1).Remove(0)

		assert.True(t, errors.Is(err, core.ErrLastRangeRemoval))
	})

	t.Run("rejects invalid range indexes", func(t *testing.T) {
		_, err := core.PointSelection(1).Replace(3, core.PointRange(2))
		assert.True(t, errors.Is(err, core.ErrRangeIndexNotFound))

		s, err := core.NewSelection([]core.Range{
			core.PointRange(1),
			core.PointRange(3),
		}, 0)
		assert.NoError(t, err)

		_, err = s.Remove(3)
		assert.True(t, errors.Is(err, core.ErrRangeIndexNotFound))
	})

	t.Run("replaces and normalizes ranges", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(1, 2),
			core.NewRange(5, 8),
		}, 0)
		assert.NoError(t, err)

		s, err = s.Replace(0, core.NewRange(4, 6))

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{core.NewRange(4, 8)}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("sets primary index", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.PointRange(1),
			core.PointRange(3),
		}, 0)
		assert.NoError(t, err)

		s, err = s.SetPrimaryIndex(1)

		assert.NoError(t, err)
		assert.Equal(t, 1, s.PrimaryIndex())
	})

	t.Run("rejects invalid primary index updates", func(t *testing.T) {
		_, err := core.PointSelection(1).SetPrimaryIndex(2)

		assert.True(t, errors.Is(err, core.ErrPrimaryIndexNotFound))
	})

	t.Run("merges all ranges from first to last", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(1, 2),
			core.NewRange(4, 6),
			core.NewRange(8, 9),
		}, 0)
		assert.NoError(t, err)

		s = s.MergeRanges()

		assert.Equal(t, []core.Range{core.NewRange(1, 9)}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("merges consecutive ranges", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(1, 3),
			core.NewRange(3, 5),
			core.NewRange(8, 9),
		}, 1)
		assert.NoError(t, err)

		s = s.MergeConsecutiveRanges()

		assert.Equal(t, []core.Range{
			core.NewRange(1, 5),
			core.NewRange(8, 9),
		}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("transforms and normalizes", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			core.NewRange(1, 2),
			core.NewRange(5, 6),
		}, 0)
		assert.NoError(t, err)

		s = s.Transform(func(r core.Range) core.Range {
			return core.NewRange(r.Anchor, r.Head+4)
		})

		assert.Equal(t, []core.Range{core.NewRange(1, 10)}, s.Ranges())
	})

	t.Run("merges adjacent line ranges", func(t *testing.T) {
		text := core.NewRope("one\ntwo\nthree\nfour")
		s, err := core.NewSelection([]core.Range{
			core.NewRange(0, 3),
			core.NewRange(4, 7),
			core.NewRange(14, 18),
		}, 0)
		assert.NoError(t, err)

		lines, err := s.LineRanges(text)

		assert.NoError(t, err)
		assert.Equal(t, []core.LineRange{
			{From: 0, To: 1},
			{From: 3, To: 3},
		}, lines)
	})

	t.Run("returns line range errors", func(t *testing.T) {
		text := core.NewRope("one")
		s := core.SingleSelection(0, 5)

		_, err := s.LineRanges(text)

		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})
}
