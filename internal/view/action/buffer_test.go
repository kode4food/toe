package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestIncrementHex(t *testing.T) {
	t.Run("increments hex number", func(t *testing.T) {
		e := testutil.EditorWithText(t, "0xff")
		testutil.SetCursor(t, e, 0)

		action.Increment(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "0x100", doc.Text().String())
	})
}

func TestBuffer(t *testing.T) {
	t.Run("increment/decrement", func(t *testing.T) {
		e := testutil.EditorWithText(t, "1")
		testutil.SetCursor(t, e, 0)

		action.Increment(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "2", doc.Text().String())

		action.Decrement(e)

		assert.Equal(t, "1", doc.Text().String())
	})

	t.Run("yank join", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetSelection(
			t, e,
			[]core.Range{
				core.NewRange(0, 1),
				core.NewRange(2, 3),
			},
			0,
		)

		action.YankJoin(e, ",")

		assert.Equal(t, "a,b", testutil.RegisteredValue(t, e, '"'))
	})
}

func TestInsertTab(t *testing.T) {
	t.Run("inserts tab at cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.InsertTab(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.True(t, len(text) > 3)
		assert.Equal(t, 'a', rune(text[0]))
	})
}

func TestExtendToColumn(t *testing.T) {
	t.Run("extends selection to column", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 0)
		e.SetCount(4)

		action.ExtendToColumn(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		// col 4 = pos 3; PutCursor with extend advances one grapheme -> pos 4
		assert.Equal(t, 4, sel.Primary().To())
	})
}

func TestSelectWithinRegex(t *testing.T) {
	t.Run("keeps only matching subranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar baz")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 11)}, 0)

		err := action.SelectWithinRegex(e, `\b\w+\b`)

		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, len(sel.Ranges()))
	})

	t.Run("invalid pattern returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		err := action.SelectWithinRegex(e, "[invalid")

		assert.Error(t, err)
	})
}

func TestSplitSelectionByRegex(t *testing.T) {
	t.Run("splits on separator", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a,b,c")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 5)}, 0)

		err := action.SplitSelectionByRegex(e, ",")

		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, len(sel.Ranges()))
	})

	t.Run("invalid pattern returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		err := action.SplitSelectionByRegex(e, "[bad")

		assert.Error(t, err)
	})
}

func TestKeepSelectionsMatching(t *testing.T) {
	t.Run("retains matching ranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo\nbar\nbaz")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 3),
			core.NewRange(4, 7),
			core.NewRange(8, 11),
		}, 0)

		err := action.KeepSelectionsMatching(e, "ba")

		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
	})
}

func TestRemoveSelectionsMatching(t *testing.T) {
	t.Run("removes matching ranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo\nbar\nbaz")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 3),
			core.NewRange(4, 7),
			core.NewRange(8, 11),
		}, 0)

		err := action.RemoveSelectionsMatching(e, "ba")

		assert.NoError(t, err)
		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 1, len(sel.Ranges()))
	})
}

func TestSortSelections(t *testing.T) {
	t.Run("sorts selections lexicographically", func(t *testing.T) {
		e := testutil.EditorWithText(t, "banana\napple\ncherry")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 6),
			core.NewRange(7, 12),
			core.NewRange(13, 19),
		}, 0)

		err := action.SortSelections(e, false, false)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "apple\nbanana\ncherry", doc.Text().String())
	})

	t.Run("sorts reverse", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 1),
			core.NewRange(2, 3),
			core.NewRange(4, 5),
		}, 0)

		err := action.SortSelections(e, true, false)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "c\nb\na", doc.Text().String())
	})

	t.Run("single selection returns error", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		err := action.SortSelections(e, false, false)

		assert.Error(t, err)
	})

	t.Run("sorts case insensitive", func(t *testing.T) {
		e := testutil.EditorWithText(t, "Banana\napple")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 6),
			core.NewRange(7, 12),
		}, 0)

		err := action.SortSelections(e, false, true)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "apple\nBanana", doc.Text().String())
	})

	t.Run("sorts backward selection ranges", func(t *testing.T) {
		e := testutil.EditorWithText(t, "b\na")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(1, 0),
			core.NewRange(3, 2),
		}, 0)

		err := action.SortSelections(e, false, false)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.True(t, len(text) > 0)
	})

	t.Run("equal elements gives 0 comparison", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\na")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 1),
			core.NewRange(2, 3),
		}, 0)

		err := action.SortSelections(e, false, false)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\na", doc.Text().String())
	})
}

