package lsp_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestDocumentOpenedPulls(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a language server process")
	}
	exe, err := os.Executable()
	assert.NoError(t, err)
	writePullDiagnosticsLanguages(t, exe)
	dir := t.TempDir()
	e := view.NewEditor(dir)
	session := lsp.Attach(t.Context(), e)
	defer func() { _ = session.Close() }()

	// the first document roots the server at the workspace, which is what
	// makes the second document out-of-workspace
	inside := openPullDocument(t, e, dir)
	assert.Eventually(t, func() bool {
		return len(inside.Diagnostics()) > 0
	}, time.Second, 10*time.Millisecond)

	outside := openPullDocument(t, e, t.TempDir())
	assert.Never(t, func() bool {
		return len(outside.Diagnostics()) > 0
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func openPullDocument(t *testing.T, e *view.Editor, dir string) *view.Document {
	t.Helper()
	path := filepath.Join(dir, "main.session")
	assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
	_, err := e.OpenFile(path)
	assert.NoError(t, err)
	doc := e.FocusedDocument()
	assert.NotNil(t, doc)
	return doc
}
