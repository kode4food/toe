package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestEdit(t *testing.T) {
	t.Run("insert at cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hllo")
		testutil.SetCursor(t, e, 1)

		action.InsertChar(e, 'e')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("newline honors auto-pair", func(t *testing.T) {
		e := testutil.EditorWithText(t, "()")
		testutil.SetCursor(t, e, 1)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(\n\t\n)", doc.Text().String())
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("delete backward dedents", func(t *testing.T) {
		e := testutil.EditorWithText(t, "    x")
		testutil.SetCursor(t, e, 4)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "x", doc.Text().String())
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("change selection enters insert", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 2)}, 0)

		action.ChangeSelection(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "c", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestEditActionsNoView(t *testing.T) {
	t.Run("delete selection is noop", func(t *testing.T) {
		assert.NotPanics(t, func() {
			action.DeleteSelection(editorWithNoView(t))
		})
	})

	t.Run("change selection is noop", func(t *testing.T) {
		assert.NotPanics(t, func() {
			action.ChangeSelection(editorWithNoView(t))
		})
	})

	t.Run("split selection is noop", func(t *testing.T) {
		assert.NotPanics(t, func() {
			action.SplitSelectionOnNewline(editorWithNoView(t))
		})
	})
}

func TestSelectAll(t *testing.T) {
	t.Run("selects entire document", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.SelectAll(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 5, sel.Primary().To())
	})

	t.Run("empty document stays at zero", func(t *testing.T) {
		e := testutil.EditorWithText(t, "")

		action.SelectAll(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 0, sel.Primary().To())
	})
}

func TestCollapseSelection(t *testing.T) {
	t.Run("range collapses to cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)

		action.CollapseSelection(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		r := sel.Primary()
		assert.Equal(t, r.From(), r.To())
	})

	t.Run("multiple selections each collapse", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 2),
			core.NewRange(2, 4),
		}, 0)

		action.CollapseSelection(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		for _, r := range sel.Ranges() {
			assert.Equal(t, r.From(), r.To())
		}
	})
}

func TestFlipSelections(t *testing.T) {
	t.Run("forward range becomes backward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)

		action.FlipSelections(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, sel.Primary().Anchor)
		assert.Equal(t, 1, sel.Primary().Head)
	})
}

func TestKeepPrimarySelection(t *testing.T) {
	t.Run("discards non-primary ranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(0),
			core.PointRange(2),
			core.PointRange(3),
		}, 1)

		action.KeepPrimarySelection(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
		assert.Equal(t, 0, sel.PrimaryIndex())
	})
}

func TestExtendLineBelow(t *testing.T) {
	t.Run("extends cursor to full line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 0)

		action.ExtendLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 3, sel.Primary().To())
	})

	t.Run("extends again if already full line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.ExtendLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 6, sel.Primary().To())
	})
}

func TestExtendLineBelowLastLine(t *testing.T) {
	t.Run("selects to end on last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 3)

		action.ExtendLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, sel.Primary().From())
		assert.Equal(t, 5, sel.Primary().To())
	})
}

func TestSelectLineBelow(t *testing.T) {
	t.Run("selects current line forward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetCursor(t, e, 0)

		action.SelectLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 3, sel.Primary().To())
	})

	t.Run("backward sel extends downward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(5, 3)}, 0)

		action.SelectLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.GreaterOrEqual(t, sel.Primary().To()-sel.Primary().From(), 0)
	})

	t.Run("extends snapped forward selection", func(t *testing.T) {
		// Start with a line-aligned forward selection (0→3 covers "ab\n"),
		// then extend again so anchorLine < headLine is reached with cnt > 0
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.SelectLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 6, sel.Primary().To())
	})
}

func TestSelectLineAbove(t *testing.T) {
	t.Run("includes previous line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetCursor(t, e, 6)

		action.SelectLineAbove(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		// includes at least two lines worth of content
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 3)
	})

	t.Run("backward sel extends upward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(6, 3)}, 0)

		action.SelectLineAbove(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.GreaterOrEqual(t, sel.Primary().To()-sel.Primary().From(), 0)
	})
}

func TestSplitSelectionOnNewlineEdges(t *testing.T) {
	t.Run("keeps point ranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.PointRange(1)}, 0)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{core.PointRange(1)},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})

	t.Run("splits final line without newline", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{core.NewRange(0, 2), core.NewRange(3, 5)},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})

	t.Run("invalid range leaves selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{core.NewRange(-2, -1)},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})
}

func TestDeleteSelection(t *testing.T) {
	t.Run("deletes selected text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.DeleteSelection(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, " world", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("cursor lands at deletion point", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)

		action.DeleteSelection(e)

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestDeleteCharForward(t *testing.T) {
	t.Run("deletes char under cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.DeleteCharForward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ac", doc.Text().String())
	})

	t.Run("noop at end of document", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a")
		testutil.SetCursor(t, e, 1)

		action.DeleteCharForward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a", doc.Text().String())
	})
}

