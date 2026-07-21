package vcs

import (
	"fmt"

	"github.com/kode4food/toe/internal/term/builtin/kit"
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

// Module returns the version-control commands
func Module(model ui.Model) command.Module {
	prev := kit.Prefixed(kit.Char('['))
	next := kit.Prefixed(kit.Char(']'))
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actChangedFilePicker,
				DocString: "Open changed file picker",
				Run: kit.Continuation(model.PickerAction(
					ui.NewChangedFilePicker,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Leader('g'),
			},
			{
				Name:      actGotoNextChange,
				DocString: "Goto next change",
				Run:       kit.Runner(action.GotoNextChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(next(kit.Char('g'))),
			},
			{
				Name:      actGotoPrevChange,
				DocString: "Goto previous change",
				Run:       kit.Runner(action.GotoPrevChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(prev(kit.Char('g'))),
			},
			{
				Name:      actGotoFirstChange,
				DocString: "Goto first change",
				Run:       kit.Runner(action.GotoFirstChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(prev(kit.Char('G'))),
			},
			{
				Name:      actGotoLastChange,
				DocString: "Goto last change",
				Run:       kit.Runner(action.GotoLastChange),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(next(kit.Char('G'))),
			},
			{
				Name:      actResetDiffChange,
				DocString: "Reset the diff changes under the selections",
				Modes:     []string{"NOR", "SEL"},
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
				Aliases: []string{"diff-reset"},
			},
		},
	}
}
