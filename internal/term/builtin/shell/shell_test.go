package shell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
)

func TestShellCommands(t *testing.T) {
	for _, name := range []string{
		"shell_pipe", "shell_insert_output", "shell_keep_pipe",
		"shell_pipe_to", "shell_append_output",
	} {
		t.Run(name+" runs without panic", func(t *testing.T) {
			e, km := test.Env(t, "abc")
			test.RunCmd(t, km, e, name)
		})
	}
}

func TestShellOption(t *testing.T) {
	t.Run("get shell returns value", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "get_option", "shell")
		assert.NotEmpty(t, res.Message)
	})

	t.Run("set then get shell", func(t *testing.T) {
		e, km := test.Env(t, "")
		test.RunCmdArgs(t, km, e, "set_option", `shell ["bash", "-c"]`)
		res := test.RunCmdArgs(t, km, e, "get_option", "shell")
		assert.Contains(t, res.Message, "bash")
	})
}
