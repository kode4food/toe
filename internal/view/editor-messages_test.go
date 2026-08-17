package view_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestMessagesDocument(t *testing.T) {
	t.Run("appends every message", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		e.AppendMessage("first")
		e.AppendMessage("second")

		doc := e.MessagesDocument()
		assert.Equal(t, "first\nsecond\n", doc.Text().String())
		assert.Equal(t, view.MessagesBufferName, doc.DisplayName())
		assert.True(t, doc.ReadOnly())
		assert.False(t, doc.Modified())
	})

	t.Run("cannot be renamed or written", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.AppendMessage("news")
		doc := e.MessagesDocument()

		doc.SetPath(filepath.Join(t.TempDir(), "log.txt"))

		assert.Equal(t, view.DocTypeLog, doc.Type())
		assert.Equal(t, "", doc.Path())
		assert.Equal(t, view.MessagesBufferName, doc.DisplayName())
		assert.True(t, errors.Is(doc.Save(e.Options(), true), view.ErrReadOnly))
	})

	t.Run("copies out without changing", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.AppendMessage("news")
		doc := e.MessagesDocument()
		path := filepath.Join(t.TempDir(), "log.txt")

		assert.NoError(t, doc.WriteCopy(path, e.Options()))

		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, "news\n", string(data))
		assert.Equal(t, "", doc.Path())
		assert.Equal(t, view.MessagesBufferName, doc.DisplayName())

		e.AppendMessage("more news")
		assert.Equal(t, "news\nmore news\n", doc.Text().String())
	})

	t.Run("listed as a buffer", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		e.AppendMessage("hello")

		var found bool
		for _, d := range e.AllDocuments() {
			found = found || d.DisplayName() == view.MessagesBufferName
		}
		assert.True(t, found)
	})
}