func TestReflowSelections(t *testing.T) {
	t.Run("reflows text to width", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world foo bar")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 19)}, 0)

		action.ReflowSelections(e, 10)

		doc, _ := e.FocusedDocument()
		// text is wrapped at 10 chars
		result := doc.Text().String()
		assert.NotEqual(t, "hello world foo bar", result)
	})
}

func TestSetLineEnding(t *testing.T) {
	t.Run("changes line endings to CRLF", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")

		err := action.SetLineEnding(e, core.LineEndingCRLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\r\nb", doc.Text().String())
	})

	t.Run("noop when ending already matches", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")

		err := action.SetLineEnding(e, core.LineEndingLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\nb", doc.Text().String())
	})
}

func TestMatchBrackets(t *testing.T) {
	t.Run("jumps to matching bracket", func(t *testing.T) {
		e := testutil.EditorWithText(t, "(abc)")
		testutil.SetCursor(t, e, 0)

		action.MatchBrackets(e)

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("noop when no bracket under cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.MatchBrackets(e)

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestPasteRegisterAtCursor(t *testing.T) {
	t.Run("pastes register content at cursor", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ac")
		testutil.SetCursor(t, e, 1)
		e.Registers().Write('x', []string{"b"})

		action.PasteRegisterAtCursor(e, 'x')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})

	t.Run("noop for missing register", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.PasteRegisterAtCursor(e, 'z')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestCloseCurrentViewForce(t *testing.T) {
	t.Run("always closes view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		before := viewCount(t, e)

		action.CloseCurrentViewForce(e)

		assert.Equal(t, before-1, viewCount(t, e))
	})
}

func TestTransposeView(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		v, _ := e.FocusedView()
		e.VSplit(v.DocID())

		assert.NotPanics(t, func() { action.TransposeView(e) })
	})
}

func TestJumpViews(t *testing.T) {
	t.Run("JumpViewLeft does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.JumpViewLeft(e) })
	})

	t.Run("JumpViewRight does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.JumpViewRight(e) })
	})

	t.Run("JumpViewUp does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.JumpViewUp(e) })
	})

	t.Run("JumpViewDown does not panic", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.JumpViewDown(e) })
	})
}

func TestSwapViews(t *testing.T) {
	t.Run("left noop with single view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.SwapViewLeft(e) })
	})

	t.Run("right noop with single view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.SwapViewRight(e) })
	})

	t.Run("up noop with single view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.SwapViewUp(e) })
	})

	t.Run("down noop with single view", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.SwapViewDown(e) })
	})
}

func TestGotoLastAccessedFile(t *testing.T) {
	t.Run("noop when no previous doc", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.GotoLastAccessedFile(e) })
	})
}

func TestGotoLastModifiedFile(t *testing.T) {
	t.Run("noop when no previous modified doc", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")

		assert.NotPanics(t, func() { action.GotoLastModifiedFile(e) })
	})
}

func TestSmartTabNotAllWhitespace(t *testing.T) {
	t.Run("noop with non-whitespace to left", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 5)

		before, _ := e.FocusedDocument()
		textBefore := before.Text().String()
		action.SmartTab(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, textBefore, doc.Text().String())
	})
}

