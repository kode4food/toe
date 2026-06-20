package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

type shellSection struct {
	Editor struct {
		Shell []string `toml:"shell"`
	} `toml:"editor"`
}

const (
	actShellPipe         = "shell_pipe"
	actShellInsertOutput = "shell_insert_output"
	actShellKeepPipe     = "shell_keep_pipe"
	actShellPipeTo       = "shell_pipe_to"
	actShellAppendOutput = "shell_append_output"
)

func shellModule(model ui.Model) command.Module {
	cfg := new(shellSection)
	return command.Module{
		Commands: map[string]command.Command{
			actShellPipe: {
				DocString: "Pipe selections through shell command",
				Run: Continuation(model.ShellAction(
					"|", action.ShellPipe,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('|')),
			},
			actShellInsertOutput: {
				DocString: "Insert shell command output before selections",
				Run: Continuation(model.ShellAction(
					"!", action.ShellInsertOutput,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('!')),
			},
			actShellKeepPipe: {
				DocString: "Filter selections with shell predicate",
				Run: Continuation(model.ShellAction(
					"$", action.ShellKeepPipe,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('$')),
			},
			actShellPipeTo: {
				DocString: "Pipe selections into shell command ignoring output",
				Run: Continuation(model.ShellAction(
					"alt+|", action.ShellPipeTo,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(alt('|')),
			},
			actShellAppendOutput: {
				DocString: "Append shell command output after selections",
				Run: Continuation(model.ShellAction(
					"alt+!", action.ShellAppendOutput,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(alt('!')),
			},
		},
		Options: []command.Option{
			{
				Key: "editor.shell",
				Get: func(e *view.Editor) (string, error) {
					return config.FormatStringSlice(e.Options().Shell), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseStringSlice(s)
					if err != nil {
						return err
					}
					e.Options().Shell = v
					return nil
				},
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = shellSection{} },
			Apply: func(e *view.Editor) {
				if len(cfg.Editor.Shell) > 0 {
					e.Options().Shell = cfg.Editor.Shell
					return
				}
				e.Options().Shell = defaultShell()
			},
		},
	}
}
