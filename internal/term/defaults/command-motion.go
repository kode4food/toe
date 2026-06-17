package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type findCharHintKey struct {
	forward   bool
	inclusive bool
}

const (
	actMoveLeft                  = "move_char_left"
	actMoveRight                 = "move_char_right"
	actMoveUp                    = "move_visual_line_up"
	actMoveDown                  = "move_visual_line_down"
	actMoveNextWordStart         = "move_next_word_start"
	actMovePrevWordStart         = "move_prev_word_start"
	actMoveNextWordEnd           = "move_next_word_end"
	actMovePrevWordEnd           = "move_prev_word_end"
	actMoveNextLongWordStart     = "move_next_long_word_start"
	actMovePrevLongWordStart     = "move_prev_long_word_start"
	actMoveNextLongWordEnd       = "move_next_long_word_end"
	actMovePrevLongWordEnd       = "move_prev_long_word_end"
	actMoveNextSubWordStart      = "move_next_sub_word_start"
	actMovePrevSubWordStart      = "move_prev_sub_word_start"
	actMoveNextSubWordEnd        = "move_next_sub_word_end"
	actMovePrevSubWordEnd        = "move_prev_sub_word_end"
	actGotoLineStart             = "goto_line_start"
	actGotoLineEnd               = "goto_line_end"
	actGotoLineEndNewline        = "goto_line_end_newline"
	actMoveLineNonWhitespace     = "goto_first_nonwhitespace"
	actMoveFileEnd               = "goto_last_line"
	actGotoWindowTop             = "goto_window_top"
	actGotoWindowCenter          = "goto_window_center"
	actGotoWindowBottom          = "goto_window_bottom"
	actGotoLastAccessedFile      = "goto_last_accessed_file"
	actGotoLastModifiedFile      = "goto_last_modified_file"
	actGotoNextBuffer            = "goto_next_buffer"
	actGotoPreviousBuffer        = "goto_previous_buffer"
	actGotoLastModification      = "goto_last_modification"
	actGotoColumn                = "goto_column"
	actGotoLine                  = "goto_line"
	actFindNextChar              = "find_next_char"
	actFindPrevChar              = "find_prev_char"
	actFindTillChar              = "find_till_char"
	actTillPrevChar              = "till_prev_char"
	actJumpForward               = "jump_forward"
	actJumpBackward              = "jump_backward"
	actSaveSelection             = "save_selection"
	actGotoLineOrFileStart       = "goto_line_or_file_start"
	actGotoLineOrExtendFileStart = "goto_line_or_extend_file_start"
	actGotoFile                  = "goto_file"
	actExtendCharLeft            = "extend_char_left"
	actExtendCharRight           = "extend_char_right"
	actExtendVisualLineUp        = "extend_visual_line_up"
	actExtendVisualLineDown      = "extend_visual_line_down"
	actExtendNextWordStart       = "extend_next_word_start"
	actExtendPrevWordStart       = "extend_prev_word_start"
	actExtendNextWordEnd         = "extend_next_word_end"
	actExtendPrevWordEnd         = "extend_prev_word_end"
	actExtendNextLongWordStart   = "extend_next_long_word_start"
	actExtendPrevLongWordStart   = "extend_prev_long_word_start"
	actExtendNextLongWordEnd     = "extend_next_long_word_end"
	actExtendPrevLongWordEnd     = "extend_prev_long_word_end"
	actExtendNextSubWordStart    = "extend_next_sub_word_start"
	actExtendPrevSubWordStart    = "extend_prev_sub_word_start"
	actExtendNextSubWordEnd      = "extend_next_sub_word_end"
	actExtendPrevSubWordEnd      = "extend_prev_sub_word_end"
	actExtendNextChar            = "extend_next_char"
	actExtendTillChar            = "extend_till_char"
	actExtendPrevChar            = "extend_prev_char"
	actExtendTillPrevChar        = "extend_till_prev_char"
	actExtendToLineStart         = "extend_to_line_start"
	actExtendToLineEnd           = "extend_to_line_end"
	actExtendToLineEndNewline    = "extend_to_line_end_newline"
	actExtendToNonWhitespace     = "extend_to_first_nonwhitespace"
	actExtendToColumn            = "extend_to_column"
	actExtendToLastLine          = "extend_to_last_line"
	actExtendToFileEnd           = "extend_to_file_end"
)

