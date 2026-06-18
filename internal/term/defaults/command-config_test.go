package defaults_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/config"
)

func TestConfigOption(t *testing.T) {
	boolKey := config.BoolOptionKeys()[0]

	t.Run("set then get round-trips", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", boolKey+" true")
		res := runCmdArgs(t, km, e, "get_option", boolKey)
		assert.Equal(t, "true", res.Message)
	})

	t.Run("toggle reports new value", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "toggle_option", boolKey)
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
		assert.Equal(t, "mocha", e.Config().Theme.Name)
	})
}

func TestConfigDocumentOptions(t *testing.T) {
	t.Run("set language updates the buffer", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_language", "go")
		doc, _ := e.FocusedDocument()
		assert.Equal(t, "go", doc.Lang())
	})

	t.Run("set language reports current without args", func(t *testing.T) {
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
