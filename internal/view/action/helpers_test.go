package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func editorWithText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	e.ResizeTree(80, 24)
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
	return e
}

func editorWithNoView(t *testing.T) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	e.CloseView(v.ID())
	return e
}

func setCursor(t *testing.T, e *view.Editor, pos int) {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	doc.SetSelectionFor(v.ID(), core.PointSelection(pos))
}

func setSelection(
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

func cursorPos(t *testing.T, e *view.Editor) int {
	t.Helper()
	v, ok := e.FocusedView()
	assert.True(t, ok)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	sel := doc.SelectionFor(v.ID())
	return sel.Primary().Cursor(doc.Text())
}

func viewCount(t *testing.T, e *view.Editor) int {
	t.Helper()
	return len(e.AllViews())
}

func registeredValue(t *testing.T, e *view.Editor, reg rune) string {
	t.Helper()
	v, ok := e.Registers().First(reg)
	assert.True(t, ok)
	return v
}
