package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/language"
)

func TestInsertChar(t *testing.T) {
	e := testutil.EditorWithText(t, "hllo")
	testutil.SetCursor(t, e, 1)
	action.InsertChar(e, 'e')
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "hello", doc.Text().String())
	assert.Equal(t, 2, testutil.CursorPos(t, e))
}

func TestInsertNewline(t *testing.T) {
	e := testutil.EditorWithText(t, "ab")
	testutil.SetCursor(t, e, 1)
	action.InsertNewline(e)
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "a\nb", doc.Text().String())
}

func TestInsertNewlineAutoPair(t *testing.T) {
	e := testutil.EditorWithText(t, "()")
	testutil.SetCursor(t, e, 1)
	action.InsertNewline(e)
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "(\n\t\n)", doc.Text().String())
	assert.Equal(t, 3, testutil.CursorPos(t, e))
}

func TestAutoPairConfig(t *testing.T) {
	t.Run("global false disables insert hook", func(t *testing.T) {
		e := testutil.EditorWithText(t, "")
		e.Options().HasAutoPairs = false

		action.InsertChar(e, '(')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(", doc.Text().String())
		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})

	t.Run("global false disables newline pair hook", func(t *testing.T) {
		e := testutil.EditorWithText(t, "()")
		e.Options().HasAutoPairs = false
		testutil.SetCursor(t, e, 1)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(\n)", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("global false disables delete hook", func(t *testing.T) {
		e := testutil.EditorWithText(t, "()")
		e.Options().HasAutoPairs = false
		testutil.SetCursor(t, e, 1)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, ")", doc.Text().String())
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("language table overrides editor default", func(t *testing.T) {
		writeCommandLanguages(t, `
[[language]]
name = "custom"

[language.auto-pairs]
'<' = '>'
`)
		lang := language.LoadLanguage("custom")
		pairs, ok := lang.AutoPairs.AutoPairs()
		assert.True(t, ok)
		pair, ok := pairs.Get('<')
		assert.True(t, ok)
		assert.Equal(t, core.Pair{Open: '<', Close: '>'}, pair)
		e := testutil.EditorWithText(t, "")
		doc, _ := e.FocusedDocument()
		doc.SetLang("custom")

		action.InsertChar(e, '(')
		action.InsertChar(e, '<')

		assert.Equal(t, "(<>", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestContinueComments(t *testing.T) {
	t.Run("insert newline continues line comment", func(t *testing.T) {
		writeCommandLanguages(t, `
[[language]]
name = "custom"
comment-token = "//"
`)
		e := testutil.EditorWithText(t, "  // hello")
		doc, _ := e.FocusedDocument()
		doc.SetLang("custom")
		testutil.SetCursor(t, e, doc.Text().LenChars())

		action.InsertNewline(e)

		assert.Equal(t, "  // hello\n  // ", doc.Text().String())
		assert.Equal(t, doc.Text().LenChars(), testutil.CursorPos(t, e))
	})

	t.Run("open above continues line comment", func(t *testing.T) {
		writeCommandLanguages(t, `
[[language]]
name = "custom"
comment-token = "//"
`)
		e := testutil.EditorWithText(t, "  // hello")
		doc, _ := e.FocusedDocument()
		doc.SetLang("custom")

		action.OpenAbove(e)

		assert.Equal(t, "  // \n  // hello", doc.Text().String())
		assert.Equal(t, 5, testutil.CursorPos(t, e))
	})

	t.Run("can disable continuation", func(t *testing.T) {
		writeCommandLanguages(t, `
[[language]]
name = "custom"
comment-token = "//"
`)
		e := testutil.EditorWithText(t, "  // hello")
		e.Options().ContinueComments = false
		doc, _ := e.FocusedDocument()
		doc.SetLang("custom")
		testutil.SetCursor(t, e, doc.Text().LenChars())

		action.InsertNewline(e)

		assert.Equal(t, "  // hello\n  ", doc.Text().String())
	})

	t.Run("open above repeated cursors", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetCount(2)

		action.OpenAbove(e)

		doc, _ := e.FocusedDocument()
		v, _ := e.FocusedView()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, "\n\nhello", doc.Text().String())
		assert.Equal(t, []core.Range{
			core.PointRange(0),
			core.PointRange(1),
		}, sel.Ranges())
	})
}

func TestDeleteCharBackward(t *testing.T) {
	e := testutil.EditorWithText(t, "hello")
	testutil.SetCursor(t, e, 3)
	action.DeleteCharBackward(e)
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "helo", doc.Text().String())
	assert.Equal(t, 2, testutil.CursorPos(t, e))
}

func TestDeleteCharBackwardAtStart(t *testing.T) {
	e := testutil.EditorWithText(t, "hi")
	action.DeleteCharBackward(e)
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "hi", doc.Text().String())
}

func TestDeleteCharForward(t *testing.T) {
	e := testutil.EditorWithText(t, "hello")
	action.DeleteCharForward(e)
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "ello", doc.Text().String())
	assert.Equal(t, 0, testutil.CursorPos(t, e))
}

func TestDeleteCharForwardAtEnd(t *testing.T) {
	e := testutil.EditorWithText(t, "hi")
	testutil.SetCursor(t, e, 2)
	action.DeleteCharForward(e)
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "hi", doc.Text().String())
}

func TestInsertCharMultiCursor(t *testing.T) {
	e := testutil.EditorWithText(t, "ab")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(), newSelection(t, []core.Range{
		core.PointRange(0),
		core.PointRange(1),
	}, 0))
	action.InsertChar(e, 'x')
	doc, _ = e.FocusedDocument()
	assert.Equal(t, "xaxb", doc.Text().String())
}

func TestUndoRedo(t *testing.T) {
	e := testutil.EditorWithText(t, "hello")
	testutil.SetCursor(t, e, 5)
	action.InsertChar(e, '!')
	doc, _ := e.FocusedDocument()
	assert.Equal(t, "hello!", doc.Text().String())

	ok := e.Undo()
	assert.True(t, ok)
	doc, _ = e.FocusedDocument()
	assert.Equal(t, "hello", doc.Text().String())

	ok = e.Redo()
	assert.True(t, ok)
	doc, _ = e.FocusedDocument()
	assert.Equal(t, "hello!", doc.Text().String())
}

func TestChangeSelection(t *testing.T) {
	e := testutil.EditorWithText(t, "hello world")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(
		v.ID(), newSelection(t, []core.Range{core.NewRange(0, 5)}, 0),
	)

	action.ChangeSelection(e)

	assert.Equal(t, " world", doc.Text().String())
	assert.Equal(t, view.ModeInsert, e.Mode())
	assert.Equal(t, "hello", registeredValue(t, e, '"'))
}
