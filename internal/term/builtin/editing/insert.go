package editing

import (
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
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

// InsertModule returns the insert-mode entry and text-insertion commands
func InsertModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actInsertRegister,
				DocString: "Insert register",
				Run:       kit.Continuation(insertRegisterAction),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('r')),
			},
			{
				Name:      actCommitUndoCheckpoint,
				DocString: "Commit changes to new checkpoint",
				Run:       kit.Runner(action.CommitUndoCheckpoint),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('s')),
			},
			{
				Name:      actDeleteWordBackward,
				DocString: "Delete previous word",
				Run:       kit.Runner(action.DeleteWordBackward),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('w'), kit.AltSpecial("backspace")),
			},
			{
				Name:      actDeleteWordForward,
				DocString: "Delete next word",
				Run:       kit.Runner(action.DeleteWordForward),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Alt('d'), kit.AltSpecial("del")),
			},
			{
				Name:      actKillToLineStart,
				DocString: "Delete till start of line",
				Run:       kit.Runner(action.KillToLineStart),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('u')),
			},
			{
				Name:      actKillToLineEnd,
				DocString: "Delete till end of line",
				Run:       kit.Runner(action.KillToLineEnd),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('k')),
			},
			{
				Name:      actDeleteCharBackward,
				DocString: "Delete previous char",
				Run:       kit.Runner(action.DeleteCharBackward),
				Modes:     []string{"INS"},
				Keys: kit.Keys(
					kit.Ctrl('h'), kit.Special("backspace"), kit.Shift("backspace"),
				),
			},
			{
				Name:      actDeleteCharForward,
				DocString: "Delete next char",
				Run:       kit.Runner(action.DeleteCharForward),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('d'), kit.Special("del")),
			},
			{
				Name:      actInsertNewline,
				DocString: "Insert newline char",
				Run:       kit.Runner(action.InsertNewline),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Ctrl('j'), kit.Special("ret")),
			},
			{
				Name: actSmartTab,
				DocString: "Insert tab if all cursors have all whitespace to " +
					"their left; otherwise, run a separate command",
				Run:   kit.Runner(action.SmartTab),
				Modes: []string{"INS"},
				Keys:  kit.Keys(kit.Special("tab")),
			},
			{
				Name:      actGotoLineEndNewline,
				DocString: "Goto newline at line end",
				Run:       kit.Runner(action.GotoLineEndNewline),
				Modes:     []string{"INS"},
				Keys:      kit.Keys(kit.Special("end")),
			},
		},
	}
}
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
