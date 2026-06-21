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
