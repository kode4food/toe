package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/action"
)

func TestMoveLeft(t *testing.T) {
	e := editorWithText(t, "hello")
	setCursor(t, e, 3)

	action.MoveLeft(e)
	assert.Equal(t, 2, cursorPos(t, e))
}

func TestMoveRight(t *testing.T) {
	e := editorWithText(t, "hello")
	action.MoveRight(e)
	assert.Equal(t, 1, cursorPos(t, e))
}

func TestMoveUp(t *testing.T) {
	e := editorWithText(t, "abc\ndef\n")
	setCursor(t, e, 4)
	action.MoveUp(e)
	assert.Equal(t, 0, cursorPos(t, e))
}

func TestMoveDown(t *testing.T) {
	e := editorWithText(t, "abc\ndef\n")
	action.MoveDown(e)
	assert.Equal(t, 4, cursorPos(t, e))
}

func TestMoveLineStart(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	setCursor(t, e, 8)
	action.MoveLineStart(e)
	assert.Equal(t, 6, cursorPos(t, e))
}

func TestMoveLineEnd(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	action.MoveLineEnd(e)
	assert.Equal(t, 4, cursorPos(t, e))
}

func TestMoveLineEndEmpty(t *testing.T) {
	e := editorWithText(t, "\n")
	action.MoveLineEnd(e)
	assert.Equal(t, 0, cursorPos(t, e))
}

func TestMoveFileStart(t *testing.T) {
	e := editorWithText(t, "abc\ndef")
	setCursor(t, e, 5)
	action.MoveFileStart(e)
	assert.Equal(t, 0, cursorPos(t, e))
}

func TestMoveFileEnd(t *testing.T) {
	e := editorWithText(t, "abc\ndef")
	action.MoveFileEnd(e)
	assert.Equal(t, 4, cursorPos(t, e))
}

func TestMoveLineNonWhitespace(t *testing.T) {
	e := editorWithText(t, "  hello")
	setCursor(t, e, 6)
	action.MoveLineNonWhitespace(e)
	assert.Equal(t, 2, cursorPos(t, e))
}

func TestMoveWordForward(t *testing.T) {
	e := editorWithText(t, "one two")
	action.MoveWordForward(e)
	assert.Equal(t, 3, cursorPos(t, e))
}

func TestMoveWordBackward(t *testing.T) {
	e := editorWithText(t, "one two")
	setCursor(t, e, 5)
	action.MoveWordBackward(e)
	assert.True(t, cursorPos(t, e) < 5)
}

func TestMoveWordEnd(t *testing.T) {
	e := editorWithText(t, "one two")
	action.MoveWordEnd(e)
	assert.True(t, cursorPos(t, e) > 0)
}

func TestMoveLongWord(t *testing.T) {
	t.Run("forward", func(t *testing.T) {
		e := editorWithText(t, "foo bar baz")
		action.MoveLongWordForward(e)
		assert.True(t, cursorPos(t, e) > 0)
	})

	t.Run("backward", func(t *testing.T) {
		e := editorWithText(t, "foo bar baz")
		setCursor(t, e, 8)
		action.MoveLongWordBackward(e)
		assert.True(t, cursorPos(t, e) < 8)
	})

	t.Run("end", func(t *testing.T) {
		e := editorWithText(t, "foo bar baz")
		action.MoveLongWordEnd(e)
		assert.True(t, cursorPos(t, e) > 0)
	})
}

func TestExtendCharRight(t *testing.T) {
	e := editorWithText(t, "hello")
	action.ExtendCharRight(e)
	a, h := selectionAnchorHead(t, e)
	assert.True(t, h > a)
}

func TestExtendCharLeft(t *testing.T) {
	e := editorWithText(t, "hello")
	setCursor(t, e, 3)
	action.ExtendCharLeft(e)
	a, h := selectionAnchorHead(t, e)
	assert.True(t, h < a)
}

func TestExtendLineDown(t *testing.T) {
	e := editorWithText(t, "abc\ndef\n")
	action.ExtendLineDown(e)
	a, h := selectionAnchorHead(t, e)
	assert.True(t, h > a)
}

func TestExtendLineUp(t *testing.T) {
	e := editorWithText(t, "abc\ndef\n")
	setCursor(t, e, 4)
	action.ExtendLineUp(e)
	a, h := selectionAnchorHead(t, e)
	assert.True(t, h < a)
}

func TestExtendNextWordStart(t *testing.T) {
	e := editorWithText(t, "one two")
	action.ExtendNextWordStart(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, a)
	assert.True(t, h > a)
}

func TestExtendPrevWordStart(t *testing.T) {
	e := editorWithText(t, "one two")
	setCursor(t, e, 5)
	action.ExtendPrevWordStart(e)
	a, h := selectionAnchorHead(t, e)
	assert.True(t, h < a)
}

func TestExtendNextWordEnd(t *testing.T) {
	e := editorWithText(t, "one two")
	action.ExtendNextWordEnd(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, a)
	assert.True(t, h > a)
}

func TestExtendNextLongWordStart(t *testing.T) {
	e := editorWithText(t, "one two")
	action.ExtendNextLongWordStart(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, a)
	assert.True(t, h > a)
}

func TestExtendPrevLongWordStart(t *testing.T) {
	e := editorWithText(t, "one two")
	setCursor(t, e, 5)
	action.ExtendPrevLongWordStart(e)
	_, h := selectionAnchorHead(t, e)
	assert.True(t, h < 5)
}

func TestExtendNextLongWordEnd(t *testing.T) {
	e := editorWithText(t, "one two")
	action.ExtendNextLongWordEnd(e)
	a, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, a)
	assert.True(t, h > a)
}

func TestExtendToLineStart(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	setCursor(t, e, 8)
	action.ExtendToLineStart(e)
	_, h := selectionAnchorHead(t, e)
	assert.True(t, h <= 6)
}

func TestExtendToLineEnd(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	action.ExtendToLineEnd(e)
	assert.Equal(t, 4, cursorPos(t, e))
	_, h := selectionAnchorHead(t, e)
	assert.True(t, h > 0)
}

func TestExtendToFileStart(t *testing.T) {
	e := editorWithText(t, "abc\ndef")
	setCursor(t, e, 5)
	action.ExtendToFileStart(e)
	_, h := selectionAnchorHead(t, e)
	assert.Equal(t, 0, h)
}

func TestExtendToLastLine(t *testing.T) {
	e := editorWithText(t, "abc\ndef")
	action.ExtendToLastLine(e)
	assert.Equal(t, 4, cursorPos(t, e))
}

func TestExtendToColumn(t *testing.T) {
	e := editorWithText(t, "hello\nworld")
	e.SetCount(3)
	action.ExtendToColumn(e)
	_, h := selectionAnchorHead(t, e)
	assert.True(t, h > 0)
}

func TestGotoNextParagraph(t *testing.T) {
	e := editorWithText(t, "a\n\nb\n")
	action.GotoNextParagraph(e)
	assert.True(t, cursorPos(t, e) > 0)
}

func TestGotoPrevParagraph(t *testing.T) {
	e := editorWithText(t, "a\n\nb\n")
	setCursor(t, e, 4)
	action.GotoPrevParagraph(e)
	assert.True(t, cursorPos(t, e) >= 0)
}
