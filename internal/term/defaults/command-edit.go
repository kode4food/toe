package defaults

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actReplace                  = "replace"
	actSwitchCase               = "switch_case"
	actSwitchToLowercase        = "switch_to_lowercase"
	actSwitchToUppercase        = "switch_to_uppercase"
	actDeleteSelection          = "delete_selection"
	actDeleteSelectionNoyank    = "delete_selection_noyank"
	actChangeSelection          = "change_selection"
	actChangeSelectionNoyank    = "change_selection_noyank"
	actUndo                     = "undo"
	actRedo                     = "redo"
	actEarlier                  = "earlier"
	actLater                    = "later"
	actIndent                   = "indent"
	actUnindent                 = "unindent"
	actReindentSelections       = "format_selections"
	actJoinSelections           = "join_selections"
	actJoinSelectionsSpace      = "join_selections_space"
	actAlignSelections          = "align_selections"
	actTrimSelections           = "trim_selections"
	actIncrement                = "increment"
	actDecrement                = "decrement"
	actRotateSelectionsBackward = "rotate_selections_backward"
	actRotateSelectionsForward  = "rotate_selections_forward"
	actRotateContentsBackward   = "rotate_contents_backward"
	actRotateContentsForward    = "rotate_contents_forward"
	actEnsureForward            = "ensure_forward"
	actRepeatLastMotion         = "repeat_last_motion"
	actSelectMode               = "select_mode"
	actInsertMode               = "insert_mode"
	actInsertAtLineStart        = "insert_at_line_start"
	actAppendMode               = "append_mode"
	actAppendToLine             = "append_to_line"
	actOpenBelow                = "open_below"
	actOpenAbove                = "open_above"
	actExitSelectMode           = "exit_select_mode"
	actNormalMode               = "normal_mode"
	actInsertRegister           = "insert_register"
	actCommitUndoCheckpoint     = "commit_undo_checkpoint"
	actDeleteWordBackward       = "delete_word_backward"
	actDeleteWordForward        = "delete_word_forward"
	actKillToLineStart          = "kill_to_line_start"
	actKillToLineEnd            = "kill_to_line_end"
	actDeleteCharBackward       = "delete_char_backward"
	actDeleteCharForward        = "delete_char_forward"
	actInsertNewline            = "insert_newline"
	actSmartTab                 = "smart_tab"
	actPageUp                   = "page_up"
	actPageDown                 = "page_down"
	actSurroundAdd              = "surround_add"
	actSurroundReplace          = "surround_replace"
	actSurroundDelete           = "surround_delete"
	actSelectTextObjectAround   = "select_textobject_around"
	actSelectTextObjectInside   = "select_textobject_inside"
	actSelectRegister           = "select_register"
	actMatchBrackets            = "match_brackets"
)

