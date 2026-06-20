package defaults_test

import (
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
