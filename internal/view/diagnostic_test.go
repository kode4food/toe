package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestDiagnostics(t *testing.T) {
	t.Run("replaces provider diagnostics", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{
				Severity: view.DiagnosticSeverityError,
				Message:  "old",
				Provider: "gopls",
			},
		})
		doc.ReplaceDiagnostics("other", []view.Diagnostic{
			{
				Severity: view.DiagnosticSeverityWarning,
				Message:  "kept",
				Provider: "other",
			},
		})
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{
				Severity: view.DiagnosticSeverityInfo,
				Message:  "new",
				Provider: "gopls",
			},
		})

		diags := doc.Diagnostics()

		assert.Len(t, diags, 2)
		assert.Equal(t, "kept", diags[0].Message)
		assert.Equal(t, "new", diags[1].Message)
	})

	t.Run("counts severities", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityError},
			{Severity: view.DiagnosticSeverityWarning},
			{Severity: view.DiagnosticSeverityInfo},
			{Severity: view.DiagnosticSeverityHint},
		})

		counts := doc.DiagnosticCounts()

		assert.Equal(t, 1, counts.Errors)
		assert.Equal(t, 1, counts.Warnings)
		assert.Equal(t, 1, counts.Info)
		assert.Equal(t, 1, counts.Hints)
	})

	t.Run("clears diagnostics", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityError},
		})

		doc.ClearDiagnostics()

		assert.Empty(t, doc.Diagnostics())
	})
}
