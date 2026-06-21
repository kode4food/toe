package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestSelectAll(t *testing.T) {
	e := editorWithText(t, "hello")
	action.SelectAll(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, a)
	assert.Equal(t, 5, h)
}

func TestCollapseSelection(t *testing.T) {
	e := editorWithText(t, "hello")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(),
		newSelection(t, []core.Range{core.NewRange(0, 3)}, 0),
	)
	action.CollapseSelection(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, a, h)
}

func TestFlipSelections(t *testing.T) {
	e := editorWithText(t, "hello")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(),
		newSelection(t, []core.Range{core.NewRange(1, 4)}, 0),
	)
	action.FlipSelections(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 4, a)
	assert.Equal(t, 1, h)
}

func TestKeepPrimarySelection(t *testing.T) {
	e := editorWithText(t, "abcdef")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(), newSelection(t, []core.Range{
		core.NewRange(0, 2),
		core.NewRange(3, 5),
	}, 0))
	action.KeepPrimarySelection(e)
	selAfter := doc.SelectionFor(v.ID())
	assert.Equal(t, 1, len(selAfter.Ranges()))
	assert.Equal(t, 0, selAfter.Primary().Anchor)
	assert.Equal(t, 2, selAfter.Primary().Head)
}

func TestDeleteSelection(t *testing.T) {
	t.Run("deletes selected range", func(t *testing.T) {
		e := editorWithText(t, "hello world")
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(),
			newSelection(t, []core.Range{core.NewRange(0, 5)}, 0),
		)
		action.DeleteSelection(e)
		doc, _ = e.FocusedDocument()
		assert.Equal(t, " world", doc.Text().String())
	})

	t.Run("deletes char under collapsed cursor", func(t *testing.T) {
		e := editorWithText(t, "abc")
		action.DeleteSelection(e)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "bc", doc.Text().String())
	})

	t.Run("enters normal mode after delete", func(t *testing.T) {
		e := editorWithText(t, "hi")
		e.SetMode(view.ModeSelect)
		action.DeleteSelection(e)
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestExtendLineBellow(t *testing.T) {
	t.Run("selects current line including newline", func(t *testing.T) {
		e := editorWithText(t, "abc\ndef\n")
		action.ExtendLineBellow(e)
		a, h := selectionAnchorHead(t, e)
		assert.Equal(t, 0, a)
		assert.Equal(t, 4, h)
	})

	t.Run("extends when line already selected", func(t *testing.T) {
		e := editorWithText(t, "abc\ndef\n")
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		doc.SetSelectionFor(v.ID(),
			newSelection(t, []core.Range{core.NewRange(0, 4)}, 0),
		)
		action.ExtendLineBellow(e)
		a, h := selectionAnchorHead(t, e)
		assert.Equal(t, 0, a)
		assert.Equal(t, 8, h)
	})
}

func TestExtendToLineBounds(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	action.ExtendToLineBounds(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, a)
	assert.True(t, h > 0)
}

func TestShrinkToLineBounds(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	action.ExtendToLineBounds(e)
	action.ShrinkToLineBounds(e)
	a, h := selectionAnchorHead(t, e)
	assert.True(t, h >= a)
}

func TestRotateSelections(t *testing.T) {
	e := editorWithText(t, "abcdef")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(), newSelection(t, []core.Range{
		core.NewRange(0, 1),
		core.NewRange(2, 3),
	}, 0))
	action.RotateSelectionsForward(e)
	action.RotateSelectionsBackward(e)
	assert.NotNil(t, doc)
}

func TestJoinSelections(t *testing.T) {
	t.Run("strips repeated language comment token", func(t *testing.T) {
		writeCommandLanguages(t, `
[[language]]
name = "custom"
comment-token = "//"
`)
		e := editorWithText(t, "// foo\n// bar\n")
		doc, _ := e.FocusedDocument()
		doc.SetLang("custom")

		action.JoinSelections(e)

		assert.Equal(t, "// foo bar\n", doc.Text().String())
	})

	t.Run("selects inserted spaces", func(t *testing.T) {
		e := editorWithText(t, "foo\nbar\n")

		action.JoinSelectionsSpace(e)

		doc, _ := e.FocusedDocument()
		v, _ := e.FocusedView()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, "foo bar\n", doc.Text().String())
		assert.Equal(t, 3, sel.Primary().Cursor(doc.Text()))
	})
}

func TestSetLineEnding(t *testing.T) {
	t.Run("converts lf to crlf", func(t *testing.T) {
		e := editorWithText(t, "a\nb\n")

		err := action.SetLineEnding(e, core.LineEndingCRLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\r\nb\r\n", doc.Text().String())
		assert.Equal(t, core.LineEndingCRLF, doc.LineEnding())
	})

	t.Run("converts crlf to lf", func(t *testing.T) {
		e := editorWithText(t, "a\r\nb\r\n")

		err := action.SetLineEnding(e, core.LineEndingLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\nb\n", doc.Text().String())
		assert.Equal(t, core.LineEndingLF, doc.LineEnding())
	})
}
