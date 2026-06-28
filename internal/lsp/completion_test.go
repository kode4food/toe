package lsp_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
)

func TestCompletion(t *testing.T) {
	t.Run("applies text edit completion", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Pr\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		items, err := session.Completions(doc, v.ID())
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		if len(items) != 1 {
			return
		}
		err = session.ApplyCompletion(doc, v.ID(), items[0])

		assert.NoError(t, err)
		assert.Equal(t, "Println\n// add\n", doc.Text().String())
	})

	t.Run("replaces typed prefix after stale edit", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeStaleCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Pr\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		items, err := session.Completions(doc, v.ID())
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		if len(items) != 1 {
			return
		}
		err = session.ApplyCompletion(doc, v.ID(), items[0])

		assert.NoError(t, err)
		assert.Equal(t, "Println\n// add\n", doc.Text().String())
	})

	t.Run("reports exited server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeExitingCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Pr\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		items, err := session.Completions(doc, v.ID())

		assert.Empty(t, items)
		assert.True(t, errors.Is(err, lsp.ErrLanguageServerExited))
		assert.Contains(t, err.Error(), "completion process exited")
		assert.NotContains(t, err.Error(), "jsonrpc2")
	})

	t.Run("hides transport error on silent exit", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSilentExitCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Pr\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		items, err := session.Completions(doc, v.ID())

		assert.Empty(t, items)
		assert.True(t, errors.Is(err, lsp.ErrLanguageServerExited))
		assert.Contains(t, err.Error(), "language server exited: session-test")
		assert.NotContains(t, err.Error(), "jsonrpc2")
		assert.NotContains(t, err.Error(), "file already closed")
	})

	t.Run("requests trigger character completion", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("x.\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		items, err := session.TriggerCompletions(doc, v.ID())

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		if len(items) != 1 {
			return
		}
		assert.Equal(t, "Println", items[0].Label)
		assert.Equal(t, "function", items[0].Kind)
	})

	t.Run("resolves documentation", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Pr\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))

		items, err := session.Completions(doc, v.ID())
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		if len(items) != 1 {
			return
		}
		resolved, err := session.ResolveCompletion(doc, v.ID(), items[0])

		assert.NoError(t, err)
		assert.Contains(t, resolved.Docs, "func Println")
		assert.Contains(t, resolved.Docs, "Println formats")
	})
}

func writeCompletionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCompletionEnv + ` = "1" }

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

func writeExitingCompletionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCompletionEnv + ` = "1", ` +
		testServerExitOnCompletionEnv + ` = "1" }

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

func writeSilentExitCompletionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCompletionEnv + ` = "1", ` +
		testServerSilentExitOnCompletionEnv + ` = "1" }

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

func writeStaleCompletionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCompletionEnv + ` = "1", ` +
		testServerStaleCompletionEnv + ` = "1" }

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
