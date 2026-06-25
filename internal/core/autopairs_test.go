package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestHookInsertOpen(t *testing.T) {
	t.Run("inserts matching close bracket", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.PointRange(5)
		change, _, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 5, change.From)
		assert.Equal(t, 5, change.To)
		assert.Equal(t, "()", change.Text())
	})

	t.Run("moves cursor between inserted pair", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.PointRange(5)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 6, next.Head)
		assert.Equal(t, 6, next.Anchor)
	})

	t.Run("does not close before alphanumeric", func(t *testing.T) {
		doc := core.NewRope("a")
		r := core.PointRange(0)
		_, _, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("inserts when next is not alpha", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(0)
		change, _, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, "()", change.Text())
	})

	t.Run("no action when next char is alpha", func(t *testing.T) {
		doc := core.NewRope("x")
		r := core.PointRange(0)
		_, _, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("forward single-grapheme after open", func(t *testing.T) {
		doc := core.NewRope("( ")
		r := core.NewRange(0, 1)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 2, next.Head)
		assert.Equal(t, 1, next.Anchor)
	})

	t.Run("backward single-grapheme after open", func(t *testing.T) {
		doc := core.NewRope("( ")
		r := core.NewRange(1, 0)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 1, next.Head)
		assert.Equal(t, 2, next.Anchor)
	})

	t.Run("forward multi-char inserts pair", func(t *testing.T) {
		doc := core.NewRope("()  ")
		r := core.NewRange(0, 3)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 0, next.Anchor)
		assert.Equal(t, 4, next.Head)
	})

	t.Run("backward multi-char inserts pair", func(t *testing.T) {
		doc := core.NewRope("()  ")
		r := core.NewRange(3, 0)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 1, next.Head)
		assert.Equal(t, 5, next.Anchor)
	})

	// 👋🏽 is 2 code points but one grapheme
	t.Run("fwd multi-rune single grapheme", func(t *testing.T) {
		doc := core.NewRope("👋🏽 ")
		r := core.NewRange(0, 2)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.GreaterOrEqual(t, next.Head, next.Anchor)
	})

	t.Run("bwd multi-rune single grapheme", func(t *testing.T) {
		doc := core.NewRope("👋🏽 ")
		r := core.NewRange(2, 0)
		_, next, ok := core.HookInsert(doc, r, '(', core.DefaultAutoPairs())
		assert.True(t, ok)
		_ = next
	})
}

