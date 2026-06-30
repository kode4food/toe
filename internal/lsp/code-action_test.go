package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
)

func TestCodeAction(t *testing.T) {
	t.Run("lists and applies action", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCodeActionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{
			core.NewRange(0, 3),
		}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)

		actions, err := session.CodeActions(doc, v.ID())
		assert.NoError(t, err)
		err = session.ApplyCodeAction(doc, v.ID(), actions[0])

		assert.Len(t, actions, 1)
		assert.NoError(t, err)
		assert.Equal(t, "new\n", doc.Text().String())
	})
}

func TestMultiCodeAction(t *testing.T) {
	t.Run("sorts and applies command action", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeMultiCodeActionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{
			core.NewRange(0, 3),
		}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)

		actions, err := session.CodeActions(doc, v.ID())
		assert.NoError(t, err)
		// 10 enabled: QuickFix + 8 refactor/source kinds + Command (disabled filtered out)
		assert.Len(t, actions, 10)
		// apply the Command action (last one after sort)
		var cmdAction view.CodeAction
		for _, a := range actions {
			if a.Title == "Run formatter" {
				cmdAction = a
			}
		}
		err = session.ApplyCodeAction(doc, v.ID(), cmdAction)
		assert.NoError(t, err)
	})
}

func writeCodeActionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCodeActionEnv + ` = "1" }

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

func TestCodeActionWithDiagnostics(t *testing.T) {
	t.Run("sorts actions by diagnostic fix status", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeDiagCodeActionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old bar\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{
			{
				Range:    view.DiagnosticRange{From: 0, To: 3},
				Message:  "old is wrong",
				Severity: view.DiagnosticSeverityError,
				Source:   "linter",
			},
			{
				Range:    view.DiagnosticRange{From: 0, To: 3},
				Message:  "old is warned",
				Severity: view.DiagnosticSeverityWarning,
			},
			{
				Range:    view.DiagnosticRange{From: 0, To: 3},
				Message:  "fyi",
				Severity: view.DiagnosticSeverityInfo,
			},
			{
				Range:    view.DiagnosticRange{From: 0, To: 3},
				Message:  "hint",
				Severity: view.DiagnosticSeverityHint,
			},
		})
		doc.SetSelectionFor(v.ID(), core.PointSelection(1))

		actions, err := session.CodeActions(doc, v.ID())

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(actions), 2)
		// First action fixes diagnostic (has diagnostics in request)
		assert.Equal(t, "Fix old", actions[0].Title)
	})
}

func writeDiagCodeActionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCodeActionEnv + ` = "1", ` +
		testServerDiagCodeActionEnv + ` = "1" }

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

func writeMultiCodeActionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCodeActionEnv + ` = "1", ` +
		testServerMultiCodeActionEnv + ` = "1" }

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

func TestCodeActionUnavailable(t *testing.T) {
	t.Run("stale action ID returns error", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		session := lsp.NewSession(t.Context(), t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		err := session.ApplyCodeAction(doc, v.ID(), view.CodeAction{ID: "server:0"})
		assert.ErrorIs(t, err, lsp.ErrCodeActionUnavailable)
	})
}
