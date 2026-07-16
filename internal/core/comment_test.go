package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestGetCommentToken(t *testing.T) {
	t.Run("nil when no matching token", func(t *testing.T) {
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

	t.Run("shorter token when only it matches", func(t *testing.T) {
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

	t.Run("lone token uncomments without space", func(t *testing.T) {
		doc := core.NewRope("#")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "", doc.String())
	})

	t.Run("explicit token overrides default", func(t *testing.T) {
		doc := core.NewRope("hello")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars()-1)}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "//")
		assert.NoError(t, err)
		doc = applyTx(doc, tx)
		assert.Equal(t, "// hello", doc.String())
	})

	t.Run("all-blank selection is no-op", func(t *testing.T) {
		doc := core.NewRope("\n\n")
		sel, _ := core.NewSelection(
			[]core.Range{core.NewRange(0, doc.LenChars())}, 0,
		)
		tx, err := core.ToggleLineComments(doc, sel, "")
		assert.NoError(t, err)
		out := applyTx(doc, tx)
		assert.Equal(t, "\n\n", out.String())
	})
}

func TestSplitLinesOfSelection(t *testing.T) {
	t.Run("multi-line range into per-line ranges", func(t *testing.T) {
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

	t.Run("final unterminated line ends at document", func(t *testing.T) {
		doc := core.NewRope("hello\nworld")
		sel, _ := core.NewSelection([]core.Range{core.NewRange(6, 11)}, 0)

		split, err := core.SplitLinesOfSelection(doc, sel)

		assert.NoError(t, err)
		ranges := split.Ranges()
		assert.Len(t, ranges, 1)
		assert.Equal(t, 6, ranges[0].Anchor)
		assert.Equal(t, 11, ranges[0].Head)
	})
}
