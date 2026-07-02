package lsp_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestSessionGopls(t *testing.T) {
	t.Run("requests completion", func(t *testing.T) {
		if os.Getenv("TOE_LSP_GOPLS_SMOKE") != "1" {
			t.Skip("set TOE_LSP_GOPLS_SMOKE=1")
		}
		gopls, err := exec.LookPath("gopls")
		if err != nil {
			t.Skip("gopls not found")
		}

		dir := t.TempDir()
		writeGoplsLanguages(t, gopls)
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "go.mod"),
			[]byte("module smoke\n\ngo 1.26\n"),
			0o644,
		))
		text := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Pr\n}\n"
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte(text), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(len(text)-3))

		res, err := session.Completions(doc, v.ID())
		items := res.Items

		assert.NoError(t, err)
		assert.NotEmpty(t, items)
	})
}

func writeGoplsLanguages(t *testing.T, gopls string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	args := ""
	if log := os.Getenv("TOE_LSP_GOPLS_LOG"); log != "" {
		args = fmt.Sprintf("args = [%q, %q]\n", "-rpc.trace", "-logfile="+log)
	}
	text := fmt.Sprintf(`
[language-server.gopls]
command = %q
%s

[[language]]
name = "go"
language-id = "go"
file-types = ["go"]
roots = ["go.mod"]
language-servers = ["gopls"]
`, gopls, args)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
