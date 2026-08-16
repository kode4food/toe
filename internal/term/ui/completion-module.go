package ui

import (
	"embed"
	"strconv"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type completionSection struct {
	Editor struct {
		Completion CompletionOptions `toml:"completion"`
	} `toml:"editor"`
}

const actCompletion = "completion"

//go:embed i18n/completion.*.json
var completionFS embed.FS

// CompletionModule returns the completion-popup navigation commands. It lives
// beside completionComponent because the popup implements every action here by
// intercepting the resolved command name
func CompletionModule(model Model) command.Module {
	cfg := new(completionSection)
	reset := func() {
		*cfg = completionSection{}
		cfg.Editor.Completion = DefaultCompletionOptions()
	}
	reset()

	return command.Module{
		Translations: i18n.LoadTranslations(completionFS),
		Commands: []command.Command{
			{
				Name:      actCompletion,
				DocString: "Complete current word",
				Run:       kit.Runner(model.CompletionAction),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('x')),
			},
			{
				Name:      CompletionAcceptAction,
				DocString: "Accept completion",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Ret, kit.Tab),
			},
			{
				Name:      CompletionCancelAction,
				DocString: "Cancel completion",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Esc),
			},
			{
				Name:      CompletionPreviousAction,
				DocString: "Previous completion",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Up, kit.Ctrl('p')),
			},
			{
				Name:      CompletionNextAction,
				DocString: "Next completion",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Down, kit.Ctrl('n')),
			},
			{
				Name:      CompletionPageUpAction,
				DocString: "Previous completion page",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.PgUp),
			},
			{
				Name:      CompletionPageDownAction,
				DocString: "Next completion page",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.PgDn),
			},
			{
				Name:      CompletionFirstAction,
				DocString: "First completion",
				Modes:     view.ModeCompletion,
				Keys:      kit.Keys(kit.Home),
			},
			{
				Name:      CompletionLastAction,
				DocString: "Last completion",
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
			).WithDoc("Pop up completions automatically"),
			completionIntOption(model, "completion.delay",
				func(o CompletionOptions) int { return o.Delay },
				func(o *CompletionOptions, v int) { o.Delay = v },
			).WithDoc("Delay in ms before completions appear"),
			completionIntOption(model, "completion.trigger-len",
				func(o CompletionOptions) int { return o.TriggerLen },
				func(o *CompletionOptions, v int) { o.TriggerLen = v },
			).WithDoc("Word length before completion triggers"),
		},
	}
}

func completionIntOption(
	model Model, key string,
	get func(CompletionOptions) int,
	set func(*CompletionOptions, int),
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
