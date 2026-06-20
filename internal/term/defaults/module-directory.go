package defaults

import (
	"strings"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

const (
	actChangeDirectory    = "change_directory"
	actShowDirectory      = "show_directory"
	actShowDirectoryStack = "show_directory_stack"
	actPushDirectory      = "push_directory"
	actPopDirectory       = "pop_directory"
)

func directoryModule() command.Module {
	return command.Module{
		Commands: map[string]command.Command{
			actChangeDirectory: {
				DocString: "Change the current working directory",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						return command.Result{
							Message: "error: no directory given",
						}
					}
					path, _ := args.First()
					if err := e.Chdir(path); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "directory: " + e.Cwd()}
				},
				Aliases:   []string{"change-current-directory", "cd"},
				Signature: minArgs(1),
			},
			actShowDirectory: {
				DocString: "Show the current working directory",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{Message: e.Cwd()}
				},
				Aliases:   []string{"show-directory", "pwd"},
				Signature: sig(),
			},
			actShowDirectoryStack: {
				DocString: "Show the directory stack as a space delimited " +
					"string",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{
						Message: strings.Join(e.DirStack(), "\n"),
					}
				},
				Aliases:   []string{"show-directory-stack"},
				Signature: sig(),
			},
			actPushDirectory: {
				DocString: "Save and then change the current directory",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						return command.Result{
							Message: "error: no directory given",
						}
					}
					path, _ := args.First()
					if err := e.PushDirectory(path); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "directory: " + e.Cwd()}
				},
				Aliases:   []string{"push-directory", "pushd"},
				Signature: minArgs(1),
			},
			actPopDirectory: {
				DocString: "Remove the top entry from the directory stack " +
					"and cd to the new top directory",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					if err := e.PopDirectory(); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "directory: " + e.Cwd()}
				},
				Aliases:   []string{"pop-directory", "popd"},
				Signature: sig(),
			},
		},
	}
}
