package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestDiagnosticPicker(t *testing.T) {
	t.Run("accept selects diagnostic", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		text := "package main\nfunc main() {}\n"
		assert.NoError(t, os.WriteFile(path, []byte(text), 0o644))

		e := view.NewEditor(dir)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc := e.Document(v.DocID())
		assert.NotNil(t, doc)
		from := strings.Index(text, "main()")
		assert.NotEqual(t, -1, from)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{
			{
				Range: view.DiagnosticRange{
					From: from,
					To:   from + len("main"),
				},
				Severity: view.DiagnosticSeverityError,
				Message:  "bad main",
				Source:   "test",
				Provider: "test",
			},
		})

		m := openDiagnosticPicker(e, files.NewDiagnosticPicker, 'd')
		_ = sendSpecial(m, tea.KeyEnter)

		v = e.FocusedView()
		assert.NotNil(t, v)
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, from, sel.Primary().Cursor(doc.Text()))
	})

	t.Run("workspace lists open documents", func(t *testing.T) {
		dir := t.TempDir()
		a := filepath.Join(dir, "a.go")
		b := filepath.Join(dir, "b.go")
		assert.NoError(t, os.WriteFile(a, []byte("package a\n"), 0o644))
		assert.NoError(t, os.WriteFile(b, []byte("package b\n"), 0o644))

		e := view.NewEditor(dir)
		docA, err := e.SwitchOrOpenDoc(a)
		assert.NoError(t, err)
		docB, err := e.SwitchOrOpenDoc(b)
		assert.NoError(t, err)
		docA.ReplaceDiagnostics("test", []view.Diagnostic{
			{
				Severity: view.DiagnosticSeverityWarning,
				Message:  "bad a",
				Source:   "test",
				Provider: "test",
			},
		})
		docB.ReplaceDiagnostics("test", []view.Diagnostic{
			{
				Severity: view.DiagnosticSeverityError,
				Message:  "bad b",
				Source:   "test",
				Provider: "test",
			},
		})

		m := openDiagnosticPicker(e, files.NewWorkspaceDiagnosticPicker, 'D')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "bad a")
		assert.Contains(t, out, "bad b")
		assert.NotContains(t, out, "code")
		assert.NotContains(t, out, "source")
		assert.Contains(t, out, "message")
		assert.Contains(t, out, "path")
		assert.Less(t,
			strings.Index(out, "message"), strings.Index(out, "path"))
		// grouping is shared with the current-file picker
		assert.Greater(t,
			sectionRow(out, "Warnings"), sectionRow(out, "Errors"),
		)
	})

	t.Run("groups by severity", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.go")
		assert.NoError(t, os.WriteFile(path, []byte("package a\n"), 0o644))

		e := view.NewEditor(dir)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc := e.Document(v.DocID())
		assert.NotNil(t, doc)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{
			{
				Severity: view.DiagnosticSeverityWarning,
				Message:  "careful", Source: "test", Provider: "test",
			},
			{
				Severity: view.DiagnosticSeverityError,
				Message:  "broken", Source: "test", Provider: "test",
			},
		})

		m := openDiagnosticPicker(e, files.NewDiagnosticPicker, 'd')
		out := stripANSI(m.View().Content)

		errors := sectionRow(out, "Errors")
		warnings := sectionRow(out, "Warnings")
		assert.GreaterOrEqual(t, errors, 0)
		assert.Greater(t, warnings, errors)
		// no info or hint diagnostics, so those labels stay hidden
		assert.Equal(t, -1, sectionRow(out, "Information"))
		assert.Equal(t, -1, sectionRow(out, "Hints"))
	})

	for _, tc := range []struct {
		name string
		sev  view.DiagnosticSeverity
		want string
	}{
		{"error icon", view.DiagnosticSeverityError, ""},
		{"warning icon", view.DiagnosticSeverityWarning, ""},
		{"info icon", view.DiagnosticSeverityInfo, ""},
		{"hint icon", view.DiagnosticSeverityHint, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "main.go")
			assert.NoError(t,
				os.WriteFile(path, []byte("package main\n"), 0o644))

			e := view.NewEditor(dir)
			v, err := e.OpenFile(path)
			assert.NoError(t, err)
			doc := e.Document(v.DocID())
			assert.NotNil(t, doc)
			doc.ReplaceDiagnostics("test", []view.Diagnostic{
				{
					Severity: tc.sev,
					Message:  "bad main",
					Source:   "test",
					Provider: "test",
				},
			})

			m := openDiagnosticPicker(e, files.NewDiagnosticPicker, 'd')
			out := stripANSI(m.View().Content)
			assert.Contains(t, out, tc.want)
			assert.NotContains(t, out, "message")
		})
	}
}

func openDiagnosticPicker(
	e *view.Editor, fn ui.PickerFunc, key rune,
) ui.Model {
	km := command.NewKeymaps()
	m := ui.New(e, km)
	event := char(key)
	if key >= 'A' && key <= 'Z' {
		event = event.WithMods(command.ModShift)
	}
	bindNormalTestAction(
		km, "diagnostic_picker", m.PickerAction(fn),
		[]command.KeyEvent{event},
	)
	m = resize(m, 120, 30)
	if key >= 'A' && key <= 'Z' {
		return sendSpecialText(m, key, string(key))
	}
	return sendKey(m, key)
}
