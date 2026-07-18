package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
)

func TestLifecycleQuit(t *testing.T) {
	t.Run("quit on clean doc signals quit", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t,
			command.SignalQuit, test.RunCmd(t, km, e, "quit").Signal)
	})

	t.Run("quit on dirty doc warns", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Contains(t, test.RunCmd(t, km, e, "quit").Message, "unsaved")
	})

	t.Run("quit! always signals quit", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Equal(t,
			command.SignalQuit, test.RunCmd(t, km, e, "quit!").Signal)
	})

	t.Run("quit! resolves in image mode", func(t *testing.T) {
		_, km := test.Env(t, "")
		cmd, ok := km.ResolveCommandIn("IMG", "q!")
		assert.True(t, ok)
		assert.Equal(t, "quit!", cmd.Name)
	})

	t.Run("quit_all on clean signals quit", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t,
			command.SignalQuit, test.RunCmd(t, km, e, "quit_all").Signal)
	})

	t.Run("quit_all on dirty warns", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Contains(t, test.RunCmd(t, km, e, "quit_all").Message, "unsaved")
	})

	t.Run("quit-all! always signals quit", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Equal(t,
			command.SignalQuit, test.RunCmd(t, km, e, "quit-all!").Signal)
	})

	t.Run("cquit on dirty warns", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Contains(t, test.RunCmd(t, km, e, "cquit").Message, "unsaved")
	})
}
