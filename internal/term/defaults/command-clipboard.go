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

func registerClipboardCommands(r *registry) {
	spc := prefixed(char(' '))

	r.RegisterCommand(actYank, command.Command{
		DocString: "Yank selection",
		Run:       Runner(action.Yank),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('y')),
	})
	r.RegisterCommand(actPasteAfter, command.Command{
		DocString: "Paste after selection",
		Run:       Runner(action.PasteAfter),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('p')),
	})
	r.RegisterCommand(actPasteBefore, command.Command{
		DocString: "Paste before selection",
		Run:       Runner(action.PasteBefore),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('P')),
	})
	r.RegisterCommand(actReplaceWithYanked, command.Command{
		DocString: "Replace with yanked text",
		Run:       Runner(action.ReplaceWithYanked),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('R')),
	})
	r.RegisterCommand(actYankToClipboard, command.Command{
		DocString: "Yank selections to clipboard",
		Run:       Runner(action.YankToClipboard),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('y'))),
	})
	r.RegisterCommand(actYankMainToClipboard, command.Command{
		DocString: "Yank main selection to clipboard",
		Run:       Runner(action.YankMainToClipboard),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('Y'))),
	})
	r.RegisterCommand(actPasteClipboardAfter, command.Command{
		DocString: "Paste clipboard after selections",
		Run:       Runner(action.PasteClipboardAfter),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('p'))),
	})
	r.RegisterCommand(actPasteClipboardBefore, command.Command{
		DocString: "Paste clipboard before selections",
		Run:       Runner(action.PasteClipboardBefore),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('P'))),
	})
	r.RegisterCommand(actClipboardReplace, command.Command{
		DocString: "Replace selections by clipboard content",
		Run:       Runner(action.ClipboardReplace),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('R'))),
		Aliases:   []string{"clipboard-paste-replace"},
		Signature: sig(),
	})
	r.RegisterCommand(actYankJoin, command.Command{
		DocString: "Yank joined selections. A separator can be provided as " +
			"first argument. Default value is newline",
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
	})
	r.RegisterCommand(actYankPrimaryClipboard, command.Command{
		DocString: "Yank selections to primary clipboard",
		Run:       Runner(action.YankToPrimaryClipboard),
		Aliases:   []string{"primary-clipboard-yank"},
		Signature: sig(),
	})
	r.RegisterCommand(actPastePrimaryClipboardAfter, command.Command{
		DocString: "Paste primary clipboard after selections",
		Run:       Runner(action.PastePrimaryClipboardAfter),
		Aliases:   []string{"primary-clipboard-paste-after"},
		Signature: sig(),
	})
	r.RegisterCommand(actPastePrimaryClipboardBefore, command.Command{
		DocString: "Paste primary clipboard before selections",
		Run:       Runner(action.PastePrimaryClipboardBefore),
		Aliases:   []string{"primary-clipboard-paste-before"},
		Signature: sig(),
	})
	r.RegisterCommand(actPrimaryClipboardReplace, command.Command{
		DocString: "Replace selections by primary clipboard",
		Run:       Runner(action.PrimaryClipboardReplace),
		Aliases:   []string{"primary-clipboard-paste-replace"},
		Signature: sig(),
	})
	r.RegisterCommand(actClearRegister, command.Command{
		DocString: "Clear given register. If no argument is provided, " +
			"clear all registers",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.ResetRegister()
			return command.Result{Message: "register cleared"}
		},
		Aliases:   []string{"clear-register"},
		Signature: sig(),
	})
	r.RegisterCommand(actShowClipboardProvider, command.Command{
		DocString: "Show clipboard provider name in status bar",
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			return command.Result{Message: action.ShowClipboardProvider()}
		},
		Aliases:   []string{"show-clipboard-provider"},
		Signature: sig(),
	})
	// ex-command aliases for yank/paste with existing implementations
	r.RegisterCommand(actYank, command.Command{
		Run:       Runner(action.Yank),
		Aliases:   []string{"clipboard-yank"},
		Signature: sig(),
	})
	r.RegisterCommand(actPasteClipboardAfter, command.Command{
		Run:       Runner(action.PasteClipboardAfter),
		Aliases:   []string{"clipboard-paste-after"},
		Signature: sig(),
	})
	r.RegisterCommand(actPasteClipboardBefore, command.Command{
		Run:       Runner(action.PasteClipboardBefore),
		Aliases:   []string{"clipboard-paste-before"},
		Signature: sig(),
	})
}
