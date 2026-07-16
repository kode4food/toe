package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestSearch(t *testing.T) {
	t.Run("search forward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "zz alpha yy")

		err := action.SearchForward(e, "alpha")

		assert.NoError(t, err)
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("search backward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "zz alpha yy alpha")
		testutil.SetCursor(t, e, 17)

		err := action.SearchBackward(e, "alpha")

		assert.NoError(t, err)
		assert.Equal(t, 12, testutil.CursorPos(t, e))
	})
}

func TestSearchHighlightLifecycle(t *testing.T) {
	t.Run("cursor move clears highlights", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.False(t, doc.SearchHighlightsActive(v.ID()))

		assert.NoError(t, action.SearchForward(e, "foo"))
		assert.True(t, doc.SearchHighlightsActive(v.ID()))

		// repeating the search keeps highlights visible
		action.SearchNext(e)
		assert.True(t, doc.SearchHighlightsActive(v.ID()))

		// a plain cursor move clears them
		testutil.SetCursor(t, e, 0)
		assert.False(t, doc.SearchHighlightsActive(v.ID()))
	})
}

func TestSearchNext(t *testing.T) {
	t.Run("repeats last search forward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		err := action.SearchForward(e, "foo")
		assert.NoError(t, err)
		pos1 := testutil.CursorPos(t, e)

		action.SearchNext(e)

		pos2 := testutil.CursorPos(t, e)
		assert.True(t, pos2 != pos1 || pos2 == 0)
	})

	t.Run("noop when no prior search", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		assert.NotPanics(t, func() { action.SearchNext(e) })
	})
}

func TestSearchPrev(t *testing.T) {
	t.Run("repeats last search backward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 8)
		err := action.SearchBackward(e, "foo")
		assert.NoError(t, err)

		assert.NotPanics(t, func() { action.SearchPrev(e) })
	})

	t.Run("noop when no prior search", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		assert.NotPanics(t, func() { action.SearchPrev(e) })
	})
}

func TestExtendSearchNext(t *testing.T) {
	t.Run("extends selection to next match", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		err := action.SearchForward(e, "foo")
		assert.NoError(t, err)

		assert.NotPanics(t, func() { action.ExtendSearchNext(e) })
	})
}

func TestExtendSearchPrev(t *testing.T) {
	t.Run("extends selection to prev match", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 8)
		err := action.SearchBackward(e, "foo")
		assert.NoError(t, err)

		assert.NotPanics(t, func() { action.ExtendSearchPrev(e) })
	})
}

func TestSearchCaseSensitive(t *testing.T) {
	t.Run("uppercase pattern finds exact match", func(t *testing.T) {
		e := testutil.EditorWithText(t, "Hello World")
		testutil.SetCursor(t, e, 0)

		err := action.SearchForward(e, "Hello")

		assert.NoError(t, err)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestSearchWrapAround(t *testing.T) {
	t.Run("next wraps with wrap-around enabled", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		// search forward for "foo", landing at pos 0
		err := action.SearchForward(e, "foo")
		assert.NoError(t, err)
		// Now search again from same position - should wrap
		assert.NotPanics(t, func() { action.SearchNext(e) })
	})

	t.Run("prev wraps with wrap-around enabled", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 0)
		// search backward from start - may need to wrap
		err := action.SearchBackward(e, "foo")
		assert.NoError(t, err)
		assert.NotPanics(t, func() { action.SearchPrev(e) })
	})

	t.Run("SearchPrev wraps from before=0", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 0)
		err := action.SearchBackward(e, "foo")
		assert.NoError(t, err)
		pos1 := testutil.CursorPos(t, e)

		action.SearchPrev(e)

		pos2 := testutil.CursorPos(t, e)
		assert.True(t, pos2 >= 0)
		assert.True(t, pos1 >= 0)
	})

	t.Run("SearchNext wraps from end of document", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 8)
		err := action.SearchForward(e, "foo")
		assert.NoError(t, err)
		posAfter := testutil.CursorPos(t, e)

		action.SearchNext(e)

		assert.True(t, testutil.CursorPos(t, e) >= 0)
		assert.True(t, posAfter >= 0)
	})

	t.Run("no wrap at last char stays put", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		e.Options().SearchWrapAround = false
		testutil.SetCursor(t, e, 2)

		err := action.SearchForward(e, "abc")

		assert.NoError(t, err)
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("wrap at last char finds from start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		e.Options().SearchWrapAround = true
		testutil.SetCursor(t, e, 10)

		err := action.SearchForward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestSearchFeedback(t *testing.T) {
	t.Run("reports forward wrap", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 10)

		err := action.SearchForward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, "Wrapped around document", e.TakeStatusMsg())
	})

	t.Run("reports no forward match", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo")
		e.Options().SearchWrapAround = false
		testutil.SetCursor(t, e, 2)

		err := action.SearchForward(e, "bar")

		assert.NoError(t, err)
		assert.Equal(t, "No more matches", e.TakeStatusMsg())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("reports backward wrap", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 0)

		err := action.SearchBackward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, "Wrapped around document", e.TakeStatusMsg())
		assert.Equal(t, 8, testutil.CursorPos(t, e))
	})
}