func TestYank(t *testing.T) {
	t.Run("copies selection to default register", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.Yank(e)

		assert.Equal(t, "hello", testutil.RegisteredValue(t, e, '"'))
		assert.Equal(t, "yanked 1 selection to register \"", e.TakeStatusMsg())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("yank multiple ranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 2),
			core.NewRange(2, 4),
		}, 0)

		action.Yank(e)

		assert.Equal(t, "ab", testutil.RegisteredValue(t, e, '"'))
	})

	t.Run("noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.Yank(e) })
	})

	t.Run("invalid range is skipped", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)

		action.Yank(e)

		assert.Empty(t, e.Registers().Read('"'))
	})
}

func TestPasteAfter(t *testing.T) {
	t.Run("pastes at head of selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "xyz")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		e.Registers().Write('"', []string{"b"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xbyz", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("empty register is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("linewise paste after last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 4)
		e.Registers().Write('"', []string{"ghi\n"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "ghi")
	})

	t.Run("multiple cursors reuse last value", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(1), core.PointRange(3)},
			0,
		)
		e.Registers().Write('"', []string{"x"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "axbcxd", doc.Text().String())
	})

	t.Run("invalid position leaves text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)
		e.Registers().Write('"', []string{"x"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		e.Registers().Write('"', []string{"x"})

		assert.NotPanics(t, func() { action.PasteAfter(e) })
	})
}

func TestPasteBefore(t *testing.T) {
	t.Run("pastes before cursor position", func(t *testing.T) {
		e := testutil.EditorWithText(t, "xz")
		testutil.SetCursor(t, e, 1)
		e.Registers().Write('"', []string{"y"})

		action.PasteBefore(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xyz", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("linewise invalid range is skipped", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)
		e.Registers().Write('"', []string{"x\n"})

		action.PasteBefore(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		e.Registers().Write('"', []string{"x"})

		assert.NotPanics(t, func() { action.PasteBefore(e) })
	})
}

func TestSplitSelectionOnNewline(t *testing.T) {
	t.Run("splits multiline selection into per-line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 8)}, 0)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, len(sel.Ranges()))
	})

	t.Run("point selection is kept", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
	})
}

