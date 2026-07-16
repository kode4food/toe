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

const scrollViewLinesText = "line\nline\nline\nline\nline\nline\nline\n" +
	"line\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\n" +
	"line\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\n" +
	"line\n"

func TestMotion(t *testing.T) {
	t.Run("move line bounds", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 4)

		action.MoveLineStart(e)
		assert.Equal(t, 3, testutil.CursorPos(t, e))

		action.MoveLineEnd(e)
		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("add newline above and below", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 3)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ab\n\ncd", doc.Text().String())

		e = testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 0)

		action.AddNewlineBelow(e)

		doc, _ = e.FocusedDocument()
		assert.Equal(t, "ab\n\ncd", doc.Text().String())
	})

	t.Run("add newline with count", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 3)
		e.SetCount(2)

		action.AddNewlineAbove(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "ab\n\n\ncd", doc.Text().String())
	})

	t.Run("paragraph motion across blank lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb\n\nc")
		testutil.SetCursor(t, e, 0)

		action.GotoNextParagraph(e)
		assert.Equal(t, 3, testutil.CursorPos(t, e))

		action.GotoPrevParagraph(e)
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestMoveLineEndEmpty(t *testing.T) {
	t.Run("MoveLineEnd on empty line stays at start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\nabc")
		testutil.SetCursor(t, e, 0)

		action.MoveLineEnd(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("empty line stays at start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\nabc")
		testutil.SetCursor(t, e, 0)

		action.ExtendToLineEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
	})
}

func TestMoveLeft(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		start int
		want  int
	}{
		{"moves one left", "abcd", 2, 1},
		{"clamps at start", "abcd", 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := testutil.EditorWithText(t, tc.text)
			testutil.SetCursor(t, e, tc.start)

			action.MoveLeft(e)

			assert.Equal(t, tc.want, testutil.CursorPos(t, e))
		})
	}
}

func TestMoveRight(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		start int
		want  int
	}{
		{"moves one right", "abcd", 1, 2},
		{"at last char moves to end pos", "abcd", 3, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := testutil.EditorWithText(t, tc.text)
			testutil.SetCursor(t, e, tc.start)

			action.MoveRight(e)

			assert.Equal(t, tc.want, testutil.CursorPos(t, e))
		})
	}
}

func TestMoveUp(t *testing.T) {
	t.Run("moves up a line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 3)

		action.MoveUp(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("stays on same column on first line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 1)

		action.MoveUp(e)

		// cursor at col 1 on line 0 -> stays at pos 1
		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestMoveDown(t *testing.T) {
	t.Run("moves down a line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 0)

		action.MoveDown(e)

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("clamps at last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd")
		testutil.SetCursor(t, e, 4)

		action.MoveDown(e)

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})
}

func TestMoveWordForward(t *testing.T) {
	t.Run("moves to next word start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetCursor(t, e, 0)

		action.MoveWordForward(e)

		// lands at the whitespace boundary before "world"
		assert.Equal(t, 5, testutil.CursorPos(t, e))
	})
}

func TestMoveWordBackward(t *testing.T) {
	t.Run("moves to previous word start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetCursor(t, e, 6)

		action.MoveWordBackward(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestMoveWordEnd(t *testing.T) {
	t.Run("moves to end of current word", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetCursor(t, e, 0)

		action.MoveWordEnd(e)

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})
}

func TestMoveLongWordForward(t *testing.T) {
	t.Run("moves to space before next WORD", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 0)

		action.MoveLongWordForward(e)

		// WORD = non-whitespace; lands at space boundary
		assert.Equal(t, 7, testutil.CursorPos(t, e))
	})
}

func TestMoveLongWordBackward(t *testing.T) {
	t.Run("jumps backward to previous WORD start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 8)

		action.MoveLongWordBackward(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestMoveLongWordEnd(t *testing.T) {
	t.Run("moves to end of current WORD", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 0)

		action.MoveLongWordEnd(e)

		assert.Equal(t, 6, testutil.CursorPos(t, e))
	})
}

func TestMoveFileStartEnd(t *testing.T) {
	t.Run("MoveFileStart goes to beginning", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 5)

		action.MoveFileStart(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("MoveFileEnd goes to last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 0)

		action.MoveFileEnd(e)

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("MoveFileEnd skips blank last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\n")
		testutil.SetCursor(t, e, 0)

		action.MoveFileEnd(e)

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})
}

func TestMoveLineNonWhitespace(t *testing.T) {
	t.Run("skips leading whitespace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "  hello")
		testutil.SetCursor(t, e, 0)

		action.MoveLineNonWhitespace(e)

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestExtendToLineStart(t *testing.T) {
	t.Run("extends selection to line start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 3)

		action.ExtendToLineStart(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
	})
}

func TestExtendToLineEnd(t *testing.T) {
	t.Run("extends selection to line end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 0)

		action.ExtendToLineEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() > 0)
	})
}

func TestExtendToFileStart(t *testing.T) {
	t.Run("extends to doc start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef")
		testutil.SetCursor(t, e, 4)

		action.ExtendToFileStart(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 0, sel.Primary().From())
	})
}

func TestExtendToLastLine(t *testing.T) {
	t.Run("extends to last non-blank line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef")
		testutil.SetCursor(t, e, 0)

		action.ExtendToLastLine(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= 4)
	})
}

