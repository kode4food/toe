package files_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
)

func TestPickerOptions(t *testing.T) {
	cases := []struct{ key, value string }{
		{key: "buffer-picker.start-position", value: "previous"},
		{key: "file-explorer.hidden", value: "true"},
		{key: "file-explorer.follow-symlinks", value: "true"},
		{key: "file-explorer.parents", value: "true"},
		{key: "file-explorer.ignore", value: "true"},
		{key: "file-explorer.git-ignore", value: "true"},
		{key: "file-explorer.git-global", value: "true"},
		{key: "file-explorer.git-exclude", value: "true"},
		{key: "file-explorer.flatten-dirs", value: "false"},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			e, km := test.Env(t, "")
			test.RunCmdArgs(t, km, e, "set-option",
				tc.key+" "+tc.value)
			res := test.RunCmdArgs(t, km, e, "get-option", tc.key)
			assert.Equal(t, tc.value, res.Message)
		})
	}

	t.Run("defaults buffer picker to top", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "get-option",
			"buffer-picker.start-position")
		assert.Equal(t, "top", res.Message)
	})

	t.Run("defaults directory flattening", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "get-option",
			"file-explorer.flatten-dirs")
		assert.Equal(t, "true", res.Message)
	})

	t.Run("resets directory flattening", func(t *testing.T) {
		e, km, reg := test.EnvWithRegistry(t, "")
		raw := map[string]any{
			"editor": map[string]any{
				"file-explorer": map[string]any{"flatten-dirs": false},
			},
		}
		assert.NoError(t, reg.ApplyTOML(e, raw))
		res := test.RunCmdArgs(t,
			km, e, "get-option", "file-explorer.flatten-dirs")
		assert.Equal(t, "false", res.Message)

		assert.NoError(t, reg.ApplyTOML(e, map[string]any{}))
		res = test.RunCmdArgs(t,
			km, e, "get-option", "file-explorer.flatten-dirs")
		assert.Equal(t, "true", res.Message)
	})

	t.Run("rejects invalid start position", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t,
			km, e, "set-option", "buffer-picker.start-position invalid")
		assert.Contains(t, res.Message, "error")
	})
}
