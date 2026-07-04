package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestStatuslineAllElements(t *testing.T) {
	t.Run("renders file-based status elements", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "note.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetRegister('a')
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineElement{
			view.StatusLineSeparator,
			view.StatusLineFileBaseName,
			view.StatusLineFileAbsolutePath,
		}
		opts.StatusLine.Center = []view.StatusLineElement{
			view.StatusLinePercent,
			view.StatusLinePrimaryLen,
			view.StatusLineFileLineEnding,
		}
		opts.StatusLine.Right = []view.StatusLineElement{
			view.StatusLineFileIndentStyle,
			view.StatusLineFileType,
			view.StatusLineRegister,
		}
		m := resize(ui.New(e, command.NewKeymaps()), 200, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "note.txt")
		assert.Contains(t, out, "reg=a")
	})
}

func TestStatuslineAltBranches(t *testing.T) {
	t.Run("crlf and tabs indent style", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package p\r\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetLineEnding(core.LineEndingCRLF)
		doc.SetIndentStyle(core.ParseIndentStyle("\t"))
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineElement{
			view.StatusLineFileLineEnding,
			view.StatusLineFileIndentStyle,
			view.StatusLineFileType,
		}
		opts.StatusLine.Center = nil
		opts.StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 200, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "crlf")
		assert.Contains(t, out, "tabs")
		assert.Contains(t, out, "go")
	})
}

func TestStatuslineReadOnly(t *testing.T) {
	t.Run("readonly indicator appears", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetReadOnly(true)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineElement{
			view.StatusLineReadOnly,
		}
		opts.StatusLine.Center = nil
		opts.StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[readonly]")
	})
}

func TestStatuslineDiagnostics(t *testing.T) {
	t.Run("renders diagnostic counts", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityError},
			{Severity: view.DiagnosticSeverityWarning},
			{Severity: view.DiagnosticSeverityHint},
		})
		e.Options().StatusLine.Left = []view.StatusLineElement{
			view.StatusLineDiagnostics,
		}
		e.Options().StatusLine.Center = nil
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "E:1")
		assert.Contains(t, out, "W:1")
		assert.Contains(t, out, "H:1")
	})

	t.Run("info and hint counts", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityInfo},
			{Severity: view.DiagnosticSeverityHint},
		})
		e.Options().StatusLine.Left = []view.StatusLineElement{
			view.StatusLineDiagnostics,
		}
		e.Options().StatusLine.Center = nil
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "I:1")
		assert.Contains(t, out, "H:1")
	})
}

func TestThemeStyleBranches(t *testing.T) {
	t.Run("ao theme covers extended style keys", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "ao"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.NotEmpty(t, out)
	})
}

func TestModeColorRender(t *testing.T) {
	t.Run("applies mode color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;245;224;220m NOR ")
	})
}

func TestStatuslineConfigRender(t *testing.T) {
	t.Run("uses configured mode label", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.Options().StatusLine.Mode.Normal = "NORMAL"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, " NORMAL ")
		assert.NotContains(t, out, " NOR ")
	})
}

func TestStatuslineTotalLines(t *testing.T) {
	t.Run("total-line-numbers appears in status", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().StatusLine.Right = []view.StatusLineElement{
			view.StatusLineTotalLines,
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1 ")
	})
}

func TestStatuslineEdgeElements(t *testing.T) {
	t.Run("narrow width drops low priority groups", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "very-long-file-name.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineElement{
			view.StatusLineMode,
			view.StatusLineFileAbsolutePath,
			view.StatusLineSelections,
			view.StatusLineTotalLines,
		}
		opts.StatusLine.Center = nil
		opts.StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 18, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " NOR ")
		assert.NotContains(t, out, "very-long-file-name")
	})

	t.Run("modified scratch appears", func(t *testing.T) {
		e := editorWithText(t, "changed")
		e.Options().StatusLine.Left = []view.StatusLineElement{
			view.StatusLineModified,
		}
		e.Options().StatusLine.Center = nil
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[modified]")
	})

	t.Run("plural selections include primary", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(0), core.PointRange(2)},
			1,
		)
		e.Options().StatusLine.Left = []view.StatusLineElement{
			view.StatusLineSelections,
			view.StatusLinePrimaryLen,
		}
		e.Options().StatusLine.Center = nil
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 2/2 sels ")
		assert.Contains(t, out, " 0 ")
	})

	t.Run("spaces indent and default file type", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.SetIndentStyle(core.Spaces(2))
		doc.SetLang("")
		e.Options().StatusLine.Left = []view.StatusLineElement{
			view.StatusLineFileIndentStyle,
			view.StatusLineFileType,
		}
		e.Options().StatusLine.Center = nil
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " spaces:2 ")
		assert.Contains(t, out, " text ")
	})
}

func TestCommandlineThemeRender(t *testing.T) {
	t.Run("applies commandline styles", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(root)
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		prompt := sendKey(m, ':').View().Content
		errOut := m.ExecTypable("not-a-command").View().Content

		assert.Contains(t, prompt, "\x1b[38;2;205;214;244m")
		assert.Contains(t, errOut, "\x1b[38;2;243;139;168m")
	})
}