func TestHookInsertClose(t *testing.T) {
	t.Run("skips over matching close bracket", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(1)
		change, _, ok := core.HookInsert(doc, r, ')', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 1, change.From)
		assert.Equal(t, 1, change.To)
	})

	t.Run("no action when close char is not next", func(t *testing.T) {
		doc := core.NewRope("(x)")
		r := core.PointRange(1)
		_, _, ok := core.HookInsert(doc, r, ')', core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("skips with single-grapheme forward sel", func(t *testing.T) {
		doc := core.NewRope(")x")
		r := core.NewRange(0, 1)
		change, next, ok := core.HookInsert(doc, r, ')', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 0, change.From)
		assert.Equal(t, 0, change.To)
		assert.Equal(t, 1, next.Anchor)
		assert.Equal(t, 2, next.Head)
	})

	t.Run("skips with multi-char forward selection", func(t *testing.T) {
		doc := core.NewRope("))x")
		r := core.NewRange(0, 2)
		change, next, ok := core.HookInsert(doc, r, ')', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 1, change.From)
		assert.Equal(t, 1, change.To)
		assert.Equal(t, 0, next.Anchor)
		assert.Equal(t, 3, next.Head)
	})
}

func TestHookInsertSame(t *testing.T) {
	t.Run("inserts matching close quote", func(t *testing.T) {
		doc := core.NewRope(" ")
		r := core.PointRange(0)
		change, _, ok := core.HookInsert(doc, r, '"', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, `""`, change.Text())
	})

	t.Run("skips past existing close quote", func(t *testing.T) {
		doc := core.NewRope(`""`)
		r := core.PointRange(1)
		change, _, ok := core.HookInsert(doc, r, '"', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 1, change.From)
		assert.Equal(t, 1, change.To)
	})

	t.Run("no action before alphanumeric", func(t *testing.T) {
		doc := core.NewRope("abc")
		r := core.PointRange(0)
		_, _, ok := core.HookInsert(doc, r, '"', core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("no action after alphanumeric", func(t *testing.T) {
		doc := core.NewRope("a ")
		r := core.PointRange(1)
		_, _, ok := core.HookInsert(doc, r, '"', core.DefaultAutoPairs())
		assert.False(t, ok)
	})
}

func TestHookInsertWhitespace(t *testing.T) {
	t.Run("inserts space pair inside bracket pair", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(1)
		change, next, ok := core.HookInsert(doc, r, ' ', core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, "  ", change.Text())
		assert.Equal(t, 2, next.Head)
		assert.Equal(t, 2, next.Anchor)
	})

	t.Run("no action when not inside a pair", func(t *testing.T) {
		doc := core.NewRope("ab")
		r := core.PointRange(1)
		_, _, ok := core.HookInsert(doc, r, ' ', core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("no action at start of doc", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(0)
		_, _, ok := core.HookInsert(doc, r, ' ', core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("no action when pair does not match", func(t *testing.T) {
		doc := core.NewRope("(]")
		r := core.PointRange(1)
		_, _, ok := core.HookInsert(doc, r, ' ', core.DefaultAutoPairs())
		assert.False(t, ok)
	})
}

func TestHookDelete(t *testing.T) {
	t.Run("deletes both chars of a pair", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(1)
		del, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 0, del.From)
		assert.Equal(t, 2, del.To)
	})

	t.Run("no action when not inside a pair", func(t *testing.T) {
		doc := core.NewRope("ab")
		r := core.PointRange(1)
		_, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("no action at start of doc", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(0)
		_, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("no action at end of doc", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.PointRange(2)
		_, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("deletes inner spaces of padded pair", func(t *testing.T) {
		doc := core.NewRope("(  )")
		r := core.PointRange(2)
		del, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 1, del.From)
		assert.Equal(t, 3, del.To)
	})

	t.Run("no whitespace delete when chars differ", func(t *testing.T) {
		doc := core.NewRope("x  y")
		r := core.PointRange(2)
		_, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.False(t, ok)
	})

	t.Run("pair delete: forward single-grapheme", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.NewRange(1, 2)
		del, next, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 0, del.From)
		assert.Equal(t, 2, del.To)
		assert.Equal(t, 0, next.Head)
		assert.Equal(t, 0, next.Anchor)
	})

	t.Run("pair delete: backward single-grapheme", func(t *testing.T) {
		doc := core.NewRope("()")
		r := core.NewRange(2, 1)
		del, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 0, del.From)
		assert.Equal(t, 2, del.To)
	})

	t.Run("pair delete: multi-char backward", func(t *testing.T) {
		doc := core.NewRope("()x")
		r := core.NewRange(3, 1)
		del, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.True(t, ok)
		assert.Equal(t, 0, del.From)
		assert.Equal(t, 2, del.To)
	})
}

func TestAutoPairsGet(t *testing.T) {
	t.Run("finds pair by opener", func(t *testing.T) {
		pair, ok := core.DefaultAutoPairs().Get('(')
		assert.True(t, ok)
		assert.Equal(t, '(', pair.Open)
		assert.Equal(t, ')', pair.Close)
	})

	t.Run("finds pair by closer", func(t *testing.T) {
		pair, ok := core.DefaultAutoPairs().Get(')')
		assert.True(t, ok)
		assert.Equal(t, '(', pair.Open)
		assert.Equal(t, ')', pair.Close)
	})

	t.Run("returns false for unknown char", func(t *testing.T) {
		_, ok := core.DefaultAutoPairs().Get('x')
		assert.False(t, ok)
	})
}

func TestHookInsertNonPair(t *testing.T) {
	t.Run("non-pair non-whitespace returns false", func(t *testing.T) {
		doc := core.NewRope("hello")
		r := core.PointRange(5)
		_, _, ok := core.HookInsert(doc, r, 'x', core.DefaultAutoPairs())
		assert.False(t, ok)
	})
}

func TestHookDeleteMismatch(t *testing.T) {
	t.Run("mismatched prev char returns false", func(t *testing.T) {
		doc := core.NewRope("[)")
		r := core.PointRange(1)
		_, _, ok := core.HookDelete(doc, r, core.DefaultAutoPairs())
		assert.False(t, ok)
	})
}

func TestPairSame(t *testing.T) {
	t.Run("quote pair is same", func(t *testing.T) {
		p, _ := core.DefaultAutoPairs().Get('"')
		assert.True(t, p.Same())
	})

	t.Run("bracket pair is not same", func(t *testing.T) {
		p, _ := core.DefaultAutoPairs().Get('(')
		assert.False(t, p.Same())
	})
}
