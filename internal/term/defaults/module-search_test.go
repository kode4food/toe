package defaults_test

import (
	"testing"

	"github.com/kode4food/toe/internal/core"
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
		setSelection(t, e, []core.Range{core.NewRange(0, 3)}, 0)
		runCmd(t, km, e, "search_selection")
	})

	t.Run("search_selection_word runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		runCmd(t, km, e, "search_selection_word")
	})

	t.Run("make_search_word_bounded runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		runCmd(t, km, e, "make_search_word_bounded")
	})
}
