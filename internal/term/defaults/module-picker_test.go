package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPickerConfig(t *testing.T) {
	t.Run("buffer picker config decodes", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"buffer-picker": map[string]any{
					"start-position": "previous",
				},
			},
		})

		assert.NoError(t, err)
	})

	t.Run("buffer picker rejects invalid start", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"buffer-picker": map[string]any{
					"start-position": "middle",
				},
			},
		})

		assert.Error(t, err)
	})

	t.Run("file explorer config decodes", func(t *testing.T) {
		e, km, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"file-explorer": map[string]any{
					"hidden":          true,
					"follow-symlinks": true,
					"parents":         true,
					"ignore":          true,
					"git-ignore":      true,
					"git-global":      true,
					"git-exclude":     true,
					"flatten-dirs":    false,
				},
			},
		})

		assert.NoError(t, err)
		res := runCmd(t, km, e, "file_explorer")
		assert.Empty(t, res.Message)
	})

	t.Run("empty config decodes", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{})

		assert.NoError(t, err)
	})
}
