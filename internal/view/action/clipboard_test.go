package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view/action"
)

func TestClipboardNoProvider(t *testing.T) {
	// the default editor clipboard is a no-op; actions must not panic
	t.Run("yank", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		assert.NotPanics(t, func() { action.YankToClipboard(e) })
	})

	t.Run("paste", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.PasteClipboardAfter(e) })
	})

	t.Run("yank primary", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		assert.NotPanics(t, func() { action.YankToPrimaryClipboard(e) })
	})

	t.Run("paste primary", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.PastePrimaryClipboardAfter(e) })
	})

	t.Run("replace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		assert.NotPanics(t, func() { action.ClipboardReplace(e) })
	})
}

func TestOSC52Clipboard(t *testing.T) {
	t.Run("write forwards to inner clipboard", func(t *testing.T) {
		inner := testutil.NewFakeClipboard()
		clip := action.NewOSC52Clipboard(inner)

		assert.NoError(t, clip.Write("hello"))

		assert.Equal(t, "hello", inner.System)
	})

	t.Run("write primary forwards to inner clipboard", func(t *testing.T) {
		inner := testutil.NewFakeClipboard()
		clip := action.NewOSC52Clipboard(inner)

		assert.NoError(t, clip.WritePrimary("hello"))

		assert.Equal(t, "hello", inner.Primary)
	})
}

func TestClipboard(t *testing.T) {
	t.Run("yank to clipboard", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.YankToClipboard(e)

		assert.Equal(t, "hello", clip.System)
		assert.Equal(t, "hello", testutil.RegisteredValue(t, e, '+'))
	})

	t.Run("paste after/before", func(t *testing.T) {
		clip := testutil.NewFakeClipboard()
		clip.System = "hello"

		e := testutil.EditorWithText(t, "x")
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		action.PasteClipboardAfter(e)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xhello", doc.Text().String())

		e = testutil.EditorWithText(t, "x")
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		action.PasteClipboardBefore(e)
		doc, _ = e.FocusedDocument()
		assert.Equal(t, "hellox", doc.Text().String())
	})

	t.Run("paste preserves yanked selections", func(t *testing.T) {
		clip := testutil.NewFakeClipboard()

		e := testutil.EditorWithText(t, "abcd")
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 1),
			core.NewRange(2, 3),
		}, 0)
		action.YankToClipboard(e)

		e = testutil.EditorWithText(t, "wxyz")
		e.SetClipboard(clip)
		e.WriteRegister('+', []string{"a", "c"})
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(1),
			core.PointRange(3),
		}, 0)

		action.PasteClipboardAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "waxycz", doc.Text().String())
	})

	t.Run("yank main to clipboard", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 5),
			core.NewRange(6, 11),
		}, 0)

		action.YankMainToClipboard(e)

		assert.Equal(t, "hello", clip.System)
	})

	t.Run("clipboard replace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		clip := testutil.NewFakeClipboard()
		clip.System = "XY"
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)

		action.ClipboardReplace(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aXYc", doc.Text().String())
	})

	t.Run("yank to primary clipboard", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		clip := testutil.NewFakeClipboard()
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.YankToPrimaryClipboard(e)

		assert.Equal(t, "hello", clip.Primary)
	})

	t.Run("paste primary after", func(t *testing.T) {
		e := testutil.EditorWithText(t, "x")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "hi"
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		action.PastePrimaryClipboardAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xhi", doc.Text().String())
	})

	t.Run("paste primary before", func(t *testing.T) {
		e := testutil.EditorWithText(t, "x")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "hi"
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)

		action.PastePrimaryClipboardBefore(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hix", doc.Text().String())
	})

	t.Run("primary replace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "Z"
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)

		action.PrimaryClipboardReplace(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aZc", doc.Text().String())
	})
}