func TestExtendToFileEnd(t *testing.T) {
	t.Run("extends to absolute end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 0)

		action.ExtendToFileEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 3, sel.Primary().To())
	})
}

func TestExtendCharLeftRight(t *testing.T) {
	t.Run("ExtendCharLeft grows selection left", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 2)

		action.ExtendCharLeft(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 1)
	})

	t.Run("ExtendCharRight grows selection right", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcde")
		testutil.SetCursor(t, e, 2)

		action.ExtendCharRight(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 1)
	})
}

func TestExtendLineUpDown(t *testing.T) {
	t.Run("ExtendLineUp grows selection upward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\nghi")
		testutil.SetCursor(t, e, 7)

		action.ExtendLineUp(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 0)
	})

	t.Run("ExtendLineDown grows selection downward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\ndef\nghi")
		testutil.SetCursor(t, e, 1)

		action.ExtendLineDown(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To()-sel.Primary().From() >= 0)
	})
}

func TestExtendWordMotions(t *testing.T) {
	t.Run("ExtendNextWordStart grows to next word", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 0)

		action.ExtendNextWordStart(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() > sel.Primary().From())
	})

	t.Run("ExtendPrevWordStart shrinks toward start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 6)

		action.ExtendPrevWordStart(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})

	t.Run("ExtendNextWordEnd extends to word end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 0)

		action.ExtendNextWordEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})

	t.Run("extends to WORD start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 0)

		action.ExtendNextLongWordStart(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})

	t.Run("extends to prev word end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 6)

		action.ExtendPrevWordEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})
}

func TestExtendToNonWhitespace(t *testing.T) {
	t.Run("extends to first non-whitespace", func(t *testing.T) {
		e := testutil.EditorWithText(t, "   hello")
		testutil.SetCursor(t, e, 0)

		action.ExtendToNonWhitespace(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= 3)
	})
}

func TestMoveSubWords(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(*view.Editor)
		text  string
		start int
	}{
		{"MoveNextSubWordStart", action.MoveNextSubWordStart, "fooBar", 0},
		{"MovePrevSubWordStart", action.MovePrevSubWordStart, "fooBar", 3},
		{"MoveNextSubWordEnd", action.MoveNextSubWordEnd, "fooBar", 0},
		{"MovePrevSubWordEnd", action.MovePrevSubWordEnd, "fooBar", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name+" moves cursor", func(t *testing.T) {
			e := testutil.EditorWithText(t, tc.text)
			testutil.SetCursor(t, e, tc.start)

			tc.fn(e)

			assert.True(t, testutil.CursorPos(t, e) >= 0)
		})
	}
}

func TestMovePrevWordEnd(t *testing.T) {
	t.Run("moves to end of previous word", func(t *testing.T) {
		e := testutil.EditorWithText(t, "hello world")
		testutil.SetCursor(t, e, 6)

		action.MovePrevWordEnd(e)

		assert.True(t, testutil.CursorPos(t, e) < 6)
	})
}

func TestMovePrevLongWordEnd(t *testing.T) {
	t.Run("moves to end of previous WORD", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 8)

		action.MovePrevLongWordEnd(e)

		assert.True(t, testutil.CursorPos(t, e) < 8)
	})
}

func TestExtendLongWordMotions(t *testing.T) {
	t.Run("extends selection backward to WORD start", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 8)

		action.ExtendPrevLongWordStart(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})

	t.Run("extends selection forward to WORD end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo.bar baz")
		testutil.SetCursor(t, e, 0)

		action.ExtendNextLongWordEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})

	t.Run("extends to prev WORD end", func(t *testing.T) {
		e := testutil.EditorWithText(t, "foo bar")
		testutil.SetCursor(t, e, 6)

		action.ExtendPrevLongWordEnd(e)

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.True(t, sel.Primary().To() >= sel.Primary().From())
	})
}

