package config

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

const (
	actQuit      = "quit"
	actQuitForce = "quit!"
)

const errorUnsavedQuitKey i18n.Key = "error.unsavedQuit"

var (
	errUnsavedQuit = i18n.NewError(errorUnsavedQuitKey)
)

//go:embed i18n/lifecycle.*.json
var lifecycleFS embed.FS

// LifecycleModule returns the quit and force-quit commands
func LifecycleModule() command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(lifecycleFS),
		Commands: []command.Command{
			{
				Name:      actQuit,
				DocString: "Quit if all documents are saved",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{Error: errUnsavedQuit}
						}
					}
					return command.Result{Signal: command.SignalQuit}
				},
				Modes:   command.AllModes,
				Aliases: []string{"q"},
			},
			{
				Name:      actQuitForce,
				DocString: "Quit, ignoring unsaved changes",
				Run: func(*view.Editor, *command.Args) command.Result {
					return command.Result{Signal: command.SignalQuit}
				},
				Modes:   command.AllModes,
				Aliases: []string{"q!"},
			},
		},
	}
}
