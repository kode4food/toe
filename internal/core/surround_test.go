package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestFindNthClosestPairsPos(t *testing.T) {
	t.Run("finds surrounding parens", func(t *testing.T) {
		doc := core.NewRope("(hello)")
		r := core.PointRange(3)
		anchor, head, err := core.FindNthClosestPairsPos(doc, r, 1)
		assert.NoError(t, err)
		assert.Equal(t, 0, anchor)
		assert.Equal(t, 6, head)
	})

	t.Run("returns error when no pair found", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.PointRange(2)
		_, _, err := core.FindNthClosestPairsPos(doc, r, 1)
		assert.True(t, errors.Is(err, core.ErrPairNotFound))
	})

	t.Run("finds pair around cursor at start", func(t *testing.T) {
		doc := core.NewRope("(a)c)")
		r := core.PointRange(0)
		_, _, err := core.FindNthClosestPairsPos(doc, r, 1)
		assert.True(t, errors.Is(err, core.ErrPairNotFound))
	})
}

func TestFindNthPairsPos(t *testing.T) {
	t.Run("finds specific char pair", func(t *testing.T) {
		doc := core.NewRope("(some) (chars)\n(newline)")
		r := core.PointRange(9)
		anchor, head, err := core.FindNthPairsPos(doc, '(', r, 1)
		assert.NoError(t, err)
		assert.Equal(t, 7, anchor)
		assert.Equal(t, 13, head)
	})

	t.Run("returns PairNotFound when no pair", func(t *testing.T) {
		doc := core.NewRope("[some]\n(chars)")
		r := core.PointRange(2)
		_, _, err := core.FindNthPairsPos(doc, '(', r, 1)
		assert.True(t, errors.Is(err, core.ErrPairNotFound))
	})

	t.Run("returns RangeExceedsText when cursor past end", func(t *testing.T) {
		doc := core.NewRope("hi")
		r := core.PointRange(2)
		_, _, err := core.FindNthPairsPos(doc, '(', r, 1)
		assert.True(t, errors.Is(err, core.ErrRangeExceedsText))
	})

	t.Run("handles same-char pair (quote)", func(t *testing.T) {
		doc := core.NewRope("some 'quoted text' on this line")
		r := core.PointRange(12)
		anchor, head, err := core.FindNthPairsPos(doc, '\'', r, 1)
		assert.NoError(t, err)
		assert.Equal(t, 5, anchor)
		assert.Equal(t, 17, head)
	})
}

func TestGetSurroundPos(t *testing.T) {
	t.Run("finds positions for each range", func(t *testing.T) {
		doc := core.NewRope("(some) (chars)\n(newline)")
		sel, err := core.NewSelection([]core.Range{
			core.PointRange(2),
			core.PointRange(9),
		}, 0)
		assert.NoError(t, err)
		positions, err := core.GetSurroundPosFor(doc, sel, '(', 1)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(positions))
	})

	t.Run("shared positions return CursorOverlap", func(t *testing.T) {
		doc := core.NewRope("[some]")
		sel, err := core.NewSelection([]core.Range{
			core.PointRange(2),
			core.PointRange(3),
		}, 0)
		assert.NoError(t, err)
		_, err = core.GetSurroundPosFor(doc, sel, '[', 1)
		assert.True(t, errors.Is(err, core.ErrSurroundCursorOverlap))
	})

	t.Run("auto-detects nearest pair", func(t *testing.T) {
		doc := core.NewRope("(hello)")
		sel, err := core.NewSelection([]core.Range{core.PointRange(3)}, 0)
		assert.NoError(t, err)
		positions, err := core.GetSurroundPos(doc, sel, 1)
		assert.NoError(t, err)
		assert.Equal(t, []int{0, 6}, positions)
	})
}