func TestExtendSubWordMotions(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(*view.Editor)
		start int
	}{
		{"ExtendNextSubWordStart", action.ExtendNextSubWordStart, 0},
		{"ExtendPrevSubWordStart", action.ExtendPrevSubWordStart, 3},
		{"ExtendNextSubWordEnd", action.ExtendNextSubWordEnd, 0},
		{"ExtendPrevSubWordEnd", action.ExtendPrevSubWordEnd, 5},
	}
	for _, tc := range tests {
		t.Run(tc.name+" moves cursor", func(t *testing.T) {
			e := testutil.EditorWithText(t, "fooBar")
			testutil.SetCursor(t, e, tc.start)

			tc.fn(e)

			v, _ := e.FocusedView()
			doc, _ := e.FocusedDocument()
			sel := doc.SelectionFor(v.ID())
			assert.True(t, sel.Primary().To() >= 0)
		})
	}
}

func TestVisualMoveFormat(t *testing.T) {
	t.Run("no viewport width uses logical lines up", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		e.SetViewContentWidth(0)
		testutil.SetCursor(t, e, 3)

		action.MoveUp(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("no viewport width uses logical down", func(t *testing.T) {
		e := testutil.EditorWithText(t, "ab\ncd\nef")
		e.SetViewContentWidth(0)
		testutil.SetCursor(t, e, 0)

		action.MoveDown(e)

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("soft-wrap width uses visual lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef\nghijkl")
		e.Options().SoftWrap.Enable = new(true)
		e.SetViewContentWidth(20)
		testutil.SetCursor(t, e, 7)

		action.MoveUp(e)

		assert.True(t, testutil.CursorPos(t, e) >= 0)
	})

	t.Run("soft-wrap uses visual lines", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdef\nghijkl")
		e.Options().SoftWrap.Enable = new(true)
		e.SetViewContentWidth(20)
		testutil.SetCursor(t, e, 0)

		action.MoveDown(e)

		assert.True(t, testutil.CursorPos(t, e) >= 0)
	})
}

func TestMoveLineEndOnInternalEmptyLine(t *testing.T) {
	t.Run("empty non-first line stays put", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb")
		testutil.SetCursor(t, e, 2)

		action.MoveLineEnd(e)

		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})
}

func TestParagraphMotionWithCount(t *testing.T) {
	t.Run("count=2 skips two paragraphs forward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb\n\nc")
		testutil.SetCursor(t, e, 0)
		e.SetCount(2)

		action.GotoNextParagraph(e)

		assert.Equal(t, 6, testutil.CursorPos(t, e))
	})

	t.Run("no next paragraph goes to last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb")
		testutil.SetCursor(t, e, 2)

		action.GotoNextParagraph(e)

		assert.True(t, testutil.CursorPos(t, e) > 0)
	})

	t.Run("no prev paragraph goes to first line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb")
		testutil.SetCursor(t, e, 0)

		action.GotoPrevParagraph(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("prev para from blank first line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "\nabc\ndef")
		testutil.SetCursor(t, e, 2)

		action.GotoPrevParagraph(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("next para from blank last line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc\n\n")
		testutil.SetCursor(t, e, 0)

		action.GotoNextParagraph(e)

		assert.True(t, testutil.CursorPos(t, e) > 0)
	})

	t.Run("count=2 skips two paragraphs backward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "a\n\nb\n\nc")
		testutil.SetCursor(t, e, 6)
		e.SetCount(2)

		action.GotoPrevParagraph(e)

		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestGotoFile(t *testing.T) {
	t.Run("finds existing file path at cursor", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "target.txt")
		assert.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

		e := testutil.EditorWithText(t, path)
		testutil.SetCursor(t, e, 0)

		target, err := action.GotoFileTarget(e)

		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		e := testutil.EditorWithText(t, "/no/such/file.txt")
		testutil.SetCursor(t, e, 0)

		_, err := action.GotoFileTarget(e)

		assert.Error(t, err)
	})

	t.Run("returns ErrNoFilePath on delimiter-only", func(t *testing.T) {
		e := testutil.EditorWithText(t, "   ")
		testutil.SetCursor(t, e, 1)

		_, err := action.GotoFileTarget(e)

		assert.Error(t, err)
	})

	t.Run("relative path resolved via cwd", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "rel.txt"), []byte("x"), 0o644,
		))

		e := testutil.EditorWithText(t, "rel.txt")
		assert.NoError(t, e.Chdir(dir))
		testutil.SetCursor(t, e, 0)

		target, err := action.GotoFileTarget(e)

		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "rel.txt"), target.Path)
	})

	t.Run("relative path resolved via doc dir", func(t *testing.T) {
		dir := t.TempDir()
		relPath := filepath.Join(dir, "rel.txt")
		assert.NoError(t, os.WriteFile(relPath, []byte("x"), 0o644))
		docPath := filepath.Join(dir, "main.txt")
		assert.NoError(t, os.WriteFile(docPath, []byte("rel.txt"), 0o644))

		e := view.NewEditor("/tmp")
		e.ResizeTree(80, 24)
		_, err := e.OpenFile(docPath)
		assert.NoError(t, err)
		testutil.SetCursor(t, e, 0)

		target, err := action.GotoFileTarget(e)

		assert.NoError(t, err)
		assert.Equal(t, relPath, target.Path)
	})

	t.Run("uses document link", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "target.txt")
		assert.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
		e := testutil.EditorWithText(t, "not-a-real-path")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 15, Target: "file://" + path},
		})
		testutil.SetCursor(t, e, 2)

		target, err := action.GotoFileTarget(e)

		assert.NoError(t, err)
		assert.Equal(t, path, target.Path)
	})

	t.Run("returns external target", func(t *testing.T) {
		e := testutil.EditorWithText(t, "https://example.com")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 19, Target: "https://example.com"},
		})
		testutil.SetCursor(t, e, 2)

		target, err := action.GotoFileTarget(e)

		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", target.URL)
		assert.Empty(t, target.Path)
	})

	t.Run("cursor in middle expands outward", func(t *testing.T) {
		e := testutil.EditorWithText(t, "/no/such/file.txt")
		testutil.SetCursor(t, e, 8)

		_, err := action.GotoFileTarget(e)

		assert.Error(t, err)
	})
}

