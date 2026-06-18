package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestTextObjectWord(t *testing.T) {
	t.Run("inside selects the word", func(t *testing.T) {
		doc := core.NewRope("foo bar baz")
		r := core.TextObjectWord(
			doc, core.NewRange(5, 5), core.TextObjectInside, false,
		)
		assert.Equal(t, "bar", sliceOf(t, doc, r))
	})

	t.Run("around includes trailing whitespace", func(t *testing.T) {
		doc := core.NewRope("foo bar baz")
		r := core.TextObjectWord(
			doc, core.NewRange(5, 5), core.TextObjectAround, false,
		)
		assert.Equal(t, "bar ", sliceOf(t, doc, r))
	})

	t.Run("around uses leading ws when no trailing", func(t *testing.T) {
		doc := core.NewRope("foo baz")
		r := core.TextObjectWord(
			doc, core.NewRange(5, 5), core.TextObjectAround, false,
		)
		assert.Equal(t, " baz", sliceOf(t, doc, r))
	})

	t.Run("long word spans punctuation", func(t *testing.T) {
		doc := core.NewRope("foo-bar baz")
		r := core.TextObjectWord(
			doc, core.NewRange(2, 2), core.TextObjectInside, true,
		)
		assert.Equal(t, "foo-bar", sliceOf(t, doc, r))
	})
}

func TestTextObjectParagraph(t *testing.T) {
	t.Run("inside selects the paragraph block", func(t *testing.T) {
		doc := core.NewRope("a\nb\n\nc\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(0, 0), core.TextObjectInside, 1,
		)
		assert.Equal(t, "a\nb\n", sliceOf(t, doc, r))
	})

	t.Run("around includes trailing blank lines", func(t *testing.T) {
		doc := core.NewRope("a\nb\n\nc\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(0, 0), core.TextObjectAround, 1,
		)
		assert.Equal(t, "a\nb\n\n", sliceOf(t, doc, r))
	})
}

func TestTextObjectPairSurround(t *testing.T) {
	t.Run("inside selects between the pair", func(t *testing.T) {
		doc := core.NewRope("(abc)")
		r := core.NewRange(2, 2).TextObjectPairSurround(
			doc, core.TextObjectInside, '(', 1)
		assert.Equal(t, "abc", sliceOf(t, doc, r))
	})

	t.Run("around includes the delimiters", func(t *testing.T) {
		doc := core.NewRope("(abc)")
		r := core.NewRange(2, 2).TextObjectPairSurround(
			doc, core.TextObjectAround, '(', 1)
		assert.Equal(t, "(abc)", sliceOf(t, doc, r))
	})
}

func sliceOf(t *testing.T, doc core.Rope, r core.Range) string {
	t.Helper()
	s, err := doc.SliceString(r.From(), r.To())
	assert.NoError(t, err)
	return s
}
