package files

import (
	"strconv"

	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type completionSection struct {
	Editor struct {
		Completion ui.CompletionOptions `toml:"completion"`
	} `toml:"editor"`
}

const (
	actCompletion         = "completion"
	actCompletionAccept   = ui.CompletionAcceptAction
	actCompletionCancel   = ui.CompletionCancelAction
	actCompletionPrevious = ui.CompletionPreviousAction
	actCompletionNext     = ui.CompletionNextAction
	actCompletionPageUp   = ui.CompletionPageUpAction
	actCompletionPageDown = ui.CompletionPageDownAction
	actCompletionFirst    = ui.CompletionFirstAction
	actCompletionLast     = ui.CompletionLastAction
)

// CompletionModule returns the completion-popup navigation commands
func CompletionModule(model ui.Model) command.Module {
	cfg := new(completionSection)
	reset := func() {
		*cfg = completionSection{}
		cfg.Editor.Completion = ui.DefaultCompletionOptions()
	}
	reset()

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actCompletion,
				DocString: "Complete current word",
				Run:       kit.Continuation(model.CompletionAction()),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('x')),
			},
			{
				Name:      actCompletionAccept,
				DocString: "Accept completion",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Ret, kit.Tab),
			},
			{
				Name:      actCompletionCancel,
				DocString: "Cancel completion",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Esc),
			},
			{
				Name:      actCompletionPrevious,
				DocString: "Previous completion",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Up, kit.Ctrl('p')),
			},
			{
				Name:      actCompletionNext,
				DocString: "Next completion",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Down, kit.Ctrl('n')),
			},
			{
				Name:      actCompletionPageUp,
				DocString: "Previous completion page",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.PgUp),
			},
			{
				Name:      actCompletionPageDown,
				DocString: "Next completion page",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.PgDn),
			},
			{
				Name:      actCompletionFirst,
				DocString: "First completion",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Home),
			},
			{
				Name:      actCompletionLast,
				DocString: "Last completion",
				Run:       kit.Runner(noopAction),
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.End),
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  reset,
			Apply: func(*view.Editor) {
				model.SetCompletionOptions(cfg.Editor.Completion)
			},
		},
		Options: []command.Option{
			kit.EditorBoolOption("completion.auto",
				func(*view.Editor) bool {
					return model.CompletionOptions().Auto
				},
				func(_ *view.Editor, next bool) {
					opts := model.CompletionOptions()
					opts.Auto = next
					model.SetCompletionOptions(opts)
				},
			),
			completionIntOption(model, "completion.delay",
				func(o ui.CompletionOptions) int { return o.Delay },
				func(o *ui.CompletionOptions, v int) { o.Delay = v },
			),
			completionIntOption(model, "completion.trigger-len",
				func(o ui.CompletionOptions) int { return o.TriggerLen },
				func(o *ui.CompletionOptions, v int) { o.TriggerLen = v },
			),
		},
	}
}

func completionIntOption(
	model ui.Model, key string,
	get func(ui.CompletionOptions) int,
	set func(*ui.CompletionOptions, int),
) command.Option {
	return command.Option{
		Key: key,
		Get: func(*view.Editor) (string, error) {
			return strconv.Itoa(get(model.CompletionOptions())), nil
		},
		Set: func(_ *view.Editor, s string) error {
			v, err := config.ParseNonNegInt(s)
			if err != nil {
				return err
			}
			opts := model.CompletionOptions()
			set(&opts, v)
			model.SetCompletionOptions(opts)
			return nil
		},
	}
}

func noopAction(*view.Editor) {}
