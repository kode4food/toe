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

	t.Run("short word stops at punctuation", func(t *testing.T) {
		doc := core.NewRope("foo-bar")
		r := core.TextObjectWord(
			doc, core.NewRange(2, 2), core.TextObjectInside, false,
		)
		assert.Equal(t, "foo", sliceOf(t, doc, r))
	})

	t.Run("whitespace cursor is empty", func(t *testing.T) {
		doc := core.NewRope("foo bar")
		r := core.TextObjectWord(
			doc, core.NewRange(3, 3), core.TextObjectInside, false,
		)
		assert.Equal(t, "", sliceOf(t, doc, r))
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

	t.Run("count selects multiple paragraphs", func(t *testing.T) {
		doc := core.NewRope("a\n\nb\n\nc\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(0, 0), core.TextObjectAround, 2,
		)
		assert.Equal(t, "a\n\nb\n\n", sliceOf(t, doc, r))
	})

	t.Run("inside count trims blank lines", func(t *testing.T) {
		doc := core.NewRope("a\n\nb\n\nc\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(0, 0), core.TextObjectInside, 2,
		)
		assert.Equal(t, "a\n\nb\n", sliceOf(t, doc, r))
	})

	t.Run("blank line before paragraph selects next", func(t *testing.T) {
		doc := core.NewRope("a\n\nb\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(2, 2), core.TextObjectInside, 1,
		)
		assert.Equal(t, "b\n", sliceOf(t, doc, r))
	})

	t.Run("cursor in second paragraph walks backward", func(t *testing.T) {
		doc := core.NewRope("a\n\nb\nc\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(4, 4), core.TextObjectInside, 1,
		)
		assert.Equal(t, "b\nc\n", sliceOf(t, doc, r))
	})

	t.Run("around at document end clamps", func(t *testing.T) {
		doc := core.NewRope("a\n\nb")
		r := core.TextObjectParagraph(
			doc, core.NewRange(3, 3), core.TextObjectAround, 2,
		)
		assert.Equal(t, "a\n\nb", sliceOf(t, doc, r))
	})

	t.Run("empty to line boundary selects next", func(t *testing.T) {
		doc := core.NewRope("empty to line\n\nparagraph boundary\n\n")
		r := core.TextObjectParagraph(
			doc, core.NewRange(14, 14), core.TextObjectInside, 1,
		)
		assert.Equal(t, "paragraph boundary\n", sliceOf(t, doc, r))
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

	t.Run("nearest pair when char omitted", func(t *testing.T) {
		doc := core.NewRope("[a (bc)]")
		r := core.NewRange(4, 4).TextObjectPairSurround(
			doc, core.TextObjectInside, 0, 1)
		assert.Equal(t, "a (bc)", sliceOf(t, doc, r))
	})

	t.Run("specific pair adjacent to range start", func(t *testing.T) {
		doc := core.NewRope("(abc)")
		r := core.NewRange(1, 4).TextObjectPairSurround(
			doc, core.TextObjectInside, '(', 1)
		assert.Equal(t, "abc", sliceOf(t, doc, r))
	})

	t.Run("backward range stays backward", func(t *testing.T) {
		doc := core.NewRope("(abc)")
		r := core.NewRange(3, 2).TextObjectPairSurround(
			doc, core.TextObjectAround, '(', 1)
		assert.Equal(t, core.DirectionBackward, r.Direction())
		assert.Equal(t, "(abc)", sliceOf(t, doc, r))
	})

	t.Run("no pair returns original range", func(t *testing.T) {
		doc := core.NewRope("abc")
		orig := core.NewRange(1, 2)

		r := orig.TextObjectPairSurround(doc, core.TextObjectInside, '(', 1)

		assert.Equal(t, orig, r)
	})
}

func sliceOf(t *testing.T, doc core.Rope, r core.Range) string {
	t.Helper()
	s, err := doc.SliceString(r.From(), r.To())
	assert.NoError(t, err)
	return s
}
