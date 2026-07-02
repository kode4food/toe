package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestFileOperations(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)

	t.Run("ignored when no capability", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.NoError(t, session.WillCreateFile(path, false))
		assert.NoError(t, session.DidCreateFile(path, false))
		assert.NoError(t, session.WillRenameFile(path, path+"2", false))
		assert.NoError(t, session.DidRenameFile(path, path+"2", false))
		assert.NoError(t, session.WillDeleteFile(path, false))
		assert.NoError(t, session.DidDeleteFile(path, false))
	})

	t.Run("invoked when server has capability", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		newPath := filepath.Join(dir, "renamed.session")
		writeFileOperationLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.NoError(t, session.WillCreateFile(path, false))
		assert.NoError(t, session.DidCreateFile(path, false))
		assert.NoError(t, session.WillRenameFile(path, newPath, false))
		assert.NoError(t, session.DidRenameFile(path, newPath, false))
		assert.NoError(t, session.WillDeleteFile(path, false))
		assert.NoError(t, session.DidDeleteFile(path, false))
	})

	t.Run("dir kind skips file-only filter", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeFileOperationLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.NoError(t, session.WillCreateFile(path, true))
		assert.NoError(t, session.WillRenameFile(path, path+"2", true))
		assert.NoError(t, session.WillDeleteFile(path, true))
	})
}

func TestFileOperationsFolderKind(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)

	t.Run("folder kind matches dir=true", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeFileOpFolderLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.NoError(t, session.WillCreateFile(dir, true))
		assert.NoError(t, session.DidCreateFile(dir, true))
		assert.NoError(t, session.WillRenameFile(dir, dir+"2", true))
		assert.NoError(t, session.DidRenameFile(dir, dir+"2", true))
		assert.NoError(t, session.WillDeleteFile(dir, true))
		assert.NoError(t, session.DidDeleteFile(dir, true))
	})
}

func TestFileOperationsWillEdit(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)

	t.Run("WillCreate applies workspace edit", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeFileOpWillEditLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.NoError(t, session.WillCreateFile(path, false))
	})
}

func TestFileOperationsErrors(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)

	t.Run("joins server errors", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		newPath := filepath.Join(dir, "renamed.session")
		writeFileOperationErrorLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.Error(t, session.WillCreateFile(path, false))
		assert.Error(t, session.WillRenameFile(path, newPath, false))
		assert.Error(t, session.WillDeleteFile(path, false))
	})
}

func writeFileOperationErrorLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerFileOperationsEnv + ` = "1", ` +
		testServerAllErrorEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeFileOpFolderLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerFileOpFolderEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeFileOpWillEditLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerFileOperationsEnv + ` = "1", ` +
		testServerFileOpWillEditEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeFileOperationLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerFileOperationsEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
