package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view/action"
)

func TestClipboardNoProvider(t *testing.T) {
	t.Run("YankToClipboard", func(t *testing.T) {
		t.Setenv("PATH", "")
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		assert.NotPanics(t, func() { action.YankToClipboard(e) })
	})

	t.Run("PasteClipboardAfter", func(t *testing.T) {
		t.Setenv("PATH", "")
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.PasteClipboardAfter(e) })
	})

	t.Run("YankToPrimaryClipboard", func(t *testing.T) {
		t.Setenv("PATH", "")
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		assert.NotPanics(t, func() { action.YankToPrimaryClipboard(e) })
	})

	t.Run("PastePrimaryClipboardAfter", func(t *testing.T) {
		t.Setenv("PATH", "")
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.PastePrimaryClipboardAfter(e) })
	})

	t.Run("ClipboardReplace", func(t *testing.T) {
		t.Setenv("PATH", "")
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		assert.NotPanics(t, func() { action.ClipboardReplace(e) })
	})

	t.Run("ShowClipboardProvider returns none", func(t *testing.T) {
		t.Setenv("PATH", "")

		result := action.ShowClipboardProvider()

		assert.Equal(t, "none", result)
	})
}

func TestClipboard(t *testing.T) {
	t.Run("show provider", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)

		assert.Equal(t, "pbcopy", action.ShowClipboardProvider())
	})

	t.Run("yank to clipboard", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.YankToClipboard(e)

		data, err := os.ReadFile(clipFile)
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
		assert.Equal(t, "hello", registeredValue(t, e, '+'))
	})

	t.Run("paste after/before", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("hello"), 0o644))

		e := testutil.EditorWithText(t, "x")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		action.PasteClipboardAfter(e)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xhello", doc.Text().String())

		e = testutil.EditorWithText(t, "x")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		action.PasteClipboardBefore(e)
		doc, _ = e.FocusedDocument()
		assert.Equal(t, "hellox", doc.Text().String())
	})

	t.Run("yank main selection to clipboard", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 5),
			core.NewRange(6, 11),
		}, 0)

		action.YankMainToClipboard(e)

		data, err := os.ReadFile(clipFile)
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})

	t.Run("clipboard replace", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)

		action.ClipboardReplace(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aXYc", doc.Text().String())
	})

	t.Run("yank to primary clipboard", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.YankToPrimaryClipboard(e)

		data, err := os.ReadFile(clipFile)
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})

	t.Run("paste primary clipboard after", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("hi"), 0o644))
		e := testutil.EditorWithText(t, "x")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		action.PastePrimaryClipboardAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xhi", doc.Text().String())
	})

	t.Run("paste primary clipboard before", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("hi"), 0o644))
		e := testutil.EditorWithText(t, "x")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		action.PastePrimaryClipboardBefore(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hix", doc.Text().String())
	})

	t.Run("primary clipboard replace", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("Z"), 0o644))
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)

		action.PrimaryClipboardReplace(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aZc", doc.Text().String())
	})
}
