package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/action"
)

func TestCloseCurrentView(t *testing.T) {
	t.Run("blocks modified view when others exist", func(t *testing.T) {
		e := editorWithText(t, "abc")
		v, _ := e.FocusedView()
		// Create two additional splits so there are 3 views total
		e.VSplit(v.DocID())
		e.VSplit(v.DocID())

		action.InsertMode(e)
		action.InsertChar(e, 'x')
		before := viewCount(t, e)

		action.CloseCurrentView(e)

		assert.Equal(t, before, viewCount(t, e))
	})

	t.Run("closes the only modified view", func(t *testing.T) {
		e := editorWithText(t, "abc")
		action.InsertMode(e)
		action.InsertChar(e, 'x')

		action.CloseCurrentView(e)

		assert.Equal(t, 0, viewCount(t, e))
	})
}

func TestCloseOtherViews(t *testing.T) {
	e := editorWithText(t, "abc")
	v, _ := e.FocusedView()
	// Create two additional splits
	e.VSplit(v.DocID())
	e.VSplit(v.DocID())

	action.CloseOtherViews(e)

	assert.Equal(t, 1, viewCount(t, e))
}
