package defaults

import (
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type editSection struct {
	Editor struct {
		AutoPairs        language.AutoPairConfig `toml:"auto-pairs"`
		ContinueComments *bool                   `toml:"continue-comments"`
		AutoSave         config.AutoSave         `toml:"auto-save"`
		AtomicSave       *bool                   `toml:"atomic-save"`
	} `toml:"editor"`
}

const (
	actReplace                  = "replace"
	actSwitchCase               = "switch_case"
	actSwitchToLowercase        = "switch_to_lowercase"
	actSwitchToUppercase        = "switch_to_uppercase"
	actDeleteSelection          = "delete_selection"
	actDeleteSelectionNoYank    = "delete_selection_noyank"
	actChangeSelection          = "change_selection"
	actChangeSelectionNoYank    = "change_selection_noyank"
	actUndo                     = "undo"
	actRedo                     = "redo"
	actEarlier                  = "earlier"
	actLater                    = "later"
	actIndent                   = "indent"
	actUnindent                 = "unindent"
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
)

func editModule() command.Module {
	cfg := new(editSection)
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actSelectMode,
				DocString: "Enter selection extend mode",
				Run:       Continuation(selectModeAction),
				Modes:     []string{"NOR"},
				Keys:      keys(char('v')),
			},
			{
				Name:      actNormalMode,
				DocString: "Enter normal mode",
				Run:       Continuation(normalModeAction),
				Keys:      keys(special("esc")),
			},
			{
				Name:      actInsertMode,
				DocString: "Insert before selection",
				Run:       Continuation(insertModeAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('i')),
			},
			{
				Name:      actInsertAtLineStart,
				DocString: "Insert at start of line",
				Run:       Continuation(insertAtLineStartAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('I')),
			},
			{
				Name:      actAppendMode,
				DocString: "Append after selection",
				Run:       Continuation(appendModeAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('a')),
			},
			{
				Name:      actAppendToLine,
				DocString: "Insert at end of line",
				Run:       Continuation(appendToLineAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('A')),
			},
			{
				Name:      actOpenBelow,
				DocString: "Open new line below selection",
				Run:       Continuation(openBelowAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('o')),
			},
			{
				Name:      actOpenAbove,
				DocString: "Open new line above selection",
				Run:       Continuation(openAboveAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('O')),
			},
			{
				Name:      actExitSelectMode,
				DocString: "Exit selection mode",
				Run:       Continuation(normalModeAction),
				Modes:     []string{"SEL"},
				Keys:      keys(special("esc")),
			},
			{
				Name:      actReplace,
				DocString: "Replace with new char",
				Run:       Continuation(replaceCharAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('r')),
			},
			{
				Name:      actDeleteSelection,
				DocString: "Delete selection",
				Run:       Runner(action.DeleteSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('d')),
			},
			{
				Name:      actDeleteSelectionNoYank,
				DocString: "Delete selection without yanking",
				Run:       Runner(action.DeleteSelectionNoYank),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('d')),
			},
			{
				Name:      actChangeSelection,
				DocString: "Change selection",
				Run:       Runner(action.ChangeSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('c')),
			},
			{
				Name:      actChangeSelectionNoYank,
				DocString: "Change selection without yanking",
				Run:       Runner(action.ChangeSelectionNoYank),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('c')),
			},
			{
				Name:      actUndo,
				DocString: "Undo change",
				Run:       Continuation(undoAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('u')),
			},
			{
				Name:      actRedo,
				DocString: "Redo change",
				Run:       Continuation(redoAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('U')),
			},
			{
				Name:      actEarlier,
				DocString: "Move backward in history",
				Run:       Continuation(earlierAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('u')),
			},
			{
				Name:      actLater,
				DocString: "Move forward in history",
				Run:       Continuation(laterAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('U')),
			},
			{
				Name:      actSwitchCase,
				DocString: "Switch (toggle) case",
				Run:       Runner(action.SwitchCase),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('~')),
			},
			{
				Name:      actSwitchToLowercase,
				DocString: "Switch to lowercase",
				Run:       Runner(action.SwitchToLowercase),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('`')),
			},
			{
				Name:      actSwitchToUppercase,
				DocString: "Switch to uppercase",
				Run:       Runner(action.SwitchToUppercase),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('`')),
			},
			{
				Name:      actRepeatLastMotion,
				DocString: "Repeat last motion",
				Run:       Runner(action.RepeatLastMotion),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('.')),
			},
			{
				Name:      actIndent,
				DocString: "Indent selection",
				Run:       Runner(action.Indent),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('>')),
			},
			{
				Name:      actUnindent,
				DocString: "Unindent selection",
				Run:       Runner(action.Unindent),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('<')),
			},
			{
				Name:      actJoinSelections,
				DocString: "Join lines inside selection",
				Run:       Runner(action.JoinSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('J')),
			},
			{
				Name:      actJoinSelectionsSpace,
				DocString: "Join lines inside selection and select spaces",
				Run:       Runner(action.JoinSelectionsSpace),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('J')),
			},
			{
				Name:      actAlignSelections,
				DocString: "Align selections in column",
				Run:       Runner(action.AlignSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('&')),
			},
			{
				Name:      actTrimSelections,
				DocString: "Trim whitespace from selections",
				Run:       Runner(action.TrimSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('_')),
			},
			{
				Name:      actRotateSelectionsBackward,
				DocString: "Rotate selections backward",
				Run:       Runner(action.RotateSelectionsBackward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('(')),
			},
			{
				Name:      actRotateSelectionsForward,
				DocString: "Rotate selections forward",
				Run:       Runner(action.RotateSelectionsForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(')')),
			},
			{
				Name:      actRotateContentsBackward,
				DocString: "Rotate selections contents backward",
				Run:       Runner(action.RotateContentsBackward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('(')),
			},
			{
				Name:      actRotateContentsForward,
				DocString: "Rotate selection contents forward",
				Run:       Runner(action.RotateContentsForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt(')')),
			},
			{
				Name:      actEnsureForward,
				DocString: "Ensure all selections face forward",
				Run:       Runner(action.EnsureForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt(':')),
			},
			{
				Name:      actIncrement,
				DocString: "Increment item under cursor",
				Run:       Runner(action.Increment),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('a')),
			},
			{
				Name:      actDecrement,
				DocString: "Decrement item under cursor",
				Run:       Runner(action.Decrement),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(ctrl('x')),
			},
		},
		Options: []command.Option{
			{
				Key: "auto-pairs",
				Get: func(e *view.Editor) (string, error) {
					return strconv.FormatBool(e.Options().HasAutoPairs), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseBool(s)
					if err != nil {
						return err
					}
					if v {
						e.Options().AutoPairMap = core.DefaultAutoPairs()
					}
					e.Options().HasAutoPairs = v
					return nil
				},
				Toggle: func(e *view.Editor) (string, error) {
					v := !e.Options().HasAutoPairs
					if v {
						e.Options().AutoPairMap = core.DefaultAutoPairs()
					}
					e.Options().HasAutoPairs = v
					return strconv.FormatBool(v), nil
				},
			},
			editorBoolOption("continue-comments",
				func(e *view.Editor) bool {
					return e.Options().ContinueComments
				},
				func(e *view.Editor, v bool) {
					e.Options().ContinueComments = v
				},
			),
			editorBoolOption("auto-save",
				func(e *view.Editor) bool {
					return e.Options().AutoSaveFocusLost
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSaveFocusLost = v
				},
			),
			editorBoolOption("auto-save.focus-lost",
				func(e *view.Editor) bool {
					return e.Options().AutoSaveFocusLost
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSaveFocusLost = v
				},
			),
			editorBoolOption("auto-save.after-delay.enable",
				func(e *view.Editor) bool {
					return e.Options().AutoSaveAfterDelay
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSaveAfterDelay = v
				},
			),
			{
				Key: "auto-save.after-delay.timeout",
				Get: func(e *view.Editor) (string, error) {
					return strconv.Itoa(e.Options().AutoSaveDelayTimeout), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParsePositiveInt(s)
					if err != nil {
						return err
					}
					e.Options().AutoSaveDelayTimeout = v
					return nil
				},
			},
			editorBoolOption("atomic-save",
				func(e *view.Editor) bool {
					return e.Options().AtomicSave
				},
				func(e *view.Editor, v bool) {
					e.Options().AtomicSave = v
				},
			),
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = editSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				ap, ok := cfg.Editor.AutoPairs.OrDefault()
				opts.AutoPairMap = ap
				opts.HasAutoPairs = ok
				opts.ContinueComments = boolOr(
					cfg.Editor.ContinueComments, true,
				)
				opts.AtomicSave = boolOr(cfg.Editor.AtomicSave, true)
				opts.AutoSaveFocusLost = boolOr(
					cfg.Editor.AutoSave.FocusLost, false,
				)
				opts.AutoSaveAfterDelay = boolOr(
					cfg.Editor.AutoSave.AfterDelay.Enable, false,
				)
				opts.AutoSaveDelayTimeout = intOr(
					cfg.Editor.AutoSave.AfterDelay.Timeout,
					view.DefaultAutoSaveDelay,
				)
			},
		},
	}
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
	n := e.Count()
	if n == 0 {
		n = 1
	}
	e.Earlier(core.UndoSteps(n))
	return nil
}

func laterAction(e *view.Editor) command.Continuation {
	n := e.Count()
	if n == 0 {
		n = 1
	}
	e.Later(core.UndoSteps(n))
	return nil
}
