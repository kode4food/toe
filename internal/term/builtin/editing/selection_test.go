package editing_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/testutil"
)

func TestSelectionSurround(t *testing.T) {
	t.Run("add wraps selection", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 4)}, 0)
		res := test.RunCmd(t, km, e, "surround_add")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('('))
		got := test.DocText(t, e)
		assert.Greater(t, len(got), len("hello"))
		assert.Contains(t, got, "(")
		assert.Contains(t, got, ")")
	})

	t.Run("delete removes pair", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := test.RunCmd(t, km, e, "surround_delete")
		res.Continuation(e, test.Char('('))
		assert.Equal(t, "abc", test.DocText(t, e))
	})

	t.Run("replace swaps pair", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := test.RunCmd(t, km, e, "surround_replace")
		next := res.Continuation(e, test.Char('('))
		assert.NotNil(t, next)
		next(e, test.Char('['))
		assert.Equal(t, "[abc]", test.DocText(t, e))
	})

	t.Run("delete removes pair (tree-sitter)", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		cursor := strings.Index(src, "alpha") + 2
		testutil.SetCursor(t, e, cursor)
		res := test.RunCmd(t, km, e, "surround_delete")
		res.Continuation(e, test.Char('('))
		assert.NotContains(t, test.DocText(t, e), "println(alpha)")
	})

	t.Run("replace swaps pair (tree-sitter)", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		cursor := strings.Index(src, "alpha") + 2
		testutil.SetCursor(t, e, cursor)
		res := test.RunCmd(t, km, e, "surround_replace")
		next := res.Continuation(e, test.Char('('))
		assert.NotNil(t, next)
		next(e, test.Char('['))
		assert.Contains(t, test.DocText(t, e), "println[alpha]")
	})
}

func TestSelectionTextObjectSyntax(t *testing.T) {
	src := "package main\n\nfunc foo(x int) {\n\tprintln(x)\n}\n"

	t.Run("inside function selects body", func(t *testing.T) {
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		cursor := strings.Index(src, "println")
		testutil.SetCursor(t, e, cursor)
		res := test.RunCmd(t, km, e, "select_textobject_inner")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('f'))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Contains(t, got, "println")
		assert.NotContains(t, got, "func foo")
	})

	t.Run("around function selects declaration", func(t *testing.T) {
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		cursor := strings.Index(src, "println")
		testutil.SetCursor(t, e, cursor)
		res := test.RunCmd(t, km, e, "select_textobject_around")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('f'))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Contains(t, got, "func foo")
		assert.Contains(t, got, "println")
	})

	t.Run("plaintext fallback for bracket objects", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := test.RunCmd(t, km, e, "select_textobject_inner")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('('))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, "abc", got)
	})
}

func TestSelectionTextObject(t *testing.T) {
	t.Run("inside selects within pair", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := test.RunCmd(t, km, e, "select_textobject_inner")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('('))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, "abc", got)
	})

	t.Run("around selects with pair", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		res := test.RunCmd(t, km, e, "select_textobject_around")
		assert.NotNil(t, res.Continuation)
		res.Continuation(e, test.Char('('))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, "(abc)", got)
	})
}

func TestSelectionMatchBrackets(t *testing.T) {
	t.Run("jumps to closing bracket (plaintext)", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 0)
		test.RunCmd(t, km, e, "match_brackets")
		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("jumps to opening bracket (plaintext)", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 4)
		test.RunCmd(t, km, e, "match_brackets")
		assert.Equal(t, 0, testutil.CursorPos(t, e))
	})

	t.Run("noop when not on bracket", func(t *testing.T) {
		e, km := test.Env(t, "(abc)")
		testutil.SetCursor(t, e, 2)
		test.RunCmd(t, km, e, "match_brackets")
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("jumps to matching bracket (tree-sitter)", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		openPos := strings.Index(src, "(alpha)")
		closePos := openPos + len("(alpha)") - 1
		testutil.SetCursor(t, e, openPos)
		test.RunCmd(t, km, e, "match_brackets")
		assert.Equal(t, closePos, testutil.CursorPos(t, e))
	})

	t.Run("bracket in string not matched (tree-sitter)", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n" +
			"\tx := \"(foo)\"\n\t_ = x\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		inStr := strings.Index(src, "(foo)") + 2
		testutil.SetCursor(t, e, inStr)
		test.RunCmd(t, km, e, "match_brackets")
		assert.Equal(t, inStr, testutil.CursorPos(t, e))
	})
}

func TestSelectionSyntax(t *testing.T) {
	t.Run("expand selects syntax node", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		pos := strings.Index(src, "alpha") + 1
		testutil.SetCursor(t, e, pos)
		test.RunCmd(t, km, e, "expand_selection")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, "alpha", got)
	})

	t.Run("shrink selects child syntax node", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		from := strings.Index(src, "func main")
		to := strings.Index(src, "}\n") + 1
		testutil.SetSelection(t, e, []core.Range{core.NewRange(from, to)}, 0)
		test.RunCmd(t, km, e, "shrink_selection")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		got, err := doc.SelectionFor(v.ID()).Primary().Fragment(doc.Text())
		assert.NoError(t, err)
		assert.Contains(t, got, "println")
	})
}

func TestSelectionRegister(t *testing.T) {
	t.Run("select register takes a char", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		res := test.RunCmd(t, km, e, "select_register")
		assert.NotNil(t, res.Continuation)
		assert.Nil(t, res.Continuation(e, test.Char('a')))
	})
}
