// Package clipboard provides the yank/paste and register command module
package clipboard

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actYank                        = "yank"
	actPasteAfter                  = "paste_after"
	actPasteBefore                 = "paste_before"
	actReplaceWithYanked           = "replace_with_yanked"
	actYankMain                    = "yank_main_selection"
	actYankJoin                    = "yank_join"
	actYankPrimaryClipboard        = "yank_to_primary_clipboard"
	actPastePrimaryClipboardAfter  = "paste_primary_clipboard_after"
	actPastePrimaryClipboardBefore = "paste_primary_clipboard_before"
	actPrimaryClipboardReplace     = "primary_clipboard_paste_replace"
	actClearRegister               = "clear_register"
	actPasteClipboardIntoPane      = "paste_clipboard_into_pane"
)

const (
	errorRegisterNameKey  i18n.Key = "error.registerName"
	statusRegisterCleared i18n.Key = "status.registerCleared"
	statusRegistersClear  i18n.Key = "status.registersCleared"
)

var (
	//go:embed i18n/document.*.json
	documentFS embed.FS

	//go:embed i18n/terminal.*.json
	terminalFS embed.FS
)

var (
	errRegisterName = i18n.NewError(errorRegisterNameKey)
)

// DocumentModule returns clipboard commands for document panes
func DocumentModule() command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(documentFS),
		Commands: []command.Command{
			{
				Name:      actYank,
				DocString: "Yank selection to clipboard or register",
				Run:       kit.Runner(action.Yank),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"clipboard-yank"},
				Keys: kit.Keys(
					kit.Char('y'), kit.LeaderPrefix(kit.Char('y')),
				),
			},
			{
				Name:      actYankMain,
				DocString: "Yank main selection to clipboard or register",
				Run:       kit.Runner(action.YankMain),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('Y'),
			},
			{
				Name:      actPasteAfter,
				DocString: "Paste clipboard or register after selection",
				Run:       kit.Runner(action.PasteAfter),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"clipboard-paste-after"},
				Keys: kit.Keys(
					kit.Char('p'), kit.LeaderPrefix(kit.Char('p')),
				),
			},
			{
				Name:      actPasteBefore,
				DocString: "Paste clipboard or register before selection",
				Run:       kit.Runner(action.PasteBefore),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"clipboard-paste-before"},
				Keys: kit.Keys(
					kit.Char('P'), kit.LeaderPrefix(kit.Char('P')),
				),
			},
			{
				Name:      actReplaceWithYanked,
				DocString: "Replace selection with clipboard or register",
				Run:       kit.Runner(action.ReplaceWithYanked),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys: kit.Keys(
					kit.Char('R'), kit.LeaderPrefix(kit.Char('R')),
				),
			},
			{
				Name: actYankJoin,
				DocString: "Yank joined selections to clipboard or " +
					"register. First argument sets the separator, a " +
					"newline by default",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					sep := "\n"
					if args != nil {
						if s, ok := args.First(); ok {
							sep = s
						}
					}
					action.YankJoin(e, sep)
					return command.Result{}
				},
				Modes:     command.DocNormalModes,
				Signature: kit.OptionalArg(),
			},
			{
				Name:      actYankPrimaryClipboard,
				DocString: "Yank selections to primary clipboard",
				Run:       kit.Runner(action.YankToPrimaryClipboard),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"primary-clipboard-yank"},
			},
			{
				Name:      actPastePrimaryClipboardAfter,
				DocString: "Paste primary clipboard after selections",
				Run:       kit.Runner(action.PastePrimaryClipboardAfter),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"primary-clipboard-paste-after"},
			},
			{
				Name:      actPastePrimaryClipboardBefore,
				DocString: "Paste primary clipboard before selections",
				Run:       kit.Runner(action.PastePrimaryClipboardBefore),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"primary-clipboard-paste-before"},
			},
			{
				Name:      actPrimaryClipboardReplace,
				DocString: "Replace selections by primary clipboard",
				Run:       kit.Runner(action.PrimaryClipboardReplace),
				Modes:     command.DocNormalModes,
			},
			{
				Name: actClearRegister,
				DocString: "Clear given register. If no argument is " +
					"provided, clear all registers",
				Run:       clearRegister,
				Modes:     command.PaneModes,
				Signature: kit.OptionalArg(),
			},
		},
	}
}

// TerminalModule returns clipboard commands used by terminal panes
func TerminalModule() command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(terminalFS),
		Commands: []command.Command{{
			Name:      actPasteClipboardIntoPane,
			DocString: "Paste clipboard into terminal",
			Run:       pasteClipboardIntoPane,
			Modes:     view.ModeTerminal,
			Keys:      kit.Leader('p'),
		}},
	}
}

func clearRegister(e *view.Editor, args *command.Args) command.Result {
	if args == nil || args.Empty() {
		e.Registers().ClearAll()
		e.ResetRegister()
		return command.Result{Message: i18n.Text(statusRegistersClear)}
	}
	name, _ := args.First()
	runes := []rune(name)
	if len(runes) != 1 {
		return command.Result{Error: errRegisterName}
	}
	e.Registers().Clear(runes[0])
	return command.Result{
		Message: i18n.Text(statusRegisterCleared, i18n.Vars{
			"register": name,
		}),
	}
}

func pasteClipboardIntoPane(e *view.Editor, _ *command.Args) command.Result {
	// bypasses document/selection paste for a pane implementing Pasteable
	pp, ok := e.Tree().Get(e.Tree().Focus()).(ui.Pasteable)
	if !ok {
		return command.Result{}
	}
	if text, ok := e.FirstRegister(view.RegisterClipboard); ok {
		pp.Paste(text)
	}
	return command.Result{}
}
