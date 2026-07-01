package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type completionSection struct {
	Editor struct {
		Completion ui.CompletionOptions `toml:"completion"`
	} `toml:"editor"`
}

const (
	actCompletion           = "completion"
	actCompletionAccept     = ui.CompletionAcceptAction
	actCompletionCancel     = ui.CompletionCancelAction
	actCompletionPrevious   = ui.CompletionPreviousAction
	actCompletionNext       = ui.CompletionNextAction
	actCompletionPageUp     = ui.CompletionPageUpAction
	actCompletionPageDown   = ui.CompletionPageDownAction
	actCompletionFirst      = ui.CompletionFirstAction
	actCompletionLast       = ui.CompletionLastAction
	actInsertRegister       = "insert_register"
	actCommitUndoCheckpoint = "commit_undo_checkpoint"
	actDeleteWordBackward   = "delete_word_backward"
	actDeleteWordForward    = "delete_word_forward"
	actKillToLineStart      = "kill_to_line_start"
	actKillToLineEnd        = "kill_to_line_end"
	actDeleteCharBackward   = "delete_char_backward"
	actDeleteCharForward    = "delete_char_forward"
	actInsertNewline        = "insert_newline"
	actSmartTab             = "smart_tab"
	actGotoLineEndNewline   = "goto_line_end_newline"
)

func insertModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actInsertRegister,
				DocString: "Insert register",
				Run:       Continuation(insertRegisterAction),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('r')),
			},
			{
				Name:      actCommitUndoCheckpoint,
				DocString: "Commit changes to new checkpoint",
				Run:       Runner(action.CommitUndoCheckpoint),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('s')),
			},
			{
				Name:      actDeleteWordBackward,
				DocString: "Delete previous word",
				Run:       Runner(action.DeleteWordBackward),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('w'), altSpecial("backspace")),
			},
			{
				Name:      actDeleteWordForward,
				DocString: "Delete next word",
				Run:       Runner(action.DeleteWordForward),
				Modes:     []string{"INS"},
				Keys:      keys(alt('d'), altSpecial("del")),
			},
			{
				Name:      actKillToLineStart,
				DocString: "Delete till start of line",
				Run:       Runner(action.KillToLineStart),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('u')),
			},
			{
				Name:      actKillToLineEnd,
				DocString: "Delete till end of line",
				Run:       Runner(action.KillToLineEnd),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('k')),
			},
			{
				Name:      actDeleteCharBackward,
				DocString: "Delete previous char",
				Run:       Runner(action.DeleteCharBackward),
				Modes:     []string{"INS"},
				Keys: keys(
					ctrl('h'), special("backspace"), shift("backspace"),
				),
			},
			{
				Name:      actDeleteCharForward,
				DocString: "Delete next char",
				Run:       Runner(action.DeleteCharForward),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('d'), special("del")),
			},
			{
				Name:      actInsertNewline,
				DocString: "Insert newline char",
				Run:       Runner(action.InsertNewline),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('j'), special("ret")),
			},
			{
				Name: actSmartTab,
				DocString: "Insert tab if all cursors have all whitespace to " +
					"their left; otherwise, run a separate command",
				Run:   Runner(action.SmartTab),
				Modes: []string{"INS"},
				Keys:  keys(special("tab")),
			},
			{
				Name:      actGotoLineEndNewline,
				DocString: "Goto newline at line end",
				Run:       Runner(action.GotoLineEndNewline),
				Modes:     []string{"INS"},
				Keys:      keys(special("end")),
			},
		},
	}
}

func completionModule(model ui.Model) command.Module {
	cfg := new(completionSection)
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actCompletion,
				DocString: "Complete current word",
				Run:       Continuation(model.CompletionAction()),
				Modes:     []string{"INS"},
				Keys:      keys(ctrl('x')),
			},
			{
				Name:      actCompletionAccept,
				DocString: "Accept completion",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("ret"), special("tab")),
			},
			{
				Name:      actCompletionCancel,
				DocString: "Cancel completion",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("esc")),
			},
			{
				Name:      actCompletionPrevious,
				DocString: "Previous completion",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("up"), ctrl('p')),
			},
			{
				Name:      actCompletionNext,
				DocString: "Next completion",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("down"), ctrl('n')),
			},
			{
				Name:      actCompletionPageUp,
				DocString: "Previous completion page",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("pageup")),
			},
			{
				Name:      actCompletionPageDown,
				DocString: "Next completion page",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("pagedown")),
			},
			{
				Name:      actCompletionFirst,
				DocString: "First completion",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("home")),
			},
			{
				Name:      actCompletionLast,
				DocString: "Last completion",
				Run:       Runner(noopAction),
				Modes:     []string{ui.CompletionMode},
				Keys:      keys(special("end")),
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = completionSection{} },
			Apply: func(*view.Editor) {
				model.SetCompletionOptions(cfg.Editor.Completion)
			},
		},
	}
}

func noopAction(*view.Editor) {}

func insertRegisterAction(e *view.Editor) command.Continuation {
	e.SetHint("^r ...")
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char != 0 && k.Mods == command.ModNone {
			action.PasteRegisterAtCursor(e, k.Code.Char)
		}
		e.SetHint("")
		return nil
	}
}
