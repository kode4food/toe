package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

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
