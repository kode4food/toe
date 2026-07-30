package config

import (
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/core"
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
				Modes:     command.DocModes,
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
				Modes:     command.PaneModes,
				Signature: kit.Sig(),
			},
			{
				Name:      actRedraw,
				DocString: "Clear and re-render the whole UI",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalClearScreen}
				},
				Modes:     command.PaneModes,
				Signature: kit.Sig(),
			},
			{
				Name:      actGoto,
				DocString: "Goto a path:line:col location",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						return command.Result{Error: errNoLineNumber}
					}
					loc, _ := args.First()
					path, at := parseGotoLocation(loc)
					if at.Line < 1 {
						return command.Result{Error: errInvalidLineNumber}
					}
					if path != "" {
						if _, err := e.OpenFile(path); err != nil {
							return command.Result{Error: err}
						}
					}
					action.GotoPosition(e, at)
					return command.Result{}
				},
				Modes:     command.DocModes,
				Aliases:   []string{"g"},
				Signature: kit.MinArgs(1),
			},
		},
	}
}

// parseGotoLocation peels up to two trailing positive-int segments of a
// "path:line:col" string as line then col; the rest is the path
func parseGotoLocation(s string) (string, core.Position) {
	parts := strings.Split(s, ":")
	var nums []int
	for len(parts) > 0 && len(nums) < 2 {
		n, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil || n < 1 {
			break
		}
		nums = append(nums, n)
		parts = parts[:len(parts)-1]
	}
	path := strings.Join(parts, ":")
	switch len(nums) {
	case 1:
		return path, core.Position{Line: nums[0]}
	case 2:
		return path, core.Position{Line: nums[1], Col: nums[0]}
	}
	return path, core.Position{}
}
