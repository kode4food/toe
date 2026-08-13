package files

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

const (
	actSaveSession    = "save_session"
	actRestoreSession = "restore_session"
)

//go:embed i18n/session.*.json
var sessionFS embed.FS

// SessionModule provides explicit commands for saving and restoring sessions
// with the registry's current runtime options
func SessionModule(r *command.Registry) command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(sessionFS),
		Commands: []command.Command{
			{
				Name:      actSaveSession,
				DocString: "Save session to the workspace session file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					values, err := r.ChangedOptionValues(e)
					if err != nil {
						return command.Result{Error: err}
					}
					path := view.WorkspaceSessionFile(e.Cwd())
					if err := e.SaveSession(path, values); err != nil {
						return command.Result{Error: err}
					}
					return command.Result{Message: "session saved"}
				},
				Modes: command.PaneModes,
			},
			{
				Name:      actRestoreSession,
				DocString: "Restore session from the workspace session file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					path := view.WorkspaceSessionFile(e.Cwd())
					values, ok, err := e.RestoreSession(path)
					if err != nil {
						return command.Result{Error: err}
					}
					if !ok {
						return command.Result{Message: "no session found"}
					}
					if err := r.ApplyOptionValues(e, values); err != nil {
						return command.Result{Error: err}
					}
					return command.Result{}
				},
				Modes: command.PaneModes,
			},
		},
	}
}
