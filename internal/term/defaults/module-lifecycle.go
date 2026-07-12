package defaults

import (
	"os"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

const (
	actQuit         = "quit"
	actQuitForce    = "quit!"
	actQuitAll      = "quit_all"
	actQuitAllForce = "quit-all!"
	actCquit        = "cquit"
	actCquitForce   = "cquit!"
)

func lifecycleModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actQuit,
				DocString: "Close the current view",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{
								Message: "document has unsaved changes " +
									"(use :quit! to force)",
							}
						}
					}
					return command.Result{Signal: command.SignalQuit}
				},
				Aliases:   []string{"q"},
				Signature: sig(),
			},
			{
				Name: actQuitForce,
				DocString: "Force close the current view, ignoring unsaved " +
					"changes",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalQuit}
				},
				Aliases:   []string{"q!"},
				Signature: sig(),
			},
			{
				Name:      actQuitAll,
				DocString: "Close all views",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{
								Message: "documents have unsaved changes " +
									"(use :quit-all! to force)",
							}
						}
					}
					return command.Result{Signal: command.SignalQuit}
				},
				Aliases:   []string{"quit-all", "qa"},
				Signature: sig(),
			},
			{
				Name:      actQuitAllForce,
				DocString: "Force close all views ignoring unsaved changes",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalQuit}
				},
				Aliases:   []string{"qa!"},
				Signature: sig(),
			},
			{
				Name:      actCquit,
				DocString: "Quit with exit code (default 1)",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{
								Message: "document has unsaved changes " +
									"(use :cquit! to force)",
							}
						}
					}
					ui.CloseAllTerminalPanes(e)
					os.Exit(1)
					return command.Result{}
				},
				Aliases:   []string{"cq"},
				Signature: sig(),
			},
			{
				Name: actCquitForce,
				DocString: "Force quit with exit code (default 1) ignoring " +
					"unsaved changes",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					ui.CloseAllTerminalPanes(e)
					os.Exit(1)
					return command.Result{}
				},
				Aliases:   []string{"cq!"},
				Signature: sig(),
			},
		},
	}
}
