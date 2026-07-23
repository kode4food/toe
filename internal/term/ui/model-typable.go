package ui

import (
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
)

var errNoSuchCommand = i18n.NewError(i18n.ErrorNoSuchCommand)

func (m Model) ExecTypable(input string) Model {
	res := execTypable(m.context, input)
	m.component.setCommandResult(res)
	return m
}

func execTypable(cx *Context, input string) command.Result {
	if input == "" {
		return command.Result{}
	}
	name, rest, _ := command.SplitCommandLine(input)
	if name == "" {
		return command.Result{}
	}
	cmd, ok := cx.Keymaps.ResolveCommandIn(cx.Editor.Mode().String(), name)
	if !ok {
		return command.Result{
			Error: errNoSuchCommand.WithVars(i18n.Vars{
				"name": name,
			}),
		}
	}
	expand := NewTokenExpander(cx.Editor)
	parsed, err := command.ParseArgs(rest, cmd.Signature, true, expand)
	if err != nil {
		return command.Result{Error: err}
	}
	return cmd.Run(cx.Editor, parsed)
}
