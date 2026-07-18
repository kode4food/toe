package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
)

func TestSupportQuit(t *testing.T) {
	t.Run("quit clean signals quit", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t, command.SignalQuit, test.RunCmd(t, km, e, "quit").Signal)
	})

	t.Run("quit dirty warns", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Contains(t, test.RunCmd(t, km, e, "quit").Message, "unsaved")
	})

	t.Run("quit all clean signals quit", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t, command.SignalQuit, test.RunCmd(t, km, e, "quit_all").Signal)
	})

	t.Run("redraw signals clear screen", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t,
			command.SignalClearScreen, test.RunCmd(t, km, e, "redraw").Signal)
	})
}

func TestSupportEchoInfo(t *testing.T) {
	t.Run("echo returns joined args", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t, "hello world",
			test.RunCmdArgs(t, km, e, "echo", "hello world").Message)
	})

	t.Run("character info describes cursor char", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetCursor(t, e, 0)
		assert.NotEmpty(t, test.RunCmd(t, km, e, "character_info").Message)
	})

	t.Run("no formatter for plain text", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		assert.Contains(t, test.RunCmd(t, km, e, "format").Message, "no formatter")
	})
}

func TestSupportGoto(t *testing.T) {
	t.Run("goto moves to the line", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\nl2\n")
		res := test.RunCmdArgs(t, km, e, "goto", "2")
		assert.NotContains(t, res.Message, "error")
		assert.Equal(t, 1, test.CursorLine(t, e))
	})

	t.Run("goto rejects junk", func(t *testing.T) {
		e, km := test.Env(t, "l0\nl1\n")
		assert.Contains(t, test.RunCmdArgs(t, km, e, "goto", "x").Message, "invalid")
	})

	t.Run("goto without args errors", func(t *testing.T) {
		e, km := test.Env(t, "l0\n")
		assert.Contains(t, test.RunCmd(t, km, e, "goto").Message, "no line number")
	})
}

func TestSupportSelectionOps(t *testing.T) {
	t.Run("sort orders multiple selections", func(t *testing.T) {
		// sort reorders the contents of multiple selections among themselves
		e, km := test.Env(t, "b\na\n")
		testutil.SetSelection(t, e,
			[]core.Range{core.NewRange(0, 1), core.NewRange(2, 3)}, 0)
		assert.NotContains(t, test.RunCmd(t, km, e, "sort").Message, "error")
		assert.Equal(t, "a\nb\n", test.DocText(t, e))
	})

	t.Run("reflow runs over a selection", func(t *testing.T) {
		e, km := test.Env(t, "one two three four five\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 20)}, 0)
		assert.NotContains(t,
			test.RunCmdArgs(t, km, e, "reflow", "10").Message, "error")
	})

	t.Run("reflow uses configured text width", func(t *testing.T) {
		e, km := test.Env(t, "one two three four five\n")
		e.Options().TextWidth = new(10)
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 20)}, 0)
		assert.Empty(t, test.RunCmd(t, km, e, "reflow").Message)
	})

	t.Run("reflow rejects bad width", func(t *testing.T) {
		e, km := test.Env(t, "abc\n")
		assert.Contains(t, test.RunCmdArgs(t, km, e, "reflow", "0").Message, "error")
	})

	t.Run("toggle comments runs", func(t *testing.T) {
		e, km := test.Env(t, "abc\n")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		assert.Empty(t, test.RunCmd(t, km, e, "toggle_comments").Message)
	})
}

func TestSupportEchoNilArgs(t *testing.T) {
	t.Run("echo with nil args returns empty", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "echo")
		assert.Empty(t, res.Message)
	})
}

func TestSupportPickerCommands(t *testing.T) {
	// each opens an overlay layer on the model and completes without chaining
	for _, name := range []string{
		"file_picker", "file_picker_in_current_dir", "file_explorer",
		"file_explorer_in_current_pane_dir", "buffer_picker",
		"jumplist_picker", "global_search", "command_palette", "last_picker",
	} {
		t.Run(name+" opens without chaining", func(t *testing.T) {
			e, km := test.Env(t, "")
			assert.Nil(t, test.RunCmd(t, km, e, name).Continuation)
		})
	}
}
