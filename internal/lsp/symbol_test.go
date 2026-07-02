package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestSymbol(t *testing.T) {
	t.Run("flattens document symbols", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeSymbolLanguages(t, exe)
		text := "func outer() {}\nvar inner int\n"
		assert.NoError(t, os.WriteFile(path, []byte(text), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		symbols, err := session.DocumentSymbols(doc)

		assert.NoError(t, err)
		assert.Equal(t, []view.Symbol{
			{
				Name: "outer", Kind: "function",
				Location: view.Location{Path: path, From: 5, To: 10},
			},
			{
				Name: "inner", Kind: "variable", Container: "outer",
				Location: view.Location{Path: path, From: 20, To: 25},
			},
		}, symbols)
	})

	t.Run("requests workspace symbols", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeWorkspaceSymbolLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		symbols, err := session.WorkspaceSymbols(doc, "main")

		assert.NoError(t, err)
		assert.Equal(t, []view.Symbol{
			{
				Name: "WorkspaceMain", Kind: "function",
				Container: "workspace",
				Location:  view.Location{Path: target, From: 3, To: 6},
			},
		}, symbols)
	})
}

func TestManySymbolKinds(t *testing.T) {
	t.Run("maps all symbol kinds", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeManySymbolLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("x\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		symbols, err := session.DocumentSymbols(doc)

		assert.NoError(t, err)
		assert.NotEmpty(t, symbols)
	})
}

func TestWorkspaceSymbolSlice(t *testing.T) {
	t.Run("handles WorkspaceSymbolSlice result", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		source := filepath.Join(dir, "main.session")
		target := filepath.Join(dir, "target.session")
		writeWorkspaceSymbolSliceLanguages(t, exe, target)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("target\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		symbols, err := session.WorkspaceSymbols(doc, "main")

		assert.NoError(t, err)
		assert.Equal(t, []view.Symbol{
			{
				Name: "WorkspaceMain", Kind: "function",
				Container: "workspace",
				Location:  view.Location{Path: target, From: 3, To: 6},
			},
		}, symbols)
	})
}

func writeSymbolLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerSymbolsEnv + ` = "1" }

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

func writeWorkspaceSymbolLanguages(t *testing.T, exe, target string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerWorkspaceSymbolsEnv + ` = "1", ` +
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

func writeWorkspaceSymbolSliceLanguages(t *testing.T, exe, target string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerWorkspaceSymbolsEnv + ` = "1", ` +
		testServerSymbolSliceEnv + ` = "1", ` +
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

func writeManySymbolLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerSymbolsEnv + ` = "1", ` +
		testServerManySymbolsEnv + ` = "1" }

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
