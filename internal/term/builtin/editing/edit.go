package editing

import (
	"embed"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
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
	actNormalMode               = "normal_mode"
)

const errorHistoryStepsKey i18n.Key = "error.historySteps"

//go:embed i18n/edit.*.json
var editFS embed.FS

var (
	errHistorySteps = i18n.NewError(errorHistoryStepsKey)
)

// EditModule returns the text-editing commands
func EditModule() command.Module {
	cfg := new(editSection)
	return command.Module{
		Translations: i18n.LoadTranslations(editFS),
		Commands: []command.Command{
			{
				Name:      actSelectMode,
				DocString: "Enter selection extend mode",
				Run:       kit.Runner(action.SelectMode),
				Modes:     view.ModeNormal,
				Keys:      kit.Keys(kit.Char('v')),
			},
			{
				Name:      actNormalMode,
				DocString: "Enter normal mode",
				Run:       kit.Runner(action.NormalMode),
				Modes:     command.DocModes,
				Keys:      kit.Keys(kit.Esc),
			},
			{
				Name:      actInsertMode,
				DocString: "Insert before selection",
				Run:       kit.Runner(action.InsertMode),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('i')),
			},
			{
				Name:      actInsertAtLineStart,
				DocString: "Insert at start of line",
				Run:       kit.Runner(action.InsertAtLineStart),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('I')),
			},
			{
				Name:      actAppendMode,
				DocString: "Append after selection",
				Run:       kit.Runner(action.AppendMode),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('a')),
			},
			{
				Name:      actAppendToLine,
				DocString: "Insert at end of line",
				Run:       kit.Runner(action.AppendToLine),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('A')),
			},
			{
				Name:      actOpenBelow,
				DocString: "Open new line below selection",
				Run:       kit.Runner(action.OpenBelow),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('o')),
			},
			{
				Name:      actOpenAbove,
				DocString: "Open new line above selection",
				Run:       kit.Runner(action.OpenAbove),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('O')),
			},
			{
				Name:      actReplace,
				DocString: "Replace with new char",
				Run:       kit.Continuation(replaceCharAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('r')),
			},
			{
				Name:      actDeleteSelection,
				DocString: "Delete selection",
				Run:       kit.Runner(action.DeleteSelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('d')),
			},
			{
				Name:      actDeleteSelectionNoYank,
				DocString: "Delete selection without yanking",
				Run:       kit.Runner(action.DeleteSelectionNoYank),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('d')),
			},
			{
				Name:      actChangeSelection,
				DocString: "Change selection",
				Run:       kit.Runner(action.ChangeSelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('c')),
			},
			{
				Name:      actChangeSelectionNoYank,
				DocString: "Change selection without yanking",
				Run:       kit.Runner(action.ChangeSelectionNoYank),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('c')),
			},
			{
				Name:      actUndo,
				DocString: "Undo change",
				Run:       kit.Runner(undoAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('u')),
			},
			{
				Name:      actRedo,
				DocString: "Redo change",
				Run:       kit.Runner(redoAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('U')),
			},
			{
				Name: actEarlier,
				DocString: "Move backward in history. Accepts a number of " +
					"steps",
				Run:       earlierAction,
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('u')),
				Signature: kit.OptionalArg(),
			},
			{
				Name:      actLater,
				DocString: "Move forward in history. Accepts a number of steps",
				Run:       laterAction,
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('U')),
				Signature: kit.OptionalArg(),
			},
			{
				Name:      actSwitchCase,
				DocString: "Switch (toggle) case",
				Run:       kit.Runner(action.SwitchCase),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('~')),
			},
			{
				Name:      actSwitchToLowercase,
				DocString: "Switch to lowercase",
				Run:       kit.Runner(action.SwitchToLowercase),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('`')),
			},
			{
				Name:      actSwitchToUppercase,
				DocString: "Switch to uppercase",
				Run:       kit.Runner(action.SwitchToUppercase),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('`')),
			},
			{
				Name:      actRepeatLastMotion,
				DocString: "Repeat last motion",
				Run:       kit.Runner(action.RepeatLastMotion),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('.')),
			},
			{
				Name:      actIndent,
				DocString: "Indent selection",
				Run:       kit.Runner(action.Indent),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('>')),
			},
			{
				Name:      actUnindent,
				DocString: "Unindent selection",
				Run:       kit.Runner(action.Unindent),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('<')),
			},
			{
				Name:      actJoinSelections,
				DocString: "Join lines inside selection",
				Run:       kit.Runner(action.JoinSelections),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('J')),
			},
			{
				Name:      actJoinSelectionsSpace,
				DocString: "Join lines inside selection and select spaces",
				Run:       kit.Runner(action.JoinSelectionsSpace),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('J')),
			},
			{
				Name:      actAlignSelections,
				DocString: "Align selections in column",
				Run:       kit.Runner(action.AlignSelections),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('&')),
			},
			{
				Name:      actTrimSelections,
				DocString: "Trim whitespace from selections",
				Run:       kit.Runner(action.TrimSelections),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('_')),
			},
			{
				Name:      actRotateSelectionsBackward,
				DocString: "Rotate selections backward",
				Run:       kit.Runner(action.RotateSelectionsBackward),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('(')),
			},
			{
				Name:      actRotateSelectionsForward,
				DocString: "Rotate selections forward",
				Run:       kit.Runner(action.RotateSelectionsForward),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char(')')),
			},
			{
				Name:      actRotateContentsBackward,
				DocString: "Rotate selections contents backward",
				Run:       kit.Runner(action.RotateContentsBackward),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('(')),
			},
			{
				Name:      actRotateContentsForward,
				DocString: "Rotate selection contents forward",
				Run:       kit.Runner(action.RotateContentsForward),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt(')')),
			},
			{
				Name:      actEnsureForward,
				DocString: "Ensure all selections face forward",
				Run:       kit.Runner(action.EnsureForward),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt(':')),
			},
			{
				Name:      actIncrement,
				DocString: "Increment item under cursor",
				Run:       kit.Runner(action.Increment),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Ctrl('a')),
			},
			{
				Name:      actDecrement,
				DocString: "Decrement item under cursor",
				Run:       kit.Runner(action.Decrement),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Ctrl('x')),
			},
		},
		Options: []command.Option{
			{
				Key: "auto-pairs",
				Get: func(e *view.Editor) (string, error) {
					return formatAutoPairs(e.Options()), nil
				},
				Set: func(e *view.Editor, s string) error {
					return setAutoPairs(e.Options(), s)
				},
				Toggle: func(e *view.Editor) (string, error) {
					v := !e.Options().HasAutoPairs
					if v {
						e.Options().AutoPairMap = core.DefaultAutoPairs()
					}
					e.Options().HasAutoPairs = v
					return strconv.FormatBool(v), nil
				},
				Complete: command.StaticCompleter("true", "false"),
			},
			kit.EditorBoolOption("continue-comments",
				func(e *view.Editor) bool {
					return e.Options().ContinueComments
				},
				func(e *view.Editor, v bool) {
					e.Options().ContinueComments = v
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

func formatAutoPairs(opts *view.Options) string {
	if !opts.HasAutoPairs {
		return "false"
	}
	if maps.Equal(opts.AutoPairMap, core.DefaultAutoPairs()) {
		return "true"
	}
	opens := make([]rune, 0, len(opts.AutoPairMap))
	for ch, pair := range opts.AutoPairMap {
		if ch == pair.Open {
			opens = append(opens, ch)
		}
	}
	slices.Sort(opens)
	values := make([]string, len(opens))
	for i, open := range opens {
		pair := opts.AutoPairMap[open]
		values[i] = fmt.Sprintf("%s = %s",
			strconv.Quote(string(pair.Open)),
			strconv.Quote(string(pair.Close)),
		)
	}
	return "{ " + strings.Join(values, ", ") + " }"
}

func setAutoPairs(opts *view.Options, value string) error {
	if enabled, err := config.ParseBool(value); err == nil {
		opts.HasAutoPairs = enabled
		if enabled {
			opts.AutoPairMap = core.DefaultAutoPairs()
		}
		return nil
	}
	var raw struct {
		Value language.AutoPairConfig `toml:"value"`
	}
	if _, err := toml.Decode("value = "+value, &raw); err != nil {
		return fmt.Errorf("%w: %s", config.ErrInvalidOption, value)
	}
	pairs, ok := raw.Value.AutoPairs()
	opts.AutoPairMap = pairs
	opts.HasAutoPairs = ok
	return nil
}

func replaceCharAction(_ *view.Editor) command.Continuation {
	return command.ReadChar(func(e *view.Editor, ch rune) command.Continuation {
		action.ReplaceChar(e, ch)
		return nil
	})
}

func undoAction(e *view.Editor) {
	e.Undo()
}

func redoAction(e *view.Editor) {
	e.Redo()
}

func earlierAction(e *view.Editor, args *command.Args) command.Result {
	n, err := historySteps(e, args)
	if err != nil {
		return command.Result{Error: err}
	}
	e.Earlier(core.UndoSteps(n))
	return command.Result{}
}

func laterAction(e *view.Editor, args *command.Args) command.Result {
	n, err := historySteps(e, args)
	if err != nil {
		return command.Result{Error: err}
	}
	e.Later(core.UndoSteps(n))
	return command.Result{}
}

func historySteps(e *view.Editor, args *command.Args) (int, error) {
	if args == nil || args.Empty() {
		return e.CountOr(1), nil
	}
	arg, _ := args.First()
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 {
		return 0, errHistorySteps
	}
	return n, nil
}
