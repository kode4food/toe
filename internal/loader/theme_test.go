package loader_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/theme"
)

func TestThemeNames(t *testing.T) {
	names := loader.ThemeNames()

	assert.Equal(t, []string{
		"frappe",
		"latte",
		"macchiato",
		"mocha",
	}, names)
}

func TestEmbeddedThemes(t *testing.T) {
	t.Run("macchiato loads with inherited palette", func(t *testing.T) {
		data, err := loader.LoadThemeTOML("macchiato")

		assert.NoError(t, err)
		assert.NotNil(t, data)
		// palette should contain macchiato overrides
		pal, ok := data["palette"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "#f4dbd6", pal["rosewater"])
		// scopes should come from the mocha parent
		assert.NotNil(t, data["ui.selection"])
		assert.NotNil(t, data["ui.statusline"])
	})

	t.Run("macchiato decodes and validates", func(t *testing.T) {
		data, err := loader.LoadThemeTOML("macchiato")
		assert.NoError(t, err)

		th, _ := theme.Decode(data)
		assert.False(t, th.Is16Color())
		assert.NoError(t, th.Validate())
		// statusline should resolve palette references to RGB
		_, ok := th.TryGet("ui.statusline")
		assert.True(t, ok)
	})
}

func TestLoadThemeTOML(t *testing.T) {
	t.Run("loads mocha", func(t *testing.T) {
		th, err := loader.LoadThemeTOML("mocha")

		assert.NoError(t, err)
		assert.Equal(t, "yellow", th["attribute"])
		palette := th["palette"].(map[string]any)
		assert.Equal(t, "#cdd6f4", palette["text"])
	})

	t.Run("rejects unsupported theme", func(t *testing.T) {
		_, err := loader.LoadThemeTOML("dark")

		assert.True(t, errors.Is(err, loader.ErrThemeNotFound))
	})
}
