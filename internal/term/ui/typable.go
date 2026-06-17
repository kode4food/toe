package ui

import (
	"fmt"

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
			fmt.Sprintf("error: no such command: `%s`", name)
	}
	expand := NewTokenExpander(cx.Editor)
	parsed, err := command.ParseArgs(rest, cmd.Signature, true, expand)
	if err != nil {
		return command.SignalNone, "error: " + err.Error()
	}
	res := cmd.Run(cx.Editor, parsed)
	return res.Signal, res.Message
}
