package config_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/builtin/test"
)

const configOptionBoolKey = "atomic-save"

func TestConfigOption(t *testing.T) {
	t.Run("set then get round-trips", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", configOptionBoolKey+" true")
		res := test.RunCmdArgs(t, km, e, "get_option", configOptionBoolKey)
		assert.Equal(t, "true", res.Message)
	})

	t.Run("toggle reports new value", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "toggle_option", configOptionBoolKey)
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("get without args is a usage error", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "get_option")
		assert.Contains(t, res.Message, "usage")
	})

	t.Run("get unknown key errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "get_option", "no_such_option")
		assert.Contains(t, res.Message, "error")
	})
}

func TestConfigTheme(t *testing.T) {
	t.Run("reports current theme without args", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "theme")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("sets a known theme with true color", func(t *testing.T) {
		t.Setenv("COLORTERM", "truecolor")
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "theme", "mocha")
		assert.NotContains(t, res.Message, "error")
		assert.Equal(t, "mocha", e.Options().Theme)
	})
}

func TestConfigDocumentOptions(t *testing.T) {
	t.Run("set language updates the buffer", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_language", "go")
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "go", doc.Lang())
	})

	t.Run("set language reports current", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "set_language")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("line ending accepts lf and crlf", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		assert.NotContains(t,
			test.RunCmdArgs(t, km, e, "set_line_ending", "crlf").Message, "error")
		assert.NotContains(t,
			test.RunCmdArgs(t, km, e, "set_line_ending", "lf").Message, "error")
	})

	t.Run("line ending rejects junk", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		res := test.RunCmdArgs(t, km, e, "set_line_ending", "bogus")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("indent style accepts tabs and spaces", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Contains(t,
			test.RunCmdArgs(t, km, e, "indent_style", "t").Message, "indent")
		assert.Contains(t,
			test.RunCmdArgs(t, km, e, "indent_style", "4").Message, "indent")
	})

	t.Run("indent style rejects out-of-range", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "indent_style", "99")
		assert.Contains(t, strings.ToLower(res.Message), "error")
	})

	t.Run("set language no document errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmd(t, km, e, "set_language")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("line ending no document errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmd(t, km, e, "set_line_ending")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("indent style no document errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmd(t, km, e, "indent_style")
		assert.Contains(t, res.Message, "error")
	})
}

func TestConfigOptions(t *testing.T) {
	boolCases := []string{
		"mouse",
		"middle-click-paste",
		"insecure",
		"editor-config",
		"auto-session",
	}
	for _, key := range boolCases {
		t.Run("toggle "+key, func(t *testing.T) {
			e, km := test.Env(t, "")
			res := test.RunCmdArgs(t, km, e, "toggle_option", key)
			assert.Contains(t, res.Message, "is now set to")
		})
	}

	t.Run("get/set default-line-ending", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "default-line-ending lf")
		res := test.RunCmdArgs(t, km, e, "get_option", "default-line-ending")
		assert.Equal(t, "lf", res.Message)
	})

	t.Run("get/set statusline separator", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "statusline.separator |")
		res := test.RunCmdArgs(t, km, e, "get_option", "statusline.separator")
		assert.Equal(t, "|", res.Message)
	})

	t.Run("get/set statusline mode names", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", `statusline.mode.normal NOR`)
		res := test.RunCmdArgs(t,
			km, e, "get_option", "statusline.mode.normal")
		assert.Equal(t, "NOR", res.Message)
	})

	t.Run("get/set statusline mode insert", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "statusline.mode.insert INS")
		res := test.RunCmdArgs(t,
			km, e, "get_option", "statusline.mode.insert")
		assert.Equal(t, "INS", res.Message)
	})

	t.Run("get/set statusline mode select", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "statusline.mode.select SEL")
		res := test.RunCmdArgs(t,
			km, e, "get_option", "statusline.mode.select")
		assert.Equal(t, "SEL", res.Message)
	})

	t.Run("get/set cursor-shape normal", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "cursor-shape.normal bar")
		res := test.RunCmdArgs(t, km, e, "get_option", "cursor-shape.normal")
		assert.Equal(t, "bar", res.Message)
	})

	t.Run("get/set cursor-shape select", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t,
			km, e, "set_option", "cursor-shape.select underline")
		res := test.RunCmdArgs(t, km, e, "get_option", "cursor-shape.select")
		assert.Equal(t, "underline", res.Message)
	})

	t.Run("get/set cursor-shape insert", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "cursor-shape.insert bar")
		res := test.RunCmdArgs(t, km, e, "get_option", "cursor-shape.insert")
		assert.Equal(t, "bar", res.Message)
	})

	t.Run("get/set theme option", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "theme mocha")
		res := test.RunCmdArgs(t, km, e, "get_option", "theme")
		assert.Equal(t, "mocha", res.Message)
	})

	t.Run("get/set default-line-ending crlf", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", "default-line-ending crlf")
		res := test.RunCmdArgs(t, km, e, "get_option", "default-line-ending")
		assert.Equal(t, "crlf", res.Message)
	})
}

