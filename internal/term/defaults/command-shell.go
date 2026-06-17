package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actShellPipe         = "shell_pipe"
	actShellInsertOutput = "shell_insert_output"
	actShellKeepPipe     = "shell_keep_pipe"
	actShellPipeTo       = "shell_pipe_to"
	actShellAppendOutput = "shell_append_output"
)

func registerShellCommands(r *registry, model ui.Model) {
	r.RegisterCommand(actShellPipe, command.Command{
		DocString: "Pipe selections through shell command",
		Run:       Continuation(model.ShellAction("|", action.ShellPipe)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('|')),
	})
	r.RegisterCommand(actShellInsertOutput, command.Command{
		DocString: "Insert shell command output before selections",
		Run:       Continuation(model.ShellAction("!", action.ShellInsertOutput)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('!')),
	})
	r.RegisterCommand(actShellKeepPipe, command.Command{
		DocString: "Filter selections with shell predicate",
		Run:       Continuation(model.ShellAction("$", action.ShellKeepPipe)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('$')),
	})
	r.RegisterCommand(actShellPipeTo, command.Command{
		DocString: "Pipe selections into shell command ignoring output",
		Run:       Continuation(model.ShellAction("alt+|", action.ShellPipeTo)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('|')),
	})
	r.RegisterCommand(actShellAppendOutput, command.Command{
		DocString: "Append shell command output after selections",
		Run:       Continuation(model.ShellAction("alt+!", action.ShellAppendOutput)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('!')),
	})
}