func registerMotionCommands(r *registry) {
	g := prefixed(char('g'))

	r.RegisterCommand(actMoveLeft, command.Command{
		DocString: "Move left",
		Run:       Runner(action.MoveLeft),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('h'), special("left")),
	})
	r.RegisterCommand(actMoveDown, command.Command{
		DocString: "Move down",
		Run:       Runner(action.MoveDown),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('j'), special("down")),
	})
	r.RegisterCommand(actMoveUp, command.Command{
		DocString: "Move up",
		Run:       Runner(action.MoveUp),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('k'), special("up")),
	})
	r.RegisterCommand(actMoveRight, command.Command{
		DocString: "Move right",
		Run:       Runner(action.MoveRight),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('l'), special("right")),
	})

	r.RegisterCommand(actMoveNextWordStart, command.Command{
		DocString: "Move to start of next word",
		Run:       Runner(action.MoveWordForward),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('w')),
	})
	r.RegisterCommand(actMovePrevWordStart, command.Command{
		DocString: "Move to start of previous word",
		Run:       Runner(action.MoveWordBackward),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('b')),
	})
	r.RegisterCommand(actMoveNextWordEnd, command.Command{
		DocString: "Move to end of next word",
		Run:       Runner(action.MoveWordEnd),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('e')),
	})
	r.RegisterCommand(actMovePrevWordEnd, command.Command{
		DocString: "Move to end of previous word",
		Run:       Runner(action.MovePrevWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actMoveNextLongWordStart, command.Command{
		DocString: "Move to start of next long word",
		Run:       Runner(action.MoveLongWordForward),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('W')),
	})
	r.RegisterCommand(actMovePrevLongWordStart, command.Command{
		DocString: "Move to start of previous long word",
		Run:       Runner(action.MoveLongWordBackward),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('B')),
	})
	r.RegisterCommand(actMoveNextLongWordEnd, command.Command{
		DocString: "Move to end of next long word",
		Run:       Runner(action.MoveLongWordEnd),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('E')),
	})
	r.RegisterCommand(actMovePrevLongWordEnd, command.Command{
		DocString: "Move to end of previous long word",
		Run:       Runner(action.MovePrevLongWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actMoveNextSubWordStart, command.Command{
		DocString: "Move to start of next sub-word",
		Run:       Runner(action.MoveNextSubWordStart),
		Signature: sig(),
	})
	r.RegisterCommand(actMovePrevSubWordStart, command.Command{
		DocString: "Move to start of previous sub-word",
		Run:       Runner(action.MovePrevSubWordStart),
		Signature: sig(),
	})
	r.RegisterCommand(actMoveNextSubWordEnd, command.Command{
		DocString: "Move to end of next sub-word",
		Run:       Runner(action.MoveNextSubWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actMovePrevSubWordEnd, command.Command{
		DocString: "Move to end of previous sub-word",
		Run:       Runner(action.MovePrevSubWordEnd),
		Signature: sig(),
	})

	r.RegisterCommand(actGotoLineStart, command.Command{
		DocString: "Goto line start",
		Run:       Runner(action.MoveLineStart),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(special("home")),
	})
	r.RegisterCommand(actGotoLineEnd, command.Command{
		DocString: "Goto line end",
		Run:       Runner(action.MoveLineEnd),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(special("end")),
	})

	r.RegisterCommand(actFindNextChar, command.Command{
		DocString: "Move to next occurrence of char",
		Run:       Continuation(findCharAction(true, true, false)),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('f')),
	})
	r.RegisterCommand(actFindTillChar, command.Command{
		DocString: "Move till next occurrence of char",
		Run:       Continuation(findCharAction(true, false, false)),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('t')),
	})
	r.RegisterCommand(actFindPrevChar, command.Command{
		DocString: "Move to previous occurrence of char",
		Run:       Continuation(findCharAction(false, true, false)),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('F')),
	})
	r.RegisterCommand(actTillPrevChar, command.Command{
		DocString: "Move till previous occurrence of char",
		Run:       Continuation(findCharAction(false, false, false)),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('T')),
	})

	r.RegisterCommand(actGotoLine, command.Command{
		DocString: "Goto line",
		Run:       Continuation(gotoLineAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(char('G')),
	})
	r.RegisterCommand(actGotoLineOrFileStart, command.Command{
		DocString: "Goto line number `<n>` else file start",
		Run:       Continuation(gotoLineOrFileStartAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(g(char('g'))),
	})
	r.RegisterCommand(actGotoFile, command.Command{
		DocString: "Goto files/URLs in selections",
		Run:       Continuation(gotoFileAction),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('f'))),
	})
	r.RegisterCommand(actGotoColumn, command.Command{
		DocString: "Goto column",
		Run:       Runner(action.ExtendToColumn),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(g(char('|'))),
	})

	r.RegisterCommand(actGotoWindowTop, command.Command{
		DocString: "Goto window top",
		Run:       Runner(action.GotoWindowTop),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('t'))),
	})
	r.RegisterCommand(actGotoWindowCenter, command.Command{
		DocString: "Goto window center",
		Run:       Runner(action.GotoWindowCenter),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('c'))),
	})
	r.RegisterCommand(actGotoWindowBottom, command.Command{
		DocString: "Goto window bottom",
		Run:       Runner(action.GotoWindowBottom),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('b'))),
	})
	r.RegisterCommand(actMoveFileEnd, command.Command{
		DocString: "Goto last line",
		Run:       Runner(action.MoveFileEnd),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(g(char('e'))),
	})
	r.RegisterCommand(actMoveLineNonWhitespace, command.Command{
		DocString: "Goto first non-blank in line",
		Run:       Runner(action.MoveLineNonWhitespace),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('s'))),
	})
	r.RegisterCommand(actGotoLastAccessedFile, command.Command{
		DocString: "Goto last accessed file",
		Run:       Runner(action.GotoLastAccessedFile),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('a'))),
	})
	r.RegisterCommand(actGotoLastModifiedFile, command.Command{
		DocString: "Goto last modified file",
		Run:       Runner(action.GotoLastModifiedFile),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('m'))),
	})
	r.RegisterCommand(actGotoNextBuffer, command.Command{
		DocString: "Goto next buffer",
		Run:       method((*view.Editor).FocusNextView),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('n'))),
	})
	r.RegisterCommand(actGotoPreviousBuffer, command.Command{
		DocString: "Goto previous buffer",
		Run:       method((*view.Editor).FocusPrevView),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('p'))),
	})
	r.RegisterCommand(actGotoLastModification, command.Command{
		DocString: "Goto last modification",
		Run:       Runner(action.GotoLastModification),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(g(char('.'))),
	})

	r.RegisterCommand(actJumpForward, command.Command{
		DocString: "Jump forward on jumplist",
		Run:       Runner(action.JumpForward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(ctrl('i'), special("tab")),
	})
	r.RegisterCommand(actJumpBackward, command.Command{
		DocString: "Jump backward on jumplist",
		Run:       Runner(action.JumpBackward),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(ctrl('o')),
	})
	r.RegisterCommand(actSaveSelection, command.Command{
		DocString: "Save current selection to jumplist",
		Run:       Runner(action.SaveSelection),
		Modes:     []string{"NOR", "SEL"},
		Keys:      keyBinding(ctrl('s')),
	})

	r.RegisterCommand(actExtendCharLeft, command.Command{
		DocString: "Extend left",
		Run:       Runner(action.ExtendCharLeft),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('h'), special("left")),
	})
	r.RegisterCommand(actExtendVisualLineDown, command.Command{
		DocString: "Extend down",
		Run:       Runner(action.ExtendLineDown),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('j'), special("down")),
	})
	r.RegisterCommand(actExtendVisualLineUp, command.Command{
		DocString: "Extend up",
		Run:       Runner(action.ExtendLineUp),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('k'), special("up")),
	})
	r.RegisterCommand(actExtendCharRight, command.Command{
		DocString: "Extend right",
		Run:       Runner(action.ExtendCharRight),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('l'), special("right")),
	})
	r.RegisterCommand(actExtendNextWordStart, command.Command{
		DocString: "Extend to start of next word",
		Run:       Runner(action.ExtendNextWordStart),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('w')),
	})
	r.RegisterCommand(actExtendPrevWordStart, command.Command{
		DocString: "Extend to start of previous word",
		Run:       Runner(action.ExtendPrevWordStart),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('b')),
	})
	r.RegisterCommand(actExtendNextWordEnd, command.Command{
		DocString: "Extend to end of next word",
		Run:       Runner(action.ExtendNextWordEnd),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('e')),
	})
	r.RegisterCommand(actExtendPrevWordEnd, command.Command{
		DocString: "Extend to end of previous word",
		Run:       Runner(action.ExtendPrevWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendNextLongWordStart, command.Command{
		DocString: "Extend to start of next long word",
		Run:       Runner(action.ExtendNextLongWordStart),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('W')),
	})
	r.RegisterCommand(actExtendPrevLongWordStart, command.Command{
		DocString: "Extend to start of previous long word",
		Run:       Runner(action.ExtendPrevLongWordStart),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('B')),
	})
	r.RegisterCommand(actExtendNextLongWordEnd, command.Command{
		DocString: "Extend to end of next long word",
		Run:       Runner(action.ExtendNextLongWordEnd),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('E')),
	})
	r.RegisterCommand(actExtendPrevLongWordEnd, command.Command{
		DocString: "Extend to end of previous long word",
		Run:       Runner(action.ExtendPrevLongWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendNextSubWordStart, command.Command{
		DocString: "Extend to start of next sub-word",
		Run:       Runner(action.ExtendNextSubWordStart),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendPrevSubWordStart, command.Command{
		DocString: "Extend to start of previous sub-word",
		Run:       Runner(action.ExtendPrevSubWordStart),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendNextSubWordEnd, command.Command{
		DocString: "Extend to end of next sub-word",
		Run:       Runner(action.ExtendNextSubWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendPrevSubWordEnd, command.Command{
		DocString: "Extend to end of previous sub-word",
		Run:       Runner(action.ExtendPrevSubWordEnd),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendNextChar, command.Command{
		DocString: "Extend to next occurrence of char",
		Run:       Continuation(findCharAction(true, true, true)),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('f')),
	})
	r.RegisterCommand(actExtendTillChar, command.Command{
		DocString: "Extend till next occurrence of char",
		Run:       Continuation(findCharAction(true, false, true)),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('t')),
	})
	r.RegisterCommand(actExtendPrevChar, command.Command{
		DocString: "Extend to previous occurrence of char",
		Run:       Continuation(findCharAction(false, true, true)),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('F')),
	})
	r.RegisterCommand(actExtendTillPrevChar, command.Command{
		DocString: "Extend till previous occurrence of char",
		Run:       Continuation(findCharAction(false, false, true)),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('T')),
	})
	r.RegisterCommand(actExtendToLineStart, command.Command{
		DocString: "Extend to line start",
		Run:       Runner(action.ExtendToLineStart),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(special("home")),
	})
	r.RegisterCommand(actExtendToLineEnd, command.Command{
		DocString: "Extend to line end",
		Run:       Runner(action.ExtendToLineEnd),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(special("end")),
	})
	r.RegisterCommand(actExtendToLineEndNewline, command.Command{
		DocString: "Extend to line end",
		Run:       Runner(action.ExtendToLineEndNewline),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendToNonWhitespace, command.Command{
		DocString: "Extend to first non-blank in line",
		Run:       Runner(action.ExtendToNonWhitespace),
		Signature: sig(),
	})

	r.RegisterCommand(actGotoLineOrExtendFileStart, command.Command{
		DocString: "Extend to line number `<n>` else file start",
		Run:       Continuation(gotoLineOrExtendFileStartAction),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(g(char('g'))),
	})
	r.RegisterCommand(actExtendToColumn, command.Command{
		DocString: "Extend to column",
		Run:       Runner(action.ExtendToColumn),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(g(char('|'))),
	})
	r.RegisterCommand(actExtendToLastLine, command.Command{
		DocString: "Extend to last line",
		Run:       Runner(action.ExtendToLastLine),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(g(char('e'))),
	})
	r.RegisterCommand(actExtendToFileEnd, command.Command{
		DocString: "Extend to file end",
		Run:       Runner(action.ExtendToFileEnd),
		Signature: sig(),
	})
}

func gotoLineAction(e *view.Editor) command.Continuation {
	if n := e.Count(); n > 0 {
		action.GotoLine(e, n)
	}
	e.ResetCount()
	return nil
}

func gotoLineOrFileStartAction(e *view.Editor) command.Continuation {
	if n := e.Count(); n > 0 {
		action.GotoLine(e, n)
	} else {
		action.MoveFileStart(e)
	}
	e.ResetCount()
	return nil
}

func gotoLineOrExtendFileStartAction(e *view.Editor) command.Continuation {
	if n := e.Count(); n > 0 {
		action.GotoLine(e, n)
	} else {
		action.ExtendToFileStart(e)
	}
	e.ResetCount()
	return nil
}

func findCharAction(fwd, inc, ext bool) command.KeyAction {
	h := map[findCharHintKey]string{
		{forward: true, inclusive: true}:   "f",
		{forward: true, inclusive: false}:  "t",
		{forward: false, inclusive: true}:  "F",
		{forward: false, inclusive: false}: "T",
	}[findCharHintKey{forward: fwd, inclusive: inc}]
	return func(e *view.Editor) command.Continuation {
		e.SetHint(h + " ...")
		return func(e *view.Editor, k command.KeyEvent) command.Continuation {
			if k.Code.Char != 0 && k.Mods == command.ModNone {
				target := k.Code.Char
				action.FindChar(action.FindCharArgs{
					Editor: e, Ch: target, Forward: fwd,
					Inclusive: inc, Extend: ext,
				})
				e.SetLastMotion(func(e *view.Editor) {
					action.FindChar(action.FindCharArgs{
						Editor: e, Ch: target, Forward: fwd,
						Inclusive: inc, Extend: ext,
					})
				})
			}
			e.SetHint("")
			return nil
		}
	}
}

func gotoFileAction(e *view.Editor) command.Continuation {
	path, err := action.GotoFile(e)
	if err != nil {
		e.SetStatusMsg("error: " + err.Error())
		return nil
	}
	if _, err2 := e.OpenFile(path); err2 != nil {
		e.SetStatusMsg("error: " + err2.Error())
	}
	return nil
}
