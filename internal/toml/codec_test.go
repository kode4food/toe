package toml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/toml"
)

func TestDecode(t *testing.T) {
	t.Run("decodes text", func(t *testing.T) {
		var out map[string]any

		assert.NoError(t, toml.Decode(`name = "go"`, &out))
		assert.Equal(t, map[string]any{"name": "go"}, out)
	})

	t.Run("reports malformed text", func(t *testing.T) {
		var out map[string]any

		assert.Error(t, toml.Decode("name =", &out))
	})
}

func TestDecodeFile(t *testing.T) {
	t.Run("decodes file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		assert.NoError(t, os.WriteFile(path, []byte(`width = 4`), 0o644))

		var out map[string]any

		assert.NoError(t, toml.DecodeFile(path, &out))
		assert.Equal(t, map[string]any{"width": int64(4)}, out)
	})

	t.Run("reports missing file", func(t *testing.T) {
		var out map[string]any
		path := filepath.Join(t.TempDir(), "missing.toml")

		assert.Error(t, toml.DecodeFile(path, &out))
	})
}

func TestEncode(t *testing.T) {
	t.Run("renders a table", func(t *testing.T) {
		text, err := toml.Encode(map[string]any{"name": "go"})

		assert.NoError(t, err)
		assert.Equal(t, "name = \"go\"\n", text)
	})

	t.Run("reports unsupported values", func(t *testing.T) {
		text, err := toml.Encode(map[string]any{"fn": func() {}})

		assert.Error(t, err)
		assert.Equal(t, "", text)
	})
}
