package editing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
)

func TestInsertRegister(t *testing.T) {
	t.Run("insert register takes a char", func(t *testing.T) {
		e, km := test.Env(t, "abc")
		testutil.SetCursor(t, e, 0)
		res := test.RunCmd(t, km, e, "insert_register")
		assert.NotNil(t, res.Continuation)
		// empty register pastes nothing; the continuation still completes
		assert.Nil(t, res.Continuation(e, test.Char('a')))
	})
}

func TestCompletionCommands(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  command.KeyEvent
	}{
		{name: "completion_previous", key: test.Special("up")},
		{name: "completion_next", key: test.Special("down")},
		{name: "completion_page_up", key: test.Special("pageup")},
		{name: "completion_page_down", key: test.Special("pagedown")},
		{name: "completion_first", key: test.Special("home")},
		{name: "completion_last", key: test.Special("end")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, km := test.Env(t, "")
			res := test.RunCmd(t, km, e, tt.name)
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
		e, _, reg := test.EnvWithRegistry(t, "")
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
		e, _, reg := test.EnvWithRegistry(t, "")
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
