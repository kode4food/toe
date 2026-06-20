package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestGrapheme(t *testing.T) {
	t.Run("newline is GraphemeKindNewline", func(t *testing.T) {
		g := core.NewGrapheme("\n", 0, 4)
		assert.Equal(t, core.GraphemeKindNewline, g.Kind())
		assert.Equal(t, 1, g.Width())
		assert.True(t, g.IsWhitespace())
		assert.True(t, g.IsWordBoundary())
	})

	t.Run("classifies tab with visual width", func(t *testing.T) {
		g := core.NewGrapheme("\t", 0, 4)
		assert.Equal(t, core.GraphemeKindTab, g.Kind())
		assert.Equal(t, 4, g.Width())

		g2 := core.NewGrapheme("\t", 2, 4)
		assert.Equal(t, 2, g2.Width())
	})

	t.Run("ascii letter is non-boundary", func(t *testing.T) {
		g := core.NewGrapheme("a", 0, 4)
		assert.Equal(t, core.GraphemeKindOther, g.Kind())
		assert.Equal(t, 1, g.Width())
		assert.False(t, g.IsWhitespace())
		assert.False(t, g.IsWordBoundary())
	})

	t.Run("space is whitespace word boundary", func(t *testing.T) {
		g := core.NewGrapheme(" ", 0, 4)
		assert.Equal(t, core.GraphemeKindOther, g.Kind())
		assert.True(t, g.IsWhitespace())
		assert.True(t, g.IsWordBoundary())
	})

	t.Run("CJK grapheme has width 2", func(t *testing.T) {
		g := core.NewGrapheme("世", 0, 4)
		assert.Equal(t, core.GraphemeKindOther, g.Kind())
		assert.Equal(t, 2, g.Width())
	})

	t.Run("ChangePosition updates tab width", func(t *testing.T) {
		g := core.NewGrapheme("\t", 0, 4)
		assert.Equal(t, 4, g.Width())
		g.ChangePosition(3, 4)
		assert.Equal(t, 1, g.Width())
	})
}

func TestTabWidthAt(t *testing.T) {
	t.Run("aligns to next tab stop", func(t *testing.T) {
		assert.Equal(t, 4, core.TabWidthAt(0, 4))
		assert.Equal(t, 3, core.TabWidthAt(1, 4))
		assert.Equal(t, 2, core.TabWidthAt(2, 4))
		assert.Equal(t, 1, core.TabWidthAt(3, 4))
		assert.Equal(t, 4, core.TabWidthAt(4, 4))
	})
}

func TestGraphemeDecoration(t *testing.T) {
	t.Run("exposes text and width", func(t *testing.T) {
		g := core.NewDecorationGrapheme("→")
		assert.Equal(t, core.GraphemeKindOther, g.Kind())
		assert.Equal(t, "→", g.Text())
		assert.Greater(t, g.Width(), 0)
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

	t.Run("combined cluster is single unit", func(t *testing.T) {
		// e + combining acute accent = é (2 bytes in UTF-8, 2 codepoints, 1 grapheme)
		doc := core.NewRope("éx")
		assert.Equal(t, 2, core.NextGraphemeBoundary(doc, 0))
		assert.Equal(t, 3, core.NextGraphemeBoundary(doc, 2))
	})
}
