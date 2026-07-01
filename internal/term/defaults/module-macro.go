package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

const (
	actRecordMacro = "record_macro"
	actReplayMacro = "replay_macro"
)

func macroModule(model ui.Model) command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actRecordMacro,
				DocString: "Record macro",
				Run:       Continuation(model.MacroRecordAction),
				Modes:     []string{"NOR"},
				Keys:      keys(char('Q')),
			},
			{
				Name:      actReplayMacro,
				DocString: "Replay macro",
				Run:       Continuation(model.MacroReplayAction),
				Modes:     []string{"NOR"},
				Keys:      keys(char('q')),
			},
		},
	}
}
