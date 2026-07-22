package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestDocumentHighlights(t *testing.T) {
	t.Run("returns highlights at cursor", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeHighlightLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("foo bar\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		highlights, err := session.DocumentHighlights(doc, v.ID())

		assert.NoError(t, err)
		assert.NotEmpty(t, highlights)
	})
}

func TestDocumentHighlightsMerge(t *testing.T) {
	t.Run("overlapping ranges merge", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeHighlightMultiLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("foo bar\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(0))

		highlights, err := session.DocumentHighlights(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, highlights, 2)
	})
}

func TestDocumentHighlightLifecycle(t *testing.T) {
	t.Run("stop clears highlights", func(t *testing.T) {
		session, doc, v := openHighlightSession(t)
		defer func() { _ = session.Close() }()
		doc.SetDocumentHighlights(v.ID(), []view.DocumentHighlight{
			{From: 0, To: 3},
		})

		_, err := session.StopLanguageServers(doc, nil)

		assert.NoError(t, err)
		assert.Empty(t, doc.DocumentHighlights(v.ID()))
	})

	t.Run("restart clears highlights", func(t *testing.T) {
		session, doc, v := openHighlightSession(t)
		defer func() { _ = session.Close() }()
		doc.SetDocumentHighlights(v.ID(), []view.DocumentHighlight{
			{From: 0, To: 3},
		})

		_, err := session.RestartLanguageServers(doc, nil)

		assert.NoError(t, err)
		assert.Empty(t, doc.DocumentHighlights(v.ID()))
	})

	t.Run("close clears highlights", func(t *testing.T) {
		session, doc, v := openHighlightSession(t)
		doc.SetDocumentHighlights(v.ID(), []view.DocumentHighlight{
			{From: 0, To: 3},
		})

		err := session.Close()

		assert.NoError(t, err)
		assert.Empty(t, doc.DocumentHighlights(v.ID()))
	})

	t.Run("reload clears highlights", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		root := t.TempDir()
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeHighlightLanguagesAt(t, root, exe)
		assert.NoError(t, os.WriteFile(path, []byte("foo bar\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetDocumentHighlights(v.ID(), []view.DocumentHighlight{
			{From: 0, To: 3},
		})
		doc.ReplaceDiagnostics("session-test", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityError},
		})
		writeNoServerLanguagesAt(t, root)

		err = session.ReloadConfig()

		assert.NoError(t, err)
		assert.Empty(t, doc.DocumentHighlights(v.ID()))
		assert.Empty(t, doc.Diagnostics())
		_, err = session.DocumentHighlights(doc, v.ID())
		assert.ErrorIs(t, err, view.ErrNoLanguageServer)
	})
}

func openHighlightSession(
	t *testing.T,
) (*lsp.Session, *view.Document, *view.View) {
	t.Helper()
	exe, err := os.Executable()
	assert.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	writeHighlightLanguages(t, exe)
	assert.NoError(t, os.WriteFile(path, []byte("foo bar\n"), 0o644))
	e := view.NewEditor(dir)
	_, err = e.OpenFile(path)
	assert.NoError(t, err)
	session := lsp.Attach(t.Context(), e)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	v, ok := e.FocusedView()
	assert.True(t, ok)
	return session, doc, v
}

func writeHighlightMultiLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerHighlightEnv + ` = "1", ` +
		testServerHighlightMultiEnv + ` = "1" }

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

func writeHighlightLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	writeHighlightLanguagesAt(t, root, exe)
}

func writeHighlightLanguagesAt(t *testing.T, root, exe string) {
	t.Helper()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerHighlightEnv + ` = "1" }

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

func writeNoServerLanguagesAt(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
}
