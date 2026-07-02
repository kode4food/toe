package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestFindNthChar(t *testing.T) {
	ch := 'a'

	t.Run("finds first match forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(1, ch, 5, core.DirectionForward)
		assert.True(t, ok)
		assert.Equal(t, 5, pos)
	})

	t.Run("finds second match forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(2, ch, 5, core.DirectionForward)
		assert.True(t, ok)
		assert.Equal(t, 10, pos)
	})

	t.Run("finds third match forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(3, ch, 5, core.DirectionForward)
		assert.True(t, ok)
		assert.Equal(t, 11, pos)
	})

	t.Run("false when not enough forward matches", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		_, ok := doc.FindNthChar(4, ch, 5, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("finds first match backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(1, ch, 5, core.DirectionBackward)
		assert.True(t, ok)
		assert.Equal(t, 4, pos)
	})

	t.Run("finds second match backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(2, ch, 5, core.DirectionBackward)
		assert.True(t, ok)
		assert.Equal(t, 1, pos)
	})

	t.Run("finds third match backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(3, ch, 5, core.DirectionBackward)
		assert.True(t, ok)
		assert.Equal(t, 0, pos)
	})

	t.Run("false when not enough backward matches", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		_, ok := doc.FindNthChar(4, ch, 5, core.DirectionBackward)
		assert.False(t, ok)
	})

	t.Run("n=0 always returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(0, ch, 0, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("character not found returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(1, 'z', 0, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("pos beyond text returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(1, ch, 20, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("at start going backward returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(1, ch, 0, core.DirectionBackward)
		assert.False(t, ok)
	})

	t.Run("unknown direction returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(1, ch, 0, core.Direction(99))
		assert.False(t, ok)
	})
}
