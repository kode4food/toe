package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
)

func TestLifecycleQuit(t *testing.T) {
	t.Run("quit on clean doc signals quit", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Equal(t, command.SignalQuit, runCmd(t, km, e, "quit").Signal)
	})

	t.Run("quit on dirty doc warns", func(t *testing.T) {
		e, km := defaultsEnv(t, "x")
		assert.Contains(t, runCmd(t, km, e, "quit").Message, "unsaved")
	})

	t.Run("quit! always signals quit", func(t *testing.T) {
		e, km := defaultsEnv(t, "x")
		assert.Equal(t, command.SignalQuit, runCmd(t, km, e, "quit!").Signal)
	})

	t.Run("quit_all on clean signals quit", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		assert.Equal(t, command.SignalQuit, runCmd(t, km, e, "quit_all").Signal)
	})

	t.Run("quit_all on dirty warns", func(t *testing.T) {
		e, km := defaultsEnv(t, "x")
		assert.Contains(t, runCmd(t, km, e, "quit_all").Message, "unsaved")
	})

	t.Run("quit-all! always signals quit", func(t *testing.T) {
		e, km := defaultsEnv(t, "x")
		assert.Equal(t,
			command.SignalQuit, runCmd(t, km, e, "quit-all!").Signal)
	})

	t.Run("cquit on dirty warns", func(t *testing.T) {
		e, km := defaultsEnv(t, "x")
		assert.Contains(t, runCmd(t, km, e, "cquit").Message, "unsaved")
	})
}
