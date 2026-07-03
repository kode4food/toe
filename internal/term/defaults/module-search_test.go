package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
)

func TestSearch(t *testing.T) {
	t.Run("search_forward runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcabc")
		runCmd(t, km, e, "search_forward")
	})

	t.Run("search_backward runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "abcabc")
		runCmd(t, km, e, "search_backward")
	})

	t.Run("enter_command_mode runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmd(t, km, e, "enter_command_mode")
	})

	t.Run("search_next runs without error", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		res := runCmd(t, km, e, "search_next")
		_ = res
	})

	t.Run("search_prev runs without error", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		res := runCmd(t, km, e, "search_prev")
		_ = res
	})

	t.Run("search_selection runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "search_selection")
	})

	t.Run("search_selection_word runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		testutil.SetCursor(t, e, 0)
		runCmd(t, km, e, "search_selection_word")
	})

	t.Run("make_search_word_bounded runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		runCmd(t, km, e, "make_search_word_bounded")
	})
}

func TestSearchOptions(t *testing.T) {
	cases := []struct{ key, val string }{
		{"editor.search.smart-case", "false"},
		{"editor.search.wrap-around", "false"},
	}
	for _, tc := range cases {
		t.Run("set/get "+tc.key, func(t *testing.T) {
			e, km := defaultsEnv(t, "")
			runCmdArgs(t, km, e, "set_option", tc.key+" "+tc.val)
			res := runCmdArgs(t, km, e, "get_option", tc.key)
			assert.Equal(t, tc.val, res.Message)
		})
	}

	t.Run("toggle smart-case", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t,
			km, e, "toggle_option", "editor.search.smart-case")
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("toggle wrap-around", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t,
			km, e, "toggle_option", "editor.search.wrap-around")
		assert.Contains(t, res.Message, "is now set to")
	})
}
