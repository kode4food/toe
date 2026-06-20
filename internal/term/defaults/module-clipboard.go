package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
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
	actShowClipboardProvider       = "show_clipboard_provider"
)

func clipboardModule() command.Module {
	spc := prefixed(char(' '))

	return command.Module{
		Commands: map[string]command.Command{
			actYank: {
				DocString: "Yank selection",
				Run:       Runner(action.Yank),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-yank"},
				Keys:      keys(char('y')),
			},
			actPasteAfter: {
				DocString: "Paste after selection",
				Run:       Runner(action.PasteAfter),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('p')),
			},
			actPasteBefore: {
				DocString: "Paste before selection",
				Run:       Runner(action.PasteBefore),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('P')),
			},
			actReplaceWithYanked: {
				DocString: "Replace with yanked text",
				Run:       Runner(action.ReplaceWithYanked),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('R')),
			},
			actYankToClipboard: {
				DocString: "Yank selections to clipboard",
				Run:       Runner(action.YankToClipboard),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('y'))),
			},
			actYankMainToClipboard: {
				DocString: "Yank main selection to clipboard",
				Run:       Runner(action.YankMainToClipboard),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('Y'))),
			},
			actPasteClipboardAfter: {
				DocString: "Paste clipboard after selections",
				Run:       Runner(action.PasteClipboardAfter),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-paste-after"},
				Keys:      keys(spc(char('p'))),
			},
			actPasteClipboardBefore: {
				DocString: "Paste clipboard before selections",
				Run:       Runner(action.PasteClipboardBefore),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-paste-before"},
				Keys:      keys(spc(char('P'))),
			},
			actClipboardReplace: {
				DocString: "Replace selections by clipboard content",
				Run:       Runner(action.ClipboardReplace),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-paste-replace"},
				Keys:      keys(spc(char('R'))),
				Signature: sig(),
			},
			actYankJoin: {
				DocString: "Yank joined selections. A separator can be " +
					"provided as first argument. Default value is newline",
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
				Aliases:   []string{"yank-join"},
				Signature: sig(),
			},
			actYankPrimaryClipboard: {
				DocString: "Yank selections to primary clipboard",
				Run:       Runner(action.YankToPrimaryClipboard),
				Aliases:   []string{"primary-clipboard-yank"},
				Signature: sig(),
			},
			actPastePrimaryClipboardAfter: {
				DocString: "Paste primary clipboard after selections",
				Run:       Runner(action.PastePrimaryClipboardAfter),
				Aliases:   []string{"primary-clipboard-paste-after"},
				Signature: sig(),
			},
			actPastePrimaryClipboardBefore: {
				DocString: "Paste primary clipboard before selections",
				Run:       Runner(action.PastePrimaryClipboardBefore),
				Aliases:   []string{"primary-clipboard-paste-before"},
				Signature: sig(),
			},
			actPrimaryClipboardReplace: {
				DocString: "Replace selections by primary clipboard",
				Run:       Runner(action.PrimaryClipboardReplace),
				Aliases:   []string{"primary-clipboard-paste-replace"},
				Signature: sig(),
			},
			actClearRegister: {
				DocString: "Clear given register. If no argument is " +
					"provided, clear all registers",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.ResetRegister()
					return command.Result{Message: "register cleared"}
				},
				Aliases:   []string{"clear-register"},
				Signature: sig(),
			},
			actShowClipboardProvider: {
				DocString: "Show clipboard provider name in status bar",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{
						Message: action.ShowClipboardProvider(),
					}
				},
				Aliases:   []string{"show-clipboard-provider"},
				Signature: sig(),
			},
		},
	}
}
