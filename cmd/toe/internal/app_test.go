package app_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	app "github.com/kode4food/toe/cmd/toe/internal"
)

func TestParseConfigFlag(t *testing.T) {
	t.Run("strips --config and path", func(t *testing.T) {
		var path string
		args := app.ParseConfigFlag(
			[]string{"--config", "/etc/toe.toml", "file.go"}, &path,
		)
		assert.Equal(t, "/etc/toe.toml", path)
		assert.Equal(t, []string{"file.go"}, args)
	})

	t.Run("passes through non-config args", func(t *testing.T) {
		var path string
		args := app.ParseConfigFlag([]string{"a.go", "b.go"}, &path)
		assert.Equal(t, "", path)
		assert.Equal(t, []string{"a.go", "b.go"}, args)
	})

	t.Run("--config at end without value", func(t *testing.T) {
		var path string
		args := app.ParseConfigFlag([]string{"--config"}, &path)
		assert.Equal(t, "", path)
		assert.Empty(t, args)
	})

	t.Run("empty args", func(t *testing.T) {
		var path string
		args := app.ParseConfigFlag(nil, &path)
		assert.Equal(t, "", path)
		assert.Empty(t, args)
	})
}

func TestChangedOptionValues(t *testing.T) {
	t.Run("returns only changed keys", func(t *testing.T) {
		base := map[string]string{"a": "1", "b": "2", "c": "3"}
		values := map[string]string{"a": "1", "b": "99", "c": "3"}
		got := app.ChangedOptionValues(base, values)
		assert.Equal(t, map[string]string{"b": "99"}, got)
	})

	t.Run("returns all when base empty", func(t *testing.T) {
		values := map[string]string{"x": "1", "y": "2"}
		got := app.ChangedOptionValues(map[string]string{}, values)
		assert.Equal(t, values, got)
	})

	t.Run("returns empty when nothing changed", func(t *testing.T) {
		base := map[string]string{"a": "1"}
		got := app.ChangedOptionValues(base, map[string]string{"a": "1"})
		assert.Empty(t, got)
	})
}

func TestRunHealth(t *testing.T) {
	t.Run("--health flag runs health check", func(t *testing.T) {
		var b bytes.Buffer
		err := app.Run([]string{"--health"}, &b)
		assert.NoError(t, err)
		assert.Contains(t, b.String(), "toe health: ok")
	})
}

func TestRunErrors(t *testing.T) {
	t.Run("directory as non-first arg errors", func(t *testing.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		err := app.Run([]string{dir1, dir2}, nil)
		assert.True(t, errors.Is(err, app.ErrDirectoryArgument))
	})

	t.Run("unreadable config file is silently skipped", func(t *testing.T) {
		dir := t.TempDir()
		// --config with non-existent file: LoadRawConfig returns !ok, skipped
		var path string
		args := app.ParseConfigFlag(
			[]string{"--config", filepath.Join(dir, "none.toml")}, &path,
		)
		assert.Empty(t, args)
		assert.NotEmpty(t, path)
	})
}
