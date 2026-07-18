package files

import (
	"strings"

	"github.com/kode4food/toe/internal/term/builtin/kit"
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

// DirectoryModule returns the working-directory commands
func DirectoryModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actChangeDirectory,
				DocString: "Change the current working directory",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					return cdResult(e, args, e.Chdir)
				},
				Modes:     command.PaneModes(),
				Aliases:   []string{"change-current-directory", "cd"},
				Signature: kit.MinArgs(1),
			},
			{
				Name:      actShowDirectory,
				DocString: "Show the current working directory",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{Message: e.Cwd()}
				},
				Modes:     command.PaneModes(),
				Aliases:   []string{"pwd"},
				Signature: kit.Sig(),
			},
			{
				Name: actShowDirectoryStack,
				DocString: "Show the directory stack as a space delimited " +
					"string",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return command.Result{
						Message: strings.Join(e.DirStack(), "\n"),
					}
				},
				Modes:     command.PaneModes(),
				Signature: kit.Sig(),
			},
			{
				Name:      actPushDirectory,
				DocString: "Save and then change the current directory",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					return cdResult(e, args, e.PushDirectory)
				},
				Modes:     command.PaneModes(),
				Aliases:   []string{"pushd"},
				Signature: kit.MinArgs(1),
			},
			{
				Name: actPopDirectory,
				DocString: "Remove the top entry from the directory stack " +
					"and cd to the new top directory",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					if err := e.PopDirectory(); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "directory: " + e.Cwd()}
				},
				Modes:     command.PaneModes(),
				Aliases:   []string{"popd"},
				Signature: kit.Sig(),
			},
		},
	}
}

func cdResult(
	e *view.Editor, args *command.Args, fn func(string) error,
) command.Result {
	if args == nil || args.Empty() {
		return command.Result{Message: "error: no directory given"}
	}
	path, _ := args.First()
	if err := fn(path); err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	return command.Result{Message: "directory: " + e.Cwd()}
}
