package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestGotoFileTarget(t *testing.T) {
	t.Run("no file path under cursor returns error", func(t *testing.T) {
		e := editorWithText(t, "   ")
		setCursor(t, e, 1)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrNoFilePath)
	})

	t.Run("URL target returns URL", func(t *testing.T) {
		e := editorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "https://example.com"},
		})
		setCursor(t, e, 2)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", target.URL)
	})

	t.Run("file:// target returns local path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "linked.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))
		e := editorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "file://" + path},
		})
		setCursor(t, e, 2)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})

	t.Run("empty target no controller errors", func(t *testing.T) {
		e := editorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: ""},
		})
		setCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrDocumentLinkTarget)
	})

	t.Run("non-existent file path errors", func(t *testing.T) {
		e := editorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "/nonexistent/path/file.txt"},
		})
		setCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.Error(t, err)
	})

	t.Run("non-localhost host errors", func(t *testing.T) {
		e := editorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "file://remotehost/path"},
		})
		setCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrDocumentLinkTarget)
	})

	t.Run("overlapping selection returns target", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ref.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))
		e := editorWithText(t, "see here\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 4, To: 8, Target: "file://" + path},
		})
		setSelection(t, e, []core.Range{core.NewRange(3, 7)}, 0)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})
}
