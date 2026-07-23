package config

import (
	"strconv"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actCharacterInfo = "character_info"
	actEcho          = "echo"
	actRedraw        = "redraw"
	actGoto          = "goto"
)

var (
	errNoLineNumber      = i18n.NewError(i18n.ErrorNoLineNumber)
	errInvalidLineNumber = i18n.NewError(i18n.ErrorInvalidLineNumber)
)

// SupportModule returns the help, echo, and misc support commands
func SupportModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name: actCharacterInfo,
				DocString: "Get info about the character under the primary " +
					"cursor",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{Message: action.CharInfo(e)}
				},
				Modes:     command.DocumentModes(),
				Aliases:   []string{"char"},
				Signature: kit.Sig(),
			},
			{
				Name:      actEcho,
				DocString: "Prints the given arguments to the status line",
				Run: func(_ *view.Editor, args *command.Args) command.Result {
					if args == nil {
						return command.Result{}
					}
					return command.Result{Message: args.Join(" ")}
				},
				Modes:     command.PaneModes(),
				Signature: kit.Sig(),
			},
			{
				Name:      actRedraw,
				DocString: "Clear and re-render the whole UI",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalClearScreen}
				},
				Modes:     command.PaneModes(),
				Signature: kit.Sig(),
			},
			{
				Name:      actGoto,
				DocString: "Goto line number",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						return command.Result{Error: errNoLineNumber}
					}
					lineStr, _ := args.First()
					n, err := strconv.Atoi(lineStr)
					if err != nil || n < 1 {
						return command.Result{Error: errInvalidLineNumber}
					}
					action.GotoLine(e, n)
					return command.Result{}
				},
				Modes:     command.DocumentModes(),
				Aliases:   []string{"g"},
				Signature: kit.MinArgs(1),
			},
		},
	}
}
