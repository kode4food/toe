package action_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestGotoFileTarget(t *testing.T) {
	t.Run("no file path under cursor returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "   ")
		testutil.SetCursor(t, e, 1)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrNoFilePath)
	})

	t.Run("URL target returns URL", func(t *testing.T) {
		e := testutil.EditorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "https://example.com"},
		})
		testutil.SetCursor(t, e, 2)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", target.URL)
	})

	t.Run("file:// target returns local path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "linked.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))
		e := testutil.EditorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "file://" + path},
		})
		testutil.SetCursor(t, e, 2)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})

	t.Run("empty target no controller errors", func(t *testing.T) {
		e := testutil.EditorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: ""},
		})
		testutil.SetCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrDocumentLinkTarget)
	})

	t.Run("non-existent file path errors", func(t *testing.T) {
		e := testutil.EditorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "/nonexistent/path/file.txt"},
		})
		testutil.SetCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.Error(t, err)
	})

	t.Run("non-localhost host errors", func(t *testing.T) {
		e := testutil.EditorWithText(t, "some text\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 4, Target: "file://remotehost/path"},
		})
		testutil.SetCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrDocumentLinkTarget)
	})

	t.Run("overlapping selection returns target", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ref.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))
		e := testutil.EditorWithText(t, "see here\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 4, To: 8, Target: "file://" + path},
		})
		testutil.SetSelection(t, e, []core.Range{core.NewRange(3, 7)}, 0)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})

	t.Run("empty target becomes file URL", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "resolved.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))
		e := testutil.EditorWithText(t, "link\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{{From: 0, To: 4, Target: ""}})
		e.SetLanguageServerController(&stubLSP{
			resolveLink: func(
				_ *view.Document, lnk view.DocumentLink,
			) (view.DocumentLink, error) {
				lnk.Target = "file://" + path
				return lnk, nil
			},
		})
		testutil.SetCursor(t, e, 2)
		target, err := action.GotoFileTarget(e)
		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})

	t.Run("controller returns error on resolve", func(t *testing.T) {
		e := testutil.EditorWithText(t, "link\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{{From: 0, To: 4, Target: ""}})
		e.SetLanguageServerController(&stubLSP{
			resolveLink: func(
				_ *view.Document, lnk view.DocumentLink,
			) (view.DocumentLink, error) {
				return lnk, errors.New("lsp failed")
			},
		})
		testutil.SetCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.Error(t, err)
	})

	t.Run("empty resolved target errors", func(t *testing.T) {
		e := testutil.EditorWithText(t, "link\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{{From: 0, To: 4, Target: ""}})
		e.SetLanguageServerController(&stubLSP{
			resolveLink: func(
				_ *view.Document, lnk view.DocumentLink,
			) (view.DocumentLink, error) {
				return lnk, nil
			},
		})
		testutil.SetCursor(t, e, 2)
		_, err := action.GotoFileTarget(e)
		assert.ErrorIs(t, err, action.ErrDocumentLinkTarget)
	})
}
