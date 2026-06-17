package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/action"
)

func TestSearchConfig(t *testing.T) {
	t.Run("smart case matches lowercase pattern", func(t *testing.T) {
		e := editorWithText(t, "zz Alpha")

		err := action.SearchForward(e, "alpha")

		assert.NoError(t, err)
		assert.Equal(t, 3, cursorPos(t, e))
	})

	t.Run("disabled smart case is sensitive", func(t *testing.T) {
		e := editorWithText(t, "zz Alpha")
		cfg := e.Config()
		cfg.Editor.Search.SmartCase = new(false)
		e.SetConfig(cfg)

		err := action.SearchForward(e, "alpha")

		assert.NoError(t, err)
		assert.Equal(t, 0, cursorPos(t, e))
	})

	t.Run("disabled wrap around stops at end", func(t *testing.T) {
		e := editorWithText(t, "foo bar")
		cfg := e.Config()
		cfg.Editor.Search.WrapAround = new(false)
		e.SetConfig(cfg)
		doc, _ := e.FocusedDocument()
		setCursor(t, e, doc.Text().LenChars())

		err := action.SearchForward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, doc.Text().LenChars(), cursorPos(t, e))
	})
}
