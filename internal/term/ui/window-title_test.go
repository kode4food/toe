package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestWindowTitle(t *testing.T) {
	title := func(t *testing.T, dir string) string {
		t.Helper()
		e := view.NewEditor(dir)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		return m.View().WindowTitle
	}

	t.Run("names the workspace root path", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "myproject")
		assert.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
		sub := filepath.Join(root, "internal", "pkg")
		assert.NoError(t, os.MkdirAll(sub, 0o755))
		assert.Equal(t, root+" - toe", title(t, sub))
	})

	t.Run("shortens a path under home", func(t *testing.T) {
		home, err := os.UserHomeDir()
		assert.NoError(t, err)
		root, err := os.MkdirTemp(home, "toe-title-")
		assert.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(root) })
		assert.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
		want := filepath.Join("~", filepath.Base(root))
		assert.Equal(t, want+" - toe", title(t, root))
	})

	t.Run("falls back to the directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.Equal(t, dir+" - toe", title(t, dir))
	})
}
