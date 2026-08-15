package editing_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestInsertRegister(t *testing.T) {
	t.Run("insert register takes a char", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetCursor(t, e, 0)
		res := test.RunCmd(t, km, e, "insert_register")
		assert.NotNil(t, res.Continuation)
		// empty register pastes nothing; the continuation still completes
		_, got := res.Continuation(e, test.Char('a'))
		assert.Equal(t, command.ContinuationDone, got)
	})
}

func TestCompletionCommands(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  command.KeyEvent
	}{
		{name: "completion_previous", key: test.Special(command.Up)},
		{name: "completion_next", key: test.Special(command.Down)},
		{name: "completion_page_up", key: test.Special(command.PageUp)},
		{name: "completion_page_down", key: test.Special(command.PageDown)},
		{name: "completion_first", key: test.Special(command.Home)},
		{name: "completion_last", key: test.Special(command.End)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, km := test.Env(t, "")
			res := test.RunCmd(t, km, e, tt.name)
			assert.Empty(t, res.Message)

			lookup, ok := km.Lookup(
				view.ModeCompletion, []command.KeyEvent{tt.key},
			)
			assert.True(t, ok)
			assert.False(t, lookup.Prefix)
			assert.Nil(t, lookup.Action(e).Continuation)
		})
	}
}

func TestSmartTab(t *testing.T) {
	t.Run("indents in leading whitespace", func(t *testing.T) {
		e, km := test.Env(t, "\tabc")
		testutil.SetCursor(t, e, 1)
		test.RunCmd(t, km, e, "smart_tab")
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "\t\tabc", doc.Text().String())
	})

	t.Run("moves past enclosing node", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		pos := strings.Index(src, "alpha") + 1
		testutil.SetCursor(t, e, pos)
		test.RunCmd(t, km, e, "smart_tab")
		v := e.FocusedView()
		assert.NotNil(t, v)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		text := doc.Text()
		cursor := doc.SelectionFor(v.ID()).Primary().Cursor(text)
		assert.Equal(t, strings.Index(src, "alpha")+len("alpha"), cursor)
		assert.Equal(t, src, text.String())
	})

	t.Run("keeps text when already at node end", func(t *testing.T) {
		src := "package main\n\nfunc main() {\n\tprintln(alpha)\n}\n"
		e, km := test.Env(t, src)
		test.RunCmdArgs(t, km, e, "set_language", "go")
		testutil.SetCursor(t, e, strings.Index(src, "alpha")+len("alpha"))
		test.RunCmd(t, km, e, "smart_tab")
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, src, doc.Text().String())
	})

	t.Run("does nothing without a syntax tree", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetCursor(t, e, 3)
		test.RunCmd(t, km, e, "smart_tab")
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "abc", doc.Text().String())
	})
}

func TestInsertTab(t *testing.T) {
	t.Run("bound to shift+tab in insert", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		e.SetMode(view.ModeInsert)
		testutil.SetCursor(t, e, 3)
		lookup, ok := km.Lookup(view.ModeInsert, []command.KeyEvent{{
			Code: command.KeyCode{Special: command.Tab},
			Mods: command.ModShift,
		}})
		assert.True(t, ok)
		assert.False(t, lookup.Prefix)
		assert.Nil(t, lookup.Action(e).Continuation)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "abc\t", doc.Text().String())
	})

	t.Run("inserts at cursor", func(t *testing.T) {
		e, km := test.Env(t, "    abc")
		testutil.SetCursor(t, e, 7)
		res := test.RunCmd(t, km, e, "insert_tab")
		assert.Nil(t, res.Continuation)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "    abc\t", doc.Text().String())
	})
}
