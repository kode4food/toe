package editing

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actToggleComments      = "toggle_comments"
	actToggleLineComments  = "toggle_line_comments"
	actToggleBlockComments = "toggle_block_comments"
)

//go:embed i18n/comment.*.json
var commentFS embed.FS

// CommentModule returns the comment toggle commands
func CommentModule() command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(commentFS),
		Commands: []command.Command{
			{
				Name:      actToggleComments,
				DocString: "Comment/uncomment selections",
				Run:       kit.Runner(action.ToggleComments),
				Modes:     command.DocNormalModes,
				Keys: kit.Keys(
					kit.Ctrl('c'), kit.LeaderPrefix(kit.Char('c')),
				),
			},
			{
				Name:      actToggleLineComments,
				DocString: "Line comment/uncomment selections",
				Run:       kit.Runner(action.ToggleLineComments),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.LeaderPrefix(kit.Alt('c'))),
			},
			{
				Name:      actToggleBlockComments,
				DocString: "Block comment/uncomment selections",
				Run:       kit.Runner(action.ToggleBlockComments),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.LeaderPrefix(kit.Char('C'))),
			},
		},
	}
}
