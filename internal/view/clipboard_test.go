package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view/action"
)

func TestYank(t *testing.T) {
	e := testutil.EditorWithText(t, "hello world")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(
		v.ID(), newSelection(t, []core.Range{core.NewRange(0, 5)}, 0),
	)

	action.Yank(e)

	assert.Equal(t, "hello", registeredValue(t, e, '"'))
}

func TestPasteAfter(t *testing.T) {
	e := testutil.EditorWithText(t, "ab")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(
		v.ID(), newSelection(t, []core.Range{core.NewRange(1, 2)}, 0),
	)
	action.Yank(e)

	blockA := newSelection(t, []core.Range{core.NewRange(0, 1)}, 0)
	doc.SetSelectionFor(v.ID(), blockA)
	action.PasteAfter(e)

	assert.Equal(t, "abb", doc.Text().String())
}

func TestPasteBefore(t *testing.T) {
	e := testutil.EditorWithText(t, "ab")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(
		v.ID(), newSelection(t, []core.Range{core.NewRange(1, 2)}, 0),
	)
	action.Yank(e)

	doc.SetSelectionFor(v.ID(), core.PointSelection(1))
	action.PasteBefore(e)

	assert.Equal(t, "abb", doc.Text().String())
}

func TestPasteAfterLinewise(t *testing.T) {
	e := testutil.EditorWithText(t, "foo\nbar")
	v, _ := e.FocusedView()
	doc, _ := e.FocusedDocument()
	e.Registers().Write('"', []string{"baz\n"})

	doc.SetSelectionFor(v.ID(), core.PointSelection(0))
	action.PasteAfter(e)

	assert.Equal(t, "foo\nbaz\nbar", doc.Text().String())
}
