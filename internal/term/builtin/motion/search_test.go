package motion_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/testutil"
)

const searchRegisterName = '/'

func TestSearch(t *testing.T) {
	t.Run("search_forward runs without panic", func(t *testing.T) {
		e, km := test.Env(t, "abcabc")
		test.RunCmd(t, km, e, "search_forward")
	})

	t.Run("search_backward runs without panic", func(t *testing.T) {
		e, km := test.Env(t, "abcabc")
		test.RunCmd(t, km, e, "search_backward")
	})

	t.Run("enter_command_mode runs without panic", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmd(t, km, e, "enter_command_mode")
	})

	t.Run("search_next runs without error", func(t *testing.T) {
		e, km := test.Env(t, "zero one two one")
		e.Registers().Set(searchRegisterName, `o\w+`)

		test.RunCmd(t, km, e, "search_next")

		assert.Equal(t, 5, testutil.CursorPos(t, e))
	})

	t.Run("search_prev runs without error", func(t *testing.T) {
		e, km := test.Env(t, "zero one two one")
		testutil.SetCursor(t, e, 16)
		e.Registers().Set(searchRegisterName, `o\w+`)

		test.RunCmd(t, km, e, "search_prev")

		assert.Equal(t, 13, testutil.CursorPos(t, e))
	})

	t.Run("search_next obeys no wrap", func(t *testing.T) {
		e, km := test.Env(t, "foo bar")
		e.Options().SearchWrapAround = false
		testutil.SetCursor(t, e, 6)
		e.Registers().Set(searchRegisterName, "foo")

		test.RunCmd(t, km, e, "search_next")

		assert.Equal(t, 6, testutil.CursorPos(t, e))
	})

	t.Run("search_prev wraps", func(t *testing.T) {
		e, km := test.Env(t, "foo bar")
		testutil.SetCursor(t, e, 0)
		e.Registers().Set(searchRegisterName, "bar")

		test.RunCmd(t, km, e, "search_prev")

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("search_selection runs", func(t *testing.T) {
		e, km := test.Env(t, "a.b")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		test.RunCmd(t, km, e, "search_selection")

		assert.Equal(t, `a\.b`,
			testutil.RegisteredValue(t, e, searchRegisterName))
	})

	t.Run("search_selection_word runs", func(t *testing.T) {
		e, km := test.Env(t, "foo bar")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)

		test.RunCmd(t, km, e, "search_selection_word")

		assert.Equal(t, `\b(?:foo)\b`,
			testutil.RegisteredValue(t, e, searchRegisterName))
	})

	t.Run("make_search_word_bounded runs", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		e.Registers().Set(searchRegisterName, "abc")

		test.RunCmd(t, km, e, "make_search_word_bounded")

		assert.Equal(t, `\babc\b`,
			testutil.RegisteredValue(t, e, searchRegisterName))
	})

	t.Run("make_search_word_bounded is idempotent", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		e.Registers().Set(searchRegisterName, `\babc\b`)

		test.RunCmd(t, km, e, "make_search_word_bounded")

		assert.Equal(t, `\babc\b`,
			testutil.RegisteredValue(t, e, searchRegisterName))
	})
}

func TestSearchOptions(t *testing.T) {
	cases := []struct{ key, val string }{
		{"search.smart-case", "false"},
		{"search.wrap-around", "false"},
	}
	for _, tc := range cases {
		t.Run("set/get "+tc.key, func(t *testing.T) {
			e, km := test.Env(t, "")
			test.RunCmdArgs(t, km, e, "set_option", tc.key+" "+tc.val)
			res := test.RunCmdArgs(t, km, e, "get_option", tc.key)
			assert.Equal(t, tc.val, res.Message)
		})
	}

	t.Run("toggle smart-case", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t,
			km, e, "toggle_option", "search.smart-case")
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("toggle wrap-around", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t,
			km, e, "toggle_option", "search.wrap-around")
		assert.Contains(t, res.Message, "is now set to")
	})
}
