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
	actYankToClipboard             = "yank_to_clipboard"
	actYankMainToClipboard         = "yank_main_selection_to_clipboard"
	actPasteClipboardAfter         = "paste_clipboard_after"
	actPasteClipboardBefore        = "paste_clipboard_before"
	actClipboardReplace            = "clipboard_paste_replace"
	actYankJoin                    = "yank_joined_to_clipboard"
	actYankPrimaryClipboard        = "yank_to_primary_clipboard"
	actPastePrimaryClipboardAfter  = "paste_primary_clipboard_after"
	actPastePrimaryClipboardBefore = "paste_primary_clipboard_before"
	actPrimaryClipboardReplace     = "primary_clipboard_paste_replace"
	actClearRegister               = "clear_register"
	actPasteClipboardIntoPane      = "paste_clipboard_into_pane"
)

var (
	//go:embed i18n/document.*.json
	documentFS embed.FS

	//go:embed i18n/terminal.*.json
	terminalFS embed.FS
)

// DocumentModule returns clipboard commands for document panes
func DocumentModule() command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(documentFS),
		Commands: []command.Command{
			{
				Name:      actYank,
				DocString: "Yank selection",
				Run:       kit.Runner(action.Yank),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"clipboard-yank"},
				Keys:      kit.Keys(kit.Char('y')),
			},
			{
				Name:      actPasteAfter,
				DocString: "Paste after selection",
				Run:       kit.Runner(action.PasteAfter),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('p')),
			},
			{
				Name:      actPasteBefore,
				DocString: "Paste before selection",
				Run:       kit.Runner(action.PasteBefore),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('P')),
			},
			{
				Name:      actReplaceWithYanked,
				DocString: "Replace with yanked text",
				Run:       kit.Runner(action.ReplaceWithYanked),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('R')),
			},
			{
				Name:      actYankToClipboard,
				DocString: "Yank selections to clipboard",
				Run:       kit.Runner(action.YankToClipboard),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('y'),
			},
			{
				Name:      actYankMainToClipboard,
				DocString: "Yank main selection to clipboard",
				Run:       kit.Runner(action.YankMainToClipboard),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('Y'),
			},
			{
				Name:      actPasteClipboardAfter,
				DocString: "Paste clipboard after selections",
				Run:       kit.Runner(action.PasteClipboardAfter),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"clipboard-paste-after"},
				Keys:      kit.Leader('p'),
			},
			{
				Name:      actPasteClipboardBefore,
				DocString: "Paste clipboard before selections",
				Run:       kit.Runner(action.PasteClipboardBefore),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"clipboard-paste-before"},
				Keys:      kit.Leader('P'),
			},
			{
				Name:      actClipboardReplace,
				DocString: "Replace selections by clipboard content",
				Run:       kit.Runner(action.ClipboardReplace),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('R'),
				Signature: command.DefaultSignature(),
			},
			{
				Name: actYankJoin,
				DocString: "Yank joined selections. A separator can " +
					"be provided as first argument. Default value is " +
					"newline",
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
				Aliases:   []string{"yank-join"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actYankPrimaryClipboard,
				DocString: "Yank selections to primary clipboard",
				Run:       kit.Runner(action.YankToPrimaryClipboard),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"primary-clipboard-yank"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actPastePrimaryClipboardAfter,
				DocString: "Paste primary clipboard after selections",
				Run:       kit.Runner(action.PastePrimaryClipboardAfter),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"primary-clipboard-paste-after"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actPastePrimaryClipboardBefore,
				DocString: "Paste primary clipboard before selections",
				Run:       kit.Runner(action.PastePrimaryClipboardBefore),
				Modes:     command.DocNormalModes,
				Aliases:   []string{"primary-clipboard-paste-before"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actPrimaryClipboardReplace,
				DocString: "Replace selections by primary clipboard",
				Run:       kit.Runner(action.PrimaryClipboardReplace),
				Modes:     command.DocNormalModes,
				Signature: command.DefaultSignature(),
			},
			{
				Name: actClearRegister,
				DocString: "Clear given register. If no argument is " +
					"provided, clear all registers",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.ResetRegister()
					return command.Result{Message: "register cleared"}
				},
				Modes:     command.PaneModes,
				Signature: command.DefaultSignature(),
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
			Signature: command.DefaultSignature(),
		}},
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
