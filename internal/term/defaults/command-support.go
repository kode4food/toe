package defaults

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

const (
	actQuit    = "quit"
	actQuitAll = "quit_all"
	actCquit   = "cquit"
	actFormat  = "format"
	actReflow  = "reflow"
	actSort    = "sort"

	actCharacterInfo = "character_info"
	actEcho          = "echo"
	actRedraw        = "redraw"
	actTutor         = "tutor"
	actGoto          = "goto"

	actFilePicker      = "file_picker"
	actFilePickerInCWD = "file_picker_in_current_dir"

	actFileExplorer         = "file_explorer"
	actFileExplorerInBufDir = "file_explorer_in_current_buffer_directory"
	actBufferPicker         = "buffer_picker"
	actJumplistPicker       = "jumplist_picker"

	actToggleComments      = "toggle_comments"
	actToggleLineComments  = "toggle_line_comments"
	actToggleBlockComments = "toggle_block_comments"

	actGotoNextParagraph = "goto_next_paragraph"
	actGotoPrevParagraph = "goto_prev_paragraph"

	actRecordMacro = "record_macro"
	actReplayMacro = "replay_macro"

	actGlobalSearch   = "global_search"
	actCommandPalette = "command_palette"
	actLastPicker     = "last_picker"
)

func registerSupportCommands(r *registry, model ui.Model) {
	spc := prefixed(char(' '))

	r.RegisterCommand(actQuit, command.Command{
		DocString: "Close the current view",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			for _, doc := range e.AllDocuments() {
				if doc.Modified() {
					return command.Result{
						Message: "document has unsaved changes (use :quit! to force)",
					}
				}
			}
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"q"},
		Signature: sig(),
	})
	r.RegisterCommand("quit!", command.Command{
		DocString: "Force close the current view, ignoring unsaved changes",
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"q!"},
		Signature: sig(),
	})
	r.RegisterCommand(actQuitAll, command.Command{
		DocString: "Close all views",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			for _, doc := range e.AllDocuments() {
				if doc.Modified() {
					return command.Result{
						Message: "documents have unsaved changes (use :quit-all! to force)",
					}
				}
			}
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"quit-all", "qa"},
		Signature: sig(),
	})
	r.RegisterCommand("quit-all!", command.Command{
		DocString: "Force close all views ignoring unsaved changes",
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"quit-all!", "qa!"},
		Signature: sig(),
	})
	r.RegisterCommand(actCquit, command.Command{
		DocString: "Quit with exit code (default 1)",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			for _, doc := range e.AllDocuments() {
				if doc.Modified() {
					return command.Result{
						Message: "document has unsaved changes (use :cquit! to force)",
					}
				}
			}
			os.Exit(1)
			return command.Result{}
		},
		Aliases:   []string{"cq"},
		Signature: sig(),
	})
	r.RegisterCommand("cquit!", command.Command{
		DocString: "Force quit with exit code (default 1) ignoring unsaved changes",
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			os.Exit(1)
			return command.Result{}
		},
		Aliases:   []string{"cq!"},
		Signature: sig(),
	})
	r.RegisterCommand(actFormat, command.Command{
		DocString: "Format the file using an external formatter or language server",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			return runFormatter(e)
		},
		Aliases:   []string{"fmt"},
		Signature: sig(),
	})
	r.RegisterCommand(actReflow, command.Command{
		DocString: "Hard-wrap the current selection of lines to a given width",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			width := config.DefaultTextWidth
			if s, err := config.GetOption(e.Config(),
				"editor.text-width"); err == nil {
				if n, err := strconv.Atoi(s); err == nil && n > 0 {
					width = n
				}
			}
			if args != nil && !args.Empty() {
				s, _ := args.First()
				n, err := strconv.Atoi(s)
				if err != nil || n < 1 {
					return command.Result{Message: "error: invalid width"}
				}
				width = n
			}
			action.ReflowSelections(e, width)
			return command.Result{}
		},
		Signature: optionalArg(),
	})
	r.RegisterCommand(actSort, command.Command{
		DocString: "Sort ranges in selection",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			reverse := args != nil && args.HasFlag("reverse")
			insensitive := args != nil && args.HasFlag("insensitive")
			err := action.SortSelections(e, reverse, insensitive)
			if err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{}
		},
		Signature: command.Signature{
			Flags: []command.Flag{
				{Name: "reverse", Alias: 'r'},
				{Name: "insensitive", Alias: 'i'},
			},
		},
	})

	r.RegisterCommand(actCharacterInfo, command.Command{
		DocString: "Get info about the character under the primary cursor",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			return command.Result{Message: action.CharInfo(e)}
		},
		Aliases:   []string{"character-info", "char"},
		Signature: sig(),
	})
	r.RegisterCommand(actEcho, command.Command{
		DocString: "Prints the given arguments to the statusline",
		Run: func(_ *view.Editor, args *command.Args) command.Result {
			if args == nil {
				return command.Result{}
			}
			return command.Result{Message: args.Join(" ")}
		},
		Signature: sig(),
	})
	r.RegisterCommand(actRedraw, command.Command{
		DocString: "Clear and re-render the whole UI",
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			return command.Result{Signal: command.SignalClearScreen}
		},
		Signature: sig(),
	})
	r.RegisterCommand(actTutor, command.Command{
		DocString: "Open the tutorial",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if _, err := e.SwitchFile(loader.RuntimeFile("tutor")); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			if doc, ok := e.FocusedDocument(); ok {
				doc.SetPath("")
			}
			return command.Result{}
		},
		Signature: sig(),
	})
	r.RegisterCommand(actGoto, command.Command{
		DocString: "Goto line number",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: no line number given"}
			}
			lineStr, _ := args.First()
			n, err := strconv.Atoi(lineStr)
			if err != nil || n < 1 {
				return command.Result{Message: "error: invalid line number"}
			}
			action.GotoLine(e, n)
			return command.Result{}
		},
		Aliases:   []string{"g"},
		Signature: minArgs(1),
	})

	r.RegisterCommand(actFilePicker, command.Command{
		DocString: "Open file picker",
		Run:       Continuation(model.PickerAction(ui.FilePicker)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('f'))),
	})
	r.RegisterCommand(actFilePickerInCWD, command.Command{
		DocString: "Open file picker at current working directory",
		Run:       Continuation(model.PickerAction(ui.FilePicker)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('F'))),
	})
	r.RegisterCommand(actFileExplorer, command.Command{
		DocString: "Open file explorer at workspace root",
		Run:       Continuation(model.PickerAction(ui.FileExplorer)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('e'))),
	})
	r.RegisterCommand(actFileExplorerInBufDir, command.Command{
		DocString: "Open file explorer at current buffer's directory",
		Run:       Continuation(model.PickerAction(ui.FileExplorerInBufferDir)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('.'))),
	})
	r.RegisterCommand(actBufferPicker, command.Command{
		DocString: "Open buffer picker",
		Run:       Continuation(model.PickerAction(ui.BufferPicker)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('b'))),
	})
	r.RegisterCommand(actJumplistPicker, command.Command{
		DocString: "Open jumplist picker",
		Run:       Continuation(model.PickerAction(ui.JumplistPicker)),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('j'))),
	})

	r.RegisterCommand(actToggleComments, command.Command{
		DocString: "Comment/uncomment selections",
		Run:       Runner(action.ToggleComments),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(ctrl('c'), spc(char('c'))),
	})
	r.RegisterCommand(actToggleLineComments, command.Command{
		DocString: "Line comment/uncomment selections",
		Run:       Runner(action.ToggleLineComments),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(alt('c'))),
	})
	r.RegisterCommand(actToggleBlockComments, command.Command{
		DocString: "Block comment/uncomment selections",
		Run:       Runner(action.ToggleBlockComments),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('C'))),
	})

	r.RegisterCommand(actGotoNextParagraph, command.Command{
		DocString: "Goto next paragraph",
		Run:       Runner(action.GotoNextParagraph),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char(']'), char('p')),
	})
	r.RegisterCommand(actGotoPrevParagraph, command.Command{
		DocString: "Goto previous paragraph",
		Run:       Runner(action.GotoPrevParagraph),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('['), char('p')),
	})

	r.RegisterCommand(actRecordMacro, command.Command{
		DocString: "Record macro",
		Run:       Continuation(model.MacroRecordAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('Q')),
	})
	r.RegisterCommand(actReplayMacro, command.Command{
		DocString: "Replay macro",
		Run:       Continuation(model.MacroReplayAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('q')),
	})

	r.RegisterCommand(actGlobalSearch, command.Command{
		DocString: "Global search in workspace folder",
		Run:       Continuation(model.GlobalSearchAction()),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('/'))),
	})
	r.RegisterCommand(actCommandPalette, command.Command{
		DocString: "Open command palette",
		Run:       Continuation(model.CommandPaletteAction()),
		Aliases:   []string{"command-palette"},
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('?'))),
	})
	r.RegisterCommand(actLastPicker, command.Command{
		DocString: "Reopen the last picker",
		Run:       Continuation(model.LastPickerAction()),
		Aliases:   []string{"last-picker"},
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(spc(char('\''))),
	})
}

