package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view/action"
)

func TestResizeView(t *testing.T) {
	t.Run("grows left split, default count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		leftID := e.Tree().Focus()
		e.VSplitNew()
		e.Tree().SetFocus(leftID)
		before := e.Views()[0].View.Area().Width

		action.ResizeViewRight(e)

		assert.Equal(t, before+1, e.Views()[0].View.Area().Width)
	})

	t.Run("grows left split by count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		leftID := e.Tree().Focus()
		e.VSplitNew()
		e.Tree().SetFocus(leftID)
		before := e.Views()[0].View.Area().Width
		e.SetCount(4)

		action.ResizeViewRight(e)

		assert.Equal(t, before+4, e.Views()[0].View.Area().Width)
		assert.Equal(t, 0, e.Count())
	})

	t.Run("shrinks left split by count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		leftID := e.Tree().Focus()
		e.VSplitNew()
		e.Tree().SetFocus(leftID)
		before := e.Views()[0].View.Area().Width
		e.SetCount(3)

		action.ResizeViewLeft(e)

		assert.Equal(t, before-3, e.Views()[0].View.Area().Width)
	})

	t.Run("grows right split by count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.VSplitNew()
		before := e.Views()[1].View.Area().Width
		e.SetCount(3)

		action.ResizeViewLeft(e)

		assert.Equal(t, before+3, e.Views()[1].View.Area().Width)
	})

	t.Run("grows top split by count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		topID := e.Tree().Focus()
		e.HSplitNew()
		e.Tree().SetFocus(topID)
		before := e.Views()[0].View.Area().Height
		e.SetCount(2)

		action.ResizeViewDown(e)

		assert.Equal(t, before+2, e.Views()[0].View.Area().Height)
	})

	t.Run("grows bottom split by count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.HSplitNew()
		before := e.Views()[1].View.Area().Height
		e.SetCount(2)

		action.ResizeViewUp(e)

		assert.Equal(t, before+2, e.Views()[1].View.Area().Height)
	})

	t.Run("shrinks bottom split by count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.HSplitNew()
		before := e.Views()[1].View.Area().Height
		e.SetCount(2)

		action.ResizeViewDown(e)

		assert.Equal(t, before-2, e.Views()[1].View.Area().Height)
	})

	t.Run("no matching axis is a no-op", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.VSplitNew()
		before := e.Views()[0].View.Area()

		action.ResizeViewDown(e)

		assert.Equal(t, before, e.Views()[0].View.Area())
	})
}
