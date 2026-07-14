package config

import (
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

const (
	actRecordMacro = "record_macro"
	actReplayMacro = "replay_macro"
)

// MacroModule returns the macro record and replay commands
func MacroModule(model ui.Model) command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actRecordMacro,
				DocString: "Record macro",
				Run:       kit.Continuation(model.MacroRecordAction),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('Q')),
			},
			{
				Name:      actReplayMacro,
				DocString: "Replay macro",
				Run:       kit.Continuation(model.MacroReplayAction),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('q')),
			},
		},
	}
}
