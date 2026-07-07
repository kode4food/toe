package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestFindMatchingBracket(t *testing.T) {
	t.Run("empty file returns false", func(t *testing.T) {
		doc := core.NewRope("")
		_, ok := core.FindMatchingBracket(doc, 0)
		assert.False(t, ok)
	})

	t.Run("simple parens", func(t *testing.T) {
		doc := core.NewRope("(hello)")
		pos, ok := core.FindMatchingBracket(doc, 0)
		assert.True(t, ok)
		assert.Equal(t, 6, pos)
		// symmetrical
		pos, ok = core.FindMatchingBracket(doc, 6)
		assert.True(t, ok)
		assert.Equal(t, 0, pos)
	})

	t.Run("nested parens outer", func(t *testing.T) {
		doc := core.NewRope("((hello))")
		pos, ok := core.FindMatchingBracket(doc, 0)
		assert.True(t, ok)
		assert.Equal(t, 8, pos)
	})

	t.Run("nested parens inner", func(t *testing.T) {
		doc := core.NewRope("((hello))")
		pos, ok := core.FindMatchingBracket(doc, 1)
		assert.True(t, ok)
		assert.Equal(t, 7, pos)
	})

	t.Run("mixed brackets", func(t *testing.T) {
		doc := core.NewRope("(paren (paren {bracket}))")
		pos, ok := core.FindMatchingBracket(doc, 0)
		assert.True(t, ok)
		assert.Equal(t, 24, pos)
	})

	t.Run("close bracket with nested parens backward", func(t *testing.T) {
		// cursor on outer `)` at 5; inner `)` at 3 increments count backward
		doc := core.NewRope("((x)y)")
		pos, ok := core.FindMatchingBracket(doc, 5)
		assert.True(t, ok)
		assert.Equal(t, 0, pos)
	})

	t.Run("not on bracket returns false", func(t *testing.T) {
		doc := core.NewRope("(hello)")
		_, ok := core.FindMatchingBracket(doc, 3)
		assert.False(t, ok)
	})

	t.Run("unmatched bracket returns false", func(t *testing.T) {
		doc := core.NewRope("(hello")
		_, ok := core.FindMatchingBracket(doc, 0)
		assert.False(t, ok)
	})

	t.Run("multiline matching", func(t *testing.T) {
		doc := core.NewRope("(prev line\n ) (middle) ( \n next line)")
		pos, ok := core.FindMatchingBracket(doc, 0)
		assert.True(t, ok)
		assert.Equal(t, 12, pos)
	})
}

func TestGetPair(t *testing.T) {
	t.Run("open paren returns paren pair", func(t *testing.T) {
		o, c := core.GetPair('(')
		assert.Equal(t, '(', o)
		assert.Equal(t, ')', c)
	})

	t.Run("close paren returns paren pair", func(t *testing.T) {
		o, c := core.GetPair(')')
		assert.Equal(t, '(', o)
		assert.Equal(t, ')', c)
	})

	t.Run("unknown char returns identity pair", func(t *testing.T) {
		o, c := core.GetPair('x')
		assert.Equal(t, 'x', o)
		assert.Equal(t, 'x', c)
	})
}

func TestBracketClassifiers(t *testing.T) {
	t.Run("IsOpenBracket", func(t *testing.T) {
		assert.True(t, core.IsOpenBracket('('))
		assert.True(t, core.IsOpenBracket('{'))
		assert.False(t, core.IsOpenBracket(')'))
		assert.False(t, core.IsOpenBracket('x'))
	})

	t.Run("IsCloseBracket", func(t *testing.T) {
		assert.True(t, core.IsCloseBracket(')'))
		assert.True(t, core.IsCloseBracket('}'))
		assert.False(t, core.IsCloseBracket('('))
		assert.False(t, core.IsCloseBracket('x'))
	})

	t.Run("IsValidBracket", func(t *testing.T) {
		assert.True(t, core.IsValidBracket('('))
		assert.True(t, core.IsValidBracket(')'))
		assert.False(t, core.IsValidBracket('"'))
		assert.False(t, core.IsValidBracket('x'))
	})

}