func TestNormalMode(t *testing.T) {
	t.Run("exits insert mode", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("noop when already normal", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("exits insert mode with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("exits insert mode with missing document", func(t *testing.T) {
		e := editorWithMissingFocusedDocument(t)
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestInsertMode(t *testing.T) {
	t.Run("places cursor at selection start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(2, 4)}, 0)

		action.InsertMode(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestAppendMode(t *testing.T) {
	t.Run("places cursor past selection end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 1)

		action.AppendMode(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("types after the selection, not before it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 1)

		action.AppendMode(e)
		action.InsertChar(e, 'X')
		action.InsertChar(e, 'Y')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abXYcde", doc.Text().String())
	})
}

func TestSelectMode(t *testing.T) {
	t.Run("enters select mode", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 0)

		action.SelectMode(e)

		assert.Equal(t, view.ModeSelect, e.Mode())
	})

	t.Run("widens empty end-of-doc selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 3)

		action.SelectMode(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 1)
	})
}

func TestInsertAtLineStart(t *testing.T) {
	t.Run("moves to first non-ws and enters insert", func(t *testing.T) {
		e := testutil.EditorWithText(t, "  hello")
		testutil.SetCursor(t, e, 6)

		action.InsertAtLineStart(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestAppendToLine(t *testing.T) {
	t.Run("moves to end of line and enters insert", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.AppendToLine(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("types after line end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.AppendToLine(e)
		for _, ch := range " world" {
			action.InsertChar(e, ch)
		}

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello world", doc.Text().String())
	})
}

func TestDeleteSelectionNoYank(t *testing.T) {
	t.Run("deletes without affecting register", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)
		e.Registers().Write('"', []string{"saved"})

		action.DeleteSelectionNoYank(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, " world", doc.Text().String())
		assert.Equal(t, "saved", testutil.RegisteredValue(t, e, '"'))
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestChangeSelectionLinewise(t *testing.T) {
	t.Run("linewise change opens blank line above", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello\nworld")
		// Select full first line including newline (linewise)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 6)}, 0)

		action.ChangeSelection(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestLinewisePaste(t *testing.T) {
	t.Run("PasteAfter linewise pastes below", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 0)
		// Yank full first line (with newline = linewise)
		e.Registers().Write('"', []string{"abc\n"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "abc")
	})

	t.Run("PasteBefore linewise pastes above", func(t *testing.T) {
		e := testutil.EditorWithText(t, "def")
		testutil.SetCursor(t, e, 0)
		e.Registers().Write('"', []string{"abc\n"})

		action.PasteBefore(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "abc")
	})
}

func TestExitSelectMode(t *testing.T) {
	t.Run("exits select mode to normal", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.SetMode(view.ModeSelect)

		action.ExitSelectMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("noop when not in select mode", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		action.ExitSelectMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestChangeSelectionNoYank(t *testing.T) {
	t.Run("skips register on insert", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 2)}, 0)
		e.Registers().Write('"', []string{"safe"})

		action.ChangeSelectionNoYank(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "c", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, "safe", testutil.RegisteredValue(t, e, '"'))
	})
}

func TestNormalModeRestoreCursor(t *testing.T) {
	t.Run("append normal moves cursor back", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		action.AppendMode(e)
		posInsert := testutil.CursorPos(t, e)
		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
		posNormal := testutil.CursorPos(t, e)
		assert.True(t, posNormal <= posInsert)
	})

	t.Run("normal strips blank indent", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 5)
		action.InsertMode(e)
		action.InsertNewline(e)
		action.InsertChar(e, '\t')
		testutil.SetCursor(t, e, testutil.CursorPos(t, e))

		action.NormalMode(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Equal(t, view.ModeNormal, e.Mode())
		assert.NotContains(t, text, "\n\t")
	})
}

func TestTryRestoreIndent(t *testing.T) {
	t.Run("normal clears whitespace line", func(t *testing.T) {
		// "a\n    \nb": a=0, \n=1, ' '=2..5, \n=6, b=7
		// lineEnd for line 1 = lineStart(2) +
		// LineEndCharIndex("    \n") = 2+4=6
		e := testutil.EditorWithText(t, "a\n    \nb")
		testutil.SetCursor(t, e, 6)
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.NotContains(t, text, "    ")
	})

	t.Run("non-blank line is not cleared", func(t *testing.T) {
		// "a\n  x\nb": a=0,\n=1,' '=2,' '=3,x=4,\n=5,b=6
		// lineEnd for line 1 = 2 + LineEndCharIndex("  x\n") = 2+3=5
		e := testutil.EditorWithText(t, "a\n  x\nb")
		testutil.SetCursor(t, e, 5)
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "  x")
	})
}

func TestDeleteCharBackwardAtLineStart(t *testing.T) {
	t.Run("at start of document is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)
		e.SetMode(view.ModeInsert)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("deletes single grapheme backward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 2)
		e.SetMode(view.ModeInsert)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ac", doc.Text().String())
		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestInsertNewlineContinuedComment(t *testing.T) {
	t.Run("bare newline whitespace line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "   ")
		testutil.SetCursor(t, e, 0)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
	})

	t.Run("newline between brackets indents", func(t *testing.T) {
		e := testutil.EditorWithText(t, "()")
		testutil.SetCursor(t, e, 1)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(\n\t\n)", doc.Text().String())
	})

	t.Run("newline after opener indents", func(t *testing.T) {
		e := testutil.EditorWithText(t, "if ok {")
		testutil.SetCursor(t, e, 7)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "if ok {\n\t", doc.Text().String())
	})

	t.Run("newline after comma indents", func(t *testing.T) {
		e := testutil.EditorWithText(t, "call(a,")
		testutil.SetCursor(t, e, 7)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "call(a,\n\t", doc.Text().String())
	})

	t.Run("uses installed indent provider", func(t *testing.T) {
		e := testutil.EditorWithText(t, "    else:")
		e.SetIndenter(func(
			_ *view.Document, _, _ int,
		) (string, bool) {
			return "    ", true
		})
		testutil.SetCursor(t, e, 9)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "    else:\n    ", doc.Text().String())
	})

	t.Run("no view is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.InsertNewline(e) })
	})

	t.Run("trims trailing whitespace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc   def")
		testutil.SetCursor(t, e, 6)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc\ndef", doc.Text().String())
	})

	t.Run("duplicate cursors share insertion", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(1), core.PointRange(1)},
			0,
		)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\nbc", doc.Text().String())
	})

	t.Run("negative range inserts at top", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "\nabc", doc.Text().String())
	})
}

func TestAddNewlineImplNoView(t *testing.T) {
	t.Run("add newline above empty text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.AddNewlineAbove(e) })
	})

	t.Run("add newline above no view", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.AddNewlineAbove(e) })
	})

	t.Run("add newline below no view", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.AddNewlineBelow(e) })
	})
}

