package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
)

func TestSelectionSurround(t *testing.T) {
	t.Run("add wraps selection", func(t *testing.T) {
		e, km := defaultsEnv(t, "hello")
		setSelection(t, e, []core.Range{core.NewRange(0, 4)}, 0)
		res := runCmd(t, km, e, "surround_add")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, command.Char('('))
		got := docText(t, e)
		assert.Greater(t, len(got), len("hello"))
		assert.Contains(t, got, "(")
		assert.Contains(t, got, ")")
	})

	t.Run("delete removes pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		setCursor(t, e, 2)
		res := runCmd(t, km, e, "surround_delete")
		res.Continuation(e, command.Char('('))
		assert.Equal(t, "abc", docText(t, e))
	})

	t.Run("replace swaps pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		setCursor(t, e, 2)
		res := runCmd(t, km, e, "surround_replace")
		next := res.Continuation(e, command.Char('('))
		assert.NotNil(t, next)
		next(e, command.Char('['))
		assert.Equal(t, "[abc]", docText(t, e))
	})
}

func TestSelectionTextObject(t *testing.T) {
	t.Run("inside selects within pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		setCursor(t, e, 2)
		res := runCmd(t, km, e, "select_textobject_inside")
		assert.NotNil(t, res.Continuation)
		assert.Nil(t, res.Continuation(e, command.Char('(')))
	})

	t.Run("around selects with pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		setCursor(t, e, 2)
		res := runCmd(t, km, e, "select_textobject_around")
		assert.Nil(t, res.Continuation(e, command.Char('(')))
	})
}

func TestSelectionRegister(t *testing.T) {
	t.Run("select register takes a char", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		res := runCmd(t, km, e, "select_register")
		assert.NotNil(t, res.Continuation)
		assert.Nil(t, res.Continuation(e, command.Char('a')))
	})
}