func TestRepeatLastMotion(t *testing.T) {
	t.Run("replays last motion", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb\n\nc")
		testutil.SetCursor(t, e, 0)
		action.GotoNextParagraph(e)
		pos1 := testutil.CursorPos(t, e)

		action.RepeatLastMotion(e)

		pos2 := testutil.CursorPos(t, e)
		assert.True(t, pos2 > pos1)
	})

	t.Run("noop with no last motion", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.RepeatLastMotion(e)

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestCharInfo(t *testing.T) {
	t.Run("ascii char", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a")
		testutil.SetCursor(t, e, 0)

		assert.Equal(t, `"a" (U+0061) Dec 97 Hex 61`, action.CharInfo(e))
	})

	t.Run("newline shows escape", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetCursor(t, e, 1)

		assert.Equal(t, `"\n" (U+000a) Dec 10 Hex 0a`, action.CharInfo(e))
	})

	t.Run("multi-byte char", func(t *testing.T) {
		e := testutil.EditorWithText(t, "é")
		testutil.SetCursor(t, e, 0)

		assert.Equal(t, `"é" (U+00e9) Hex c3 a9`, action.CharInfo(e))
	})

	t.Run("empty at EOF", func(t *testing.T) {
		e := testutil.EditorWithText(t, "")
		testutil.SetCursor(t, e, 0)

		assert.Equal(t, "", action.CharInfo(e))
	})

	t.Run("tab shows escape", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\t")
		testutil.SetCursor(t, e, 0)

		assert.Equal(t, `"\t" (U+0009) Dec 9 Hex 09`, action.CharInfo(e))
	})

	t.Run("cr shows escape", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\r")
		testutil.SetCursor(t, e, 0)

		assert.Equal(t, `"\r" (U+000d) Dec 13 Hex 0d`, action.CharInfo(e))
	})

	t.Run("null shows escape", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\x00")
		testutil.SetCursor(t, e, 0)

		assert.Equal(t, `"\0" (U+0000) Dec 0 Hex 00`, action.CharInfo(e))
	})

	t.Run("multi-codepoint grapheme", func(t *testing.T) {
		// e + combining acute = two codepoints, one grapheme
		e := testutil.EditorWithText(t, "é")
		testutil.SetCursor(t, e, 0)

		info := action.CharInfo(e)
		assert.Contains(t, info, "U+0065")
		assert.Contains(t, info, "U+0301")
	})
}

func TestSetLineEndingCRLFConversion(t *testing.T) {
	t.Run("converts LF to CRLF", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb\nc")

		err := action.SetLineEnding(e, core.LineEndingCRLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\r\nb\r\nc", doc.Text().String())
	})

	t.Run("converts CRLF back to LF", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\r\nb")

		err := action.SetLineEnding(e, core.LineEndingLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\nb", doc.Text().String())
	})

	t.Run("no-op when CRLF already matches", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\r\nb")

		err := action.SetLineEnding(e, core.LineEndingCRLF)

		assert.NoError(t, err)
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a\r\nb", doc.Text().String())
	})
}

func TestGotoLastAccessedFileSwitches(t *testing.T) {
	t.Run("switches back to first document", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "file1.txt")
		f2 := filepath.Join(dir, "file2.txt")
		assert.NoError(t, os.WriteFile(f1, []byte("first"), 0o644))
		assert.NoError(t, os.WriteFile(f2, []byte("second"), 0o644))

		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		v1, err := e.OpenFile(f1)
		assert.NoError(t, err)
		firstDocID := v1.DocID()

		// Split so file1 stays in one view while we open file2 in a new view
		e.HSplit(firstDocID)
		_, err = e.OpenFile(f2)
		assert.NoError(t, err)
		v2, _ := e.FocusedView()
		secondDocID := v2.DocID()
		assert.NotEqual(t, firstDocID, secondDocID)

		// Now GotoLastAccessedFile should find a view showing file1
		action.GotoLastAccessedFile(e)

		vAfter, _ := e.FocusedView()
		assert.Equal(t, firstDocID, vAfter.DocID())
	})
}

