package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestRange(t *testing.T) {
	t.Run("reports bounds and length", func(t *testing.T) {
		r := core.Range{Anchor: 9, Head: 4}

		assert.Equal(t, 4, r.From())
		assert.Equal(t, 9, r.To())
		assert.Equal(t, 5, r.Len())
		assert.False(t, r.IsEmpty())
	})

	t.Run("reports empty point", func(t *testing.T) {
		r := core.PointRange(3)

		assert.True(t, r.IsEmpty())
		assert.Equal(t, 3, r.From())
		assert.Equal(t, 3, r.To())
	})

	t.Run("reports direction and flips", func(t *testing.T) {
		r := core.Range{Anchor: 8, Head: 2}

		assert.Equal(t, core.DirectionBackward, r.Direction())
		assert.Equal(t, core.DirectionForward, r.Flip().Direction())
		assert.Equal(t, core.Range{Anchor: 2, Head: 8},
			r.WithDirection(core.DirectionForward),
		)
	})

	t.Run("checks overlap and containment", func(t *testing.T) {
		r := core.Range{Anchor: 2, Head: 7}

		assert.True(t, r.Overlaps(core.Range{Anchor: 5, Head: 9}))
		assert.False(t, r.Overlaps(core.Range{Anchor: 7, Head: 9}))
		assert.True(t, core.PointRange(7).Overlaps(core.Range{
			Anchor: 7,
			Head:   9,
		}))
		assert.True(t, r.ContainsRange(core.Range{Anchor: 3, Head: 6}))
		assert.True(t, r.Contains(4))
		assert.False(t, r.Contains(7))
	})

	t.Run("computes line range", func(t *testing.T) {
		text := core.NewRope("one\ntwo\nthree")
		r := core.Range{Anchor: 2, Head: 7}

		lr, err := r.LineSpan(text)

		assert.NoError(t, err)
		assert.Equal(t, core.Span{From: 0, To: 1}, lr)
	})

	t.Run("extends forward and backward ranges", func(t *testing.T) {
		assert.Equal(t, core.Range{Anchor: 2, Head: 9},
			core.Range{Anchor: 4, Head: 7}.Extend(core.Span{From: 2, To: 9}),
		)
		assert.Equal(t, core.Range{Anchor: 9, Head: 2},
			core.Range{Anchor: 7, Head: 4}.Extend(core.Span{From: 2, To: 9}),
		)
	})

	t.Run("merges with direction behavior", func(t *testing.T) {
		assert.Equal(t, core.Range{Anchor: 2, Head: 9},
			core.Range{
				Anchor: 2,
				Head:   5,
			}.Merge(core.Range{Anchor: 4, Head: 9}),
		)
		assert.Equal(t, core.Range{Anchor: 9, Head: 2},
			core.Range{
				Anchor: 7,
				Head:   2,
			}.Merge(core.Range{Anchor: 9, Head: 4}),
		)
	})

	t.Run("line range out of bounds errors", func(t *testing.T) {
		doc := core.NewRope("hi")
		_, err := core.Range{Anchor: 0, Head: 100}.LineSpan(doc)
		assert.True(t, errors.Is(err, core.ErrRopeIndexOutOfRange))
	})

	t.Run("fragment out of bounds errors", func(t *testing.T) {
		doc := core.NewRope("hi")
		_, err := core.Range{Anchor: 0, Head: 100}.Fragment(doc)
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
		r := core.Range{Anchor: 0, Head: 3}
		aligned := r.GraphemeAligned(doc)
		assert.Equal(t, 0, aligned.From())
		assert.Equal(t, 3, aligned.To())
	})

	t.Run("grapheme aligned backward range", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.Range{Anchor: 4, Head: 1}
		aligned := r.GraphemeAligned(doc)
		assert.Equal(t, 1, aligned.From())
		assert.Equal(t, 4, aligned.To())
	})

	t.Run("put cursor forward crossing anchor", func(t *testing.T) {
		doc := core.NewRope("hello world")
		r := core.Range{Anchor: 3, Head: 7}
		result := r.PutCursor(doc, 1, true)
		assert.Equal(t, 1, result.From())
	})

	t.Run("put cursor backward crossing anchor", func(t *testing.T) {
		doc := core.NewRope("hello world")
		r := core.Range{Anchor: 7, Head: 3}
		result := r.PutCursor(doc, 9, true)
		assert.Equal(t, 10, result.To())
	})
}
