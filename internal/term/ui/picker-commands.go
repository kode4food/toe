package ui

import (
	"strings"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type commandPaletteSource struct {
	PickerBase
	km *command.Keymaps
}

// CommandPalettePicker opens a picker listing all registered commands
func CommandPalettePicker(e *view.Editor, km *command.Keymaps) *Picker {
	return NewPicker(e, &commandPaletteSource{
		PickerBase: PickerBase{
			id:          "command-palette",
			columns:     []string{"name", "bindings", "doc"},
			proportions: []int{0, 1, 2},
		},
		km: km,
	})
}

func (c *commandPaletteSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	cmds := c.km.Commands()
	items := make([]PickerItem, 0, len(cmds))
	mode := e.Mode().String()
	for _, cmd := range cmds {
		if cmd.Run == nil || len(cmd.Aliases) == 0 {
			continue
		}
		name := cmd.Aliases[0]
		items = append(items, PickerItem{
			Display: name,
			Columns: []string{
				name, commandKeyString(c.km, mode, cmd.Name), cmd.DocString,
			},
			SortKey: name,
			Payload: cmd,
		})
	}
	return items, nil, func() {}
}

func (c *commandPaletteSource) Accept(
	e *view.Editor, item PickerItem, _ PickerAcceptAction,
) {
	cmd, ok := item.Payload.(command.Command)
	if !ok || cmd.Run == nil {
		return
	}
	cmd.Run(e, nil)
}

func (c *commandPaletteSource) SkipPreview() {}

func commandKeyString(km *command.Keymaps, mode, name string) string {
	return commandModeKeyString(km.Bindings(mode, name))
}

func commandModeKeyString(bindings []command.KeyBinding) string {
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		for _, seq := range binding {
			if len(seq) == 0 {
				continue
			}
			parts = append(parts, commandKeySeqString(seq))
		}
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
