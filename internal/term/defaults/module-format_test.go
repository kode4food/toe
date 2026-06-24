package defaults_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCommands(t *testing.T) {
	t.Run("format on plain text reports no formatter", func(t *testing.T) {
		e, km := defaultsEnv(t, "hello\n")
		res := runCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "no formatter")
	})

	t.Run("reflow runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "hello world\n")
		runCmd(t, km, e, "reflow")
	})

	t.Run("sort runs without panic", func(t *testing.T) {
		e, km := defaultsEnv(t, "b\na\nc\n")
		runCmd(t, km, e, "sort")
	})

	t.Run("format_selections runs", func(t *testing.T) {
		e, km := defaultsEnv(t, "  hello\n")
		runCmd(t, km, e, "format_selections")
	})
}

func TestFormatWithFormatter(t *testing.T) {
	t.Run("formatter noop returns no message", func(t *testing.T) {
		writeFormatterConfig(t, `command = "cat"`)
		e, km := defaultsEnv(t, "hello\n")
		res := runCmd(t, km, e, "format")
		assert.Equal(t, "", res.Message)
	})

	t.Run("formatter changes text applies diff", func(t *testing.T) {
		writeFormatterConfig(t, `command = "tr"
args = ["a-z", "A-Z"]`)
		e, km := defaultsEnv(t, "hello\n")
		res := runCmd(t, km, e, "format")
		assert.Equal(t, "", res.Message)
		assert.Equal(t, "HELLO\n", docText(t, e))
	})

	t.Run("failing formatter reports error", func(t *testing.T) {
		writeFormatterConfig(t, `command = "false"`)
		e, km := defaultsEnv(t, "hello\n")
		res := runCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "error")
	})
}

func writeFormatterConfig(t *testing.T, fmtToml string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	content := fmt.Sprintf(
		"[[language]]\nname = \"text\"\n[language.formatter]\n%s\n", fmtToml,
	)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(content), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
