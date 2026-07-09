package lsp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestSessionConcurrentClientStart(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)

	marker := filepath.Join(t.TempDir(), "starts")
	dir := t.TempDir()
	writeConcurrentStartLanguages(t, exe, marker)
	path := filepath.Join(dir, "main.session")
	assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))

	e := view.NewEditor(dir)
	_, err = e.OpenFile(path)
	assert.NoError(t, err)
	session := lsp.Attach(t.Context(), e)
	defer func() { _ = session.Close() }()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	v, ok := e.FocusedView()
	assert.True(t, ok)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_ = session.PullDiagnostics(doc)
			_, _ = session.DocumentLinks(doc)
			_, _ = session.DocumentColors(doc)
			_, _ = session.InlayHints(doc, v.ID())
			_ = session.WorkspaceCommands(doc)
		})
	}
	wg.Wait()

	data, err := os.ReadFile(marker)
	assert.NoError(t, err)
	starts := strings.Count(string(data), "x\n")
	assert.Equal(t, 1, starts, "server process should start exactly once")
}

func writeConcurrentStartLanguages(t *testing.T, exe, marker string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = %q, %s = "50" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerStartMarkerEnv, marker, testServerInitDelayMsEnv)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}
