package config

import (
	"embed"
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actCharacterInfo = "character_info"
	actEcho          = "echo"
	actRedraw        = "redraw"
	actGoto          = "goto"
	actAbout         = "about"
)

const (
	errorNoLineNumberKey      i18n.Key = "error.noLineNumber"
	errorInvalidLineNumberKey i18n.Key = "error.invalidLineNumber"
)

var (
	errNoLineNumber      = i18n.NewError(errorNoLineNumberKey)
	errInvalidLineNumber = i18n.NewError(errorInvalidLineNumberKey)
)

//go:embed i18n/support.*.json
var supportFS embed.FS

// SupportModule returns miscellaneous support commands
func SupportModule(model ui.Model) command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(supportFS),
		Commands: []command.Command{
			{
				Name:      actAbout,
				DocString: "Show version and license information",
				Run:       kit.Runner(model.AboutAction),
				Modes:     command.PaneModes,
			},
			{
				Name: actCharacterInfo,
				DocString: "Get info about the character under the primary " +
					"cursor",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{Message: action.CharInfo(e)}
				},
				Modes:   command.DocModes,
				Aliases: []string{"char"},
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
				Modes: command.PaneModes,
				Signature: command.Signature{
					Positionals: command.Positionals{Min: 1, Max: -1},
				},
			},
			{
				Name:      actRedraw,
				DocString: "Clear and re-render the whole UI",
				Run: func(*view.Editor, *command.Args) command.Result {
					return command.Result{Signal: command.SignalClearScreen}
				},
				Modes: command.PaneModes,
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
				Signature: kit.RequiredArg(),
			},
		},
	}
}

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
		return path, core.Position{Line: nums[1], Column: nums[0]}
	}
	return path, core.Position{}
}
