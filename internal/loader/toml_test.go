package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
)

func TestMergeTOMLValues(t *testing.T) {
	t.Run("merges named array entries", func(t *testing.T) {
		left := []any{
			map[string]any{
				"name": "toml",
				"language-server": map[string]any{
					"command": "taplo",
					"args":    []any{"lsp", "stdio"},
				},
			},
		}
		right := []any{
			map[string]any{
				"name": "toml",
				"language-server": map[string]any{
					"command": "/usr/bin/taplo",
				},
			},
		}

		merged := loader.MergeTOMLValues(left, right, 3)

		assert.Equal(t, []any{
			map[string]any{
				"name": "toml",
				"language-server": map[string]any{
					"command": "/usr/bin/taplo",
					"args":    []any{"lsp", "stdio"},
				},
			},
		}, merged)
	})

	t.Run("right value replaces at depth zero", func(t *testing.T) {
		left := map[string]any{"a": map[string]any{"b": 1}}
		right := map[string]any{"a": map[string]any{"c": 2}}

		merged := loader.MergeTOMLValues(left, right, 1)

		assert.Equal(t, map[string]any{
			"a": map[string]any{"c": 2},
		}, merged)
	})
}

func TestLoadMergedTOML(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.toml")
	local := filepath.Join(dir, "local.toml")
	err := os.WriteFile(global, []byte(`
[editor]
text-width = 80

[editor.soft-wrap]
enable = true
wrap-indicator = "↪ "
`), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(local, []byte(`
[editor]
text-width = 72

[editor.soft-wrap]
wrap-indicator = "» "
`), 0o644)
	assert.NoError(t, err)

	merged, ok := loader.LoadMergedTOML([]string{global, local}, 3)

	assert.True(t, ok)
	editor := merged["editor"].(map[string]any)
	assert.Equal(t, int64(72), editor["text-width"])
	soft := editor["soft-wrap"].(map[string]any)
	assert.Equal(t, true, soft["enable"])
	assert.Equal(t, "» ", soft["wrap-indicator"])
}

func TestLoadMergedTOMLWithBase(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "local.toml")
	base := map[string]any{
		"language": []any{
			map[string]any{
				"name":       "markdown",
				"text-width": int64(100),
				"soft-wrap": map[string]any{
					"wrap-indicator": "↪ ",
				},
			},
		},
	}
	err := os.WriteFile(local, []byte(`
[[language]]
name = "markdown"
text-width = 80
`), 0o644)
	assert.NoError(t, err)

	merged, ok := loader.LoadMergedTOMLWithBase(base, []string{local}, 3)

	assert.True(t, ok)
	lang, ok := namedTOMLValue(merged["language"], "markdown")
	assert.True(t, ok)
	soft := lang["soft-wrap"].(map[string]any)
	assert.Equal(t, int64(80), lang["text-width"])
	assert.Equal(t, "↪ ", soft["wrap-indicator"])
}

func TestDefaultLanguagesTOML(t *testing.T) {
	langs, ok := loader.LoadDefaultLanguagesTOML()

	assert.True(t, ok)
	assert.NotEmpty(t, langs["language"])
}

func TestLoadMergedTOMLLanguageDepth(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.toml")
	local := filepath.Join(dir, "local.toml")
	err := os.WriteFile(global, []byte(`
[[language]]
name = "markdown"
soft-wrap.enable = true
soft-wrap.wrap-indicator = "↪ "
`), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(local, []byte(`
[[language]]
name = "markdown"
soft-wrap.wrap-indicator = "» "
`), 0o644)
	assert.NoError(t, err)

	merged, ok := loader.LoadMergedTOML([]string{global, local}, 3)

	assert.True(t, ok)
	lang, ok := namedTOMLValue(merged["language"], "markdown")
	assert.True(t, ok)
	soft := lang["soft-wrap"].(map[string]any)
	assert.Nil(t, soft["enable"])
	assert.Equal(t, "» ", soft["wrap-indicator"])
}

func namedTOMLValue(value any, name string) (map[string]any, bool) {
	switch values := value.(type) {
	case []any:
		for _, item := range values {
			m, ok := item.(map[string]any)
			if ok && m["name"] == name {
				return m, true
			}
		}
	case []map[string]any:
		for _, m := range values {
			if m["name"] == name {
				return m, true
			}
		}
	}
	return nil, false
}
