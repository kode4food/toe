package files

import (
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

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
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actCompletion,
				DocString: "Complete current word",
				Run:       kit.Continuation(model.CompletionAction()),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('x')),
			},
			{
				Name:      actCompletionAccept,
				DocString: "Accept completion",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("ret"), kit.Special("tab")),
			},
			{
				Name:      actCompletionCancel,
				DocString: "Cancel completion",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("esc")),
			},
			{
				Name:      actCompletionPrevious,
				DocString: "Previous completion",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("up"), kit.Ctrl('p')),
			},
			{
				Name:      actCompletionNext,
				DocString: "Next completion",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("down"), kit.Ctrl('n')),
			},
			{
				Name:      actCompletionPageUp,
				DocString: "Previous completion page",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("pageup")),
			},
			{
				Name:      actCompletionPageDown,
				DocString: "Next completion page",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("pagedown")),
			},
			{
				Name:      actCompletionFirst,
				DocString: "First completion",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("home")),
			},
			{
				Name:      actCompletionLast,
				DocString: "Last completion",
				Run:       kit.Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      kit.Keys(kit.Special("end")),
			},
		},
	}
}

func noopAction(*view.Editor) {}
