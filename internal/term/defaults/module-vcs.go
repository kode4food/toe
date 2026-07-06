package defaults

import (
	"fmt"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actGotoNextChange    = "goto_next_change"
	actGotoPrevChange    = "goto_prev_change"
	actGotoFirstChange   = "goto_first_change"
	actGotoLastChange    = "goto_last_change"
	actResetDiffChange   = "reset_diff_change"
	actChangedFilePicker = "changed_file_picker"
)

func vcsModule(model ui.Model) command.Module {
	prev := prefixed(char('['))
	next := prefixed(char(']'))
	spc := prefixed(char(' '))
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actChangedFilePicker,
				DocString: "Open changed file picker",
				Run: Continuation(model.PickerAction(
					ui.NewChangedFilePicker,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('g'))),
			},
			{
				Name:      actGotoNextChange,
				DocString: "Goto next change",
				Run:       Runner(action.GotoNextChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(next(char('g'))),
			},
			{
				Name:      actGotoPrevChange,
				DocString: "Goto previous change",
				Run:       Runner(action.GotoPrevChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(prev(char('g'))),
			},
			{
				Name:      actGotoFirstChange,
				DocString: "Goto first change",
				Run:       Runner(action.GotoFirstChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(prev(char('G'))),
			},
			{
				Name:      actGotoLastChange,
				DocString: "Goto last change",
				Run:       Runner(action.GotoLastChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(next(char('G'))),
			},
			{
				Name:      actResetDiffChange,
				DocString: "Reset the diff changes under the selections",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					n, err := action.ResetDiffChange(e)
					if err != nil {
						return command.Result{
							Message: "error: " + err.Error(),
						}
					}
					msg := fmt.Sprintf("Reset %d change", n)
					if n != 1 {
						msg += "s"
					}
					return command.Result{Message: msg}
				},
				Aliases: []string{"reset-diff-change", "diff-reset"},
			},
		},
	}
}