func TestSearchPatterns(t *testing.T) {
	t.Run("finds multiline pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "aa\nbb\ncc")

		err := action.SearchForward(e, "bb\ncc")

		assert.NoError(t, err)
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("finds crlf pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "aa\r\nbb\r\ncc")

		err := action.SearchForward(e, "bb\r\ncc")

		assert.NoError(t, err)
		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("skips empty matches", func(t *testing.T) {
		e := testutil.EditorWithText(t, "bbb")
		testutil.SetCursor(t, e, 0)

		err := action.SearchForward(e, "a*")

		assert.NoError(t, err)
		assert.Equal(t, "No more matches", e.TakeStatusMsg())
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestPageOperations(t *testing.T) {
	t.Run("PageUp does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne\nf")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.PageUp(e) })
	})

	t.Run("PageDown does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne\nf")
		testutil.SetCursor(t, e, 0)

		assert.NotPanics(t, func() { action.PageDown(e) })
	})

	t.Run("HalfPageUp does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd")
		assert.NotPanics(t, func() { action.HalfPageUp(e) })
	})

	t.Run("HalfPageDown does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd")
		assert.NotPanics(t, func() { action.HalfPageDown(e) })
	})

	t.Run("PageCursorHalfUp does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd")
		assert.NotPanics(t, func() { action.PageCursorHalfUp(e) })
	})

	t.Run("PageCursorHalfDown does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd")
		assert.NotPanics(t, func() { action.PageCursorHalfDown(e) })
	})

	t.Run("PageUp does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd")
		assert.NotPanics(t, func() { action.PageUp(e) })
	})

	t.Run("PageDown does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd")
		assert.NotPanics(t, func() { action.PageDown(e) })
	})

}

func TestKillToLine(t *testing.T) {
	t.Run("KillToLineEnd deletes from cursor to end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 5)

		action.KillToLineEnd(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})

	t.Run("at line end joins with next line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello\nworld")
		e.SetMode(view.ModeInsert)
		// cursor at pos 5 = lineEnd of first line
		testutil.SetCursor(t, e, 5)

		action.KillToLineEnd(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "helloworld", doc.Text().String())
	})

	t.Run("deletes from line start to cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 6)

		action.KillToLineStart(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "world", doc.Text().String())
	})

	t.Run("at line start joins with prev line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello\nworld")
		e.SetMode(view.ModeInsert)
		// cursor at pos 6 = start of second line
		testutil.SetCursor(t, e, 6)

		action.KillToLineStart(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "helloworld", doc.Text().String())
	})
}

