package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

const (
	actSaveSession    = "save_session"
	actRestoreSession = "restore_session"
)

// sessionModule provides explicit save/restore commands for layout and open
// documents. Option persistence is handled by auto-session at startup/shutdown
func sessionModule(model ui.Model) command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actSaveSession,
				DocString: "Save session to the workspace session file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					path := view.WorkspaceSessionFile(e.Cwd())
					if err := e.SaveSession(path, nil); err != nil {
						return command.Result{
							Message: "error: " + err.Error(),
						}
					}
					return command.Result{Message: "session saved"}
				},
				Aliases:   []string{"save-session"},
				Signature: sig(),
			},
			{
				Name:      actRestoreSession,
				DocString: "Restore session from the workspace session file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					path := view.WorkspaceSessionFile(e.Cwd())
					_, ok, err := e.RestoreSession(path)
					if err != nil {
						return command.Result{
							Message: "error: " + err.Error(),
						}
					}
					if !ok {
						return command.Result{Message: "no session found"}
					}
					model.RestoreTerminalPanes(e)
					return command.Result{}
				},
				Aliases:   []string{"restore-session"},
				Signature: sig(),
			},
		},
	}
}
