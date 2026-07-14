package editing

import (
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/kit"
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

// EditModule returns the text-editing commands
func EditModule() command.Module {
	cfg := new(editSection)
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actSelectMode,
				DocString: "Enter selection extend mode",
				Run:       kit.Continuation(selectModeAction),
				Modes:     []string{"NOR"},
				Keys:      kit.Keys(kit.Char('v')),
			},
			{
				Name:      actNormalMode,
				DocString: "Enter normal mode",
				Run:       kit.Continuation(normalModeAction),
				Keys:      kit.Keys(kit.Special("esc")),
			},
			{
				Name:      actInsertMode,
				DocString: "Insert before selection",
				Run:       kit.Continuation(insertModeAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('i')),
			},
			{
				Name:      actInsertAtLineStart,
				DocString: "Insert at start of line",
				Run:       kit.Continuation(insertAtLineStartAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('I')),
			},
			{
				Name:      actAppendMode,
				DocString: "Append after selection",
				Run:       kit.Continuation(appendModeAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('a')),
			},
			{
				Name:      actAppendToLine,
				DocString: "Insert at end of line",
				Run:       kit.Continuation(appendToLineAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('A')),
			},
			{
				Name:      actOpenBelow,
				DocString: "Open new line below selection",
				Run:       kit.Continuation(openBelowAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('o')),
			},
			{
				Name:      actOpenAbove,
				DocString: "Open new line above selection",
				Run:       kit.Continuation(openAboveAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('O')),
			},
			{
				Name:      actExitSelectMode,
				DocString: "Exit selection mode",
				Run:       kit.Continuation(normalModeAction),
				Modes:     []string{"SEL"},
				Keys:      kit.Keys(kit.Special("esc")),
			},
			{
				Name:      actReplace,
				DocString: "Replace with new char",
				Run:       kit.Continuation(replaceCharAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('r')),
			},
			{
				Name:      actDeleteSelection,
				DocString: "Delete selection",
				Run:       kit.Runner(action.DeleteSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('d')),
			},
			{
				Name:      actDeleteSelectionNoYank,
				DocString: "Delete selection without yanking",
				Run:       kit.Runner(action.DeleteSelectionNoYank),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('d')),
			},
			{
				Name:      actChangeSelection,
				DocString: "Change selection",
				Run:       kit.Runner(action.ChangeSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('c')),
			},
			{
				Name:      actChangeSelectionNoYank,
				DocString: "Change selection without yanking",
				Run:       kit.Runner(action.ChangeSelectionNoYank),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('c')),
			},
			{
				Name:      actUndo,
				DocString: "Undo change",
				Run:       kit.Continuation(undoAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('u')),
			},
			{
				Name:      actRedo,
				DocString: "Redo change",
				Run:       kit.Continuation(redoAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('U')),
			},
			{
				Name:      actEarlier,
				DocString: "Move backward in history",
				Run:       kit.Continuation(earlierAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('u')),
			},
			{
				Name:      actLater,
				DocString: "Move forward in history",
				Run:       kit.Continuation(laterAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('U')),
			},
			{
				Name:      actSwitchCase,
				DocString: "Switch (toggle) case",
				Run:       kit.Runner(action.SwitchCase),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('~')),
			},
			{
				Name:      actSwitchToLowercase,
				DocString: "Switch to lowercase",
				Run:       kit.Runner(action.SwitchToLowercase),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('`')),
			},
			{
				Name:      actSwitchToUppercase,
				DocString: "Switch to uppercase",
				Run:       kit.Runner(action.SwitchToUppercase),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('`')),
			},
			{
				Name:      actRepeatLastMotion,
				DocString: "Repeat last motion",
				Run:       kit.Runner(action.RepeatLastMotion),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('.')),
			},
			{
				Name:      actIndent,
				DocString: "Indent selection",
				Run:       kit.Runner(action.Indent),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('>')),
			},
			{
				Name:      actUnindent,
				DocString: "Unindent selection",
				Run:       kit.Runner(action.Unindent),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('<')),
			},
			{
				Name:      actJoinSelections,
				DocString: "Join lines inside selection",
				Run:       kit.Runner(action.JoinSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('J')),
			},
			{
				Name:      actJoinSelectionsSpace,
				DocString: "Join lines inside selection and select spaces",
				Run:       kit.Runner(action.JoinSelectionsSpace),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('J')),
			},
			{
				Name:      actAlignSelections,
				DocString: "Align selections in column",
				Run:       kit.Runner(action.AlignSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('&')),
			},
			{
				Name:      actTrimSelections,
				DocString: "Trim whitespace from selections",
				Run:       kit.Runner(action.TrimSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('_')),
			},
			{
				Name:      actRotateSelectionsBackward,
				DocString: "Rotate selections backward",
				Run:       kit.Runner(action.RotateSelectionsBackward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('(')),
			},
			{
				Name:      actRotateSelectionsForward,
				DocString: "Rotate selections forward",
				Run:       kit.Runner(action.RotateSelectionsForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char(')')),
			},
			{
				Name:      actRotateContentsBackward,
				DocString: "Rotate selections contents backward",
				Run:       kit.Runner(action.RotateContentsBackward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('(')),
			},
			{
				Name:      actRotateContentsForward,
				DocString: "Rotate selection contents forward",
				Run:       kit.Runner(action.RotateContentsForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt(')')),
			},
			{
				Name:      actEnsureForward,
				DocString: "Ensure all selections face forward",
				Run:       kit.Runner(action.EnsureForward),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt(':')),
			},
			{
				Name:      actIncrement,
				DocString: "Increment item under cursor",
				Run:       kit.Runner(action.Increment),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Ctrl('a')),
			},
			{
				Name:      actDecrement,
				DocString: "Decrement item under cursor",
				Run:       kit.Runner(action.Decrement),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Ctrl('x')),
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
			kit.EditorBoolOption("continue-comments",
				func(e *view.Editor) bool {
					return e.Options().ContinueComments
				},
				func(e *view.Editor, v bool) {
					e.Options().ContinueComments = v
				},
			),
			kit.EditorBoolOption("auto-save",
				func(e *view.Editor) bool {
					return e.Options().AutoSaveFocusLost
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSaveFocusLost = v
				},
			),
			kit.EditorBoolOption("auto-save.focus-lost",
				func(e *view.Editor) bool {
					return e.Options().AutoSaveFocusLost
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSaveFocusLost = v
				},
			),
			kit.EditorBoolOption("auto-save.after-delay.enable",
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
			kit.EditorBoolOption("atomic-save",
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
				opts.ContinueComments = kit.BoolOr(
					cfg.Editor.ContinueComments, true,
				)
				opts.AtomicSave = kit.BoolOr(cfg.Editor.AtomicSave, true)
				opts.AutoSaveFocusLost = kit.BoolOr(
					cfg.Editor.AutoSave.FocusLost, false,
				)
				opts.AutoSaveAfterDelay = kit.BoolOr(
					cfg.Editor.AutoSave.AfterDelay.Enable, false,
				)
				opts.AutoSaveDelayTimeout = kit.IntOr(
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