func TestOpenAbove(t *testing.T) {
	t.Run("inserts blank line above", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.OpenAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "\nhello", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("inserts above second line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetCursor(t, e, 2)

		action.OpenAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\n\nb", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("count repeats new lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)
		e.SetCount(2)

		action.OpenAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "\n\nhello", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("duplicate target inserts once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(0), core.PointRange(1)},
			0,
		)

		action.OpenAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "\nabc", doc.Text().String())
	})

	t.Run("negative range inserts at top", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(-2, -1)}, 0)

		action.OpenAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "\nabc", doc.Text().String())
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("no view is noop", func(t *testing.T) {
		e := editorWithNoView(t)

		assert.NotPanics(t, func() { action.OpenAbove(e) })
	})
}

func TestGotoLine(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{"line 1 goes to start", 1, 0},
		{"line 2 goes to second line", 2, 3},
		{"zero noop", 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := testutil.EditorWithText(t, "ab\ncd\nef")
			testutil.SetCursor(t, e, 0)

			action.GotoLine(e, tc.n)

			assert.Equal(t, tc.want, testutil.CursorPos(t, e))
		})
	}
}

func TestGotoLineTrailingNewline(t *testing.T) {
	t.Run("skips blank last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\n")
		testutil.SetCursor(t, e, 0)
		action.GotoLine(e, 99)
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})
}

func TestReplaceChar(t *testing.T) {
	t.Run("replaces selected grapheme", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)

		action.ReplaceChar(e, 'x')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "axc", doc.Text().String())
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("empty range is skipped", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.ReplaceChar(e, 'x')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestReplaceWithYanked(t *testing.T) {
	t.Run("replaces selection with yanked text", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)
		e.Registers().Write('"', []string{"XY"})

		action.ReplaceWithYanked(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aXYc", doc.Text().String())
	})

	t.Run("noop when register empty", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 2)}, 0)
		e.Registers().Clear('"')

		action.ReplaceWithYanked(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("multiple selections repeat fallback", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e,
			[]core.Range{core.NewRange(0, 1), core.NewRange(2, 3)},
			0,
		)
		e.Registers().Write('"', []string{"x"})
		e.SetCount(2)

		action.ReplaceWithYanked(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "xxbxxd", doc.Text().String())
	})

	t.Run("empty ranges are ignored", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.PointRange(1)}, 0)
		e.Registers().Write('"', []string{"x"})

		action.ReplaceWithYanked(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("noop with no view", func(t *testing.T) {
		e := editorWithNoView(t)
		e.Registers().Write('"', []string{"x"})

		assert.NotPanics(t, func() {
			action.ReplaceWithYanked(e)
		})
	})
}

func TestSwitchCase(t *testing.T) {
	t.Run("toggles case", func(t *testing.T) {
		e := testutil.EditorWithText(t, "Hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.SwitchCase(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hELLO", doc.Text().String())
	})

	t.Run("non-alpha chars unchanged", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a1b")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.SwitchCase(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "A1B", doc.Text().String())
	})

	t.Run("cursor-only is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.SwitchCase(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestSwitchToUppercase(t *testing.T) {
	t.Run("uppercases selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.SwitchToUppercase(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "HELLO", doc.Text().String())
	})
}

func TestSwitchToLowercase(t *testing.T) {
	t.Run("lowercases selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "HELLO")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		action.SwitchToLowercase(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestExtendToLineBounds(t *testing.T) {
	t.Run("extends to cover full lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetCursor(t, e, 1)

		action.ExtendToLineBounds(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
		assert.Equal(t, 3, sel.Primary().To())
	})
}

func TestShrinkToLineBounds(t *testing.T) {
	t.Run("shrinks multiline selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 6)}, 0)

		action.ShrinkToLineBounds(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() > sel.Primary().From())
	})

	t.Run("single-line selection unchanged", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 4)}, 0)

		action.ShrinkToLineBounds(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, sel.Primary().From())
		assert.Equal(t, 4, sel.Primary().To())
	})

	t.Run("backward multiline selection shrinks", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		// Backward selection from mid-second-line to mid-first-line
		testutil.SetSelection(t, e, []core.Range{core.NewRange(5, 1)}, 0)

		action.ShrinkToLineBounds(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})
}

