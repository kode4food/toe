package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestFindNthChar(t *testing.T) {
	m := core.RuneMatcher('a')

	t.Run("finds first match forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(1, m, 5, core.DirectionForward)
		assert.True(t, ok)
		assert.Equal(t, 5, pos)
	})

	t.Run("finds second match forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(2, m, 5, core.DirectionForward)
		assert.True(t, ok)
		assert.Equal(t, 10, pos)
	})

	t.Run("finds third match forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(3, m, 5, core.DirectionForward)
		assert.True(t, ok)
		assert.Equal(t, 11, pos)
	})

	t.Run("returns false when not enough matches forward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		_, ok := doc.FindNthChar(4, m, 5, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("finds first match backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(1, m, 5, core.DirectionBackward)
		assert.True(t, ok)
		assert.Equal(t, 4, pos)
	})

	t.Run("finds second match backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(2, m, 5, core.DirectionBackward)
		assert.True(t, ok)
		assert.Equal(t, 1, pos)
	})

	t.Run("finds third match backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		pos, ok := doc.FindNthChar(3, m, 5, core.DirectionBackward)
		assert.True(t, ok)
		assert.Equal(t, 0, pos)
	})

	t.Run("returns false when not enough matches backward", func(t *testing.T) {
		doc := core.NewRope("aa ⌚aa \r\n aa")
		_, ok := doc.FindNthChar(4, m, 5, core.DirectionBackward)
		assert.False(t, ok)
	})

	t.Run("n=0 always returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(0, m, 0, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("character not found returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		z := core.RuneMatcher('z')
		_, ok := doc.FindNthChar(1, z, 0, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("pos beyond text returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(1, m, 20, core.DirectionForward)
		assert.False(t, ok)
	})

	t.Run("at start going backward returns false", func(t *testing.T) {
		doc := core.NewRope("abc")
		_, ok := doc.FindNthChar(1, m, 0, core.DirectionBackward)
		assert.False(t, ok)
	})

	t.Run("FuncMatcher works", func(t *testing.T) {
		doc := core.NewRope("a1b2c3")
		pos, ok := doc.FindNthChar(
			2,
			core.FuncMatcher(func(r rune) bool { return r >= '0' && r <= '9' }),
			0, core.DirectionForward,
		)
		assert.True(t, ok)
		assert.Equal(t, 3, pos)
	})
}