func registerEditCommands(r *registry) {
	r.RegisterCommand(actSelectMode, command.Command{
		DocString: "Enter selection extend mode",
		Run:       Continuation(selectModeAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('v')),
	})
	r.RegisterCommand(actNormalMode, command.Command{
		DocString: "Enter normal mode",
		Run:       Continuation(normalModeAction),
		Keys:      keyBinding(special("esc")),
	})
	r.RegisterCommand(actInsertMode, command.Command{
		DocString: "Insert before selection",
		Run:       Continuation(insertModeAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('i')),
	})
	r.RegisterCommand(actInsertAtLineStart, command.Command{
		DocString: "Insert at start of line",
		Run:       Continuation(insertAtLineStartAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('I')),
	})
	r.RegisterCommand(actAppendMode, command.Command{
		DocString: "Append after selection",
		Run:       Continuation(appendModeAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('a')),
	})
	r.RegisterCommand(actAppendToLine, command.Command{
		DocString: "Insert at end of line",
		Run:       Continuation(appendToLineAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('A')),
	})
	r.RegisterCommand(actOpenBelow, command.Command{
		DocString: "Open new line below selection",
		Run:       Continuation(openBelowAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('o')),
	})
	r.RegisterCommand(actOpenAbove, command.Command{
		DocString: "Open new line above selection",
		Run:       Continuation(openAboveAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('O')),
	})
	r.RegisterCommand(actExitSelectMode, command.Command{
		DocString: "Exit selection mode",
		Run:       Continuation(normalModeAction),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(special("esc")),
	})

	r.RegisterCommand(actReplace, command.Command{
		DocString: "Replace with new char",
		Run:       Continuation(replaceCharAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('r')),
	})
	r.RegisterCommand(actDeleteSelection, command.Command{
		DocString: "Delete selection",
		Run:       Runner(action.DeleteSelection),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('d')),
	})
	r.RegisterCommand(actDeleteSelectionNoyank, command.Command{
		DocString: "Delete selection without yanking",
		Run:       Runner(action.DeleteSelectionNoyank),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('d')),
	})
	r.RegisterCommand(actChangeSelection, command.Command{
		DocString: "Change selection",
		Run:       Runner(action.ChangeSelection),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('c')),
	})
	r.RegisterCommand(actChangeSelectionNoyank, command.Command{
		DocString: "Change selection without yanking",
		Run:       Runner(action.ChangeSelectionNoyank),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('c')),
	})

	r.RegisterCommand(actUndo, command.Command{
		DocString: "Undo change",
		Run:       Continuation(undoAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('u')),
	})
	r.RegisterCommand(actRedo, command.Command{
		DocString: "Redo change",
		Run:       Continuation(redoAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('U')),
	})
	r.RegisterCommand(actEarlier, command.Command{
		DocString: "Move backward in history",
		Run:       Continuation(earlierAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('u')),
	})
	r.RegisterCommand(actLater, command.Command{
		DocString: "Move forward in history",
		Run:       Continuation(laterAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('U')),
	})

	r.RegisterCommand(actSwitchCase, command.Command{
		DocString: "Switch (toggle) case",
		Run:       Runner(action.SwitchCase),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('~')),
	})
	r.RegisterCommand(actSwitchToLowercase, command.Command{
		DocString: "Switch to lowercase",
		Run:       Runner(action.SwitchToLowercase),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('`')),
	})
	r.RegisterCommand(actSwitchToUppercase, command.Command{
		DocString: "Switch to uppercase",
		Run:       Runner(action.SwitchToUppercase),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('`')),
	})

	r.RegisterCommand(actRepeatLastMotion, command.Command{
		DocString: "Repeat last motion",
		Run:       Runner(action.RepeatLastMotion),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('.')),
	})

	r.RegisterCommand(actIndent, command.Command{
		DocString: "Indent selection",
		Run:       Runner(action.Indent),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('>')),
	})
	r.RegisterCommand(actUnindent, command.Command{
		DocString: "Unindent selection",
		Run:       Runner(action.Unindent),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('<')),
	})
	r.RegisterCommand(actReindentSelections, command.Command{
		DocString: "Format selection",
		Run:       Runner(action.ReindentSelections),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('=')),
	})
	r.RegisterCommand(actJoinSelections, command.Command{
		DocString: "Join lines inside selection",
		Run:       Runner(action.JoinSelections),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('J')),
	})
	r.RegisterCommand(actJoinSelectionsSpace, command.Command{
		DocString: "Join lines inside selection and select spaces",
		Run:       Runner(action.JoinSelectionsSpace),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('J')),
	})

	r.RegisterCommand(actAlignSelections, command.Command{
		DocString: "Align selections in column",
		Run:       Runner(action.AlignSelections),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('&')),
	})
	r.RegisterCommand(actTrimSelections, command.Command{
		DocString: "Trim whitespace from selections",
		Run:       Runner(action.TrimSelections),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('_')),
	})

	r.RegisterCommand(actRotateSelectionsBackward, command.Command{
		DocString: "Rotate selections backward",
		Run:       Runner(action.RotateSelectionsBackward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('(')),
	})
	r.RegisterCommand(actRotateSelectionsForward, command.Command{
		DocString: "Rotate selections forward",
		Run:       Runner(action.RotateSelectionsForward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char(')')),
	})
	r.RegisterCommand(actRotateContentsBackward, command.Command{
		DocString: "Rotate selections contents backward",
		Run:       Runner(action.RotateContentsBackward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt('(')),
	})
	r.RegisterCommand(actRotateContentsForward, command.Command{
		DocString: "Rotate selection contents forward",
		Run:       Runner(action.RotateContentsForward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt(')')),
	})
	r.RegisterCommand(actEnsureForward, command.Command{
		DocString: "Ensure all selections face forward",
		Run:       Runner(action.EnsureForward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(alt(':')),
	})

	r.RegisterCommand(actIncrement, command.Command{
		DocString: "Increment item under cursor",
		Run:       Runner(action.Increment),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(ctrl('a')),
	})
	r.RegisterCommand(actDecrement, command.Command{
		DocString: "Decrement item under cursor",
		Run:       Runner(action.Decrement),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(ctrl('x')),
	})
}

func selectModeAction(e *view.Editor) command.Continuation {
	action.SelectMode(e)
	return nil
}

func normalModeAction(e *view.Editor) command.Continuation {
	action.NormalMode(e)
	return nil
}

func insertModeAction(e *view.Editor) command.Continuation {
	action.InsertMode(e)
	return nil
}

func insertAtLineStartAction(e *view.Editor) command.Continuation {
	action.InsertAtLineStart(e)
	return nil
}

func appendModeAction(e *view.Editor) command.Continuation {
	action.AppendMode(e)
	return nil
}

func appendToLineAction(e *view.Editor) command.Continuation {
	action.AppendToLine(e)
	return nil
}

func openBelowAction(e *view.Editor) command.Continuation {
	action.MoveLineEnd(e)
	action.InsertNewline(e)
	e.SetMode(view.ModeInsert)
	return nil
}

func openAboveAction(e *view.Editor) command.Continuation {
	action.MoveLineStart(e)
	action.InsertNewline(e)
	action.MoveUp(e)
	e.SetMode(view.ModeInsert)
	return nil
}

func replaceCharAction(e *view.Editor) command.Continuation {
	e.SetHint("r ...")
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char != 0 && k.Mods == command.ModNone {
			action.ReplaceChar(e, k.Code.Char)
		}
		e.SetHint("")
		return nil
	}
}

func undoAction(e *view.Editor) command.Continuation {
	e.Undo()
	return nil
}

func redoAction(e *view.Editor) command.Continuation {
	e.Redo()
	return nil
}

func earlierAction(e *view.Editor) command.Continuation {
	if n := e.Count(); n > 0 {
		e.Earlier(core.UndoSteps(n))
	} else {
		e.Earlier(core.UndoSteps(1))
	}
	return nil
}

func laterAction(e *view.Editor) command.Continuation {
	if n := e.Count(); n > 0 {
		e.Later(core.UndoSteps(n))
	} else {
		e.Later(core.UndoSteps(1))
	}
	return nil
}