// runFormatter runs the configured external formatter for the focused
// document's language. If no formatter is configured, it reports the gap
// via the status message. If the formatter succeeds and its output differs
// from the current buffer, the buffer is replaced via a transaction
func runFormatter(e *view.Editor) command.Result {
	v, ok := e.FocusedView()
	if !ok {
		return command.Result{}
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return command.Result{}
	}
	if doc.Readonly() {
		return command.Result{Message: "error: buffer is read-only"}
	}

	lang := config.LoadLanguage(doc.Lang())
	if lang.Formatter == nil {
		return command.Result{
			Message: "no formatter configured for " + doc.Lang(),
		}
	}

	text := doc.Text().String()
	cmd := exec.Command(lang.Formatter.Command, lang.Formatter.Args...)
	cmd.Stdin = bytes.NewBufferString(text)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		msg := lang.Formatter.Command + ": " + err.Error()
		if errOut.Len() > 0 {
			msg = lang.Formatter.Command + ": " + errOut.String()
		}
		return command.Result{Message: "error: " + msg}
	}

	formatted := out.String()
	if formatted == text {
		return command.Result{}
	}

	rope := doc.Text()
	n := rope.LenChars()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, n, formatted),
	})
	if err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	sel := doc.SelectionFor(v.ID())
	tx := core.NewTransaction(rope).WithChanges(cs).WithSelection(sel)
	if err := e.Apply(tx); err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	return command.Result{}
}
