package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
)

func TestSupportQuit(t *testing.T) {
	t.Run("quit clean signals quit", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Equal(t, command.SignalQuit, runCmd(t, km, e, "quit").Signal)
	})

	t.Run("quit dirty warns", func(t *testing.T) {
		e, km := defaultsEnv(t, "x")
		assert.Contains(t, runCmd(t, km, e, "quit").Message, "unsaved")
	})

	t.Run("quit all clean signals quit", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Equal(t, command.SignalQuit, runCmd(t, km, e, "quit_all").Signal)
	})

	t.Run("redraw signals clear screen", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Equal(t,
			command.SignalClearScreen, runCmd(t, km, e, "redraw").Signal)
	})
}

func TestSupportEchoInfo(t *testing.T) {
	t.Run("echo returns joined args", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Equal(t, "hello world",
			runCmdArgs(t, km, e, "echo", "hello world").Message)
	})

	t.Run("character info describes cursor char", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		assert.NotEmpty(t, runCmd(t, km, e, "character_info").Message)
	})

	t.Run("no formatter for plain text", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		assert.Contains(t, runCmd(t, km, e, "format").Message, "no formatter")
	})
}

func TestSupportGoto(t *testing.T) {
	t.Run("goto moves to the line", func(t *testing.T) {
		e, km := defaultsEnv(t, "l0\nl1\nl2\n")
		res := runCmdArgs(t, km, e, "goto", "2")
		assert.NotContains(t, res.Message, "error")
		assert.Equal(t, 1, cursorLine(t, e))
	})

	t.Run("goto rejects junk", func(t *testing.T) {
		e, km := defaultsEnv(t, "l0\nl1\n")
		assert.Contains(t, runCmdArgs(t, km, e, "goto", "x").Message, "invalid")
	})

	t.Run("goto without args errors", func(t *testing.T) {
		e, km := defaultsEnv(t, "l0\n")
		assert.Contains(t, runCmd(t, km, e, "goto").Message, "no line number")
	})
}

func TestSupportSelectionOps(t *testing.T) {
	t.Run("sort orders multiple selections", func(t *testing.T) {
		// sort reorders the contents of multiple selections among themselves
		e, km := defaultsEnv(t, "b\na\n")
		setSelection(t, e,
			[]core.Range{core.NewRange(0, 1), core.NewRange(2, 3)}, 0)
		assert.NotContains(t, runCmd(t, km, e, "sort").Message, "error")
		assert.Equal(t, "a\nb\n", docText(t, e))
	})

	t.Run("reflow runs over a selection", func(t *testing.T) {
		e, km := defaultsEnv(t, "one two three four five\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 20)}, 0)
		assert.NotContains(t,
			runCmdArgs(t, km, e, "reflow", "10").Message, "error")
	})

	t.Run("reflow uses configured text width", func(t *testing.T) {
		e, km := defaultsEnv(t, "one two three four five\n")
		width := 10
		e.Options().TextWidth = &width
		setSelection(t, e, []core.Range{core.NewRange(0, 20)}, 0)
		assert.Empty(t, runCmd(t, km, e, "reflow").Message)
	})

	t.Run("reflow rejects bad width", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\n")
		assert.Contains(t, runCmdArgs(t, km, e, "reflow", "0").Message, "error")
	})

	t.Run("toggle comments runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		assert.Empty(t, runCmd(t, km, e, "toggle_comments").Message)
	})
}

func TestSupportEchoNilArgs(t *testing.T) {
	t.Run("echo with nil args returns empty", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "echo")
		assert.Empty(t, res.Message)
	})
}

func TestSupportTutor(t *testing.T) {
	t.Run("tutor runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmd(t, km, e, "tutor")
	})
}

func TestSupportPickerCommands(t *testing.T) {
	// each opens an overlay layer on the model and completes without chaining
	for _, name := range []string{
		"file_picker", "file_picker_in_current_dir", "file_explorer",
		"file_explorer_in_current_buffer_directory", "buffer_picker",
		"jumplist_picker", "global_search", "command_palette", "last_picker",
	} {
		t.Run(name+" opens without chaining", func(t *testing.T) {
			e, km := defaultsEnv(t, "")
			assert.Nil(t, runCmd(t, km, e, name).Continuation)
		})
	}
}
