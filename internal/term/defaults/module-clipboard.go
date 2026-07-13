package defaults

import (
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

func clipboardModule() command.Module {
	spc := prefixed(char(' '))
	cw := prefixed(ctrl('w'))

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actYank,
				DocString: "Yank selection",
				Run:       Runner(action.Yank),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-yank"},
				Keys:      keys(char('y')),
			},
			{
				Name:      actPasteAfter,
				DocString: "Paste after selection",
				Run:       Runner(action.PasteAfter),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('p')),
			},
			{
				Name:      actPasteBefore,
				DocString: "Paste before selection",
				Run:       Runner(action.PasteBefore),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('P')),
			},
			{
				Name:      actReplaceWithYanked,
				DocString: "Replace with yanked text",
				Run:       Runner(action.ReplaceWithYanked),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('R')),
			},
			{
				Name:      actYankToClipboard,
				DocString: "Yank selections to clipboard",
				Run:       Runner(action.YankToClipboard),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('y'))),
			},
			{
				Name:      actYankMainToClipboard,
				DocString: "Yank main selection to clipboard",
				Run:       Runner(action.YankMainToClipboard),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('Y'))),
			},
			{
				Name:      actPasteClipboardAfter,
				DocString: "Paste clipboard after selections",
				Run:       Runner(action.PasteClipboardAfter),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-paste-after"},
				Keys:      keys(spc(char('p'))),
			},
			{
				Name:      actPasteClipboardIntoPane,
				DocString: "Paste clipboard into terminal",
				Run:       pasteClipboardIntoPane,
				Modes:     []string{"TRM"},
				Keys:      keys(cw(char('p'))),
				Signature: sig(),
			},
			{
				Name:      actPasteClipboardBefore,
				DocString: "Paste clipboard before selections",
				Run:       Runner(action.PasteClipboardBefore),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-paste-before"},
				Keys:      keys(spc(char('P'))),
			},
			{
				Name:      actClipboardReplace,
				DocString: "Replace selections by clipboard content",
				Run:       Runner(action.ClipboardReplace),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"clipboard-paste-replace"},
				Keys:      keys(spc(char('R'))),
				Signature: sig(),
			},
			{
				Name: actYankJoin,
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
			{
				Name:      actYankPrimaryClipboard,
				DocString: "Yank selections to primary clipboard",
				Run:       Runner(action.YankToPrimaryClipboard),
				Aliases:   []string{"primary-clipboard-yank"},
				Signature: sig(),
			},
			{
				Name:      actPastePrimaryClipboardAfter,
				DocString: "Paste primary clipboard after selections",
				Run:       Runner(action.PastePrimaryClipboardAfter),
				Aliases:   []string{"primary-clipboard-paste-after"},
				Signature: sig(),
			},
			{
				Name:      actPastePrimaryClipboardBefore,
				DocString: "Paste primary clipboard before selections",
				Run:       Runner(action.PastePrimaryClipboardBefore),
				Aliases:   []string{"primary-clipboard-paste-before"},
				Signature: sig(),
			},
			{
				Name:      actPrimaryClipboardReplace,
				DocString: "Replace selections by primary clipboard",
				Run:       Runner(action.PrimaryClipboardReplace),
				Aliases:   []string{"primary-clipboard-paste-replace"},
				Signature: sig(),
			},
			{
				Name: actClearRegister,
				DocString: "Clear given register. If no argument is " +
					"provided, clear all registers",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.ResetRegister()
					return command.Result{Message: "register cleared"}
				},
				Aliases:   []string{"clear-register"},
				Signature: sig(),
			},
		},
	}
}

func pasteClipboardIntoPane(e *view.Editor, _ *command.Args) command.Result {
	// bypasses document/selection paste for a pane implementing RawPane
	pp, ok := e.Tree().Get(e.Tree().Focus()).(ui.RawPane)
	if !ok {
		return command.Result{}
	}
	if text, ok := e.FirstRegister(view.RegisterClipboard); ok {
		pp.Paste(text)
	}
	return command.Result{}
}