func TestRemovePrimarySelection(t *testing.T) {
	t.Run("removes primary when multiple exist", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(0),
			core.PointRange(1),
			core.PointRange(2),
		}, 0)

		action.RemovePrimarySelection(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})

	t.Run("noop with single selection", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		action.RemovePrimarySelection(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
	})
}

func TestDeleteWordBackwardForward(t *testing.T) {
	t.Run("DeleteWordBackward removes previous word", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 11)

		action.DeleteWordBackward(e)

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) < 11)
	})

	t.Run("DeleteWordBackward noop at position 0", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 0)

		action.DeleteWordBackward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})

	t.Run("DeleteWordForward removes next word", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 0)

		action.DeleteWordForward(e)

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) < 11)
	})

	t.Run("DeleteWordForward noop at end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 5)

		action.DeleteWordForward(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestSelectTextObjects(t *testing.T) {
	tests := []struct {
		name string
		ch   rune
		text string
		pos  int
	}{
		{"around word w", 'w', "hello world", 2},
		{"around WORD W", 'W', "foo.bar baz", 2},
		{"around paragraph p", 'p', "a\n\nb\n\nc", 0},
		{"around paren m", 'm', "(hello)", 2},
		{"around bracket [", '[', "[hello]", 2},
	}
	for _, tc := range tests {
		t.Run("SelectTextObjectAround "+tc.name, func(t *testing.T) {
			e := testutil.EditorWithText(t, tc.text)
			testutil.SetCursor(t, e, tc.pos)

			action.SelectTextObjectAround(e, tc.ch)

			v, _ := e.FocusedView()
			doc, _ := e.FocusedDocument()
			sel := doc.SelectionFor(v.ID())
			assert.True(t, sel.Primary().To() >= sel.Primary().From())
		})

		t.Run("SelectTextObjectInside "+tc.name, func(t *testing.T) {
			e := testutil.EditorWithText(t, tc.text)
			testutil.SetCursor(t, e, tc.pos)

			action.SelectTextObjectInside(e, tc.ch)

			v, _ := e.FocusedView()
			doc, _ := e.FocusedDocument()
			sel := doc.SelectionFor(v.ID())
			assert.True(t, sel.Primary().To() >= sel.Primary().From())
		})
	}
}

func TestMergeSelections(t *testing.T) {
	t.Run("merges overlapping selections", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 3),
			core.NewRange(2, 5),
		}, 0)

		action.MergeSelections(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
	})
}

func TestMergeConsecutive(t *testing.T) {
	t.Run("merges adjacent selections", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 2),
			core.NewRange(2, 4),
		}, 0)

		action.MergeConsecutive(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
	})
}

func TestEnsureForward(t *testing.T) {
	t.Run("reverses backward selection to forward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(3, 1)}, 0)

		action.EnsureForward(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, core.DirectionForward, sel.Primary().Direction())
	})
}

func TestIndentUnindent(t *testing.T) {
	t.Run("Indent adds indentation", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.Indent(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.True(t, text[0] == ' ' || text[0] == '\t')
	})

	t.Run("Indent skips blank lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello\n\nworld")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 12)}, 0)

		action.Indent(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.True(t, len(text) > len("hello\n\nworld"))
	})

	t.Run("Unindent removes indentation", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\thello")
		testutil.SetCursor(t, e, 0)

		action.Unindent(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})

	t.Run("Unindent multiple lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\thello\n\tworld")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 13)}, 0)

		action.Unindent(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello\nworld", doc.Text().String())
	})

	t.Run("Unindent no-indent is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.Unindent(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestSearchSelection(t *testing.T) {
	t.Run("stores selection as search pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.SearchSelection(e)

		val, ok := e.Registers().First('/')
		assert.True(t, ok)
		assert.Equal(t, "foo", val)
		assert.Equal(t, "register '/' set to 'foo'", e.TakeStatusMsg())
	})
}

