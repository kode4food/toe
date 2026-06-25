package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
)

func TestEditorConfigGlob(t *testing.T) {
	t.Run("double-star matches subdirectory", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "src", "pkg")
		err := os.MkdirAll(dir, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(root, ".editorconfig"), []byte(`
root = true

[*.go]
indent_style = tab

[src/**/*.go]
indent_style = space
indent_size = 4
`), 0o644)
		assert.NoError(t, err)

		cfg := config.FindEditorConfig(filepath.Join(dir, "main.go"))
		assert.NotNil(t, cfg.IndentStyle)
		assert.False(t, cfg.IndentStyle.IsTabs())
		assert.Equal(t, uint8(4), cfg.IndentStyle.Width())
	})

	t.Run("star-slash prefix matches subdir file", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "sub")
		err := os.MkdirAll(dir, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(root, ".editorconfig"), []byte(`
root = true

[*/*.go]
indent_style = space
indent_size = 2
`), 0o644)
		assert.NoError(t, err)

		cfg := config.FindEditorConfig(filepath.Join(dir, "main.go"))
		assert.NotNil(t, cfg.IndentStyle)
		assert.False(t, cfg.IndentStyle.IsTabs())
	})

	t.Run("brace alternation matches extensions", func(t *testing.T) {
		root := t.TempDir()
		err := os.WriteFile(filepath.Join(root, ".editorconfig"), []byte(`
root = true

[*.{go,rs}]
indent_style = tab
`), 0o644)
		assert.NoError(t, err)

		cfgGo := config.FindEditorConfig(filepath.Join(root, "main.go"))
		assert.NotNil(t, cfgGo.IndentStyle)
		assert.True(t, cfgGo.IndentStyle.IsTabs())

		cfgRS := config.FindEditorConfig(filepath.Join(root, "lib.rs"))
		assert.NotNil(t, cfgRS.IndentStyle)
		assert.True(t, cfgRS.IndentStyle.IsTabs())

		cfgPy := config.FindEditorConfig(filepath.Join(root, "script.py"))
		assert.Nil(t, cfgPy.IndentStyle)
	})
}

func TestEditorConfig(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(root, ".editorconfig"), []byte(`
root = true

[*]
insert_final_newline = false
trim_trailing_whitespace = true

[*.go]
indent_style = space
indent_size = 2
tab_width = 8
end_of_line = crlf
max_line_length = 100
`), 0o644)
	assert.NoError(t, err)

	cfg := config.FindEditorConfig(filepath.Join(dir, "main.go"))

	assert.NotNil(t, cfg.IndentStyle)
	assert.False(t, cfg.IndentStyle.IsTabs())
	assert.Equal(t, uint8(2), cfg.IndentStyle.Width())
	assert.Equal(t, 8, *cfg.TabWidth)
	assert.Equal(t, core.LineEndingCRLF, *cfg.LineEnding)
	assert.True(t, *cfg.TrimTrailingWhitespace)
	assert.False(t, *cfg.InsertFinalNewline)
	assert.Equal(t, 100, *cfg.MaxLineLength)
}
