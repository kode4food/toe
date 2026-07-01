package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actToggleComments      = "toggle_comments"
	actToggleLineComments  = "toggle_line_comments"
	actToggleBlockComments = "toggle_block_comments"
)

func commentModule() command.Module {
	spc := prefixed(char(' '))

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actToggleComments,
				DocString: "Comment/uncomment selections",
				Run:       Runner(action.ToggleComments),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('c'), spc(char('c'))),
			},
			{
				Name:      actToggleLineComments,
				DocString: "Line comment/uncomment selections",
				Run:       Runner(action.ToggleLineComments),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(alt('c'))),
			},
			{
				Name:      actToggleBlockComments,
				DocString: "Block comment/uncomment selections",
				Run:       Runner(action.ToggleBlockComments),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('C'))),
			},
		},
	}
}
