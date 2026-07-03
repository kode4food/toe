package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view/action"
)

func TestSearchConfig(t *testing.T) {
	t.Run("smart case matches lowercase pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "zz Alpha")

		err := action.SearchForward(e, "alpha")

		assert.NoError(t, err)
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("disabled smart case is sensitive", func(t *testing.T) {
		e := testutil.EditorWithText(t, "zz Alpha")
		e.Options().SearchSmartCase = false

		err := action.SearchForward(e, "alpha")

		assert.NoError(t, err)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("disabled wrap around stops at end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		e.Options().SearchWrapAround = false
		doc, _ := e.FocusedDocument()
		testutil.SetCursor(t, e, doc.Text().LenChars())

		err := action.SearchForward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, doc.Text().LenChars(), testutil.CursorPos(t, e))
	})
}
