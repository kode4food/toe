package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestCommandPalettePicker(t *testing.T) {
	t.Run("lists registered commands", func(t *testing.T) {
		m, _ := paletteModel(t)
		assert.Contains(t, stripANSI(m.View().Content), "palette_probe")
	})

	t.Run("shows key bindings", func(t *testing.T) {
		m, _ := paletteModel(t)
		m = resize(m, 60, 30)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "bindings")
		assert.Contains(t, out, "description")
		assert.Contains(t, out, "gp")
		assert.Contains(t, out, "A command palette row")
	})

	t.Run("runs an argument-free command", func(t *testing.T) {
		m, e := paletteModel(t)
		for _, ch := range "palette_probe" {
			m = sendKey(m, ch)
		}
		_ = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, view.ModeInsert, e.Mode())
	})

	t.Run("prompts for a command taking args", func(t *testing.T) {
		m, e := paletteModel(t)
		for _, ch := range "arg_probe" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, view.ModeNormal, e.Mode())
		assert.Contains(t, stripANSI(m.View().Content), ": arg_probe")
	})

	t.Run("runs the prompted command with its args", func(t *testing.T) {
		m, e := paletteModel(t)
		for _, ch := range "arg_probe" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		for _, ch := range "hello" {
			m = sendKey(m, ch)
		}
		_ = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, view.ModeSelect, e.Mode())
	})

	t.Run("filters by mode", func(t *testing.T) {
		root := t.TempDir()
		e := view.NewEditor(root)
		openRenderImagePane(t, e, writeRenderImage(t, root, 4, 4, nil))
		km := command.NewKeymaps()
		_ = km.Register("image_probe", command.Command{
			DocString: "Image command",
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{}
			},
			Aliases: []string{"image_probe"},
			Modes:   view.ModeImage,
		})
		_ = km.Register("document_probe", command.Command{
			DocString: "Document command",
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{}
			},
			Aliases: []string{"document_probe"},
			Modes:   view.ModeNormal,
		})
		m := ui.New(e, km)
		m = m.WithInitialPicker(m.CommandPalettePicker)
		m = resize(m, 80, 24)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "image_probe")
		assert.NotContains(t, out, "document_probe")
	})
}

// paletteModel binds a probe command and opens the palette with 'p', returning
// the editor so the test can observe the accepted command
func paletteModel(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_ = km.Register("palette_probe", command.Command{
		DocString: "A command palette row with enough doc text to overflow",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.SetMode(view.ModeInsert)
			return command.Result{}
		},
		Aliases: []string{"palette_probe"},
		Modes:   view.ModeNormal,
		Keys: map[view.Mode]command.KeyBinding{
			view.ModeAny: {{char('g'), char('p')}},
		},
	})
	_ = km.Register("arg_probe", command.Command{
		DocString: "A command palette row taking an optional argument",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if arg, ok := args.First(); ok && arg == "hello" {
				e.SetMode(view.ModeSelect)
			}
			return command.Result{}
		},
		Aliases: []string{"arg_probe"},
		Modes:   view.ModeNormal,
		Signature: command.Signature{
			Positionals: command.Positionals{Min: 0, Max: 1},
		},
	})
	km.Bind(view.ModeNormal, "palette_probe", []command.KeyEvent{
		char('a'), char('b'), char('c'), char('d'), char('e'), char('f'),
		char('g'), char('h'), char('i'), char('j'), char('k'), char('l'),
	})
	bindNormalTestAction(
		km, "open_palette", m.PickerAction(m.CommandPalettePicker),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 100, 30)
	return sendKey(m, 'p'), e
}
