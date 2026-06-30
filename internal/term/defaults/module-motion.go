package defaults

import (
	"strconv"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

type (
	findCharHintKey struct {
		forward   bool
		inclusive bool
	}

	motionSection struct {
		Editor struct {
			ScrollOff   *int `toml:"scrolloff"`
			ScrollLines *int `toml:"scroll-lines"`
		} `toml:"editor"`
	}
)

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
	actMoveLineNonWhitespace     = "goto_first_nonwhitespace"
	actMoveFileEnd               = "goto_last_line"
	actGotoWindowTop             = "goto_window_top"
	actGotoWindowCenter          = "goto_window_center"
	actGotoWindowBottom          = "goto_window_bottom"
	actGotoLastAccessedFile      = "goto_last_accessed_file"
	actGotoLastModifiedFile      = "goto_last_modified_file"
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
	actGotoNextParagraph         = "goto_next_paragraph"
	actGotoPrevParagraph         = "goto_prev_paragraph"
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

func motionModule() command.Module {
	cfg := new(motionSection)
	g := prefixed(char('g'))
	prev := prefixed(char('['))
	next := prefixed(char(']'))

	return command.Module{
		Commands: map[string]command.Command{
			actMoveLeft: {
				DocString: "Move left",
				Run:       Runner(action.MoveLeft),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   keyBinding(char('h'), special("left")),
					"INS": keyBinding(special("left")),
				},
			},
			actMoveDown: {
				DocString: "Move down",
				Run:       Runner(action.MoveDown),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   keyBinding(char('j'), special("down")),
					"INS": keyBinding(special("down")),
				},
			},
			actMoveUp: {
				DocString: "Move up",
				Run:       Runner(action.MoveUp),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   keyBinding(char('k'), special("up")),
					"INS": keyBinding(special("up")),
				},
			},
			actMoveRight: {
				DocString: "Move right",
				Run:       Runner(action.MoveRight),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   keyBinding(char('l'), special("right")),
					"INS": keyBinding(special("right")),
				},
			},
			actMoveNextWordStart: {
				DocString: "Move to start of next word",
				Run:       Runner(action.MoveWordForward),
				Modes:     []string{"NOR"},
				Keys:      keys(char('w')),
			},
			actMovePrevWordStart: {
				DocString: "Move to start of previous word",
				Run:       Runner(action.MoveWordBackward),
				Modes:     []string{"NOR"},
				Keys:      keys(char('b')),
			},
			actMoveNextWordEnd: {
				DocString: "Move to end of next word",
				Run:       Runner(action.MoveWordEnd),
				Modes:     []string{"NOR"},
				Keys:      keys(char('e')),
			},
			actMovePrevWordEnd: {
				DocString: "Move to end of previous word",
				Run:       Runner(action.MovePrevWordEnd),
				Signature: sig(),
			},
			actMoveNextLongWordStart: {
				DocString: "Move to start of next long word",
				Run:       Runner(action.MoveLongWordForward),
				Modes:     []string{"NOR"},
				Keys:      keys(char('W')),
			},
			actMovePrevLongWordStart: {
				DocString: "Move to start of previous long word",
				Run:       Runner(action.MoveLongWordBackward),
				Modes:     []string{"NOR"},
				Keys:      keys(char('B')),
			},
			actMoveNextLongWordEnd: {
				DocString: "Move to end of next long word",
				Run:       Runner(action.MoveLongWordEnd),
				Modes:     []string{"NOR"},
				Keys:      keys(char('E')),
			},
			actMovePrevLongWordEnd: {
				DocString: "Move to end of previous long word",
				Run:       Runner(action.MovePrevLongWordEnd),
				Signature: sig(),
			},
			actMoveNextSubWordStart: {
				DocString: "Move to start of next sub-word",
				Run:       Runner(action.MoveNextSubWordStart),
				Signature: sig(),
			},
			actMovePrevSubWordStart: {
				DocString: "Move to start of previous sub-word",
				Run:       Runner(action.MovePrevSubWordStart),
				Signature: sig(),
			},
			actMoveNextSubWordEnd: {
				DocString: "Move to end of next sub-word",
				Run:       Runner(action.MoveNextSubWordEnd),
				Signature: sig(),
			},
			actMovePrevSubWordEnd: {
				DocString: "Move to end of previous sub-word",
				Run:       Runner(action.MovePrevSubWordEnd),
				Signature: sig(),
			},
			actGotoLineStart: {
				DocString: "Goto line start",
				Run:       Runner(action.MoveLineStart),
				Modes:     []string{"NOR", "INS"},
				Keys:      keys(special("home")),
			},
			actGotoLineEnd: {
				DocString: "Goto line end",
				Run:       Runner(action.MoveLineEnd),
				Modes:     []string{"NOR"},
				Keys:      keys(special("end")),
			},
			actFindNextChar: {
				DocString: "Move to next occurrence of char",
				Run:       Continuation(findCharAction(true, true, false)),
				Modes:     []string{"NOR"},
				Keys:      keys(char('f')),
			},
			actFindTillChar: {
				DocString: "Move till next occurrence of char",
				Run:       Continuation(findCharAction(true, false, false)),
				Modes:     []string{"NOR"},
				Keys:      keys(char('t')),
			},
			actFindPrevChar: {
				DocString: "Move to previous occurrence of char",
				Run:       Continuation(findCharAction(false, true, false)),
				Modes:     []string{"NOR"},
				Keys:      keys(char('F')),
			},
			actTillPrevChar: {
				DocString: "Move till previous occurrence of char",
				Run:       Continuation(findCharAction(false, false, false)),
				Modes:     []string{"NOR"},
				Keys:      keys(char('T')),
			},
			actGotoLine: {
				DocString: "Goto line",
				Run:       Continuation(gotoLineAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('G')),
			},
			actGotoLineOrFileStart: {
				DocString: "Goto line number `<n>` else file start",
				Run:       Continuation(gotoLineOrFileStartAction),
				Modes:     []string{"NOR"},
				Keys:      keys(g(char('g'))),
			},
			actGotoFile: {
				DocString: "Goto files/URLs in selections",
				Run:       Continuation(gotoFileAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('f'))),
			},
			actGotoColumn: {
				DocString: "Goto column",
				Run:       Runner(action.ExtendToColumn),
				Modes:     []string{"NOR"},
				Keys:      keys(g(char('|'))),
			},
			actGotoWindowTop: {
				DocString: "Goto window top",
				Run:       Runner(action.GotoWindowTop),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('t'))),
			},
			actGotoWindowCenter: {
				DocString: "Goto window center",
				Run:       Runner(action.GotoWindowCenter),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('c'))),
			},
			actGotoWindowBottom: {
				DocString: "Goto window bottom",
				Run:       Runner(action.GotoWindowBottom),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('b'))),
			},
			actMoveFileEnd: {
				DocString: "Goto last line",
				Run:       Runner(action.MoveFileEnd),
				Modes:     []string{"NOR"},
				Keys:      keys(g(char('e'))),
			},
			actMoveLineNonWhitespace: {
				DocString: "Goto first non-blank in line",
				Run:       Runner(action.MoveLineNonWhitespace),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('s'))),
			},
			actGotoLastAccessedFile: {
				DocString: "Goto last accessed file",
				Run:       Runner(action.GotoLastAccessedFile),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('a'))),
			},
			actGotoLastModifiedFile: {
				DocString: "Goto last modified file",
				Run:       Runner(action.GotoLastModifiedFile),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('m'))),
			},
			actGotoLastModification: {
				DocString: "Goto last modification",
				Run:       Runner(action.GotoLastModification),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('.'))),
			},
			actJumpForward: {
				DocString: "Jump forward on jumplist",
				Run:       Runner(action.JumpForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('i'), special("tab")),
			},
			actJumpBackward: {
				DocString: "Jump backward on jumplist",
				Run:       Runner(action.JumpBackward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('o')),
			},
			actSaveSelection: {
				DocString: "Save current selection to jumplist",
				Run:       Runner(action.SaveSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('s')),
			},
			actGotoNextParagraph: {
				DocString: "Goto next paragraph",
				Run:       Runner(action.GotoNextParagraph),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(next(char('p'))),
			},
			actGotoPrevParagraph: {
				DocString: "Goto previous paragraph",
				Run:       Runner(action.GotoPrevParagraph),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(prev(char('p'))),
			},
			actExtendCharLeft: {
				DocString: "Extend left",
				Run:       Runner(action.ExtendCharLeft),
				Modes:     []string{"SEL"},
				Keys:      keys(char('h'), special("left")),
			},
			actExtendVisualLineDown: {
				DocString: "Extend down",
				Run:       Runner(action.ExtendLineDown),
				Modes:     []string{"SEL"},
				Keys:      keys(char('j'), special("down")),
			},
			actExtendVisualLineUp: {
				DocString: "Extend up",
				Run:       Runner(action.ExtendLineUp),
				Modes:     []string{"SEL"},
				Keys:      keys(char('k'), special("up")),
			},
			actExtendCharRight: {
				DocString: "Extend right",
				Run:       Runner(action.ExtendCharRight),
				Modes:     []string{"SEL"},
				Keys:      keys(char('l'), special("right")),
			},
			actExtendNextWordStart: {
				DocString: "Extend to start of next word",
				Run:       Runner(action.ExtendNextWordStart),
				Modes:     []string{"SEL"},
				Keys:      keys(char('w')),
			},
			actExtendPrevWordStart: {
				DocString: "Extend to start of previous word",
				Run:       Runner(action.ExtendPrevWordStart),
				Modes:     []string{"SEL"},
				Keys:      keys(char('b')),
			},
			actExtendNextWordEnd: {
				DocString: "Extend to end of next word",
				Run:       Runner(action.ExtendNextWordEnd),
				Modes:     []string{"SEL"},
				Keys:      keys(char('e')),
			},
			actExtendPrevWordEnd: {
				DocString: "Extend to end of previous word",
				Run:       Runner(action.ExtendPrevWordEnd),
				Signature: sig(),
			},
			actExtendNextLongWordStart: {
				DocString: "Extend to start of next long word",
				Run:       Runner(action.ExtendNextLongWordStart),
				Modes:     []string{"SEL"},
				Keys:      keys(char('W')),
			},
			actExtendPrevLongWordStart: {
				DocString: "Extend to start of previous long word",
				Run:       Runner(action.ExtendPrevLongWordStart),
				Modes:     []string{"SEL"},
				Keys:      keys(char('B')),
			},
			actExtendNextLongWordEnd: {
				DocString: "Extend to end of next long word",
				Run:       Runner(action.ExtendNextLongWordEnd),
				Modes:     []string{"SEL"},
				Keys:      keys(char('E')),
			},
			actExtendPrevLongWordEnd: {
				DocString: "Extend to end of previous long word",
				Run:       Runner(action.ExtendPrevLongWordEnd),
				Signature: sig(),
			},
			actExtendNextSubWordStart: {
				DocString: "Extend to start of next sub-word",
				Run:       Runner(action.ExtendNextSubWordStart),
				Signature: sig(),
			},
			actExtendPrevSubWordStart: {
				DocString: "Extend to start of previous sub-word",
				Run:       Runner(action.ExtendPrevSubWordStart),
				Signature: sig(),
			},
			actExtendNextSubWordEnd: {
				DocString: "Extend to end of next sub-word",
				Run:       Runner(action.ExtendNextSubWordEnd),
				Signature: sig(),
			},
			actExtendPrevSubWordEnd: {
				DocString: "Extend to end of previous sub-word",
				Run:       Runner(action.ExtendPrevSubWordEnd),
				Signature: sig(),
			},
			actExtendNextChar: {
				DocString: "Extend to next occurrence of char",
				Run:       Continuation(findCharAction(true, true, true)),
				Modes:     []string{"SEL"},
				Keys:      keys(char('f')),
			},
			actExtendTillChar: {
				DocString: "Extend till next occurrence of char",
				Run:       Continuation(findCharAction(true, false, true)),
				Modes:     []string{"SEL"},
				Keys:      keys(char('t')),
			},
			actExtendPrevChar: {
				DocString: "Extend to previous occurrence of char",
				Run:       Continuation(findCharAction(false, true, true)),
				Modes:     []string{"SEL"},
				Keys:      keys(char('F')),
			},
			actExtendTillPrevChar: {
				DocString: "Extend till previous occurrence of char",
				Run:       Continuation(findCharAction(false, false, true)),
				Modes:     []string{"SEL"},
				Keys:      keys(char('T')),
			},
			actExtendToLineStart: {
				DocString: "Extend to line start",
				Run:       Runner(action.ExtendToLineStart),
				Modes:     []string{"SEL"},
				Keys:      keys(special("home")),
			},
			actExtendToLineEnd: {
				DocString: "Extend to line end",
				Run:       Runner(action.ExtendToLineEnd),
				Modes:     []string{"SEL"},
				Keys:      keys(special("end")),
			},
			actExtendToLineEndNewline: {
				DocString: "Extend to line end",
				Run:       Runner(action.ExtendToLineEndNewline),
				Signature: sig(),
			},
			actExtendToNonWhitespace: {
				DocString: "Extend to first non-blank in line",
				Run:       Runner(action.ExtendToNonWhitespace),
				Signature: sig(),
			},
			actGotoLineOrExtendFileStart: {
				DocString: "Extend to line number `<n>` else file start",
				Run:       Continuation(gotoLineOrExtendFileStartAction),
				Modes:     []string{"SEL"},
				Keys:      keys(g(char('g'))),
			},
			actExtendToColumn: {
				DocString: "Extend to column",
				Run:       Runner(action.ExtendToColumn),
				Modes:     []string{"SEL"},
				Keys:      keys(g(char('|'))),
			},
			actExtendToLastLine: {
				DocString: "Extend to last line",
				Run:       Runner(action.ExtendToLastLine),
				Modes:     []string{"SEL"},
				Keys:      keys(g(char('e'))),
			},
			actExtendToFileEnd: {
				DocString: "Extend to file end",
				Run:       Runner(action.ExtendToFileEnd),
				Signature: sig(),
			},
		},
		Options: []command.Option{
			{
				Key: "editor.scrolloff",
				Get: func(e *view.Editor) (string, error) {
					return strconv.Itoa(e.Options().ScrollOff), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseNonNegInt(s)
					if err != nil {
						return err
					}
					e.Options().ScrollOff = v
					return nil
				},
			},
			{
				Key: "editor.scroll-lines",
				Get: func(e *view.Editor) (string, error) {
					return strconv.Itoa(e.Options().ScrollLines), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParsePositiveInt(s)
					if err != nil {
						return err
					}
					e.Options().ScrollLines = v
					return nil
				},
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = motionSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				opts.ScrollOff = intOr(
					cfg.Editor.ScrollOff, view.DefaultScrollOff,
				)
				opts.ScrollLines = intOr(
					cfg.Editor.ScrollLines, view.DefaultScrollLines,
				)
			},
		},
	}
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
	target, err := action.GotoFileTarget(e)
	if err != nil {
		e.SetStatusMsg("error: " + err.Error())
		return nil
	}
	if target.URL != "" {
		if err2 := action.OpenExternalURL(target.URL); err2 != nil {
			e.SetStatusMsg("error: " + err2.Error())
		}
		return nil
	}
	if _, err2 := e.OpenFile(target.Path); err2 != nil {
		e.SetStatusMsg("error: " + err2.Error())
	}
	return nil
}