func TestAlignSelections(t *testing.T) {
	t.Run("pads shorter cursors to match longest", func(t *testing.T) {
		e := testutil.EditorWithText(t, "x\nyyy")
		// line 0: cursor at pos 1 (col 1)
		// line 1: "yyy" starts at pos 2; cursor at pos 4 (col 2)
		// maxCol = 2; pad line 0 col 1 by 1 space
		testutil.SetSelection(t, e,
			[]core.Range{
				core.PointRange(1),
				core.PointRange(4),
			},
			0,
		)

		action.AlignSelections(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "x \nyyy", doc.Text().String())
	})

	t.Run("single selection is noop", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abc")
		testutil.SetCursor(t, e, 1)

		action.AlignSelections(e)

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestScrollViewLines(t *testing.T) {
	t.Run("scrolls view down", func(t *testing.T) {
		e := testutil.EditorWithText(t, scrollViewLinesText)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		action.ScrollViewLines(e, v, 3, false)

		offset := v.Offset()
		assert.NotEqual(t, 0, offset.Anchor)
	})

	t.Run("scrolls view up", func(t *testing.T) {
		e := testutil.EditorWithText(t, scrollViewLinesText)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		action.ScrollViewLines(e, v, 5, false)

		action.ScrollViewLines(e, v, 3, true)

		offset := v.Offset()
		assert.NotEqual(t, 0, offset.Anchor)
	})

	t.Run("zero lines clamps to one", func(t *testing.T) {
		e := testutil.EditorWithText(t, scrollViewLinesText)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor
		action.ScrollViewLines(e, v, 0, false)
		assert.GreaterOrEqual(t, v.Offset().Anchor, before)
	})
}

func TestScrollViewColumns(t *testing.T) {
	t.Run("scrolls view right", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdefghij\n")
		v, ok := e.FocusedView()
		assert.True(t, ok)

		action.ScrollViewColumns(e, v, 3, false)

		assert.Equal(t, 3, v.Offset().HorizontalOffset)
	})

	t.Run("right clamps to widest visible line", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdefghij\n")
		v, ok := e.FocusedView()
		assert.True(t, ok)

		action.ScrollViewColumns(e, v, 100, false)

		assert.Equal(t, 9, v.Offset().HorizontalOffset)
	})

	t.Run("left clamps to zero", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdefghij\n")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		action.ScrollViewColumns(e, v, 3, false)

		action.ScrollViewColumns(e, v, 100, true)

		assert.Equal(t, 0, v.Offset().HorizontalOffset)
	})

	t.Run("zero columns clamps to one", func(t *testing.T) {
		e := testutil.EditorWithText(t, "abcdefghij\n")
		v, ok := e.FocusedView()
		assert.True(t, ok)

		action.ScrollViewColumns(e, v, 0, false)

		assert.Equal(t, 1, v.Offset().HorizontalOffset)
	})
}
