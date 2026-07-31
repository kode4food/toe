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
	actInsertTab            = "insert_tab"
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
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('r')),
			},
			{
				Name:      actCommitUndoCheckpoint,
				DocString: "Commit changes to new checkpoint",
				Run:       kit.Runner(action.CommitUndoCheckpoint),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('s')),
			},
			{
				Name:      actDeleteWordBackward,
				DocString: "Delete previous word",
				Run:       kit.Runner(action.DeleteWordBackward),
				Modes:     view.ModeInsert,
				Keys: kit.Keys(
					kit.Ctrl('w'), kit.AltSpecial(command.Backspace),
				),
			},
			{
				Name:      actDeleteWordForward,
				DocString: "Delete next word",
				Run:       kit.Runner(action.DeleteWordForward),
				Modes:     view.ModeInsert,
				Keys: kit.Keys(
					kit.Alt('d'), kit.AltSpecial(command.Delete),
				),
			},
			{
				Name:      actKillToLineStart,
				DocString: "Delete till start of line",
				Run:       kit.Runner(action.KillToLineStart),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('u')),
			},
			{
				Name:      actKillToLineEnd,
				DocString: "Delete till end of line",
				Run:       kit.Runner(action.KillToLineEnd),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('k')),
			},
			{
				Name:      actDeleteCharBackward,
				DocString: "Delete previous char",
				Run:       kit.Runner(action.DeleteCharBackward),
				Modes:     view.ModeInsert,
				Keys: kit.Keys(
					kit.Ctrl('h'), kit.Bksp, kit.Shift(command.Backspace),
				),
			},
			{
				Name:      actDeleteCharForward,
				DocString: "Delete next char",
				Run:       kit.Runner(action.DeleteCharForward),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('d'), kit.Del),
			},
			{
				Name:      actInsertNewline,
				DocString: "Insert newline char",
				Run:       kit.Runner(action.InsertNewline),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Ctrl('j'), kit.Ret),
			},
			{
				Name: actSmartTab,
				DocString: "Insert tab in leading whitespace; otherwise " +
					"move past the enclosing syntax node",
				Run:   kit.Runner(smartTab),
				Modes: view.ModeInsert,
				Keys:  kit.Keys(kit.Tab),
			},
			{
				Name:      actInsertTab,
				DocString: "Insert tab at each cursor",
				Run:       kit.Runner(action.InsertTab),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.Shift(command.Tab)),
			},
			{
				Name:      actGotoLineEndNewline,
				DocString: "Goto newline at line end",
				Run:       kit.Runner(action.GotoLineEndNewline),
				Modes:     view.ModeInsert,
				Keys:      kit.Keys(kit.End),
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
