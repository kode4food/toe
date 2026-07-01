package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestModelLifecycle(t *testing.T) {
	newModel := func() ui.Model {
		e := view.NewEditor(t.TempDir())
		return resize(ui.New(e, command.NewKeymaps()), 80, 24)
	}

	t.Run("init produces a renderable model", func(t *testing.T) {
		m := newModel()
		_ = m.Init()
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("view before resize is empty", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps())
		v := m.View()

		assert.Equal(t, "", v.Content)
		assert.True(t, v.AltScreen)
	})

	t.Run("completion options round trip", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps())

		assert.Equal(t,
			ui.CompletionIconsCodicon, m.CompletionOptions().Icons,
		)

		m.SetCompletionOptions(ui.CompletionOptions{
			Icons: ui.CompletionIconsASCII,
		})

		assert.Equal(t, ui.CompletionIconsASCII, m.CompletionOptions().Icons)

		m.SetCompletionOptions(ui.CompletionOptions{})

		assert.Equal(t,
			ui.CompletionIconsCodicon, m.CompletionOptions().Icons,
		)
	})

	t.Run("with startup cmd renders", func(t *testing.T) {
		m := newModel().WithStartupCmd(nil)
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("initial picker mounts on resize", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps()).WithInitialPicker(ui.FilePicker)
		m = resize(m, 80, 24)
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("initial nil picker is ignored", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps()).WithInitialPicker(
			func(_ *view.Editor) *ui.Picker { return nil },
		)
		m = resize(m, 80, 24)
		assert.NotEmpty(t, m.View().Content)
	})
}

func TestCommandPaletteAction(t *testing.T) {
	t.Run("opens command palette picker", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "palette", m.CommandPaletteAction(),
			[]command.KeyEvent{char('f')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 'f')
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func TestLastPickerAction(t *testing.T) {
	t.Run("no last picker is noop", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "last", m.LastPickerAction(), []command.KeyEvent{char('l')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 'l')
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("reopens last picker", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "palette", m.CommandPaletteAction(),
			[]command.KeyEvent{char('f')},
		)
		bindNormalTestAction(
			km, "last", m.LastPickerAction(), []command.KeyEvent{char('l')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 'f')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		m = m2.(ui.Model)
		m = sendKey(m, 'l')
		assert.NotEmpty(t, m.View().Content)
	})
}

func TestShellAction(t *testing.T) {
	t.Run("opens shell prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("shell", command.Command{
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				fn := func(_ *view.Editor, _ string) error { return nil }
				return command.Result{Continuation: m.ShellAction("$", fn)(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{
				"*": {[][]command.KeyEvent{{char('!')}}},
			},
		})
		m = resize(m, 80, 24)
		m = sendKey(m, '!')
		out := m.View().Content
		assert.Contains(t, out, "$")
	})
}
