package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actCommandMode           = "enter_command_mode"
	actSearch                = "search_forward"
	actSearchReverse         = "search_backward"
	actSearchNext            = "search_next"
	actSearchPrev            = "search_prev"
	actSearchSelectionWord   = "search_selection_word"
	actMakeSearchWordBounded = "make_search_word_bounded"
	actSearchSelection       = "search_selection"
	actExtendSearchNext      = "extend_search_next"
	actExtendSearchPrev      = "extend_search_prev"
)

func registerSearchCommands(r *registry, model ui.Model) {
	r.RegisterCommand(actCommandMode, command.Command{
		DocString: "Enter command mode",
		Run:       Continuation(model.CmdModeAction()),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char(':')),
	})
	r.RegisterCommand(actSearch, command.Command{
		DocString: "Search for regex pattern",
		Run:       Continuation(model.SearchAction(true)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('/')),
	})
	r.RegisterCommand(actSearchReverse, command.Command{
		DocString: "Reverse search for regex pattern",
		Run:       Continuation(model.SearchAction(false)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('?')),
	})
	r.RegisterCommand(actSearchNext, command.Command{
		DocString: "Select next search match",
		Run:       Runner(action.SearchNext),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('n')),
	})
	r.RegisterCommand(actSearchPrev, command.Command{
		DocString: "Select previous search match",
		Run:       Runner(action.SearchPrev),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('N')),
	})
	r.RegisterCommand(actSearchSelectionWord, command.Command{
		DocString: "Use current selection as the search pattern," +
			" automatically wrapping with `\\b` on word boundaries",
		Run:   Runner(action.SearchSelectionWord),
		Modes: []string{"NOR", "SEL"},
		Keys:  keyBinding(char('*')),
	})
	r.RegisterCommand(actMakeSearchWordBounded, command.Command{
		DocString: "Modify current search to make it word bounded",
		Run:       Runner(action.MakeSearchWordBounded),
		Signature: sig(),
	})
	r.RegisterCommand(actSearchSelection, command.Command{
		DocString: "Use current selection as search pattern",
		Run:       Runner(action.SearchSelection),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('*')),
	})
	r.RegisterCommand(actExtendSearchNext, command.Command{
		DocString: "Add next search match to selection",
		Run:       Runner(action.ExtendSearchNext),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('n')),
	})
	r.RegisterCommand(actExtendSearchPrev, command.Command{
		DocString: "Add previous search match to selection",
		Run:       Runner(action.ExtendSearchPrev),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('N')),
	})
}
