package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestDiagnostics(t *testing.T) {
	t.Run("stores published diagnostics", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("😀x\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()

		err = session.PublishDiagnostics(
			t.Context(),
			&protocol.PublishDiagnosticsParams{
				URI: uri.File(path),
				Diagnostics: []protocol.Diagnostic{
					{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 2},
						},
						Severity: protocol.DiagnosticSeverityError,
						Source:   protocol.NewOptional("gopls"),
						Message:  protocol.String("bad identifier"),
					},
				},
			},
		)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		diags := doc.Diagnostics()

		assert.NoError(t, err)
		assert.Len(t, diags, 1)
		assert.Equal(t, view.DiagnosticSeverityError, diags[0].Severity)
		assert.Equal(t, "bad identifier", diags[0].Message)
		assert.Equal(t, "gopls", diags[0].Source)
		assert.Equal(t, "lsp", diags[0].Provider)
		assert.Equal(t, view.DiagnosticRange{From: 0, To: 1}, diags[0].Range)
	})

	t.Run("ignores stale versions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer session.Close()

		err = session.PublishDiagnostics(
			t.Context(),
			&protocol.PublishDiagnosticsParams{
				URI:     uri.File(path),
				Version: protocol.NewOptional[int32](999),
				Diagnostics: []protocol.Diagnostic{
					{
						Range:   protocol.Range{},
						Message: protocol.String("stale"),
					},
				},
			},
		)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.NoError(t, err)
		assert.Empty(t, doc.Diagnostics())
	})
}
