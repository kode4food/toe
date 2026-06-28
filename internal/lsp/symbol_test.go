package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
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
		defer session.Close()
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
