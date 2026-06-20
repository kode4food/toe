package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
)

func TestMotionGotoLine(t *testing.T) {
	t.Run("goto line honors count", func(t *testing.T) {
		e, km := defaultsEnv(t, "l0\nl1\nl2\nl3\n")
		e.SetCount(2)
		runCmd(t, km, e, "goto_line")
		assert.Equal(t, 1, cursorLine(t, e))
	})

	t.Run("goto line or file start without count", func(t *testing.T) {
		e, km := defaultsEnv(t, "l0\nl1\nl2\n")
		setCursor(t, e, 5)
		runCmd(t, km, e, "goto_line_or_file_start")
		assert.Equal(t, 0, cursorPos(t, e))
	})

	t.Run("goto line or extend file start", func(t *testing.T) {
		e, km := defaultsEnv(t, "l0\nl1\nl2\n")
		setCursor(t, e, 5)
		runCmd(t, km, e, "goto_line_or_extend_file_start")
		assert.Equal(t, 0, cursorPos(t, e))
	})
}

func TestMotionFindChar(t *testing.T) {
	t.Run("find next char lands on target", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcdef")
		setCursor(t, e, 0)
		res := runCmd(t, km, e, "find_next_char")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, command.Char('c'))
		assert.Equal(t, 2, cursorPos(t, e))
	})

	t.Run("find till char stops before target", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcdef")
		setCursor(t, e, 0)
		res := runCmd(t, km, e, "find_till_char")
		res.Continuation(e, command.Char('c'))
		assert.Equal(t, 1, cursorPos(t, e))
	})

	t.Run("non-char key cancels find", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcdef")
		setCursor(t, e, 0)
		res := runCmd(t, km, e, "find_next_char")
		res.Continuation(e, command.Special("esc"))
		assert.Equal(t, 0, cursorPos(t, e))
	})
}

func TestMotionGotoFile(t *testing.T) {
	// no path under the cursor: the action reports an error via status and the
	// command completes without a continuation
	t.Run("no path under cursor is handled", func(t *testing.T) {
		e, km := defaultsEnv(t, "not a path")
		setCursor(t, e, 0)
		assert.Nil(t, runCmd(t, km, e, "goto_file").Continuation)
	})
}

func TestMotionParagraph(t *testing.T) {
	t.Run("next paragraph moves cursor down", func(t *testing.T) {
		e, km := defaultsEnv(t, "a\n\nb\n")
		setCursor(t, e, 0)
		runCmd(t, km, e, "goto_next_paragraph")
		assert.Greater(t, cursorPos(t, e), 0)
	})

	t.Run("prev paragraph runs from end", func(t *testing.T) {
		e, km := defaultsEnv(t, "a\n\nb\n")
		setCursor(t, e, 4)
		runCmd(t, km, e, "goto_prev_paragraph")
		assert.Less(t, cursorPos(t, e), 4)
	})
}