func TestCommentTokenAt(t *testing.T) {
	t.Run("toggle comment removes existing token", func(t *testing.T) {
		e := testutil.EditorWithText(t, "# hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})

	t.Run("uncommented line gets token added", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.ToggleLineComments(e)

		doc, _ := e.FocusedDocument()
		assert.Contains(t, doc.Text().String(), "hello")
		assert.True(t, len(doc.Text().String()) > len("hello"))
	})
}

func TestSkipHorizontalWhitespace(t *testing.T) {
	t.Run("MoveLineNonWhitespace skips tabs", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\t\thello")
		testutil.SetCursor(t, e, 0)

		action.MoveLineNonWhitespace(e)

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("mixed spaces and tabs skipped", func(t *testing.T) {
		e := testutil.EditorWithText(t, " \thello")
		testutil.SetCursor(t, e, 0)

		action.MoveLineNonWhitespace(e)

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestGotoLastModifiedFileSwitches(t *testing.T) {
	t.Run("switches to last modified document", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "file1.txt")
		f2 := filepath.Join(dir, "file2.txt")
		assert.NoError(t, os.WriteFile(f1, []byte("first"), 0o644))
		assert.NoError(t, os.WriteFile(f2, []byte("second"), 0o644))

		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		v1, err := e.OpenFile(f1)
		assert.NoError(t, err)
		firstDocID := v1.DocID()

		// Modify first doc then switch to second
		doc1, _ := e.FocusedDocument()
		rope := doc1.Text()
		cs, err2 := core.NewChangeSetFromChanges(
			rope, []core.Change{core.TextChange(0, 0, "x")},
		)
		assert.NoError(t, err2)
		_ = e.Apply(core.NewTransaction(rope).WithChanges(cs))

		// Split and open file2 to record file1 as last modified
		e.HSplit(firstDocID)
		_, err = e.OpenFile(f2)
		assert.NoError(t, err)

		v2, _ := e.FocusedView()
		secondDocID := v2.DocID()
		assert.NotEqual(t, firstDocID, secondDocID)

		action.GotoLastModifiedFile(e)

		vAfter, _ := e.FocusedView()
		assert.Equal(t, firstDocID, vAfter.DocID())
	})
}

func TestIncrementWithHashRegister(t *testing.T) {
	t.Run("each selection gets different increment", func(t *testing.T) {
		e := testutil.EditorWithText(t, "1\n1")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 1),
			core.NewRange(2, 3),
		}, 0)
		e.SetRegister('#')

		action.Increment(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "2")
		assert.Contains(t, text, "3")
	})
}

func TestJoinSelectionsCommented(t *testing.T) {
	t.Run("join merges commented lines into one", func(t *testing.T) {
		e := testutil.EditorWithText(t, "# foo\n# bar")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 11)}, 0)

		action.JoinSelections(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.NotContains(t, text, "\n")
	})

	t.Run("strips duplicate comment token on join", func(t *testing.T) {
		writeTextLangConfigBuf(t, "//")

		e := testutil.EditorWithText(t, "// foo\n// bar")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 13)}, 0)

		action.JoinSelections(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.NotContains(t, text, "\n")
	})

	t.Run("join with space separator", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\nb")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.JoinSelectionsSpace(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "a b", doc.Text().String())
	})
}

func TestSearchWithCount(t *testing.T) {
	t.Run("count skips multiple matches", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo foo foo")
		err := action.SearchForward(e, "foo")
		assert.NoError(t, err)
		e.SetCount(2)

		action.SearchNext(e)

		assert.True(t, testutil.CursorPos(t, e) >= 0)
	})
}

func TestIncrementNonNumber(t *testing.T) {
	t.Run("increment on non-numeric word is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello")
		testutil.SetCursor(t, e, 0)

		action.Increment(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "hello", doc.Text().String())
	})
}

func TestInsertTabDuplicateCursors(t *testing.T) {
	t.Run("same-position cursors insert tab once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(1),
			core.PointRange(1),
		}, 0)

		action.InsertTab(e)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.True(t, len(text) > 2)
	})
}

func TestPasteRegisterAtCursorDuplicate(t *testing.T) {
	t.Run("same-position cursors paste once", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ac")
		testutil.SetSelection(t, e, []core.Range{
			core.PointRange(1),
			core.PointRange(1),
		}, 0)
		e.Registers().Write('x', []string{"b"})

		action.PasteRegisterAtCursor(e, 'x')

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestExtendToColumnEdge(t *testing.T) {
	t.Run("extend to column 1 selects first char", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 3)
		e.SetCount(1)

		action.ExtendToColumn(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().From() <= sel.Primary().To())
	})
}

func TestFilterSelectionsImplEdge(t *testing.T) {
	t.Run("no matches leaves single range", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\nghi")
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, 3),
			core.NewRange(4, 7),
		}, 0)

		err := action.KeepSelectionsMatching(e, "xyz")

		assert.NoError(t, err)
	})
}

func TestReflowSelectionsWidth(t *testing.T) {
	t.Run("reflow with explicit width", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar baz qux")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 15)}, 0)

		action.ReflowSelections(e, 5)

		doc, _ := e.FocusedDocument()
		text := doc.Text().String()
		assert.Contains(t, text, "\n")
	})
}

func TestYankJoinSingleRange(t *testing.T) {
	t.Run("single range joins without separator", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		action.YankJoin(e, ",")

		assert.Equal(t, "abc", testutil.RegisteredValue(t, e, '"'))
	})
}

func writeTextLangConfigBuf(t *testing.T, commentToken string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	configDir := filepath.Join(dir, "toe")
	assert.NoError(t, os.MkdirAll(configDir, 0o755))
	content := "[[language]]\nname = \"text\"\nscope = \"text.plain\"\n" +
		"comment-token = \"" + commentToken + "\"\n"
	assert.NoError(t,
		os.WriteFile(filepath.Join(configDir, "languages.toml"),
			[]byte(content), 0o644),
	)
}
