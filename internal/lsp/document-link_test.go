package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestDocumentLinks(t *testing.T) {
	t.Run("returns links with target", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeDocumentLinkLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(path, []byte("foo bar\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		links, err := session.DocumentLinks(doc)

		assert.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "file://"+target, links[0].Target)
		assert.Equal(t, 0, links[0].From)
		assert.Equal(t, 3, links[0].To)
	})
}

func TestResolveDocumentLink(t *testing.T) {
	t.Run("resolves link target", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeDocumentLinkResolveLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(path, []byte("foo bar\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		links, err := session.DocumentLinks(doc)
		assert.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Empty(t, links[0].Target)

		resolved, err := session.ResolveDocumentLink(doc, links[0])
		assert.NoError(t, err)
		assert.Equal(t, "file://"+target, resolved.Target)
	})
}

func writeDocumentLinkLanguages(t *testing.T, exe, target string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerDocumentLinkEnv + ` = "1", ` +
		testServerNavigationTargetEnv + ` = "` + target + `" }

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

func writeDocumentLinkResolveLanguages(t *testing.T, exe, target string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerDocumentLinkEnv + ` = "1", ` +
		testServerDocumentLinkResolveEnv + ` = "1", ` +
		testServerNavigationTargetEnv + ` = "` + target + `" }

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