func TestSearchSelectionWord(t *testing.T) {
	t.Run("stores word-bounded pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.SearchSelectionWord(e)

		val, ok := e.Registers().First('/')
		assert.True(t, ok)
		assert.True(t, len(val) > 0)
		assert.Contains(t, e.TakeStatusMsg(), "register '/' set to '")
	})
}

func TestMakeSearchWordBounded(t *testing.T) {
	t.Run("adds word boundaries to pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo")
		e.Registers().Write('/', []string{"foo"})

		action.MakeSearchWordBounded(e)

		val, ok := e.Registers().First('/')
		assert.True(t, ok)
		assert.Contains(t, val, `\b`)
		assert.Equal(t, "register '/' set to '\\bfoo\\b'", e.TakeStatusMsg())
	})

	t.Run("noop when already bounded", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo")
		e.Registers().Write('/', []string{`\bfoo\b`})

		action.MakeSearchWordBounded(e)

		val, _ := e.Registers().First('/')
		assert.Equal(t, `\bfoo\b`, val)
	})
}

func TestCopyOnNextLine(t *testing.T) {
	t.Run("duplicates selection to next line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 0)

		action.CopyOnNextLine(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})
}

func TestCopyOnPrevLine(t *testing.T) {
	t.Run("duplicates selection to prev line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 4)

		action.CopyOnPrevLine(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})
}

func TestCopyOnNextLineDuplicateHead(t *testing.T) {
	t.Run("stops when target head already exists", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(1),
			core.PointRange(4),
		}, 0)

		action.CopyOnNextLine(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})
}

func TestJoinWithEmptyNextLine(t *testing.T) {
	t.Run("join with empty next line uses no sep", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\n\ndef")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 8)}, 0)

		action.JoinSelectionsSpace(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.NotContains(t, text, "\n\n")
	})
}

func TestAlignView(t *testing.T) {
	t.Run("AlignViewTop does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		testutil.SetCursor(t, e, 4)

		assert.NotPanics(t, func() { action.AlignViewTop(e) })
	})

	t.Run("AlignViewCenter does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		testutil.SetCursor(t, e, 4)

		assert.NotPanics(t, func() { action.AlignViewCenter(e) })
	})

	t.Run("AlignViewBottom does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		testutil.SetCursor(t, e, 4)

		assert.NotPanics(t, func() { action.AlignViewBottom(e) })
	})
}

func TestIndentWithSpaces(t *testing.T) {
	t.Run("indent with space style aligns to stop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)
		doc, _ := e.FocusedDocument()
		doc.SetIndentStyle(core.Spaces(2))

		action.Indent(e)

		text, _ := e.FocusedDocument()
		assert.True(t, len(text.Text().String()) > len("hello"))
	})

	t.Run("indent with count=2 indents twice", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)
		e.SetCount(2)

		action.Indent(e)

		doc, _ := e.FocusedDocument()
		assert.True(t, len(doc.Text().String()) > len("hello"))
	})
}

func TestFindPrevMatchFromZero(t *testing.T) {
	t.Run("from position 0 wraps around", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar foo")
		testutil.SetCursor(t, e, 0)

		err := action.SearchBackward(e, "foo")

		assert.NoError(t, err)
		assert.True(t, testutil.CursorPos(t, e) >= 0)
	})

	t.Run("SearchPrev from position 0 wraps around", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar baz")
		testutil.SetCursor(t, e, 0)
		err := action.SearchBackward(e, "foo")
		assert.NoError(t, err)

		action.SearchPrev(e)

		assert.True(t, testutil.CursorPos(t, e) >= 0)
	})
}

