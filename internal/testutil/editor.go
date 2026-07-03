package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// EditorWithText creates an editor seeded with text and the cursor at pos 0
func EditorWithText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	e.ResizeTree(80, 24)
	SetEditorText(t, e, text)
	return e
}

// SetEditorText inserts text into an existing editor's focused document
func SetEditorText(t *testing.T, e *view.Editor, text string) {
	t.Helper()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, 0, text),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).
		WithChanges(cs).
		WithSelection(core.PointSelection(0))
	assert.NoError(t, e.Apply(tx))
}

// CursorPos returns the cursor position in the focused document
func CursorPos(t *testing.T, e *view.Editor) int {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel := doc.SelectionFor(v.ID())
	return sel.Primary().Cursor(doc.Text())
}

// SetCursor moves the focused view's cursor to pos
func SetCursor(t *testing.T, e *view.Editor, pos int) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	doc.SetSelectionFor(v.ID(), core.PointSelection(pos))
}

// SetSelection replaces the focused view's selection
func SetSelection(
	t *testing.T, e *view.Editor, ranges []core.Range, primary int,
) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel, err := core.NewSelection(ranges, primary)
	assert.NoError(t, err)
	doc.SetSelectionFor(v.ID(), sel)
}
