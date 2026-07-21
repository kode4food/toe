package motion

import (
	"strconv"

	"github.com/kode4food/toe/internal/term/builtin/kit"
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
	actJumplistPicker            = "jumplist_picker"
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

// CursorModule returns the cursor-motion and goto commands
func CursorModule() command.Module {
	cfg := new(motionSection)
	g := kit.Prefixed(kit.Char('g'))
	prev := kit.Prefixed(kit.Char('['))
	next := kit.Prefixed(kit.Char(']'))

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actMoveLeft,
				DocString: "Move left",
				Run:       kit.Runner(action.MoveLeft),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   kit.KeyBinding(kit.Char('h'), kit.Left),
					"INS": kit.KeyBinding(kit.Left),
				},
			},
			{
				Name:      actMoveDown,
				DocString: "Move down",
				Run:       kit.Runner(action.MoveDown),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   kit.KeyBinding(kit.Char('j'), kit.Down),
					"INS": kit.KeyBinding(kit.Down),
				},
			},
			{
				Name:      actMoveUp,
				DocString: "Move up",
				Run:       kit.Runner(action.MoveUp),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   kit.KeyBinding(kit.Char('k'), kit.Up),
					"INS": kit.KeyBinding(kit.Up),
				},
			},
			{
				Name:      actMoveRight,
				DocString: "Move right",
				Run:       kit.Runner(action.MoveRight),
				Modes:     []string{"NOR", "INS"},
				Keys: map[string][]command.KeyBinding{
					"*":   kit.KeyBinding(kit.Char('l'), kit.Right),
					"INS": kit.KeyBinding(kit.Right),
				},
			},
			{
				Name:      actMoveNextWordStart,
				DocString: "Move to start of next word",
				Run:       kit.Runner(action.MoveWordForward),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('w')),
			},
			{
				Name:      actMovePrevWordStart,
				DocString: "Move to start of previous word",
				Run:       kit.Runner(action.MoveWordBackward),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('b')),
			},
			{
				Name:      actMoveNextWordEnd,
				DocString: "Move to end of next word",
				Run:       kit.Runner(action.MoveWordEnd),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('e')),
			},
			{
				Name:      actMovePrevWordEnd,
				DocString: "Move to end of previous word",
				Run:       kit.Runner(action.MovePrevWordEnd),
				Modes:     []string{"NOR"},
				Signature: kit.Sig(),
			},
			{
				Name:      actMoveNextLongWordStart,
				DocString: "Move to start of next long word",
				Run:       kit.Runner(action.MoveLongWordForward),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('W')),
			},
			{
				Name:      actMovePrevLongWordStart,
				DocString: "Move to start of previous long word",
				Run:       kit.Runner(action.MoveLongWordBackward),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('B')),
			},
			{
				Name:      actMoveNextLongWordEnd,
				DocString: "Move to end of next long word",
				Run:       kit.Runner(action.MoveLongWordEnd),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('E')),
			},
			{
				Name:      actMovePrevLongWordEnd,
				DocString: "Move to end of previous long word",
				Run:       kit.Runner(action.MovePrevLongWordEnd),
				Modes:     []string{"NOR"},
				Signature: kit.Sig(),
			},
			{
				Name:      actMoveNextSubWordStart,
				DocString: "Move to start of next sub-word",
				Run:       kit.Runner(action.MoveNextSubWordStart),
				Modes:     []string{"NOR"},
				Signature: kit.Sig(),
			},
			{
				Name:      actMovePrevSubWordStart,
				DocString: "Move to start of previous sub-word",
				Run:       kit.Runner(action.MovePrevSubWordStart),
				Modes:     []string{"NOR"},
				Signature: kit.Sig(),
			},
			{
				Name:      actMoveNextSubWordEnd,
				DocString: "Move to end of next sub-word",
				Run:       kit.Runner(action.MoveNextSubWordEnd),
				Modes:     []string{"NOR"},
				Signature: kit.Sig(),
			},
			{
				Name:      actMovePrevSubWordEnd,
				DocString: "Move to end of previous sub-word",
				Run:       kit.Runner(action.MovePrevSubWordEnd),
				Modes:     []string{"NOR"},
				Signature: kit.Sig(),
			},
			{
				Name:      actGotoLineStart,
				DocString: "Goto line start",
				Run:       kit.Runner(action.MoveLineStart),
				Modes:     []string{"NOR", "INS"},
				Keys:      kit.Keys(kit.Home),
			},
			{
				Name:      actGotoLineEnd,
				DocString: "Goto line end",
				Run:       kit.Runner(action.MoveLineEnd),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.End),
			},
			{
				Name:      actFindNextChar,
				DocString: "Move to next occurrence of char",
				Run:       kit.Continuation(findCharAction(true, true, false)),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('f')),
			},
			{
				Name:      actFindTillChar,
				DocString: "Move till next occurrence of char",
				Run:       kit.Continuation(findCharAction(true, false, false)),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('t')),
			},
			{
				Name:      actFindPrevChar,
				DocString: "Move to previous occurrence of char",
				Run:       kit.Continuation(findCharAction(false, true, false)),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('F')),
			},
			{
				Name:      actTillPrevChar,
				DocString: "Move till previous occurrence of char",
				Run: kit.Continuation(
					findCharAction(false, false, false),
				),
				Modes: []string{"NOR"},
				Keys:  kit.Keys(kit.Char('T')),
			},
			{
				Name:      actGotoLine,
				DocString: "Goto line",
				Run:       kit.Continuation(gotoLineAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('G')),
			},
			{
				Name:      actGotoLineOrFileStart,
				DocString: "Goto line number `<n>` else file start",
				Run:       kit.Continuation(gotoLineOrFileStartAction),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(g(kit.Char('g'))),
			},
			{
				Name:      actGotoFile,
				DocString: "Goto files/URLs in selections",
				Run:       kit.Continuation(gotoFileAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('f'))),
			},
			{
				Name:      actGotoColumn,
				DocString: "Goto column",
				Run:       kit.Runner(action.ExtendToColumn),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(g(kit.Char('|'))),
			},
			{
				Name:      actGotoWindowTop,
				DocString: "Goto window top",
				Run:       kit.Runner(action.GotoWindowTop),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('t'))),
			},
			{
				Name:      actGotoWindowCenter,
				DocString: "Goto window center",
				Run:       kit.Runner(action.GotoWindowCenter),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('c'))),
			},
			{
				Name:      actGotoWindowBottom,
				DocString: "Goto window bottom",
				Run:       kit.Runner(action.GotoWindowBottom),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('b'))),
			},
			{
				Name:      actMoveFileEnd,
				DocString: "Goto last line",
				Run:       kit.Runner(action.MoveFileEnd),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(g(kit.Char('e'))),
			},
			{
				Name:      actMoveLineNonWhitespace,
				DocString: "Goto first non-blank in line",
				Run:       kit.Runner(action.MoveLineNonWhitespace),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('s'))),
			},
			{
				Name:      actGotoLastAccessedFile,
				DocString: "Goto last accessed file",
				Run:       kit.Runner(action.GotoLastAccessedFile),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('a'))),
			},
			{
				Name:      actGotoLastModifiedFile,
				DocString: "Goto last modified file",
				Run:       kit.Runner(action.GotoLastModifiedFile),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('m'))),
			},
			{
				Name:      actGotoLastModification,
				DocString: "Goto last modification",
				Run:       kit.Runner(action.GotoLastModification),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('.'))),
			},
			{
				Name:      actJumpForward,
				DocString: "Jump forward on jumplist",
				Run:       kit.Runner(action.JumpForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Ctrl('i'), kit.Tab),
			},
			{
				Name:      actJumpBackward,
				DocString: "Jump backward on jumplist",
				Run:       kit.Runner(action.JumpBackward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Ctrl('o')),
			},
			{
				Name:      actSaveSelection,
				DocString: "Save current selection to jumplist",
				Run:       kit.Runner(action.SaveSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Ctrl('s')),
			},
			{
				Name:      actGotoNextParagraph,
				DocString: "Goto next paragraph",
				Run:       kit.Runner(action.GotoNextParagraph),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(next(kit.Char('p'))),
			},
			{
				Name:      actGotoPrevParagraph,
				DocString: "Goto previous paragraph",
				Run:       kit.Runner(action.GotoPrevParagraph),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(prev(kit.Char('p'))),
			},
			{
				Name:      actExtendCharLeft,
				DocString: "Extend left",
				Run:       kit.Runner(action.ExtendCharLeft),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('h'), kit.Left),
			},
			{
				Name:      actExtendVisualLineDown,
				DocString: "Extend down",
				Run:       kit.Runner(action.ExtendLineDown),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('j'), kit.Down),
			},
			{
				Name:      actExtendVisualLineUp,
				DocString: "Extend up",
				Run:       kit.Runner(action.ExtendLineUp),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('k'), kit.Up),
			},
			{
				Name:      actExtendCharRight,
				DocString: "Extend right",
				Run:       kit.Runner(action.ExtendCharRight),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('l'), kit.Right),
			},
			{
				Name:      actExtendNextWordStart,
				DocString: "Extend to start of next word",
				Run:       kit.Runner(action.ExtendNextWordStart),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('w')),
			},
			{
				Name:      actExtendPrevWordStart,
				DocString: "Extend to start of previous word",
				Run:       kit.Runner(action.ExtendPrevWordStart),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('b')),
			},
			{
				Name:      actExtendNextWordEnd,
				DocString: "Extend to end of next word",
				Run:       kit.Runner(action.ExtendNextWordEnd),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('e')),
			},
			{
				Name:      actExtendPrevWordEnd,
				DocString: "Extend to end of previous word",
				Run:       kit.Runner(action.ExtendPrevWordEnd),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendNextLongWordStart,
				DocString: "Extend to start of next long word",
				Run:       kit.Runner(action.ExtendNextLongWordStart),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('W')),
			},
			{
				Name:      actExtendPrevLongWordStart,
				DocString: "Extend to start of previous long word",
				Run:       kit.Runner(action.ExtendPrevLongWordStart),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('B')),
			},
			{
				Name:      actExtendNextLongWordEnd,
				DocString: "Extend to end of next long word",
				Run:       kit.Runner(action.ExtendNextLongWordEnd),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('E')),
			},
			{
				Name:      actExtendPrevLongWordEnd,
				DocString: "Extend to end of previous long word",
				Run:       kit.Runner(action.ExtendPrevLongWordEnd),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendNextSubWordStart,
				DocString: "Extend to start of next sub-word",
				Run:       kit.Runner(action.ExtendNextSubWordStart),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendPrevSubWordStart,
				DocString: "Extend to start of previous sub-word",
				Run:       kit.Runner(action.ExtendPrevSubWordStart),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendNextSubWordEnd,
				DocString: "Extend to end of next sub-word",
				Run:       kit.Runner(action.ExtendNextSubWordEnd),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendPrevSubWordEnd,
				DocString: "Extend to end of previous sub-word",
				Run:       kit.Runner(action.ExtendPrevSubWordEnd),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendNextChar,
				DocString: "Extend to next occurrence of char",
				Run:       kit.Continuation(findCharAction(true, true, true)),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('f')),
			},
			{
				Name:      actExtendTillChar,
				DocString: "Extend till next occurrence of char",
				Run:       kit.Continuation(findCharAction(true, false, true)),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('t')),
			},
			{
				Name:      actExtendPrevChar,
				DocString: "Extend to previous occurrence of char",
				Run:       kit.Continuation(findCharAction(false, true, true)),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('F')),
			},
			{
				Name:      actExtendTillPrevChar,
				DocString: "Extend till previous occurrence of char",
				Run:       kit.Continuation(findCharAction(false, false, true)),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Char('T')),
			},
			{
				Name:      actExtendToLineStart,
				DocString: "Extend to line start",
				Run:       kit.Runner(action.ExtendToLineStart),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Home),
			},
			{
				Name:      actExtendToLineEnd,
				DocString: "Extend to line end",
				Run:       kit.Runner(action.ExtendToLineEnd),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.End),
			},
			{
				Name:      actExtendToLineEndNewline,
				DocString: "Extend to line end",
				Run:       kit.Runner(action.ExtendToLineEndNewline),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actExtendToNonWhitespace,
				DocString: "Extend to first non-blank in line",
				Run:       kit.Runner(action.ExtendToNonWhitespace),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actGotoLineOrExtendFileStart,
				DocString: "Extend to line number `<n>` else file start",
				Run:       kit.Continuation(gotoLineOrExtendFileStartAction),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(g(kit.Char('g'))),
			},
			{
				Name:      actExtendToColumn,
				DocString: "Extend to column",
				Run:       kit.Runner(action.ExtendToColumn),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(g(kit.Char('|'))),
			},
			{
				Name:      actExtendToLastLine,
				DocString: "Extend to last line",
				Run:       kit.Runner(action.ExtendToLastLine),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(g(kit.Char('e'))),
			},
			{
				Name:      actExtendToFileEnd,
				DocString: "Extend to file end",
				Run:       kit.Runner(action.ExtendToFileEnd),
				Modes:     []string{"SEL"},
				Signature: kit.Sig(),
			},
		},
		Options: []command.Option{
			{
				Key: "scrolloff",
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
				Key: "scroll-lines",
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
				opts.ScrollOff = kit.IntOr(
					cfg.Editor.ScrollOff, view.DefaultScrollOff,
				)
				opts.ScrollLines = kit.IntOr(
					cfg.Editor.ScrollLines, view.DefaultScrollLines,
				)
			},
		},
		Labels: []command.PrefixLabel{
			kit.Label("Goto", kit.Char('g'), "NOR", "SEL"),
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
					Editor:    e,
					Ch:        target,
					Forward:   fwd,
					Inclusive: inc,
					Extend:    ext,
				})
				e.SetLastMotion(func(e *view.Editor) {
					action.FindChar(action.FindCharArgs{
						Editor:    e,
						Ch:        target,
						Forward:   fwd,
						Inclusive: inc,
						Extend:    ext,
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
