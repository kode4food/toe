package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
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