func TestConfigOptionErrors(t *testing.T) {
	t.Run("set_option nil args is usage error", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "set_option")
		assert.Contains(t, res.Message, "usage")
	})

	t.Run("set_option unknown key errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "set_option", "no_such_option true")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("toggle_option nil args is usage error", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "toggle_option")
		assert.Contains(t, res.Message, "usage")
	})
}

func TestConfigThemeErrors(t *testing.T) {
	t.Run("unknown name errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "theme", "no-such-theme-xyz")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("invalid current falls back to default", func(t *testing.T) {
		t.Setenv("COLORTERM", "truecolor")
		e, km := test.Env(t, "")
		e.Options().Theme = "no-such-theme-xyz"
		res := test.RunCmd(t, km, e, "theme")
		assert.NotContains(t, res.Message, "error")
	})
}

func TestConfigThemeExtra(t *testing.T) {
	t.Run("theme default alias loads mocha", func(t *testing.T) {
		t.Setenv("COLORTERM", "truecolor")
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "theme", "default")
		assert.NotContains(t, res.Message, "error")
		assert.Equal(t, "mocha", e.Options().Theme)
	})

	t.Run("set_language text sets empty lang", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_language", "text")
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "", doc.Lang())
	})

	t.Run("cursor-shape invalid value errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(
			t, km, e, "set_option", "cursor-shape.normal bogus_invalid",
		)
		assert.Contains(t, res.Message, "error")
	})

	t.Run("get default-line-ending empty returns empty", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "get_option", "default-line-ending")
		assert.NotContains(t, res.Message, "error")
	})
}

func TestConfigCommands(t *testing.T) {
	t.Run("toggle non-toggleable option errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(
			t, km, e, "toggle_option", "default-line-ending",
		)
		assert.Contains(t, res.Message, "error")
	})

	t.Run("line ending no args on crlf doc shows crlf", func(t *testing.T) {
		e, km := test.Env(t, "abc\r\n")
		test.RunCmdArgs(t, km, e, "set_line_ending", "crlf")
		res := test.RunCmd(t, km, e, "set_line_ending")
		assert.Equal(t, "crlf", res.Message)
	})

	t.Run("encoding always returns utf-8", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "encoding")
		assert.Equal(t, "utf-8", res.Message)
	})

	t.Run("config_reload no fn errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "config_reload")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("config_reload fn set succeeds", func(t *testing.T) {
		e, km := test.Env(t, "")
		e.SetConfigReload(func() error { return nil })
		res := test.RunCmd(t, km, e, "config_reload")
		assert.Equal(t, "config reloaded", res.Message)
	})

	t.Run("config_open runs without panic", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e, km := test.Env(t, "")
		test.RunCmd(t, km, e, "config_open")
	})

	t.Run("config_open_workspace rejects untrusted", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "config_open_workspace")
		assert.Contains(t, res.Message, "workspace untrusted")
	})

	t.Run("config_open_workspace opens trusted", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := test.Env(t, "")
		assert.NoError(t, loader.TrustWorkspace(e.Cwd()))
		res := test.RunCmd(t, km, e, "config_open_workspace")
		assert.NotContains(t, res.Message, "error")
	})

	t.Run("log_open runs without panic", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := test.Env(t, "")
		test.RunCmd(t, km, e, "log_open")
	})

	t.Run("workspace_trust runs", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "workspace_trust")
		assert.NotContains(t, res.Message, "error")
	})

	t.Run("workspace_untrust runs", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "workspace_untrust")
		assert.NotContains(t, res.Message, "error")
	})

	t.Run("line ending no args shows current", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		res := test.RunCmd(t, km, e, "set_line_ending")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("indent_style without args shows current", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "indent_style")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("config_open no home returns error", func(t *testing.T) {
		t.Setenv("HOME", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "config_open")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("log_open no home returns error", func(t *testing.T) {
		t.Setenv("HOME", "")
		t.Setenv("XDG_DATA_HOME", "")
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "log_open")
		assert.Contains(t, res.Message, "error")
	})
}
