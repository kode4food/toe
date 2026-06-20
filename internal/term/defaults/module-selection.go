package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actCopyOnNextLine             = "copy_on_next_line"
	actCopyOnPrevLine             = "copy_on_prev_line"
	actSelectWithinRegex          = "select_within_regex"
	actSplitSelectionByRegex      = "split_selection_by_regex"
	actKeepSelectionsMatching     = "keep_selections_matching"
	actRemoveSelectionsMatching   = "remove_selections_matching"
	actSplitSelectionOnNewline    = "split_selection_on_newline"
	actMergeSelections            = "merge_selections"
	actMergeConsecutiveSelections = "merge_consecutive_selections"
	actCollapseSelection          = "collapse_selection"
	actFlipSelections             = "flip_selections"
	actSelectAll                  = "select_all"
	actSelectLineAbove            = "select_line_above"
	actSelectLineBelow            = "select_line_below"
	actExtendLineBellow           = "extend_line_below"
	actExtendToLineBounds         = "extend_to_line_bounds"
	actShrinkToLineBounds         = "shrink_to_line_bounds"
	actKeepPrimarySelection       = "keep_primary_selection"
	actRemovePrimarySelection     = "remove_primary_selection"
	actMatchBrackets              = "match_brackets"
	actSurroundAdd                = "surround_add"
	actSurroundReplace            = "surround_replace"
	actSurroundDelete             = "surround_delete"
	actSelectTextObjectAround     = "select_textobject_around"
	actSelectTextObjectInside     = "select_textobject_inside"
	actAddNewlineAbove            = "add_newline_above"
	actAddNewlineBelow            = "add_newline_below"
	actSelectRegister             = "select_register"
)

func selectionModule(model ui.Model) command.Module {
	m := prefixed(char('m'))
	prev := prefixed(char('['))
	next := prefixed(char(']'))

	return command.Module{
		Commands: map[string]command.Command{
			actCopyOnNextLine: {
				DocString: "Copy selection on next line",
				Run:       Runner(action.CopyOnNextLine),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('C')),
			},
			actCopyOnPrevLine: {
				DocString: "Copy selection on previous line",
				Run:       Runner(action.CopyOnPrevLine),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('C')),
			},
			actSelectWithinRegex: {
				DocString: "Select all regex matches inside selections",
				Run: Continuation(model.RegexAction(
					"select:", action.SelectWithinRegex,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('s')),
			},
			actSplitSelectionByRegex: {
				DocString: "Split selections on regex matches",
				Run: Continuation(model.RegexAction(
					"split:", action.SplitSelectionByRegex,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('S')),
			},
			actKeepSelectionsMatching: {
				DocString: "Keep selections matching regex",
				Run: Continuation(model.RegexAction(
					"keep:", action.KeepSelectionsMatching,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('K')),
			},
			actRemoveSelectionsMatching: {
				DocString: "Remove selections matching regex",
				Run: Continuation(model.RegexAction(
					"remove:", action.RemoveSelectionsMatching,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(alt('K')),
			},
			actSplitSelectionOnNewline: {
				DocString: "Split selection on newlines",
				Run:       Runner(action.SplitSelectionOnNewline),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('s')),
			},
			actMergeSelections: {
				DocString: "Merge selections",
				Run:       Runner(action.MergeSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('-')),
			},
			actMergeConsecutiveSelections: {
				DocString: "Merge consecutive selections",
				Run:       Runner(action.MergeConsecutive),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('_')),
			},
			actCollapseSelection: {
				DocString: "Collapse selection into single cursor",
				Run:       Runner(action.CollapseSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(';')),
			},
			actFlipSelections: {
				DocString: "Flip selection cursor and anchor",
				Run:       Runner(action.FlipSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt(';')),
			},
			actSelectAll: {
				DocString: "Select whole document",
				Run:       Runner(action.SelectAll),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('%')),
			},
			actSelectLineAbove: {
				DocString: "Select line above",
				Run:       Runner(action.SelectLineAbove),
				Signature: sig(),
			},
			actSelectLineBelow: {
				DocString: "Select line below",
				Run:       Runner(action.SelectLineBelow),
				Signature: sig(),
			},
			actExtendLineBellow: {
				DocString: "Select current line, if already selected, extend" +
					" to next line",
				Run:   Runner(action.ExtendLineBellow),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('x')),
			},
			actExtendToLineBounds: {
				DocString: "Extend selection to line bounds",
				Run:       Runner(action.ExtendToLineBounds),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('X')),
			},
			actShrinkToLineBounds: {
				DocString: "Shrink selection to line bounds",
				Run:       Runner(action.ShrinkToLineBounds),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('x')),
			},
			actKeepPrimarySelection: {
				DocString: "Keep primary selection",
				Run:       Runner(action.KeepPrimarySelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(',')),
			},
			actRemovePrimarySelection: {
				DocString: "Remove primary selection",
				Run:       Runner(action.RemovePrimarySelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt(',')),
			},
			actMatchBrackets: {
				DocString: "Goto matching bracket",
				Run:       Runner(action.MatchBrackets),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('m'))),
			},
			actSurroundAdd: {
				DocString: "Surround add",
				Run:       Continuation(surroundAddAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('s'))),
			},
			actSurroundReplace: {
				DocString: "Surround replace",
				Run:       Continuation(surroundReplaceAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('r'))),
			},
			actSurroundDelete: {
				DocString: "Surround delete",
				Run:       Continuation(surroundDeleteAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('d'))),
			},
			actSelectTextObjectAround: {
				DocString: "Select around object",
				Run:       Continuation(textObjectAction(true)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('a'))),
			},
			actSelectTextObjectInside: {
				DocString: "Select inside object",
				Run:       Continuation(textObjectAction(false)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('i'))),
			},
			actAddNewlineAbove: {
				DocString: "Add newline above",
				Run:       Runner(action.AddNewlineAbove),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(prev(char(' '))),
			},
			actAddNewlineBelow: {
				DocString: "Add newline below",
				Run:       Runner(action.AddNewlineBelow),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(next(char(' '))),
			},
			actSelectRegister: {
				DocString: "Select register",
				Run:       Continuation(selectRegisterAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('"')),
			},
		},
	}
}

