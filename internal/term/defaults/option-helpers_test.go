package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestOptionHelperDefaults(t *testing.T) {
	t.Run("apply empty uses all defaults", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{}))
		assert.True(t, e.Options().SearchSmartCase)
		assert.True(t, e.Options().SearchWrapAround)
		assert.Equal(t, view.DefaultScrollOff, e.Options().ScrollOff)
		assert.Equal(t, view.DefaultScrollLines, e.Options().ScrollLines)
		assert.Equal(t, view.DefaultTheme, e.Options().Theme)
		assert.NotEmpty(t, e.Options().Shell)
	})

	t.Run("non-nil bools applied", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"search": map[string]any{
					"smart-case":  false,
					"wrap-around": false,
				},
			},
		}))
		assert.False(t, e.Options().SearchSmartCase)
		assert.False(t, e.Options().SearchWrapAround)
	})

	t.Run("non-nil ints applied", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"scrolloff":    5,
				"scroll-lines": 8,
			},
		}))
		assert.Equal(t, 5, e.Options().ScrollOff)
		assert.Equal(t, 8, e.Options().ScrollLines)
	})

	t.Run("non-empty string theme applied", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{
			"theme": "base16",
		}))
		assert.Equal(t, "base16", e.Options().Theme)
	})

	t.Run("line-number string applied", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{"line-number": "relative"},
		}))
		assert.Equal(t, view.LineNumber("relative"), e.Options().LineNumber)
	})

	t.Run("bufferline string applied", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{"bufferline": "always"},
		}))
		assert.Equal(t, view.BufferLine("always"), e.Options().BufferLine)
	})

	t.Run("shell config applied", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"shell": []any{"bash", "-c"},
			},
		}))
		assert.Equal(t, []string{"bash", "-c"}, e.Options().Shell)
	})
}
