package shell

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

type section struct {
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

const (
	promptPipeKey         i18n.Key = "prompt.pipe"
	promptInsertOutputKey i18n.Key = "prompt.insertOutput"
	promptFilterKey       i18n.Key = "prompt.filter"
	promptPipeToKey       i18n.Key = "prompt.pipeTo"
	promptAppendOutputKey i18n.Key = "prompt.appendOutput"
)

//go:embed i18n/shell.*.json
var shellFS embed.FS

// Module returns the shell-pipe and shell command bindings
func Module(model ui.Model) command.Module {
	cfg := new(section)
	return command.Module{
		Translations: i18n.LoadTranslations(shellFS),
		Commands: []command.Command{
			{
				Name:      actShellPipe,
				DocString: "Pipe selections through shell command",
				Run: kit.Runner(model.ShellAction(
					promptPipeKey, action.ShellPipe,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('|')),
			},
			{
				Name:      actShellInsertOutput,
				DocString: "Insert shell command output before selections",
				Run: kit.Runner(model.ShellAction(
					promptInsertOutputKey,
					action.ShellInsertOutput,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('!')),
			},
			{
				Name:      actShellKeepPipe,
				DocString: "Filter selections with shell predicate",
				Run: kit.Runner(model.ShellAction(
					promptFilterKey,
					action.ShellKeepPipe,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('$')),
			},
			{
				Name:      actShellPipeTo,
				DocString: "Pipe selections into shell command ignoring output",
				Run: kit.Runner(model.ShellAction(
					promptPipeToKey, action.ShellPipeTo,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Alt('|')),
			},
			{
				Name:      actShellAppendOutput,
				DocString: "Append shell command output after selections",
				Run: kit.Runner(model.ShellAction(
					promptAppendOutputKey,
					action.ShellAppendOutput,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Alt('!')),
			},
		},
		Options: []command.Option{
			{
				Key:       "shell",
				DocString: "Shell used to run external commands",
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
			Reset:  func() { *cfg = section{} },
			Apply: func(e *view.Editor) {
				if len(cfg.Editor.Shell) > 0 {
					e.Options().Shell = cfg.Editor.Shell
					return
				}
				e.Options().Shell = view.DefaultShell()
			},
		},
	}
}
