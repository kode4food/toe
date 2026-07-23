package config

import (
	"os"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
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

var (
	errUnsavedQuit    = i18n.NewError(i18n.ErrorUnsavedQuit)
	errUnsavedQuitAll = i18n.NewError(i18n.ErrorUnsavedQuitAll)
	errUnsavedCquit   = i18n.NewError(i18n.ErrorUnsavedCquit)
)

// LifecycleModule returns the quit and force-quit commands
func LifecycleModule() command.Module {
	modes := []string{"NOR", "SEL", "INS", "TRM", "IMG"}
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actQuit,
				DocString: "Close the current view",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{Error: errUnsavedQuit}
						}
					}
					return command.Result{Signal: command.SignalQuit}
				},
				Modes:     modes,
				Aliases:   []string{"q"},
				Signature: kit.Sig(),
			},
			{
				Name: actQuitForce,
				DocString: "Force close the current view, ignoring unsaved " +
					"changes",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalQuit}
				},
				Modes:     modes,
				Aliases:   []string{"q!"},
				Signature: kit.Sig(),
			},
			{
				Name:      actQuitAll,
				DocString: "Close all views",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{Error: errUnsavedQuitAll}
						}
					}
					return command.Result{Signal: command.SignalQuit}
				},
				Modes:     modes,
				Aliases:   []string{"qa"},
				Signature: kit.Sig(),
			},
			{
				Name:      actQuitAllForce,
				DocString: "Force close all views ignoring unsaved changes",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Signal: command.SignalQuit}
				},
				Modes:     modes,
				Aliases:   []string{"qa!"},
				Signature: kit.Sig(),
			},
			{
				Name:      actCquit,
				DocString: "Quit with exit code (default 1)",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{Error: errUnsavedCquit}
						}
					}
					ui.CloseAllTerminalPanes(e)
					os.Exit(1)
					return command.Result{}
				},
				Modes:     modes,
				Aliases:   []string{"cq"},
				Signature: kit.Sig(),
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
				Modes:     modes,
				Aliases:   []string{"cq!"},
				Signature: kit.Sig(),
			},
		},
	}
}
