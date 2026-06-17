package ui

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type commandPaletteSource struct {
	pickerMeta
	km *command.Keymaps
}

// CommandPalettePicker opens a picker listing all registered commands
func CommandPalettePicker(e *view.Editor, km *command.Keymaps) *Picker {
	return NewPicker(e, &commandPaletteSource{
		pickerMeta: pickerMeta{
			title:   "Command palette",
			columns: []string{"name", "description"},
		},
		km: km,
	})
}

func (c *commandPaletteSource) Load(
	_ *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	cmds := c.km.Commands()
	items := make([]PickerItem, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd.Run == nil || len(cmd.Aliases) == 0 {
			continue
		}
		name := cmd.Aliases[0]
		items = append(items, PickerItem{
			Display: name,
			Columns: []string{name, cmd.DocString},
			SortKey: name,
			Payload: cmd,
		})
	}
	return items, nil, func() {}
}

func (c *commandPaletteSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, c.Columns(), c.Primary())
}

func (c *commandPaletteSource) Accept(e *view.Editor, item PickerItem) {
	cmd, ok := item.Payload.(command.Command)
	if !ok || cmd.Run == nil {
		return
	}
	cmd.Run(e, nil)
}
