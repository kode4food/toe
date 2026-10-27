package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchableTreeDirsExcludesGit(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".git", "src"} {
		assert.NoError(t, os.Mkdir(filepath.Join(root, dir), 0o755))
	}
	assert.NoError(t, os.WriteFile(
		filepath.Join(root, "top.txt"), []byte("x"), 0o644,
	))

	dirs := watchableTreeDirs(root)

	assert.Contains(t, dirs, root)
	assert.Contains(t, dirs, filepath.Join(root, "src", "..."))
	assert.False(t, slices.ContainsFunc(dirs, func(d string) bool {
		return d == filepath.Join(root, ".git") ||
			d == filepath.Join(root, ".git", "...")
	}), "recursive watch targets must not include .git: %v", dirs)
}
