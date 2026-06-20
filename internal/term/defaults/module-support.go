package defaults

import (
	"strconv"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actCharacterInfo = "character_info"
	actEcho          = "echo"
	actRedraw        = "redraw"
	actTutor         = "tutor"
	actGoto          = "goto"
)

func supportModule() command.Module {
	return command.Module{
		Commands: map[string]command.Command{
			actCharacterInfo: {
				DocString: "Get info about the character under the primary " +
					"cursor",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{Message: action.CharInfo(e)}
				},
				Aliases:   []string{"character-info", "char"},
				Signature: sig(),
			},
			actEcho: {
				DocString: "Prints the given arguments to the statusline",
				Run: func(_ *view.Editor, args *command.Args) command.Result {
					if args == nil {
						return command.Result{}
					}
					return command.Result{Message: args.Join(" ")}
				},
				Signature: sig(),
			},
			actRedraw: {
				DocString: "Clear and re-render the whole UI",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalClearScreen}
				},
				Signature: sig(),
			},
			actTutor: {
				DocString: "Open the tutorial",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					_, err := e.SwitchFile(loader.RuntimeFile("tutor"))
					if err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					if doc, ok := e.FocusedDocument(); ok {
						doc.SetPath("")
					}
					return command.Result{}
				},
				Signature: sig(),
			},
			actGoto: {
				DocString: "Goto line number",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						return command.Result{
							Message: "error: no line number given",
						}
					}
					lineStr, _ := args.First()
					n, err := strconv.Atoi(lineStr)
					if err != nil || n < 1 {
						return command.Result{
							Message: "error: invalid line number",
						}
					}
					action.GotoLine(e, n)
					return command.Result{}
				},
				Aliases:   []string{"g"},
				Signature: minArgs(1),
			},
		},
	}
}
