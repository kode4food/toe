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

	t.Run("line range out of bounds errors", func(t *testing.T) {
		doc := core.NewRope("hi")
		_, err := core.NewRange(0, 100).LineRange(doc)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("fragment out of bounds errors", func(t *testing.T) {
		doc := core.NewRope("hi")
		_, err := core.NewRange(0, 100).Fragment(doc)
		assert.Error(t, err)
	})

	t.Run("put cursor forward crossing anchor", func(t *testing.T) {
		doc := core.NewRope("hello world")
		// Forward range [3,7), move cursor to 1 (crosses anchor 3 backwards)
		r := core.NewRange(3, 7)
		result := r.PutCursor(doc, 1, true)
		assert.Equal(t, 1, result.From())
	})

	t.Run("put cursor backward crossing anchor", func(t *testing.T) {
		doc := core.NewRope("hello world")
		// Backward range: anchor=7, head=3; move cursor to 9 (crosses anchor 7
		// forwards). PrevGraphemeBoundary(7)=6, NextGraphemeBoundary(9)=10 →
		// [6,10)
		r := core.NewRange(7, 3)
		result := r.PutCursor(doc, 9, true)
		assert.Equal(t, 10, result.To())
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

	t.Run("into single is no-op when already single", func(t *testing.T) {
		s := core.SingleSelection(2, 5)
		assert.Equal(t, s.Ranges(), s.IntoSingle().Ranges())
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
