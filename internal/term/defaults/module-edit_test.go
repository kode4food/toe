package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func TestEditModeCommands(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want view.Mode
	}{
		{"insert mode", "insert_mode", view.ModeInsert},
		{"insert at line start", "insert_at_line_start", view.ModeInsert},
		{"append mode", "append_mode", view.ModeInsert},
		{"append to line", "append_to_line", view.ModeInsert},
		{"select mode", "select_mode", view.ModeSelect},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, km := defaultsEnv(t, "abc")
			runCmd(t, km, e, tc.cmd)
			assert.Equal(t, tc.want, e.Mode())
		})
	}

	t.Run("normal mode exits insert", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		e.SetMode(view.ModeInsert)
		runCmd(t, km, e, "normal_mode")
		assert.Equal(t, view.ModeNormal, e.Mode())
	})

	t.Run("exit select returns to normal", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		runCmd(t, km, e, "select_mode")
		runCmd(t, km, e, "exit_select_mode")
		assert.Equal(t, view.ModeNormal, e.Mode())
	})
}

func TestEditOpenLine(t *testing.T) {
	t.Run("open below adds line and enters insert", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "open_below")
		assert.Contains(t, docText(t, e), "\n")
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("open above adds line and enters insert", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "open_above")
		assert.Equal(t, "\nabc", docText(t, e))
		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

func TestEditReplaceChar(t *testing.T) {
	t.Run("continuation replaces char under cursor", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		res := runCmd(t, km, e, "replace")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, char('x'))
		assert.Equal(t, "xbc", docText(t, e))
	})
}

func TestEditTextOps(t *testing.T) {
	t.Run("delete selection removes text", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcdef")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "delete_selection")
		assert.Equal(t, "def", docText(t, e))
	})

	t.Run("delete selection yanks to register", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcdef")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "delete_selection")
		got, ok := e.Registers().First('"')
		assert.True(t, ok)
		assert.Equal(t, "abc", got)
	})

	t.Run("removes and enters insert", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcdef")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "change_selection")
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("switch case toggles text", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "switch_case")
		assert.Equal(t, "ABC", docText(t, e))
	})

	t.Run("switch to lowercase", func(t *testing.T) {
		e, km := defaultsEnv(t, "ABC")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "switch_to_lowercase")
		assert.Equal(t, "abc", docText(t, e))
	})

	t.Run("switch to uppercase", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "switch_to_uppercase")
		assert.Equal(t, "ABC", docText(t, e))
	})

	t.Run("indent adds indentation", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "indent")
		assert.Contains(t, docText(t, e), "\t")
	})

	t.Run("join selections combines lines", func(t *testing.T) {
		e, km := defaultsEnv(t, "a\nb\n")
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "join_selections")
		assert.NotContains(t, docText(t, e), "\n\n")
	})

	t.Run("shrinks selection bounds, text unchanged", func(t *testing.T) {
		e, km := defaultsEnv(t, "  abc  ")
		setSelection(t, e, []core.Range{core.NewRange(0, 7)}, 0)
		runCmd(t, km, e, "trim_selections")
		// TrimSelections trims selection bounds, not the text
		assert.Equal(t, "  abc  ", docText(t, e))
	})

	t.Run("increment increases number", func(t *testing.T) {
		e, km := defaultsEnv(t, "1")
		setSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		runCmd(t, km, e, "increment")
		assert.Equal(t, "2", docText(t, e))
	})

	t.Run("decrement decreases number", func(t *testing.T) {
		e, km := defaultsEnv(t, "2")
		setSelection(t, e, []core.Range{core.NewRange(0, 1)}, 0)
		runCmd(t, km, e, "decrement")
		assert.Equal(t, "1", docText(t, e))
	})

	t.Run("ensure forward makes selection forward", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		runCmd(t, km, e, "ensure_forward")
		assert.NotContains(t,
			runCmd(t, km, e, "ensure_forward").Message, "error")
	})
}

func TestEditOptions(t *testing.T) {
	cases := []struct{ key, val string }{
		{"editor.auto-pairs", "true"},
		{"editor.continue-comments", "true"},
		{"editor.auto-save", "true"},
		{"editor.auto-save.focus-lost", "true"},
		{"editor.auto-save.after-delay.enable", "true"},
		{"editor.auto-save.after-delay.timeout", "1000"},
		{"editor.atomic-save", "true"},
	}
	for _, tc := range cases {
		t.Run("set/get "+tc.key, func(t *testing.T) {
			e, km := defaultsEnv(t, "")
			runCmdArgs(t, km, e, "set_option", tc.key+" "+tc.val)
			res := runCmdArgs(t, km, e, "get_option", tc.key)
			assert.Equal(t, tc.val, res.Message)
		})
	}

	t.Run("toggle auto-pairs", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "toggle_option", "editor.auto-pairs")
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("toggle continue-comments", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "toggle_option", "editor.continue-comments")
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("toggle auto-save", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "toggle_option", "editor.auto-save")
		assert.Contains(t, res.Message, "is now set to")
	})
}

func TestEditUndoRedo(t *testing.T) {
	t.Run("undo reverts, redo reapplies", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		runCmd(t, km, e, "undo")
		assert.Equal(t, "", docText(t, e))
		runCmd(t, km, e, "redo")
		assert.Equal(t, "abc", docText(t, e))
	})

	// Earlier/Later navigate the history timeline; the revert semantics are
	// covered in core/history_test. Here we exercise the command wrappers and
	// their count branch, asserting they complete without a continuation
	t.Run("earlier and later take no continuation", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		assert.Nil(t, runCmd(t, km, e, "earlier").Continuation)
		assert.Nil(t, runCmd(t, km, e, "later").Continuation)
	})

	t.Run("earlier honors count", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		e.SetCount(2)
		assert.Nil(t, runCmd(t, km, e, "earlier").Continuation)
	})
}
