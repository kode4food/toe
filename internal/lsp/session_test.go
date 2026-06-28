package lsp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/uri"
)

func TestSession(t *testing.T) {
	t.Run("opens configured server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		file := filepath.Join(dir, "main.session")
		marker := filepath.Join(dir, "did-open")
		writeSessionLanguages(t, exe, marker)
		assert.NoError(t, os.WriteFile(file, []byte("hello\n"), 0o644))

		e := view.NewEditor(dir)
		_, err = e.OpenFile(file)
		assert.NoError(t, err)

		session := lsp.Attach(t.Context(), e)
		defer session.Close()

		assert.Eventually(t, func() bool {
			got, err := os.ReadFile(marker)
			return err == nil && string(got) == string(uri.File(file))
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("serves workspace folders", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		file := filepath.Join(dir, "main.session")
		marker := filepath.Join(dir, "did-open")
		writeWorkspaceFolderLanguages(t, exe, marker)
		assert.NoError(t, os.WriteFile(file, []byte("hello\n"), 0o644))

		e := view.NewEditor(dir)
		_, err = e.OpenFile(file)
		assert.NoError(t, err)

		session := lsp.Attach(t.Context(), e)
		defer session.Close()

		assert.Eventually(t, func() bool {
			got, err := os.ReadFile(marker)
			want := string(uri.File(file)) + "\n" + string(uri.File(dir))
			return err == nil && string(got) == want
		}, time.Second, 10*time.Millisecond)
	})

}

func writeSessionLanguages(t *testing.T, exe, marker string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`
[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = %q }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerDidOpenFileEnv, marker)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeWorkspaceFolderLanguages(t *testing.T, exe, marker string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`
[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = %q, %s = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerDidOpenFileEnv, marker,
		testServerWorkspaceFoldersEnv)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
