package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestLifecycleQuit(t *testing.T) {
	t.Run("quit on clean doc signals quit", func(t *testing.T) {
		e, km := test.Env(t, "")
		assert.Equal(t,
			command.SignalQuit, test.RunCmd(t, km, e, "quit").Signal)
	})

	t.Run("quit warns on any dirty doc", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		testutil.SetEditorText(t, e, "dirty")
		e.NewDocument()
		assert.Contains(t, test.RunCmd(t, km, e, "quit").Message, "unsaved")
	})

	t.Run("quit! always signals quit", func(t *testing.T) {
		e, km := test.Env(t, "x")
		assert.Equal(t,
			command.SignalQuit, test.RunCmd(t, km, e, "quit!").Signal)
	})

	t.Run("quit! resolves in pane modes", func(t *testing.T) {
		_, km := test.Env(t, "")
		for _, mode := range []view.Mode{view.ModeImage, view.ModeBinary} {
			cmd := km.ResolveCommandIn(mode, "q!")
			assert.NotNil(t, cmd)
			assert.Equal(t, "quit!", cmd.Name)
		}
	})

}