func selectRegisterAction(e *view.Editor) command.Continuation {
	e.SetHint(`" ...`)
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char != 0 && k.Mods == command.ModNone {
			e.SetRegister(k.Code.Char)
		}
		e.SetHint("")
		return nil
	}
}

func surroundAddAction(e *view.Editor) command.Continuation {
	e.SetHint("ms ...")
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char != 0 && k.Mods == command.ModNone {
			action.SurroundAdd(e, k.Code.Char)
		}
		e.SetHint("")
		return nil
	}
}

func surroundReplaceAction(e *view.Editor) command.Continuation {
	e.SetHint("mr ...")
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			e.SetHint("")
			return nil
		}
		from := k.Code.Char
		e.SetHint("mr " + string(from) + " ...")
		return func(e *view.Editor, k command.KeyEvent) command.Continuation {
			if k.Code.Char != 0 && k.Mods == command.ModNone {
				action.SurroundReplace(e, from, k.Code.Char)
			}
			e.SetHint("")
			return nil
		}
	}
}

func surroundDeleteAction(e *view.Editor) command.Continuation {
	e.SetHint("md ...")
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char != 0 && k.Mods == command.ModNone {
			action.SurroundDelete(e, k.Code.Char)
		}
		e.SetHint("")
		return nil
	}
}

func textObjectAction(around bool) command.KeyAction {
	h, fn := "mi", action.SelectTextObjectInside
	if around {
		h, fn = "ma", action.SelectTextObjectAround
	}
	return func(e *view.Editor) command.Continuation {
		e.SetHint(h + " ...")
		return func(e *view.Editor, k command.KeyEvent) command.Continuation {
			if k.Code.Char != 0 && k.Mods == command.ModNone {
				fn(e, k.Code.Char)
			}
			e.SetHint("")
			return nil
		}
	}
}
