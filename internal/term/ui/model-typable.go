package ui

import (
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
)

func (m Model) ExecTypable(input string) Model {
	_, m.component.cmdMsg = execTypable(m.context, input)
	return m
}

func execTypable(cx *Context, input string) (command.Signal, string) {
	if input == "" {
		return command.SignalNone, ""
	}
	name, rest, _ := command.SplitCommandLine(input)
	if name == "" {
		return command.SignalNone, ""
	}
	cmd, ok := cx.Keymaps.ResolveCommand(name)
	if !ok {
		return command.SignalNone,
			i18n.Text(i18n.ErrorNoSuchCommand, i18n.Vars{
				"name": name,
			})
	}
	expand := NewTokenExpander(cx.Editor)
	parsed, err := command.ParseArgs(rest, cmd.Signature, true, expand)
	if err != nil {
		return command.SignalNone, i18n.Text(
			i18n.ErrorMessage, i18n.Vars{"message": err},
		)
	}
	res := cmd.Run(cx.Editor, parsed)
	return res.Signal, res.Message
}
