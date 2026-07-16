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
		assert.Contains(t, out, "doc")
		assert.Contains(t, out, "gp")
		assert.Contains(t, out, "A command palette row")
	})

	t.Run("accepts and runs the command", func(t *testing.T) {
		m, e := paletteModel(t)
		for _, ch := range "palette_probe" {
			m = sendKey(m, ch)
		}
		_ = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, view.ModeInsert, e.Mode())
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
		Modes:   []string{"NOR"},
		Keys: map[string][]command.KeyBinding{
			"*": {
				{
					{char('g'), char('p')},
				},
			},
		},
	})
	km.Bind("NOR", "palette_probe", []command.KeyEvent{
		char('a'), char('b'), char('c'), char('d'), char('e'), char('f'),
		char('g'), char('h'), char('i'), char('j'), char('k'), char('l'),
	})
	bindNormalTestAction(
		km, "open_palette",
		m.PickerAction(func(ed *view.Editor) *ui.Picker {
			return ui.CommandPalettePicker(ed, km)
		}),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 100, 30)
	return sendKey(m, 'p'), e
}
