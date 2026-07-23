package files_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
)

func TestDirectoryChange(t *testing.T) {
	t.Run("cd changes the working directory", func(t *testing.T) {
		e, km := test.Env(t, "")
		sub := filepath.Join(e.Cwd(), "sub")
		assert.NoError(t, os.Mkdir(sub, 0o755))
		res := test.RunCmdArgs(t, km, e, "change_directory", sub)
		assert.Contains(t, res.Message, "directory:")
		assert.True(t, strings.HasSuffix(e.Cwd(), "sub"))
	})

	t.Run("cd to a missing directory errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "change_directory",
			filepath.Join(e.Cwd(), "nope"))
		assert.Contains(t, res.Message, "error")
	})

	t.Run("cd without args errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "change_directory")
		assert.Error(t, res.Error)
		assert.Contains(t, res.Message, "error")
	})

	t.Run("pwd reports the working directory", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "show_directory")
		assert.Equal(t, e.Cwd(), res.Message)
	})
}

func TestDirectoryStack(t *testing.T) {
	t.Run("push then pop restores the directory", func(t *testing.T) {
		e, km := test.Env(t, "")
		start := e.Cwd()
		sub := filepath.Join(start, "sub")
		assert.NoError(t, os.Mkdir(sub, 0o755))

		test.RunCmdArgs(t, km, e, "push_directory", sub)
		assert.True(t, strings.HasSuffix(e.Cwd(), "sub"))
		assert.NotEmpty(t, test.RunCmd(t, km, e, "show_directory_stack").Message)

		test.RunCmd(t, km, e, "pop_directory")
		assert.Equal(t, start, e.Cwd())
	})

	t.Run("pop on empty stack errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "pop_directory")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("push without args errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "push_directory")
		assert.Contains(t, res.Message, "error")
	})
}
