package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
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
		res.Continuation(e, command.Char('x'))
		assert.Equal(t, "xbc", docText(t, e))
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
