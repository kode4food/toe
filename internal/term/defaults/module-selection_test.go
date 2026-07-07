package defaults_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
)

func TestSelectionSurround(t *testing.T) {
	t.Run("add wraps selection", func(t *testing.T) {
		e, km := defaultsEnv(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 4)}, 0)
		res := runCmd(t, km, e, "surround_add")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, char('('))
		got := docText(t, e)
		assert.Greater(t, len(got), len("hello"))
		assert.Contains(t, got, "(")
		assert.Contains(t, got, ")")
	})

	t.Run("delete removes pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := runCmd(t, km, e, "surround_delete")
		res.Continuation(e, char('('))
		assert.Equal(t, "abc", docText(t, e))
	})

	t.Run("replace swaps pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := runCmd(t, km, e, "surround_replace")
		next := res.Continuation(e, char('('))
		assert.NotNil(t, next)
		next(e, char('['))
		assert.Equal(t, "[abc]", docText(t, e))
	})
}

func TestSelectionTextObject(t *testing.T) {
	t.Run("inside selects within pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := runCmd(t, km, e, "select_textobject_inside")
		assert.NotNil(t, res.Continuation)
		assert.Nil(t, res.Continuation(e, char('(')))
	})

	t.Run("around selects with pair", func(t *testing.T) {
		e, km := defaultsEnv(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := runCmd(t, km, e, "select_textobject_around")
		assert.Nil(t, res.Continuation(e, char('(')))
	})
}

func TestSelectionSyntax(t *testing.T) {
	t.Run("expand selects syntax node", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := defaultsEnv(t, src)
		runCmdArgs(t, km, e, "set_language", "go")
		pos := strings.Index(src, "alpha") + 1
		testutil.SetCursor(t, e, pos)
		runCmd(t, km, e, "expand_selection")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, "alpha", got)
	})
}

func TestSelectionRegister(t *testing.T) {
	t.Run("select register takes a char", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		res := runCmd(t, km, e, "select_register")
		assert.NotNil(t, res.Continuation)
		assert.Nil(t, res.Continuation(e, char('a')))
	})
}
