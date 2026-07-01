package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestFindBlockComments(t *testing.T) {
	t.Run("uncommented returns Uncommented", func(t *testing.T) {
		doc := core.NewRope("1\n2\n3")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		toks := []core.BlockCommentToken{{Start: "/*", End: "*/"}}
		ok, changes, err := core.FindBlockComments(toks, doc, sel)
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Len(t, changes, 1)
		assert.Equal(t, core.CommentChangeUncommented, changes[0].Kind)
		assert.Equal(t, 0, changes[0].StartPos)
		assert.Equal(t, 4, changes[0].EndPos)
		assert.Equal(t, "/*", changes[0].StartToken)
		assert.Equal(t, "*/", changes[0].EndToken)
	})

	t.Run("already-commented returns Commented", func(t *testing.T) {
		doc := core.NewRope("/* 1\n2\n3 */")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		toks := []core.BlockCommentToken{{Start: "/*", End: "*/"}}
		ok, changes, err := core.FindBlockComments(toks, doc, sel)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Len(t, changes, 1)
		assert.Equal(t, core.CommentChangeCommented, changes[0].Kind)
	})

	t.Run("whitespace-only returns Whitespace", func(t *testing.T) {
		doc := core.NewRope("   ")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		toks := []core.BlockCommentToken{{Start: "/*", End: "*/"}}
		ok, changes, err := core.FindBlockComments(toks, doc, sel)
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Len(t, changes, 1)
		assert.Equal(t, core.CommentChangeWhitespace, changes[0].Kind)
	})

	t.Run("uses default token when none provided", func(t *testing.T) {
		doc := core.NewRope("hello")
		sel, _ := core.NewSelection([]core.Range{core.NewRange(0, 5)}, 0)
		ok, changes, err := core.FindBlockComments(nil, doc, sel)
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Len(t, changes, 1)
		assert.Equal(t, "/*", changes[0].StartToken)
	})

	t.Run("longest token wins when multiple match", func(t *testing.T) {
		doc := core.NewRope("/** text **/")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		toks := []core.BlockCommentToken{
			{Start: "/*", End: "*/"},
			{Start: "/**", End: "**/"},
		}
		ok, changes, err := core.FindBlockComments(toks, doc, sel)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "/**", changes[0].StartToken)
	})
}

func TestToggleBlockComments(t *testing.T) {
	applyTx := func(doc core.Rope, tx core.Transaction) core.Rope {
		out, err := tx.Apply(doc)
		assert.NoError(t, err)
		return out
	}

	toks := []core.BlockCommentToken{{Start: "/*", End: "*/"}}

	t.Run("wraps uncommented in block comment", func(t *testing.T) {
		doc := core.NewRope("1\n2\n3")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		tx, err := core.ToggleBlockComments(doc, sel, toks)
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "/* 1\n2\n3 */", doc.String())
	})

	t.Run("removes block comment delimiters", func(t *testing.T) {
		doc := core.NewRope("/* 1\n2\n3 */")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		tx, err := core.ToggleBlockComments(doc, sel, toks)
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "1\n2\n3", doc.String())
	})

	t.Run("comment-only content", func(t *testing.T) {
		doc := core.NewRope("/* */")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		tx, err := core.ToggleBlockComments(doc, sel, toks)
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "", doc.String())
	})

	t.Run("whitespace-only is no-op", func(t *testing.T) {
		doc := core.NewRope("   ")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		tx, err := core.ToggleBlockComments(doc, sel, toks)
		assert.NoError(t, err)
		out, err := tx.Apply(doc)
		assert.NoError(t, err)
		assert.Equal(t, "   ", out.String())
	})
}