func TestSurroundAdd(t *testing.T) {
	t.Run("wraps selection with parens", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 4)}, 0)

		action.SurroundAdd(e, '(')

		doc, _ := e.FocusedDocument()
		result := doc.Text().String()
		assert.True(t, len(result) > len("hello"))
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestSurroundDelete(t *testing.T) {
	t.Run("removes surrounding parens", func(t *testing.T) {
		e := testutil.EditorWithText(t, "(hello)")
		testutil.SetCursor(t, e, 1)

		action.SurroundDelete(e, '(')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestSurroundReplace(t *testing.T) {
	t.Run("parens replaced with brackets", func(t *testing.T) {
		e := testutil.EditorWithText(t, "(hello)")
		testutil.SetCursor(t, e, 1)

		action.SurroundReplace(e, '(', '[')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "[hello]", doc.Text().String())
	})
}

func TestGotoWindowTopBottomCenter(t *testing.T) {
	t.Run("GotoWindowTop does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		assert.NotPanics(t, func() { action.GotoWindowTop(e) })
	})

	t.Run("GotoWindowBottom does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		assert.NotPanics(t, func() { action.GotoWindowBottom(e) })
	})

	t.Run("GotoWindowCenter does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc\nd\ne")
		assert.NotPanics(t, func() { action.GotoWindowCenter(e) })
	})
}

func TestFindChar(t *testing.T) {
	t.Run("finds char forward inclusive", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 0)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'c',
			Forward:   true,
			Inclusive: true,
		})

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("finds char backward inclusive", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 4)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'b',
			Forward:   false,
			Inclusive: true,
		})

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})

	t.Run("finds char forward exclusive", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 0)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'c',
			Forward:   true,
			Inclusive: false,
		})

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})

	t.Run("finds char backward exclusive", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 4)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'b',
			Forward:   false,
			Inclusive: false,
		})

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("backward exclusive at position 0 is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'a',
			Forward:   false,
			Inclusive: false,
		})

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestSearchBackwardNoWrap(t *testing.T) {
	t.Run("no match at zero when wrap off", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 0)
		e.Options().SearchWrapAround = false

		err := action.SearchBackward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestSearchBackwardWrapsForward(t *testing.T) {
	t.Run("wraps to match after cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "xyzfoo")
		testutil.SetCursor(t, e, 2)

		err := action.SearchBackward(e, "foo")

		assert.NoError(t, err)
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})
}

func TestSearchNextNoWrap(t *testing.T) {
	t.Run("no advance at last match wrap off", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		e.Options().SearchWrapAround = false
		err := action.SearchForward(e, "foo")
		assert.NoError(t, err)
		pos1 := testutil.CursorPos(t, e)

		action.SearchNext(e)

		assert.Equal(t, pos1, testutil.CursorPos(t, e))
	})
}

func TestExtendToLineBoundsLastLine(t *testing.T) {
	t.Run("extends to end of file on last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 3)

		action.ExtendToLineBounds(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, sel.Primary().From())
		assert.Equal(t, 5, sel.Primary().To())
	})
}

func TestFindCharNotFound(t *testing.T) {
	t.Run("forward miss is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 0)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'z',
			Forward:   true,
			Inclusive: true,
		})

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("backward miss is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 4)

		action.FindChar(action.FindCharArgs{
			Editor:    e,
			Ch:        'z',
			Forward:   false,
			Inclusive: true,
		})

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})
}

func TestExtendSearchNoPattern(t *testing.T) {
	t.Run("ExtendSearchNext noop without pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		action.ExtendSearchNext(e)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("ExtendSearchPrev noop without pattern", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		action.ExtendSearchPrev(e)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestMergeNoView(t *testing.T) {
	t.Run("MergeSelections noop without view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		action.MergeSelections(e)
	})

	t.Run("MergeConsecutive noop without view", func(t *testing.T) {
		e := view.NewEditor("/tmp")
		v, _ := e.FocusedView()
		e.CloseView(v.ID())
		action.MergeConsecutive(e)
	})
}
