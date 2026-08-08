package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

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
			{Anchor: 10, Head: 12},
			{Anchor: 2, Head: 6},
			{Anchor: 5, Head: 9},
		}, 1)

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{
			{Anchor: 2, Head: 9},
			{Anchor: 10, Head: 12},
		}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
		assert.Equal(t, core.Range{Anchor: 2, Head: 9}, s.Primary())
	})

	t.Run("makes a single primary selection", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			{Anchor: 1, Head: 2},
			{Anchor: 4, Head: 6},
		}, 1)
		assert.NoError(t, err)

		s = s.IntoSingle()

		assert.Equal(t, []core.Range{{
			Anchor: 4,
			Head:   6,
		}}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("into single is no-op when already single", func(t *testing.T) {
		s := core.SingleSelection(core.Range{Anchor: 2, Head: 5})
		assert.Equal(t, s.Ranges(), s.IntoSingle().Ranges())
	})

	t.Run("pushes range as primary and normalizes", func(t *testing.T) {
		s := core.SingleSelection(core.Range{
			Anchor: 1,
			Head:   3,
		}).Push(core.Range{Anchor: 2, Head: 5})

		assert.Equal(t, []core.Range{{
			Anchor: 1,
			Head:   5,
		}}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("removes range and adjusts primary", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			{Anchor: 1, Head: 2},
			{Anchor: 4, Head: 6},
			{Anchor: 8, Head: 9},
		}, 2)
		assert.NoError(t, err)

		s, err = s.Remove(1)

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{
			{Anchor: 1, Head: 2},
			{Anchor: 8, Head: 9},
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
			{Anchor: 1, Head: 2},
			{Anchor: 5, Head: 8},
		}, 0)
		assert.NoError(t, err)

		s, err = s.Replace(0, core.Range{Anchor: 4, Head: 6})

		assert.NoError(t, err)
		assert.Equal(t, []core.Range{{
			Anchor: 4,
			Head:   8,
		}}, s.Ranges())
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
			{Anchor: 1, Head: 2},
			{Anchor: 4, Head: 6},
			{Anchor: 8, Head: 9},
		}, 0)
		assert.NoError(t, err)

		s = s.MergeRanges()

		assert.Equal(t, []core.Range{{
			Anchor: 1,
			Head:   9,
		}}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("single range survives repeated merge", func(t *testing.T) {
		s := core.PointSelection(3)

		assert.Equal(t, s, s.MergeConsecutiveRanges())
	})

	t.Run("merges consecutive ranges", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			{Anchor: 1, Head: 3},
			{Anchor: 3, Head: 5},
			{Anchor: 8, Head: 9},
		}, 1)
		assert.NoError(t, err)

		s = s.MergeConsecutiveRanges()

		assert.Equal(t, []core.Range{
			{Anchor: 1, Head: 5},
			{Anchor: 8, Head: 9},
		}, s.Ranges())
		assert.Equal(t, 0, s.PrimaryIndex())
	})

	t.Run("transforms and normalizes", func(t *testing.T) {
		s, err := core.NewSelection([]core.Range{
			{Anchor: 1, Head: 2},
			{Anchor: 5, Head: 6},
		}, 0)
		assert.NoError(t, err)

		s = s.Transform(func(r core.Range) core.Range {
			return core.Range{Anchor: r.Anchor, Head: r.Head + 4}
		})

		assert.Equal(t, []core.Range{{
			Anchor: 1,
			Head:   10,
		}}, s.Ranges())
	})

	t.Run("merges adjacent line ranges", func(t *testing.T) {
		text := core.NewRope("one\ntwo\nthree\nfour")
		s, err := core.NewSelection([]core.Range{
			{Anchor: 0, Head: 3},
			{Anchor: 4, Head: 7},
			{Anchor: 14, Head: 18},
		}, 0)
		assert.NoError(t, err)

		lines, err := s.LineRanges(text)

		assert.NoError(t, err)
		assert.Equal(t, []core.Span{
			{From: 0, To: 1},
			{From: 3, To: 3},
		}, lines)
	})

	t.Run("returns line range errors", func(t *testing.T) {
		text := core.NewRope("one")
		s := core.SingleSelection(core.Range{Anchor: 0, Head: 5})

		_, err := s.LineRanges(text)

		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})
}

func TestSelectionEqual(t *testing.T) {
	ranges := []core.Range{
		{Anchor: 1, Head: 3},
		{Anchor: 5, Head: 7},
	}

	t.Run("identical selections are equal", func(t *testing.T) {
		a, err := core.NewSelection(ranges, 1)
		assert.NoError(t, err)
		b, err := core.NewSelection(ranges, 1)
		assert.NoError(t, err)

		assert.True(t, a.Equal(b))
	})

	t.Run("different ranges are not equal", func(t *testing.T) {
		a, err := core.NewSelection(ranges, 1)
		assert.NoError(t, err)
		b, err := core.NewSelection([]core.Range{
			{Anchor: 1, Head: 3},
			{Anchor: 5, Head: 8},
		}, 1)
		assert.NoError(t, err)

		assert.False(t, a.Equal(b))
	})

	t.Run("different primary index is not equal", func(t *testing.T) {
		a, err := core.NewSelection(ranges, 0)
		assert.NoError(t, err)
		b, err := core.NewSelection(ranges, 1)
		assert.NoError(t, err)

		assert.False(t, a.Equal(b))
	})
}
