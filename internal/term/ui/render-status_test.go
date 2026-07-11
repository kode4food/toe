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
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineSeparator},
			{Element: view.StatusLineFileBaseName},
			{Element: view.StatusLineFileAbsolutePath},
			{Element: view.StatusLinePercent},
			{Element: view.StatusLinePrimaryLen},
			{Element: view.StatusLineFileLineEnding},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileIndentStyle},
			{Element: view.StatusLineFileType},
			{Element: view.StatusLineRegister},
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
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineFileLineEnding},
			{Element: view.StatusLineFileIndentStyle},
			{Element: view.StatusLineFileType},
		}
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
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineReadOnly},
		}
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
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineDiagnostics},
		}
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
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineDiagnostics},
		}
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
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineTotalLines},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1 ")
	})
}

func TestStatuslineElementRegistry(t *testing.T) {
	cases := []struct {
		element view.StatusLineElement
		setup   func(t *testing.T) *view.Editor
		left    []view.StatusLineItem
		want    string
	}{
		{element: view.StatusLineMode, want: " NOR "},
		{element: view.StatusLineSeparator, want: "│"},
		{
			element: view.StatusLineFileBaseName,
			setup:   fileEditor,
			want:    "note.txt",
		},
		{
			element: view.StatusLineFileName,
			setup:   fileEditor,
			want:    "note.txt",
		},
		{
			element: view.StatusLineFileAbsolutePath,
			setup:   fileEditor,
			want:    "note.txt",
		},
		{
			element: view.StatusLineReadOnly,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				doc, ok := e.FocusedDocument()
				assert.True(t, ok)
				doc.SetReadOnly(true)
				return e
			},
			want: "[readonly]",
		},
		{
			element: view.StatusLineModified,
			setup: func(t *testing.T) *view.Editor {
				return editorWithText(t, "changed")
			},
			want: "[modified]",
		},
		{element: view.StatusLineFileEncoding, want: " utf-8 "},
		{element: view.StatusLineFileLineEnding, want: " lf "},
		{
			element: view.StatusLineFileIndentStyle,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				doc, ok := e.FocusedDocument()
				assert.True(t, ok)
				doc.SetIndentStyle(core.ParseIndentStyle("\t"))
				return e
			},
			want: " tabs ",
		},
		{element: view.StatusLineFileType, want: " text "},
		{
			element: view.StatusLineDiagnostics,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				doc, ok := e.FocusedDocument()
				assert.True(t, ok)
				doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
					{Severity: view.DiagnosticSeverityError},
				})
				return e
			},
			want: "E:1",
		},
		{element: view.StatusLineSelections, want: " 1 sel "},
		{
			element: view.StatusLinePrimaryLen,
			setup: func(t *testing.T) *view.Editor {
				e := editorWithText(t, "abcd")
				testutil.SetSelection(t, e,
					[]core.Range{core.NewRange(0, 2)}, 0,
				)
				return e
			},
			want: " 2 ",
		},
		{element: view.StatusLinePosition, want: " 1:1 "},
		{element: view.StatusLinePercent, want: "%"},
		{element: view.StatusLineTotalLines, want: " 1 "},
		{
			element: view.StatusLineSpacer,
			left: []view.StatusLineItem{
				{Element: view.StatusLineFileType},
				{Element: view.StatusLineSpacer},
				{Element: view.StatusLineFileType},
			},
			want: "text   text",
		},
		{
			element: view.StatusLineVersionControl,
			setup: func(t *testing.T) *view.Editor {
				testutil.RequireGit(t)
				e, s := repoEditor(t, "one\n", "one\nCHANGED\n")
				t.Cleanup(s.Close)
				return e
			},
			want: " main ",
		},
		{
			element: view.StatusLineSpinner,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				e.SetLanguageServerController(&completionController{busy: true})
				return e
			},
			want: "⠋",
		},
		{
			element: view.StatusLineRegister,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				e.SetRegister('a')
				return e
			},
			want: "reg=a",
		},
	}

	covered := make(map[view.StatusLineElement]bool, len(cases))
	for _, tc := range cases {
		covered[tc.element] = true
		t.Run(string(tc.element), func(t *testing.T) {
			e := view.NewEditor(t.TempDir())
			if tc.setup != nil {
				e = tc.setup(t)
			}
			left := tc.left
			if left == nil {
				left = []view.StatusLineItem{{Element: tc.element}}
			}
			e.Options().StatusLine.Left = left
			e.Options().StatusLine.Right = []view.StatusLineItem{
				{Element: view.StatusLineSpacer},
			}
			m := resize(ui.New(e, command.NewKeymaps()), 200, 24)

			out := stripANSI(m.View().Content)

			assert.Contains(t, out, tc.want)
		})
	}

	t.Run("covers every element", func(t *testing.T) {
		for _, e := range view.AllStatusLineElements {
			assert.True(t, covered[e], string(e))
		}
	})
}

func TestStatuslineEncoding(t *testing.T) {
	t.Run("utf-8 without bom", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "plain.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileEncoding},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " utf-8 ")
	})

	t.Run("utf-8 with bom", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "bom.txt")
		data := append([]byte{0xef, 0xbb, 0xbf}, []byte("hello\n")...)
		assert.NoError(t, os.WriteFile(path, data, 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileEncoding},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " utf-8-bom ")
	})
}

func TestStatuslineEdgeElements(t *testing.T) {
	t.Run("narrow width drops rightmost first", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "very-long-file-name.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineMode, Pinned: true},
			{Element: view.StatusLineSelections},
			{Element: view.StatusLineFileAbsolutePath},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineSpacer},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 14, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " NOR ")
		assert.Contains(t, out, "1 sel")
		assert.NotContains(t, out, "very-long-file-name")
	})

	t.Run("right section drops from its left", func(t *testing.T) {
		e := editorWithText(t, "hello")
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineSpacer},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileType},
			{Element: view.StatusLinePosition},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 7, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1:1 ")
		assert.NotContains(t, out, "text")
	})

	t.Run("pinned element survives narrow width", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "very-long-file-name.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineMode},
			{Element: view.StatusLineFileAbsolutePath},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLinePosition, Pinned: true},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 12, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1:1 ")
		assert.NotContains(t, out, "very-long-file-name")
	})

	t.Run("modified scratch appears", func(t *testing.T) {
		e := editorWithText(t, "changed")
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineModified},
		}
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
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineSelections},
			{Element: view.StatusLinePrimaryLen},
		}
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
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineFileIndentStyle},
			{Element: view.StatusLineFileType},
		}
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

func fileEditor(t *testing.T) *view.Editor {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
	e := view.NewEditor(root)
	_, err := e.OpenFile(path)
	assert.NoError(t, err)
	return e
}
