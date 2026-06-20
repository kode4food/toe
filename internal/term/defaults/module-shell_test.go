package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellCommands(t *testing.T) {
	for _, name := range []string{
		"shell_pipe", "shell_insert_output", "shell_keep_pipe",
		"shell_pipe_to", "shell_append_output",
	} {
		t.Run(name+" runs without panic", func(t *testing.T) {
			e, km := defaultsEnv(t, "abc")
			runCmd(t, km, e, name)
		})
	}
}

func TestShellOption(t *testing.T) {
	t.Run("get editor.shell returns value", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmdArgs(t, km, e, "get_option", "editor.shell")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("set then get editor.shell", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		runCmdArgs(t, km, e, "set_option", `editor.shell ["bash", "-c"]`)
		res := runCmdArgs(t, km, e, "get_option", "editor.shell")
		assert.Contains(t, res.Message, "bash")
	})
}
