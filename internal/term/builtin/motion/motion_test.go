package motion_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
)

func TestMotionGotoLine(t *testing.T) {
	t.Run("goto line honors count", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\nl2\nl3\n")
		e.SetCount(2)
		test.RunCmd(t, km, e, "goto_line")
		assert.Equal(t, 1, test.CursorLine(t, e))
	})

	t.Run("goto line or file start without count", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\nl2\n")
		testutil.SetCursor(t, e, 5)
		test.RunCmd(t, km, e, "goto_line_or_file_start")
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("goto line or extend file start", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\nl2\n")
		testutil.SetCursor(t, e, 5)
		test.RunCmd(t, km, e, "goto_line_or_extend_file_start")
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestMotionFindChar(t *testing.T) {
	t.Run("find next char lands on target", func(t *testing.T) {
		e, km := test.Env(t, "abcdef")
		testutil.SetCursor(t, e, 0)
		res := test.RunCmd(t, km, e, "find_next_char")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('c'))
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("find till char stops before target", func(t *testing.T) {
		e, km := test.Env(t, "abcdef")
		testutil.SetCursor(t, e, 0)
		res := test.RunCmd(t, km, e, "find_till_char")
		res.Continuation(e, test.Char('c'))
		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})

	t.Run("non-char key cancels find", func(t *testing.T) {
		e, km := test.Env(t, "abcdef")
		testutil.SetCursor(t, e, 0)
		res := test.RunCmd(t, km, e, "find_next_char")
		res.Continuation(e, test.Special(command.Escape))
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})
}

func TestMotionGotoFile(t *testing.T) {
	t.Run("no path under cursor is handled", func(t *testing.T) {
		e, km := test.Env(t, "not a path")
		testutil.SetCursor(t, e, 0)
		assert.Nil(t, test.RunCmd(t, km, e, "goto_file").Continuation)
	})

	t.Run("valid file path opens file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "hello.txt")
		assert.NoError(t, os.WriteFile(target, []byte("hi"), 0o644))

		e, km := test.Env(t, target)
		testutil.SetCursor(t, e, 0)

		assert.Nil(t, test.RunCmd(t, km, e, "goto_file").Continuation)
	})

	t.Run("directory path reports open error", func(t *testing.T) {
		dir := t.TempDir()
		e, km := test.Env(t, dir)
		testutil.SetCursor(t, e, 0)

		assert.Nil(t, test.RunCmd(t, km, e, "goto_file").Continuation)

		assert.Contains(t, e.TakeStatusMsg(), "error:")
	})
}

func TestMotionParagraph(t *testing.T) {
	t.Run("next paragraph moves cursor down", func(t *testing.T) {
		e, km := test.Env(t, "a\n\nb\n")
		testutil.SetCursor(t, e, 0)
		test.RunCmd(t, km, e, "goto_next_paragraph")
		assert.Greater(t, testutil.CursorPos(t, e), 0)
	})

	t.Run("prev paragraph runs from end", func(t *testing.T) {
		e, km := test.Env(t, "a\n\nb\n")
		testutil.SetCursor(t, e, 4)
		test.RunCmd(t, km, e, "goto_prev_paragraph")
		assert.Less(t, testutil.CursorPos(t, e), 4)
	})
}

func TestMotionOptions(t *testing.T) {
	t.Run("get/set scrolloff", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "scrolloff 5")
		res := test.RunCmdArgs(t, km, e, "get_option", "scrolloff")
		assert.Equal(t, "5", res.Message)
	})

	t.Run("get/set scroll-lines", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "scroll-lines 3")
		res := test.RunCmdArgs(t, km, e, "get_option", "scroll-lines")
		assert.Equal(t, "3", res.Message)
	})
}

func TestMotionGotoLineWithCount(t *testing.T) {
	t.Run("file start count", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\nl2\n")
		e.SetCount(2)
		test.RunCmd(t, km, e, "goto_line_or_file_start")
		assert.Equal(t, 1, test.CursorLine(t, e))
	})

	t.Run("extend file start count", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\nl2\n")
		e.SetCount(2)
		test.RunCmd(t, km, e, "goto_line_or_extend_file_start")
		assert.Equal(t, 1, test.CursorLine(t, e))
	})
}
