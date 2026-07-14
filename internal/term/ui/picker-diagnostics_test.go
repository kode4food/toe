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
		doc, ok := e.Document(v.DocID())
		assert.True(t, ok)
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

		v, ok = e.FocusedView()
		assert.True(t, ok)
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
		assert.Contains(t, out, "severity")
	})

	for _, tc := range []struct {
		name string
		sev  view.DiagnosticSeverity
		want string
	}{
		{"colors error severity", view.DiagnosticSeverityError, "ERROR"},
		{"colors warning severity", view.DiagnosticSeverityWarning, "WARN"},
		{"colors info severity", view.DiagnosticSeverityInfo, "INFO"},
		{"colors hint severity", view.DiagnosticSeverityHint, "HINT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "main.go")
			assert.NoError(t,
				os.WriteFile(path, []byte("package main\n"), 0o644))

			e := view.NewEditor(dir)
			v, err := e.OpenFile(path)
			assert.NoError(t, err)
			doc, ok := e.Document(v.DocID())
			assert.True(t, ok)
			doc.ReplaceDiagnostics("test", []view.Diagnostic{
				{
					Severity: tc.sev,
					Message:  "bad main",
					Source:   "test",
					Provider: "test",
				},
			})

			m := openDiagnosticPicker(e, files.NewDiagnosticPicker, 'd')
			assert.Contains(t, stripANSI(m.View().Content), tc.want)
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
