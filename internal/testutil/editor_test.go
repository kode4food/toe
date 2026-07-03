package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
)

func TestEditorWithText(t *testing.T) {
	e := testutil.EditorWithText(t, "hello")

	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	assert.Equal(t, "hello", doc.Text().String())
	assert.Equal(t, 0, testutil.CursorPos(t, e))
}

func TestSetEditorText(t *testing.T) {
	e := testutil.EditorWithText(t, "")
	testutil.SetEditorText(t, e, "world")

	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	assert.Equal(t, "world", doc.Text().String())
}

func TestCursorPos(t *testing.T) {
	e := testutil.EditorWithText(t, "abc")
	testutil.SetCursor(t, e, 2)

	assert.Equal(t, 2, testutil.CursorPos(t, e))
}

func TestSetCursor(t *testing.T) {
	e := testutil.EditorWithText(t, "hello")
	testutil.SetCursor(t, e, 3)

	assert.Equal(t, 3, testutil.CursorPos(t, e))
}

func TestSetSelection(t *testing.T) {
	e := testutil.EditorWithText(t, "hello")
	testutil.SetSelection(t, e,
		[]core.Range{core.NewRange(1, 4)}, 0,
	)

	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel := doc.SelectionFor(v.ID())
	r := sel.Primary()
	assert.Equal(t, 1, r.Anchor)
	assert.Equal(t, 4, r.Head)
}
