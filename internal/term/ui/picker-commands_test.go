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

	t.Run("accepts and runs the command", func(t *testing.T) {
		m, e := paletteModel(t)
		for _, ch := range "palette_probe" {
			m = sendKey(m, ch)
		}
		_ = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, view.ModeInsert, e.Mode())
	})
}

// paletteModel registers a probe command that switches to insert mode, then
// binds 'p' to open the command palette. The editor is returned so the test
// can observe the probe command's effect after it is accepted
func paletteModel(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	km.Register("palette_probe", command.Command{
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.SetMode(view.ModeInsert)
			return command.Result{}
		},
		Aliases: []string{"palette_probe"},
		Modes:   []string{"NOR"},
	})
	bindNormalTestAction(
		km, "open_palette",
		m.PickerAction(func(ed *view.Editor) *ui.Picker {
			return ui.CommandPalettePicker(ed, km)
		}),
		[]command.KeyEvent{command.Char('p')},
	)
	m = resize(m, 100, 30)
	return sendKey(m, 'p'), e
}
