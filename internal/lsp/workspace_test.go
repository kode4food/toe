package lsp_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view/language"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace(t *testing.T) {
	t.Run("rejects outside workspace", func(t *testing.T) {
		root := t.TempDir()
		other := t.TempDir()

		_, ok := lsp.ResolveWorkspace(lsp.WorkspaceRequest{
			FilePath:  filepath.Join(other, "main.go"),
			Workspace: root,
		})

		assert.False(t, ok)
	})

	t.Run("uses top marker", func(t *testing.T) {
		root := t.TempDir()
		project := filepath.Join(root, "project")
		pkg := filepath.Join(project, "pkg")
		writeFile(t, filepath.Join(project, "go.mod"))
		writeFile(t, filepath.Join(pkg, "go.mod"))
		file := writeFile(t, filepath.Join(pkg, "main.go"))

		ws, ok := lsp.ResolveWorkspace(lsp.WorkspaceRequest{
			FilePath:    file,
			Workspace:   root,
			RootMarkers: []string{"go.mod"},
		})

		assert.True(t, ok)
		assert.Equal(t, project, ws.Path)
	})

	t.Run("workspace fallback", func(t *testing.T) {
		root := t.TempDir()
		file := writeFile(t, filepath.Join(root, "main.go"))

		ws, ok := lsp.ResolveWorkspace(lsp.WorkspaceRequest{
			FilePath:  file,
			Workspace: root,
		})

		assert.True(t, ok)
		assert.Equal(t, root, ws.Path)
	})

	t.Run("cwd workspace without marker", func(t *testing.T) {
		root := t.TempDir()
		file := writeFile(t, filepath.Join(root, "main.go"))

		_, ok := lsp.ResolveWorkspace(lsp.WorkspaceRequest{
			FilePath:       file,
			Workspace:      root,
			WorkspaceIsCWD: true,
		})

		assert.False(t, ok)
	})

	t.Run("root dir stop falls back to workspace", func(t *testing.T) {
		root := t.TempDir()
		pkg := filepath.Join(root, "pkg")
		file := writeFile(t, filepath.Join(pkg, "main.go"))

		ws, ok := lsp.ResolveWorkspace(lsp.WorkspaceRequest{
			FilePath:  file,
			Workspace: root,
			RootDirs:  []string{"pkg"},
		})

		assert.True(t, ok)
		assert.Equal(t, root, ws.Path)
	})

	t.Run("root dir stop keeps marker", func(t *testing.T) {
		root := t.TempDir()
		pkg := filepath.Join(root, "pkg")
		writeFile(t, filepath.Join(pkg, "go.mod"))
		file := writeFile(t, filepath.Join(pkg, "main.go"))

		ws, ok := lsp.ResolveWorkspace(lsp.WorkspaceRequest{
			FilePath:    file,
			Workspace:   root,
			RootMarkers: []string{"go.mod"},
			RootDirs:    []string{"pkg"},
		})

		assert.True(t, ok)
		assert.Equal(t, pkg, ws.Path)
	})

	t.Run("required root patterns", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "go.mod"))

		ok, err := lsp.RequiredRootFound(root, []string{"go.*"})

		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("missing required root", func(t *testing.T) {
		root := t.TempDir()
		_, _, err := lsp.Start(t.Context(), "test", language.Server{
			Command:              "unused",
			RequiredRootPatterns: []string{"go.mod"},
		}, root, nil)

		assert.True(t, errors.Is(err, lsp.ErrRequiredRoot))
	})
}

func writeFile(t *testing.T, path string) string {
	t.Helper()
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoError(t, os.WriteFile(path, nil, 0o644))
	return path
}
