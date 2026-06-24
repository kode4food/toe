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

	t.Run("grapheme aligned point range", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.PointRange(2)
		aligned := r.GraphemeAligned(doc)
		assert.Equal(t, 2, aligned.From())
		assert.Equal(t, 2, aligned.To())
	})

	t.Run("grapheme aligned forward range", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.NewRange(0, 3)
		aligned := r.GraphemeAligned(doc)
		assert.Equal(t, 0, aligned.From())
		assert.Equal(t, 3, aligned.To())
	})

	t.Run("grapheme aligned backward range", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.NewRange(4, 1)
		aligned := r.GraphemeAligned(doc)
		assert.Equal(t, 1, aligned.From())
		assert.Equal(t, 4, aligned.To())
	})

	t.Run("put cursor forward crossing anchor", func(t *testing.T) {
		doc := core.NewRope("hello world")
		r := core.NewRange(3, 7)
		result := r.PutCursor(doc, 1, true)
		assert.Equal(t, 1, result.From())
	})

	t.Run("put cursor backward crossing anchor", func(t *testing.T) {
		doc := core.NewRope("hello world")
		r := core.NewRange(7, 3)
		result := r.PutCursor(doc, 9, true)
		assert.Equal(t, 10, result.To())
	})
}
