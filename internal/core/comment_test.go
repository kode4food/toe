package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestGetCommentToken(t *testing.T) {
	t.Run("returns nil when line has no matching token", func(t *testing.T) {
		doc := core.NewRope("··") // multibyte, no comment
		_, ok := core.GetCommentToken(doc, []string{"//", "///"}, 0)
		assert.False(t, ok)
	})

	t.Run("returns longest matching token", func(t *testing.T) {
		doc := core.NewRope("    /// amogus")
		tok, ok := core.GetCommentToken(doc, []string{"///", "//"}, 0)
		assert.True(t, ok)
		assert.Equal(t, "///", tok)
	})

	t.Run("returns shorter token when only it matches", func(t *testing.T) {
		doc := core.NewRope("// hello")
		tok, ok := core.GetCommentToken(doc, []string{"//", "///"}, 0)
		assert.True(t, ok)
		assert.Equal(t, "//", tok)
	})

	t.Run("returns false for blank line", func(t *testing.T) {
		doc := core.NewRope("\n")
		_, ok := core.GetCommentToken(doc, []string{"//"}, 0)
		assert.False(t, ok)
	})

	t.Run("returns false for out-of-range line", func(t *testing.T) {
		doc := core.NewRope("hello")
		_, ok := core.GetCommentToken(doc, []string{"//"}, 99)
		assert.False(t, ok)
	})
}

func TestToggleLineComments(t *testing.T) {
	applyTx := func(doc core.Rope, tx core.Transaction) core.Rope {
		out, err := tx.Apply(doc)
		assert.NoError(t, err)
		return out
	}

	t.Run("comment adds token to non-blank lines", func(t *testing.T) {
		doc := core.NewRope("  1\n\n  2\n  3")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "  # 1\n\n  # 2\n  # 3", doc.String())
	})

	t.Run("uncomment removes token from lines", func(t *testing.T) {
		doc := core.NewRope("  # 1\n\n  # 2\n  # 3")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "  1\n\n  2\n  3", doc.String())
	})

	t.Run("uncomment zero-margin comments", func(t *testing.T) {
		doc := core.NewRope("  #1\n\n  #2\n  #3")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "  1\n\n  2\n  3", doc.String())
	})

	t.Run("uncomment lone comment token with no space", func(t *testing.T) {
		doc := core.NewRope("#")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "", doc.String())
	})

	t.Run("explicit token is used instead of default", func(t *testing.T) {
		doc := core.NewRope("hello")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "//")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "// hello", doc.String())
	})
}

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

	t.Run("whitespace-only returns Whitespace and false", func(t *testing.T) {
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
}

func TestSplitLinesOfSelection(t *testing.T) {
	t.Run("splits multi-line range into per-line ranges", func(t *testing.T) {
		doc := core.NewRope("abc\ndef\nghi")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		split, err := core.SplitLinesOfSelection(doc, sel)
		assert.NoError(t, err)
		ranges := split.Ranges()
		assert.Len(t, ranges, 3)
		assert.Equal(t, 0, ranges[0].Anchor)
		assert.Equal(t, 4, ranges[0].Head) // "abc\n"
		assert.Equal(t, 4, ranges[1].Anchor)
		assert.Equal(t, 8, ranges[1].Head) // "def\n"
		assert.Equal(t, 8, ranges[2].Anchor)
		assert.Equal(t, 11, ranges[2].Head) // "ghi"
	})

	t.Run("single-line range produces one sub-range", func(t *testing.T) {
		doc := core.NewRope("hello\nworld")
		sel, _ := core.NewSelection([]core.Range{core.NewRange(0, 5)}, 0)
		split, err := core.SplitLinesOfSelection(doc, sel)
		assert.NoError(t, err)
		assert.Len(t, split.Ranges(), 1)
	})
}
