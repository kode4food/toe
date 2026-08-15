package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestOpenAbove(t *testing.T) {
	t.Run("inserts blank line above", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "\nhello", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("inserts above second line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetCursor(t, e, 2)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "a\n\nb", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("count repeats new lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)
		e.SetCount(2)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "\n\nhello", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("shared target keeps every cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(0), core.PointRange(1)},
			0,
		)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "\n\nabc", doc.Text().String())
		v := e.FocusedView()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})

	t.Run("opens above the first selected line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   5,
		}}, 0)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "\na\nb\nc", doc.Text().String())
	})

	t.Run("backward selection opens above too", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 5,
			Head:   0,
		}}, 0)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "\na\nb\nc", doc.Text().String())
	})

	t.Run("negative range is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: -2,
			Head:   -1,
		}}, 0)

		action.OpenAbove(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("no view is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.OpenAbove(e) })
	})
}

func TestOpenBelow(t *testing.T) {
	t.Run("inserts blank line below", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "hello\n", doc.Text().String())
		assert.Equal(t, 6, testutil.CursorPos(t, e))
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("inserts below first line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetCursor(t, e, 0)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "a\n\nb", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("count repeats new lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)
		e.SetCount(2)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "hello\n\n", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("shared target keeps every cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(0), core.PointRange(1)},
			0,
		)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "abc\n\n", doc.Text().String())
		v := e.FocusedView()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})

	t.Run("opens below the last selected line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 0,
			Head:   3,
		}}, 0)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "a\nb\n\nc", doc.Text().String())
	})

	t.Run("backward selection opens below too", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 3,
			Head:   0,
		}}, 0)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "a\nb\n\nc", doc.Text().String())
	})

	t.Run("keeps the line's indent", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\tabc")
		testutil.SetCursor(t, e, 2)

		action.OpenBelow(e)

		doc := e.FocusedDocument()
		assert.Equal(t, "\tabc\n\t", doc.Text().String())
	})

	t.Run("no view is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.OpenBelow(e) })
	})
}
