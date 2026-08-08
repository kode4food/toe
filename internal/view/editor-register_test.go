package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestEditorRegisters(t *testing.T) {
	t.Run("selection text is computed", func(t *testing.T) {
		e := testutil.EditorWithText(t, "alpha beta")
		testutil.SetSelection(t, e, []core.Range{
			{Anchor: 0, Head: 5},
			{Anchor: 6, Head: 10},
		}, 0)

		assert.Equal(t, []string{"alpha", "beta"}, e.ReadRegister('.'))
	})

	t.Run("selection indices are computed", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(0),
			core.PointRange(2),
		}, 0)

		assert.Equal(t, []string{"1", "2"}, e.ReadRegister('#'))
	})

	t.Run("document path is computed", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.SetPath("/tmp/example.txt")

		assert.Equal(t, []string{"/tmp/example.txt"}, e.ReadRegister('%'))
	})

	t.Run("clipboard register preserves selections", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{
			{Anchor: 0, Head: 1},
			{Anchor: 2, Head: 3},
		}, 0)
		e.SetRegister('+')

		action.Yank(e)

		assert.Equal(t, "a\nc", clip.System)
		assert.Equal(t, []string{"a", "c"}, e.ReadRegister('+'))
	})

	t.Run("first register returns first value", func(t *testing.T) {
		e := testutil.EditorWithText(t, "alpha beta")
		testutil.SetSelection(t, e, []core.Range{
			{Anchor: 0, Head: 5},
			{Anchor: 6, Head: 10},
		}, 0)

		val, ok := e.FirstRegister('.')

		assert.True(t, ok)
		assert.Equal(t, "alpha", val)
	})

	t.Run("empty first register is not ok", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")

		val, ok := e.FirstRegister('q')

		assert.False(t, ok)
		assert.Equal(t, "", val)
	})

	t.Run("external clipboard reads as one value", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		clip := testutil.NewFakeClipboard()
		clip.System = "a\nc"
		e.SetClipboard(clip)

		assert.Equal(t, []string{"a\nc"}, e.ReadRegister('+'))
	})
}
