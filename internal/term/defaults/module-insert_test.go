package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
)

func TestInsertRegister(t *testing.T) {
	t.Run("insert register takes a char", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		testutil.SetCursor(t, e, 0)
		res := runCmd(t, km, e, "insert_register")
		assert.NotNil(t, res.Continuation)
		// empty register pastes nothing; the continuation still completes
		assert.Nil(t, res.Continuation(e, char('a')))
	})
}

func TestCompletionCommands(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  command.KeyEvent
	}{
		{name: "completion_previous", key: special("up")},
		{name: "completion_next", key: special("down")},
		{name: "completion_page_up", key: special("pageup")},
		{name: "completion_page_down", key: special("pagedown")},
		{name: "completion_first", key: special("home")},
		{name: "completion_last", key: special("end")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, km := defaultsEnv(t, "")
			res := runCmd(t, km, e, tt.name)
			assert.Empty(t, res.Message)

			act, found, prefix := km.Lookup(
				ui.CompletionMode, []command.KeyEvent{tt.key},
			)
			assert.True(t, found)
			assert.False(t, prefix)
			assert.Nil(t, act(e))
		})
	}
}

func TestCompletionConfig(t *testing.T) {
	t.Run("icon mode decodes", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"completion": map[string]any{
					"icons": string(ui.CompletionIconsNone),
				},
			},
		})

		assert.NoError(t, err)
	})

	t.Run("invalid icon mode errors", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"completion": map[string]any{
					"icons": "automatic",
				},
			},
		})

		assert.Error(t, err)
	})
}