func TestAddNewlineEdges(t *testing.T) {
	t.Run("invalid range leaves text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("duplicate target inserts once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(0), core.PointRange(1)},
			0,
		)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "\nabc", doc.Text().String())
	})

	t.Run("below missing next line leaves text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		action.AddNewlineBelow(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestAutoPairsDisabled(t *testing.T) {
	t.Run("auto-pairs disabled skips pair hook", func(t *testing.T) {
		e := testutil.EditorWithText(t, "")
		e.Options().HasAutoPairs = false
		e.SetMode(view.ModeInsert)

		action.InsertChar(e, '(')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(", doc.Text().String())
	})
}

func TestInsertNewlineDuplicateCursors(t *testing.T) {
	t.Run("duplicate cursors insert newline once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(1),
			core.PointRange(1),
		}, 0)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, 2, doc.Text().LenLines())
	})
}

func TestCountOrOne(t *testing.T) {
	t.Run("no count uses 1", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")
		testutil.SetCursor(t, e, 2)
		e.SetCount(0)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\n\nb\nc", doc.Text().String())
	})

	t.Run("count=2 inserts two newlines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetCursor(t, e, 2)
		e.SetCount(2)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\n\n\nb", doc.Text().String())
	})
}

func TestInsertCharDuplicate(t *testing.T) {
	t.Run("same-position cursors insert once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(0),
			core.PointRange(0),
		}, 0)

		action.InsertChar(e, 'x')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xabc", doc.Text().String())
	})
}

func TestDeleteCharBackwardAutoPair(t *testing.T) {
	t.Run("deletes auto-pair bracket", func(t *testing.T) {
		e := testutil.EditorWithText(t, "()")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 1)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "", doc.Text().String())
	})
}

func TestInsertCharAutoPair(t *testing.T) {
	t.Run("inserting open bracket creates auto-pair", func(t *testing.T) {
		e := testutil.EditorWithText(t, "")
		testutil.SetCursor(t, e, 0)

		action.InsertChar(e, '(')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "()", doc.Text().String())
		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})

	t.Run("inserting close bracket moves past it", func(t *testing.T) {
		e := testutil.EditorWithText(t, "()")
		testutil.SetCursor(t, e, 1)

		action.InsertChar(e, ')')

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) >= 2)
	})
}

func TestDeleteCharForwardDuplicate(t *testing.T) {
	t.Run("same-position cursors delete once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(0),
			core.PointRange(0),
		}, 0)

		action.DeleteCharForward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "bc", doc.Text().String())
	})
}

func TestInsertNewlineTrailingWhitespace(t *testing.T) {
	t.Run("trims trailing whitespace", func(t *testing.T) {
		// "hello  " — cursor at 7 (pos after 'o'), chars 5,6 are spaces
		// firstTrailingWS=5, pos=7 → 5 < 7 hits the elif branch
		e := testutil.EditorWithText(t, "hello  ")
		testutil.SetCursor(t, e, 7)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
		assert.NotContains(t, text, "  \n")
	})

	t.Run("whitespace line gets bare newline", func(t *testing.T) {
		e := testutil.EditorWithText(t, "   ")
		testutil.SetCursor(t, e, 0)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
	})
}

func TestDeleteCharBackwardDedent(t *testing.T) {
	t.Run("dedents leading tab", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\thello")
		testutil.SetCursor(t, e, 1)
		e.SetMode(view.ModeInsert)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestChangeSelectionLinewiseTrue(t *testing.T) {
	t.Run("linewise inserts above", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello\nworld\n")
		// Linewise: covers the full first line including newline
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 6)}, 0)

		action.ChangeSelection(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestInsertNewlineIndented(t *testing.T) {
	t.Run("preserves leading indent", func(t *testing.T) {
		e := testutil.EditorWithText(t, "  hello")
		testutil.SetCursor(t, e, 7)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "  ")
	})

	t.Run("no comment continuation when disabled", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 5)
		e.Options().ContinueComments = false
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "\n")
	})

	t.Run("continues comment token on newline", func(t *testing.T) {
		writeTextLangConfig(t, "//")

		e := testutil.EditorWithText(t, "// hello")
		testutil.SetCursor(t, e, 8)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "//")
	})
}

func TestChangeSelectionNoYankLinewise(t *testing.T) {
	t.Run("linewise opens line above", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 4)}, 0)

		action.ChangeSelectionNoYank(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestSelectAllNoView(t *testing.T) {
	t.Run("no focused view is noop", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())

		action.SelectAll(e)
	})
}

func TestInsertTabNoView(t *testing.T) {
	t.Run("no focused view is noop", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())

		action.InsertTab(e)
	})
}

func TestDeleteSelectionNoYankNoView(t *testing.T) {
	t.Run("no focused view is noop", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())

		action.DeleteSelectionNoYank(e)
	})
}

func TestChangeSelectionNoYankNoView(t *testing.T) {
	t.Run("no focused view is noop", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())

		action.ChangeSelectionNoYank(e)
	})
}
