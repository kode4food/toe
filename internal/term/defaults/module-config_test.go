package defaults_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const configOptionBoolKey = "editor.atomic-save"

func TestConfigOption(t *testing.T) {
	t.Run("set then get round-trips", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", configOptionBoolKey+" true")
		res := runCmdArgs(t, km, e, "get_option", configOptionBoolKey)
		assert.Equal(t, "true", res.Message)
	})

	t.Run("toggle reports new value", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "toggle_option", configOptionBoolKey)
		assert.Contains(t, res.Message, "is now set to")
	})

	t.Run("get without args is a usage error", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "get_option")
		assert.Contains(t, res.Message, "usage")
	})

	t.Run("get unknown key errors", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "get_option", "no_such_option")
		assert.Contains(t, res.Message, "error")
	})
}

func TestConfigTheme(t *testing.T) {
	t.Run("reports current theme without args", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "theme")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("sets a known theme with true color", func(t *testing.T) {
		t.Setenv("COLORTERM", "truecolor")
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "theme", "mocha")
		assert.NotContains(t, res.Message, "error")
		assert.Equal(t, "mocha", e.Options().Theme)
	})
}

func TestConfigDocumentOptions(t *testing.T) {
	t.Run("set language updates the buffer", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_language", "go")
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "go", doc.Lang())
	})

	t.Run("set language reports current", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "set_language")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("line ending accepts lf and crlf", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		assert.NotContains(t,
			runCmdArgs(t, km, e, "set_line_ending", "crlf").Message, "error")
		assert.NotContains(t,
			runCmdArgs(t, km, e, "set_line_ending", "lf").Message, "error")
	})

	t.Run("line ending rejects junk", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		res := runCmdArgs(t, km, e, "set_line_ending", "bogus")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("indent style accepts tabs and spaces", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Contains(t,
			runCmdArgs(t, km, e, "indent_style", "t").Message, "indent")
		assert.Contains(t,
			runCmdArgs(t, km, e, "indent_style", "4").Message, "indent")
	})

	t.Run("indent style rejects out-of-range", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "indent_style", "99")
		assert.Contains(t, strings.ToLower(res.Message), "error")
	})
}

func TestConfigOptions(t *testing.T) {
	boolCases := []string{
		"editor.mouse",
		"editor.middle-click-paste",
		"editor.insecure",
		"editor.editor-config",
	}
	for _, key := range boolCases {
		t.Run("toggle "+key, func(t *testing.T) {
			e, km := defaultsEnv(t, "")
			res := runCmdArgs(t, km, e, "toggle_option", key)
			assert.Contains(t, res.Message, "is now set to")
		})
	}

	t.Run("get/set default-line-ending", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.default-line-ending lf")
		res := runCmdArgs(t, km, e, "get_option", "editor.default-line-ending")
		assert.Equal(t, "lf", res.Message)
	})

	t.Run("get/set statusline separator", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.statusline.separator |")
		res := runCmdArgs(t, km, e, "get_option", "editor.statusline.separator")
		assert.Equal(t, "|", res.Message)
	})

	t.Run("get/set statusline mode names", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", `editor.statusline.mode.normal NOR`)
		res := runCmdArgs(t,
			km, e, "get_option", "editor.statusline.mode.normal")
		assert.Equal(t, "NOR", res.Message)
	})

	t.Run("get/set statusline mode insert", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.statusline.mode.insert INS")
		res := runCmdArgs(t,
			km, e, "get_option", "editor.statusline.mode.insert")
		assert.Equal(t, "INS", res.Message)
	})

	t.Run("get/set statusline mode select", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.statusline.mode.select SEL")
		res := runCmdArgs(t,
			km, e, "get_option", "editor.statusline.mode.select")
		assert.Equal(t, "SEL", res.Message)
	})

	t.Run("get/set cursor-shape normal", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.cursor-shape.normal bar")
		res := runCmdArgs(t, km, e, "get_option", "editor.cursor-shape.normal")
		assert.Equal(t, "bar", res.Message)
	})

	t.Run("get/set cursor-shape select", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t,
			km, e, "set_option", "editor.cursor-shape.select underline")
		res := runCmdArgs(t, km, e, "get_option", "editor.cursor-shape.select")
		assert.Equal(t, "underline", res.Message)
	})

	t.Run("get/set cursor-shape insert", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.cursor-shape.insert bar")
		res := runCmdArgs(t, km, e, "get_option", "editor.cursor-shape.insert")
		assert.Equal(t, "bar", res.Message)
	})

	t.Run("get/set theme option", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "theme mocha")
		res := runCmdArgs(t, km, e, "get_option", "theme")
		assert.Equal(t, "mocha", res.Message)
	})

	t.Run("get/set default-line-ending crlf", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", "editor.default-line-ending crlf")
		res := runCmdArgs(t, km, e, "get_option", "editor.default-line-ending")
		assert.Equal(t, "crlf", res.Message)
	})
}

func TestConfigOptionErrors(t *testing.T) {
	t.Run("set_option nil args is usage error", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "set_option")
		assert.Contains(t, res.Message, "usage")
	})

	t.Run("set_option unknown key errors", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "set_option", "no_such_option true")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("toggle_option nil args is usage error", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "toggle_option")
		assert.Contains(t, res.Message, "usage")
	})
}

func TestConfigThemeExtra(t *testing.T) {
	t.Run("theme default alias loads mocha", func(t *testing.T) {
		t.Setenv("COLORTERM", "truecolor")
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "theme", "default")
		assert.NotContains(t, res.Message, "error")
		assert.Equal(t, "mocha", e.Options().Theme)
	})

	t.Run("set_language text sets empty lang", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_language", "text")
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "", doc.Lang())
	})

	t.Run("cursor-shape invalid value errors", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(
			t, km, e, "set_option", "editor.cursor-shape.normal bogus_invalid",
		)
		assert.Contains(t, res.Message, "error")
	})

	t.Run("get default-line-ending empty returns empty", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "get_option", "editor.default-line-ending")
		assert.NotContains(t, res.Message, "error")
	})
}

func TestConfigCommands(t *testing.T) {
	t.Run("toggle non-toggleable option errors", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(
			t, km, e, "toggle_option", "editor.default-line-ending",
		)
		assert.Contains(t, res.Message, "error")
	})

	t.Run("line ending no args on crlf doc shows crlf", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc\r\n")
		runCmdArgs(t, km, e, "set_line_ending", "crlf")
		res := runCmd(t, km, e, "set_line_ending")
		assert.Equal(t, "crlf", res.Message)
	})

	t.Run("encoding always returns utf-8", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "encoding")
		assert.Equal(t, "utf-8", res.Message)
	})

	t.Run("config_reload errors without reload fn", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "config_reload")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("config_open runs without panic", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e, km := defaultsEnv(t, "")
		runCmd(t, km, e, "config_open")
	})

	t.Run("config_open_workspace runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmd(t, km, e, "config_open_workspace")
	})

	t.Run("log_open runs without panic", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := defaultsEnv(t, "")
		runCmd(t, km, e, "log_open")
	})

	t.Run("workspace_trust runs", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "workspace_trust")
		assert.NotContains(t, res.Message, "error")
	})

	t.Run("workspace_untrust runs", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "workspace_untrust")
		assert.NotContains(t, res.Message, "error")
	})

	t.Run("line ending no args shows current", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		res := runCmd(t, km, e, "set_line_ending")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("indent_style without args shows current", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "indent_style")
		assert.NotEmpty(t, res.Message)
	})
}
