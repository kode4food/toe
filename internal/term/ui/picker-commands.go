package ui

import (
	"strings"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type commandPaletteSource struct {
	PickerBase
	keymaps *command.Keymaps
}

// CommandPalettePicker opens a picker listing all registered commands
func CommandPalettePicker(e *view.Editor, km *command.Keymaps) *Picker {
	return NewPicker(e, &commandPaletteSource{
		PickerBase: PickerBase{
			Ident:       "command-palette",
			Label:       "Command Palette",
			Cols:        []string{"name", "bindings", "doc"},
			Proportions: []int{0, 1, 2},
		},
		keymaps: km,
	})
}

// Load lists every command available in the current mode
func (c *commandPaletteSource) Load(e *view.Editor) PickerLoad {
	mode := e.Mode()
	cmds := c.keymaps.CommandsIn(mode)
	items := make([]*PickerItem, 0, len(cmds))
	var slab PickerItemSlab
	for _, cmd := range cmds {
		if cmd.Run == nil || len(cmd.Aliases) == 0 {
			continue
		}
		name := cmd.Aliases[0]
		items = append(items, slab.Add(PickerItem{
			Display: name,
			Columns: []string{
				name, commandKeyString(c.keymaps, mode, cmd.Name),
				cmd.DocString,
			},
			SortKey: name,
			Payload: cmd,
		}))
	}
	return PickerLoad{Items: items, Stop: func() {}}
}

// Accept runs the chosen command
func (c *commandPaletteSource) Accept(
	e *view.Editor, item *PickerItem, _ PickerAcceptAction,
) {
	cmd, ok := item.Payload.(*command.Command)
	if !ok || cmd.Run == nil || len(cmd.Aliases) == 0 {
		return
	}
	if c.keymaps.ResolveCommandIn(e.Mode(), cmd.Aliases[0]) == nil {
		return
	}
	cmd.Run(e, nil)
}

// SkipPreview leaves the palette without a preview pane
func (c *commandPaletteSource) SkipPreview() {}

func commandKeyString(km *command.Keymaps, mode view.Mode, name string) string {
	return commandModeKeyString(km.Bindings(mode, name))
}

func commandModeKeyString(bindings command.KeyBinding) string {
	parts := make([]string, 0, len(bindings))
	for _, seq := range bindings {
		if len(seq) == 0 {
			continue
		}
		parts = append(parts, commandKeySeqString(seq))
	}
	return strings.Join(parts, " ")
}

func commandKeySeqString(seq []command.KeyEvent) string {
	var b strings.Builder
	for _, ev := range seq {
		b.WriteString(commandKeyEventString(ev))
	}
	return b.String()
}

func commandKeyEventString(ev command.KeyEvent) string {
	s := ev.String()
	if s == " " {
		s = "space"
	}
	if len([]rune(s)) > 1 {
		return "<" + s + ">"
	}
	return s
}
