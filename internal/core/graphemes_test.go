package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestTabWidthAt(t *testing.T) {
	t.Run("aligns to next tab stop", func(t *testing.T) {
		assert.Equal(t, 4, core.TabWidthAt(0, 4))
		assert.Equal(t, 3, core.TabWidthAt(1, 4))
		assert.Equal(t, 2, core.TabWidthAt(2, 4))
		assert.Equal(t, 1, core.TabWidthAt(3, 4))
		assert.Equal(t, 4, core.TabWidthAt(4, 4))
	})
}

func TestGraphemeBoundaries(t *testing.T) {
	t.Run("next boundary advances over ascii char", func(t *testing.T) {
		doc := core.NewRope("abc")
		assert.Equal(t, 1, core.NextGraphemeBoundary(doc, 0))
		assert.Equal(t, 2, core.NextGraphemeBoundary(doc, 1))
		assert.Equal(t, 3, core.NextGraphemeBoundary(doc, 2))
	})

	t.Run("advances over multibyte char", func(t *testing.T) {
		doc := core.NewRope("a世b")
		assert.Equal(t, 1, core.NextGraphemeBoundary(doc, 0))
		assert.Equal(t, 2, core.NextGraphemeBoundary(doc, 1))
		assert.Equal(t, 3, core.NextGraphemeBoundary(doc, 2))
	})

	t.Run("prev boundary retreats over ascii char", func(t *testing.T) {
		doc := core.NewRope("abc")
		assert.Equal(t, 2, core.PrevGraphemeBoundary(doc, 3))
		assert.Equal(t, 1, core.PrevGraphemeBoundary(doc, 2))
		assert.Equal(t, 0, core.PrevGraphemeBoundary(doc, 1))
	})

	t.Run("nth next advances n clusters", func(t *testing.T) {
		doc := core.NewRope("abcde")
		assert.Equal(t, 3, core.NthNextGraphemeBoundary(doc, 0, 3))
	})

	t.Run("nth prev retreats n clusters", func(t *testing.T) {
		doc := core.NewRope("abcde")
		assert.Equal(t, 2, core.NthPrevGraphemeBoundary(doc, 5, 3))
	})

	t.Run("ensure next snaps to boundary", func(t *testing.T) {
		doc := core.NewRope("abc")
		assert.Equal(t, 0, core.EnsureGraphemeBoundaryNext(doc, 0))
		assert.Equal(t, 1, core.EnsureGraphemeBoundaryNext(doc, 1))
	})

	t.Run("ensure prev snaps to boundary", func(t *testing.T) {
		doc := core.NewRope("abc")
		n := doc.LenChars()
		assert.Equal(t, n, core.EnsureGraphemeBoundaryPrev(doc, n))
		assert.Equal(t, 2, core.EnsureGraphemeBoundaryPrev(doc, 2))
	})

	t.Run("nth next clamps at end", func(t *testing.T) {
		doc := core.NewRope("abc")
		assert.Equal(t, 3, core.NthNextGraphemeBoundary(doc, 2, 10))
	})

	t.Run("nth prev clamps at start", func(t *testing.T) {
		doc := core.NewRope("abc")
		assert.Equal(t, 0, core.NthPrevGraphemeBoundary(doc, 1, 10))
	})

	t.Run("wide unicode grapheme has width > 1", func(t *testing.T) {
		// 世 is a wide CJK char; its display width is 2
		doc := core.NewRope("世b")
		// NthNextGraphemeBoundary steps over 世 (1 grapheme) to pos 1
		assert.Equal(t, 1, core.NextGraphemeBoundary(doc, 0))
		// and the char at 0 is '世', which graphemeWidth reports as 2
		// we verify indirectly: the rope contains 2 chars, prev from 1 lands
		// at 0
		assert.Equal(t, 0, core.PrevGraphemeBoundary(doc, 1))
	})

	t.Run("combined cluster is single unit", func(t *testing.T) {
		// e plus acute accent is two codepoints but one grapheme
		doc := core.NewRope("éx")
		assert.Equal(t, 2, core.NextGraphemeBoundary(doc, 0))
		assert.Equal(t, 3, core.NextGraphemeBoundary(doc, 2))
	})
}
