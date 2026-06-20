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
		Commands: map[string]command.Command{
			actToggleComments: {
				DocString: "Comment/uncomment selections",
				Run:       Runner(action.ToggleComments),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('c'), spc(char('c'))),
			},
			actToggleLineComments: {
				DocString: "Line comment/uncomment selections",
				Run:       Runner(action.ToggleLineComments),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(alt('c'))),
			},
			actToggleBlockComments: {
				DocString: "Block comment/uncomment selections",
				Run:       Runner(action.ToggleBlockComments),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('C'))),
			},
		},
	}
}
