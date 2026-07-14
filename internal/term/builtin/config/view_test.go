package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/testutil"
)

func TestViewScrollCommands(t *testing.T) {
	for _, name := range []string{
		"page_up", "page_down",
		"page_cursor_half_up", "page_cursor_half_down",
		"half_page_up", "half_page_down",
		"page_cursor_up", "page_cursor_down",
		"center_cursor_line", "align_view_top", "align_view_bottom",
		"scroll_up", "scroll_down",
	} {
		t.Run(name+" runs without error", func(t *testing.T) {
			e, km := test.Env(t, "l0\nl1\nl2\nl3\nl4\n")
			testutil.SetCursor(t, e, 5)
			res := test.RunCmd(t, km, e, name)
			assert.Empty(t, res.Message)
		})
	}
}

func TestViewSplit(t *testing.T) {
	t.Run("vsplit opens second view", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "vsplit")
		assert.Equal(t, before+1, len(e.AllViews()))
	})

	t.Run("split opens second view", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "split")
		assert.Equal(t, before+1, len(e.AllViews()))
	})

	t.Run("vsplit_new opens a new empty view", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "vsplit_new")
		assert.Equal(t, before+1, len(e.AllViews()))
	})

	t.Run("hsplit_new opens a new empty view", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "hsplit_new")
		assert.Equal(t, before+1, len(e.AllViews()))
	})

	t.Run("wclose reduces view count", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmd(t, km, e, "vsplit")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "wclose")
		assert.Equal(t, before-1, len(e.AllViews()))
	})

	t.Run("wclose! reduces view count", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		test.RunCmd(t, km, e, "vsplit")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "wclose!")
		assert.Equal(t, before-1, len(e.AllViews()))
	})
}

func TestViewNavigation(t *testing.T) {
	for _, name := range []string{
		"jump_view_left", "jump_view_right",
		"jump_view_up", "jump_view_down",
		"rotate_view", "transpose_view",
	} {
		t.Run(name+" runs without error", func(t *testing.T) {
			e, km := test.Env(t, "abc")
			res := test.RunCmd(t, km, e, name)
			assert.Empty(t, res.Message)
		})
	}
}

func TestViewOptions(t *testing.T) {
	cases := []struct{ key, val string }{
		{"line-number", "absolute"},
		{"cursorline", "true"},
		{"cursorcolumn", "true"},
		{"text-width", "72"},
		{"soft-wrap.enable", "true"},
		{"soft-wrap.max-wrap", "10"},
		{"soft-wrap.max-indent-retain", "20"},
		{"soft-wrap.wrap-at-text-width", "true"},
		{"bufferline", "always"},
	}
	for _, tc := range cases {
		t.Run("set/get "+tc.key, func(t *testing.T) {
			e, km := test.Env(t, "")
			test.RunCmdArgs(t, km, e, "set_option", tc.key+" "+tc.val)
			res := test.RunCmdArgs(t, km, e, "get_option", tc.key)
			assert.Equal(t, tc.val, res.Message)
		})
	}

	t.Run("toggle cursorline", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "toggle_option", "cursorline")
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("toggle soft-wrap.enable", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "toggle_option", "soft-wrap.enable")
		assert.Contains(t, res.Message, "is now set to")
	})
}

func TestViewWonly(t *testing.T) {
	t.Run("wonly closes other views", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		test.RunCmd(t, km, e, "vsplit")
		test.RunCmd(t, km, e, "wonly")
		assert.Equal(t, 1, len(e.AllViews()))
	})
}

func TestViewOptionsExtra(t *testing.T) {
	cases := []struct{ key, val string }{
		{"rulers", "[80, 100]"},
		{"whitespace.render", "all"},
		{"indent-guides.render", "true"},
		{"indent-guides.skip-levels", "2"},
		{"indent-guides.character", `"│"`},
		{"gutters.line-numbers.min-width", "3"},
		{"soft-wrap.wrap-indicator", `"↩"`},
	}
	for _, tc := range cases {
		t.Run("set/get "+tc.key, func(t *testing.T) {
			e, km := test.Env(t, "")
			test.RunCmdArgs(t, km, e, "set_option", tc.key+" "+tc.val)
			res := test.RunCmdArgs(t, km, e, "get_option", tc.key)
			assert.NotEmpty(t, res)
		})
	}

	t.Run("toggle indent-guides.render", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t,
			km, e, "toggle_option", "indent-guides.render")
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("toggle soft-wrap.wrap-at-text-width", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(
			t, km, e, "toggle_option", "soft-wrap.wrap-at-text-width",
		)
		assert.Contains(t, res.Message, "is now set to")
	})
}

func TestNullableIntDefault(t *testing.T) {
	t.Run("get text-width returns default unset", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "get_option", "text-width")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("set invalid text-width reports error", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "set_option", "text-width bad")
		assert.NotEmpty(t, res.Message)
	})
}

func TestViewSwapCommands(t *testing.T) {
	for _, name := range []string{
		"swap_view_left", "swap_view_right",
		"swap_view_up", "swap_view_down",
	} {
		t.Run(name+" runs without panic", func(t *testing.T) {
			e, km := test.Env(t, "abc")
			test.RunCmd(t, km, e, name)
		})
	}
}
