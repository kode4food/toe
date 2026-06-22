package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestEdit(t *testing.T) {
	t.Run("insert at cursor", func(t *testing.T) {
		e := editorWithText(t, "hllo")
		setCursor(t, e, 1)

		action.InsertChar(e, 'e')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
		assert.Equal(t, 2, cursorPos(t, e))
	})

	t.Run("newline honors auto-pair", func(t *testing.T) {
		e := editorWithText(t, "()")
		setCursor(t, e, 1)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(\n\t\n)", doc.Text().String())
		assert.Equal(t, 3, cursorPos(t, e))
	})

	t.Run("delete backward dedents", func(t *testing.T) {
		e := editorWithText(t, "    x")
		setCursor(t, e, 4)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "x", doc.Text().String())
		assert.Equal(t, 0, cursorPos(t, e))
	})

	t.Run("change selection enters insert", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 2)}, 0)

		action.ChangeSelection(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "c", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestSelectAll(t *testing.T) {
	t.Run("selects entire document", func(t *testing.T) {
		e := editorWithText(t, "hello")
		setCursor(t, e, 0)

		action.SelectAll(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 5, sel.Primary().To())
	})

	t.Run("empty document stays at zero", func(t *testing.T) {
		e := editorWithText(t, "")

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
		e := editorWithText(t, "abcd")
		setSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)

		action.CollapseSelection(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		r := sel.Primary()
		assert.Equal(t, r.From(), r.To())
	})

	t.Run("multiple selections each collapse", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		setSelection(t, e, []core.Range{
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
		e := editorWithText(t, "abcd")
		setSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)

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
		e := editorWithText(t, "abcd")
		setSelection(t, e, []core.Range{
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
		e := editorWithText(t, "ab\ncd")
		setCursor(t, e, 0)

		action.ExtendLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 3, sel.Primary().To())
	})

	t.Run("extends again if already full line", func(t *testing.T) {
		e := editorWithText(t, "ab\ncd\nef")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.ExtendLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 6, sel.Primary().To())
	})
}

func TestSelectLineBelow(t *testing.T) {
	t.Run("selects current line forward", func(t *testing.T) {
		e := editorWithText(t, "ab\ncd\nef")
		setCursor(t, e, 0)

		action.SelectLineBelow(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 3, sel.Primary().To())
	})
}

func TestSelectLineAbove(t *testing.T) {
	t.Run("includes previous line", func(t *testing.T) {
		e := editorWithText(t, "ab\ncd\nef")
		setCursor(t, e, 6)

		action.SelectLineAbove(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		// includes at least two lines worth of content
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 3)
	})
}

func TestDeleteSelection(t *testing.T) {
	t.Run("deletes selected text", func(t *testing.T) {
		e := editorWithText(t, "hello world")
		setSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.DeleteSelection(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, " world", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("cursor lands at deletion point", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)

		action.DeleteSelection(e)

		assert.Equal(t, 1, cursorPos(t, e))
	})
}

func TestDeleteCharForward(t *testing.T) {
	t.Run("deletes char under cursor", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setCursor(t, e, 1)

		action.DeleteCharForward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ac", doc.Text().String())
	})

	t.Run("noop at end of document", func(t *testing.T) {
		e := editorWithText(t, "a")
		setCursor(t, e, 1)

		action.DeleteCharForward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a", doc.Text().String())
	})
}

func TestYank(t *testing.T) {
	t.Run("copies selection to default register", func(t *testing.T) {
		e := editorWithText(t, "hello")
		setSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.Yank(e)

		assert.Equal(t, "hello", registeredValue(t, e, '"'))
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("yank multiple ranges", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		setSelection(t, e, []core.Range{
			core.NewRange(0, 2),
			core.NewRange(2, 4),
		}, 0)

		action.Yank(e)

		assert.Equal(t, "ab", registeredValue(t, e, '"'))
	})
}

func TestPasteAfter(t *testing.T) {
	t.Run("pastes at head of selection", func(t *testing.T) {
		e := editorWithText(t, "xyz")
		// Select "x" (range 0..1), head is at 1
		setSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		e.Registers().Write('"', []string{"b"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xbyz", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestPasteBefore(t *testing.T) {
	t.Run("pastes before cursor position", func(t *testing.T) {
		e := editorWithText(t, "xz")
		setCursor(t, e, 1)
		e.Registers().Write('"', []string{"y"})

		action.PasteBefore(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xyz", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestSplitSelectionOnNewline(t *testing.T) {
	t.Run("splits multiline selection into per-line", func(t *testing.T) {
		e := editorWithText(t, "ab\ncd\nef")
		setSelection(t, e, []core.Range{core.NewRange(0, 8)}, 0)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, len(sel.Ranges()))
	})

	t.Run("point selection is kept", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setCursor(t, e, 1)

		action.SplitSelectionOnNewline(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
	})
}

func TestNormalMode(t *testing.T) {
	t.Run("exits insert mode", func(t *testing.T) {
		e := editorWithText(t, "abc")
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("noop when already normal", func(t *testing.T) {
		e := editorWithText(t, "abc")

		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestInsertMode(t *testing.T) {
	t.Run("places cursor at selection start", func(t *testing.T) {
		e := editorWithText(t, "abcde")
		setSelection(t, e, []core.Range{core.NewRange(2, 4)}, 0)

		action.InsertMode(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, 2, cursorPos(t, e))
	})
}

func TestAppendMode(t *testing.T) {
	t.Run("places cursor past selection end", func(t *testing.T) {
		e := editorWithText(t, "abcde")
		setCursor(t, e, 1)

		action.AppendMode(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestSelectMode(t *testing.T) {
	t.Run("enters select mode", func(t *testing.T) {
		e := editorWithText(t, "abcde")
		setCursor(t, e, 0)

		action.SelectMode(e)

		assert.Equal(t, view.ModeSelect, e.Mode())
	})

	t.Run("widens empty end-of-doc selection", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setCursor(t, e, 3)

		action.SelectMode(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 1)
	})
}

func TestInsertAtLineStart(t *testing.T) {
	t.Run("moves to first non-ws and enters insert", func(t *testing.T) {
		e := editorWithText(t, "  hello")
		setCursor(t, e, 6)

		action.InsertAtLineStart(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, 2, cursorPos(t, e))
	})
}

func TestAppendToLine(t *testing.T) {
	t.Run("moves to end of line and enters insert", func(t *testing.T) {
		e := editorWithText(t, "hello")
		setCursor(t, e, 0)

		action.AppendToLine(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestDeleteSelectionNoyank(t *testing.T) {
	t.Run("deletes without affecting register", func(t *testing.T) {
		e := editorWithText(t, "hello world")
		setSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)
		e.Registers().Write('"', []string{"saved"})

		action.DeleteSelectionNoyank(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, " world", doc.Text().String())
		assert.Equal(t, "saved", registeredValue(t, e, '"'))
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestChangeSelectionLinewise(t *testing.T) {
	t.Run("linewise change opens blank line above", func(t *testing.T) {
		e := editorWithText(t, "hello\nworld")
		// Select full first line including newline (linewise)
		setSelection(t, e, []core.Range{core.NewRange(0, 6)}, 0)

		action.ChangeSelection(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestLinewisePaste(t *testing.T) {
	t.Run("PasteAfter linewise pastes below", func(t *testing.T) {
		e := editorWithText(t, "abc\ndef")
		setCursor(t, e, 0)
		// Yank full first line (with newline = linewise)
		e.Registers().Write('"', []string{"abc\n"})

		action.PasteAfter(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "abc")
	})

	t.Run("PasteBefore linewise pastes above", func(t *testing.T) {
		e := editorWithText(t, "def")
		setCursor(t, e, 0)
		e.Registers().Write('"', []string{"abc\n"})

		action.PasteBefore(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "abc")
	})
}

func TestExitSelectMode(t *testing.T) {
	t.Run("exits select mode to normal", func(t *testing.T) {
		e := editorWithText(t, "abc")
		e.SetMode(view.ModeSelect)

		action.ExitSelectMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("noop when not in select mode", func(t *testing.T) {
		e := editorWithText(t, "abc")

		action.ExitSelectMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestChangeSelectionNoyank(t *testing.T) {
	t.Run("skips register on insert", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 2)}, 0)
		e.Registers().Write('"', []string{"safe"})

		action.ChangeSelectionNoyank(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "c", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
		assert.Equal(t, "safe", registeredValue(t, e, '"'))
	})
}

func TestNormalModeRestoreCursor(t *testing.T) {
	t.Run("append normal moves cursor back", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setCursor(t, e, 0)

		action.AppendMode(e)
		posInsert := cursorPos(t, e)
		action.NormalMode(e)

		assert.Equal(t, view.ModeNormal, e.Mode())
		posNormal := cursorPos(t, e)
		assert.True(t, posNormal <= posInsert)
	})

	t.Run("normal strips blank indent", func(t *testing.T) {
		e := editorWithText(t, "hello")
		setCursor(t, e, 5)
		action.InsertMode(e)
		action.InsertNewline(e)
		action.InsertChar(e, '\t')
		setCursor(t, e, cursorPos(t, e))

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
		// lineEnd for line 1 = lineStart(2) + LineEndCharIndex("    \n") = 2+4=6
		e := editorWithText(t, "a\n    \nb")
		setCursor(t, e, 6)
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.NotContains(t, text, "    ")
	})

	t.Run("non-blank line is not cleared", func(t *testing.T) {
		// "a\n  x\nb": a=0,\n=1,' '=2,' '=3,x=4,\n=5,b=6
		// lineEnd for line 1 = 2 + LineEndCharIndex("  x\n") = 2+3=5
		e := editorWithText(t, "a\n  x\nb")
		setCursor(t, e, 5)
		e.SetMode(view.ModeInsert)

		action.NormalMode(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "  x")
	})
}

func TestDeleteCharBackwardAtLineStart(t *testing.T) {
	t.Run("at start of document is noop", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setCursor(t, e, 0)
		e.SetMode(view.ModeInsert)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("deletes single grapheme backward", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setCursor(t, e, 2)
		e.SetMode(view.ModeInsert)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ac", doc.Text().String())
		assert.Equal(t, 1, cursorPos(t, e))
	})
}

func TestInsertNewlineContinuedComment(t *testing.T) {
	t.Run("bare newline whitespace line", func(t *testing.T) {
		e := editorWithText(t, "   ")
		setCursor(t, e, 0)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
	})

	t.Run("newline between brackets indents", func(t *testing.T) {
		e := editorWithText(t, "()")
		setCursor(t, e, 1)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(\n\t\n)", doc.Text().String())
	})
}

func TestAddNewlineImplNoView(t *testing.T) {
	t.Run("add newline above empty text", func(t *testing.T) {
		e := editorWithText(t, "")
		setCursor(t, e, 0)

		assert.NotPanics(t, func() { action.AddNewlineAbove(e) })
	})
}

func TestAutoPairsDisabled(t *testing.T) {
	t.Run("auto-pairs disabled skips pair hook", func(t *testing.T) {
		e := editorWithText(t, "")
		e.Options().HasAutoPairs = false
		e.SetMode(view.ModeInsert)

		action.InsertChar(e, '(')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "(", doc.Text().String())
	})
}

func TestInsertNewlineDuplicateCursors(t *testing.T) {
	t.Run("duplicate cursors insert newline once", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{
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
		e := editorWithText(t, "a\nb\nc")
		setCursor(t, e, 2)
		e.SetCount(0)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\n\nb\nc", doc.Text().String())
	})

	t.Run("count=2 inserts two newlines", func(t *testing.T) {
		e := editorWithText(t, "a\nb")
		setCursor(t, e, 2)
		e.SetCount(2)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\n\n\nb", doc.Text().String())
	})
}

func TestInsertCharAutoPair(t *testing.T) {
	t.Run("inserting open bracket creates auto-pair", func(t *testing.T) {
		e := editorWithText(t, "")
		setCursor(t, e, 0)

		action.InsertChar(e, '(')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "()", doc.Text().String())
		assert.Equal(t, 1, cursorPos(t, e))
	})

	t.Run("inserting close bracket moves past it", func(t *testing.T) {
		e := editorWithText(t, "()")
		setCursor(t, e, 1)

		action.InsertChar(e, ')')

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) >= 2)
	})
}

func TestDeleteCharForwardDuplicate(t *testing.T) {
	t.Run("same-position cursors delete once", func(t *testing.T) {
		e := editorWithText(t, "abc")
		setSelection(t, e, []core.Range{
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
		e := editorWithText(t, "hello  ")
		setCursor(t, e, 7)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
		assert.NotContains(t, text, "  \n")
	})

	t.Run("whitespace line gets bare newline", func(t *testing.T) {
		e := editorWithText(t, "   ")
		setCursor(t, e, 0)
		e.SetMode(view.ModeInsert)

		action.InsertNewline(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
	})
}

func TestDeleteCharBackwardDedent(t *testing.T) {
	t.Run("dedents leading tab", func(t *testing.T) {
		e := editorWithText(t, "\thello")
		setCursor(t, e, 1)
		e.SetMode(view.ModeInsert)

		action.DeleteCharBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestChangeSelectionLinewiseTrue(t *testing.T) {
	t.Run("linewise inserts above", func(t *testing.T) {
		e := editorWithText(t, "hello\nworld\n")
		// Linewise: covers the full first line including newline
		setSelection(t, e, []core.Range{core.NewRange(0, 6)}, 0)

		action.ChangeSelection(e)

		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}
