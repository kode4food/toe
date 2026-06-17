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
	actAddNewlineAbove            = "add_newline_above"
	actAddNewlineBelow            = "add_newline_below"
)

func registerSelectionCommands(r *registry, model ui.Model) {
	m := prefixed(char('m'))
	prev := prefixed(char('['))
	next := prefixed(char(']'))

	r.RegisterCommand(actCopyOnNextLine, command.Command{
		DocString: "Copy selection on next line",
		Run:       Runner(action.CopyOnNextLine),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('C')),
	})
	r.RegisterCommand(actCopyOnPrevLine, command.Command{
		DocString: "Copy selection on previous line",
		Run:       Runner(action.CopyOnPrevLine),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt('C')),
	})

	r.RegisterCommand(actSelectWithinRegex, command.Command{
		DocString: "Select all regex matches inside selections",
		Run: Continuation(model.RegexAction(
			"select:", action.SelectWithinRegex,
		)),
		Modes: []string{"NOR"},
		Keys:  keyBinding(char('s')),
	})
	r.RegisterCommand(actSplitSelectionByRegex, command.Command{
		DocString: "Split selections on regex matches",
		Run: Continuation(model.RegexAction(
			"split:", action.SplitSelectionByRegex,
		)),
		Modes: []string{"NOR"},
		Keys:  keyBinding(char('S')),
	})
	r.RegisterCommand(actKeepSelectionsMatching, command.Command{
		DocString: "Keep selections matching regex",
		Run: Continuation(model.RegexAction(
			"keep:", action.KeepSelectionsMatching,
		)),
		Modes: []string{"NOR"},
		Keys:  keyBinding(char('K')),
	})
	r.RegisterCommand(actRemoveSelectionsMatching, command.Command{
		DocString: "Remove selections matching regex",
		Run: Continuation(model.RegexAction(
			"remove:", action.RemoveSelectionsMatching,
		)),
		Modes: []string{"NOR"},
		Keys:  keyBinding(alt('K')),
	})
	r.RegisterCommand(actSplitSelectionOnNewline, command.Command{
		DocString: "Split selection on newlines",
		Run:       Runner(action.SplitSelectionOnNewline),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt('s')),
	})
	r.RegisterCommand(actMergeSelections, command.Command{
		DocString: "Merge selections",
		Run:       Runner(action.MergeSelections),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt('-')),
	})
	r.RegisterCommand(actMergeConsecutiveSelections, command.Command{
		DocString: "Merge consecutive selections",
		Run:       Runner(action.MergeConsecutive),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt('_')),
	})
	r.RegisterCommand(actCollapseSelection, command.Command{
		DocString: "Collapse selection into single cursor",
		Run:       Runner(action.CollapseSelection),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char(';')),
	})
	r.RegisterCommand(actFlipSelections, command.Command{
		DocString: "Flip selection cursor and anchor",
		Run:       Runner(action.FlipSelections),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt(';')),
	})
	r.RegisterCommand(actSelectAll, command.Command{
		DocString: "Select whole document",
		Run:       Runner(action.SelectAll),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('%')),
	})
	r.RegisterCommand(actSelectLineAbove, command.Command{
		DocString: "Select line above",
		Run:       Runner(action.SelectLineAbove),
		Signature: sig(),
	})
	r.RegisterCommand(actSelectLineBelow, command.Command{
		DocString: "Select line below",
		Run:       Runner(action.SelectLineBelow),
		Signature: sig(),
	})
	r.RegisterCommand(actExtendLineBellow, command.Command{
		DocString: "Select current line, if already selected, extend" +
			" to next line",
		Run:   Runner(action.ExtendLineBellow),
		Modes: []string{"NOR"},
		Keys:  keyBinding(char('x')),
	})
	r.RegisterCommand(actExtendToLineBounds, command.Command{
		DocString: "Extend selection to line bounds",
		Run:       Runner(action.ExtendToLineBounds),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('X')),
	})
	r.RegisterCommand(actShrinkToLineBounds, command.Command{
		DocString: "Shrink selection to line bounds",
		Run:       Runner(action.ShrinkToLineBounds),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt('x')),
	})
	r.RegisterCommand(actKeepPrimarySelection, command.Command{
		DocString: "Keep primary selection",
		Run:       Runner(action.KeepPrimarySelection),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char(',')),
	})
	r.RegisterCommand(actRemovePrimarySelection, command.Command{
		DocString: "Remove primary selection",
		Run:       Runner(action.RemovePrimarySelection),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(alt(',')),
	})

	r.RegisterCommand(actMatchBrackets, command.Command{
		DocString: "Goto matching bracket",
		Run:       Runner(action.MatchBrackets),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(m(char('m'))),
	})
	r.RegisterCommand(actSurroundAdd, command.Command{
		DocString: "Surround add",
		Run:       Continuation(surroundAddAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(m(char('s'))),
	})
	r.RegisterCommand(actSurroundReplace, command.Command{
		DocString: "Surround replace",
		Run:       Continuation(surroundReplaceAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(m(char('r'))),
	})
	r.RegisterCommand(actSurroundDelete, command.Command{
		DocString: "Surround delete",
		Run:       Continuation(surroundDeleteAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(m(char('d'))),
	})
	r.RegisterCommand(actSelectTextObjectAround, command.Command{
		DocString: "Select around object",
		Run:       Continuation(textObjectAction(true)),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(m(char('a'))),
	})
	r.RegisterCommand(actSelectTextObjectInside, command.Command{
		DocString: "Select inside object",
		Run:       Continuation(textObjectAction(false)),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(m(char('i'))),
	})

	r.RegisterCommand(actAddNewlineAbove, command.Command{
		DocString: "Add newline above",
		Run:       Runner(action.ExtendLineAbove),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(prev(char(' '))),
	})
	r.RegisterCommand(actAddNewlineBelow, command.Command{
		DocString: "Add newline below",
		Run:       Runner(action.ExtendLine),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(next(char(' '))),
	})

	r.RegisterCommand(actSelectRegister, command.Command{
		DocString: "Select register",
		Run:       Continuation(selectRegisterAction),
		Modes:     []string{"NOR"},
		Keys:      keyBinding(char('"')),
	})

	r.RegisterCommand(actCollapseSelection, command.Command{
		DocString: "Collapse selection into single cursor",
		Run:       Runner(action.CollapseSelection),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char(';')),
	})
	r.RegisterCommand(actFlipSelections, command.Command{
		DocString: "Flip selection cursor and anchor",
		Run:       Runner(action.FlipSelections),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt(';')),
	})
	r.RegisterCommand(actKeepPrimarySelection, command.Command{
		DocString: "Keep primary selection",
		Run:       Runner(action.KeepPrimarySelection),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char(',')),
	})
	r.RegisterCommand(actRemovePrimarySelection, command.Command{
		DocString: "Remove primary selection",
		Run:       Runner(action.RemovePrimarySelection),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt(',')),
	})
	r.RegisterCommand(actSelectAll, command.Command{
		DocString: "Select whole document",
		Run:       Runner(action.SelectAll),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('%')),
	})
	r.RegisterCommand(actExtendLineBellow, command.Command{
		DocString: "Select current line, if already selected, extend" +
			" to next line",
		Run:   Runner(action.ExtendLineBellow),
		Modes: []string{"SEL"},
		Keys:  keyBinding(char('x')),
	})
	r.RegisterCommand(actExtendToLineBounds, command.Command{
		DocString: "Extend selection to line bounds",
		Run:       Runner(action.ExtendToLineBounds),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('X')),
	})
	r.RegisterCommand(actShrinkToLineBounds, command.Command{
		DocString: "Shrink selection to line bounds",
		Run:       Runner(action.ShrinkToLineBounds),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt('x')),
	})
	r.RegisterCommand(actCopyOnNextLine, command.Command{
		DocString: "Copy selection on next line",
		Run:       Runner(action.CopyOnNextLine),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('C')),
	})
	r.RegisterCommand(actCopyOnPrevLine, command.Command{
		DocString: "Copy selection on previous line",
		Run:       Runner(action.CopyOnPrevLine),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt('C')),
	})

	r.RegisterCommand(actSelectWithinRegex, command.Command{
		DocString: "Select all regex matches inside selections",
		Run: Continuation(model.RegexAction(
			"select:", action.SelectWithinRegex,
		)),
		Modes: []string{"SEL"},
		Keys:  keyBinding(char('s')),
	})
	r.RegisterCommand(actSplitSelectionByRegex, command.Command{
		DocString: "Split selections on regex matches",
		Run: Continuation(model.RegexAction(
			"split:", action.SplitSelectionByRegex,
		)),
		Modes: []string{"SEL"},
		Keys:  keyBinding(char('S')),
	})
	r.RegisterCommand(actKeepSelectionsMatching, command.Command{
		DocString: "Keep selections matching regex",
		Run: Continuation(model.RegexAction(
			"keep:", action.KeepSelectionsMatching,
		)),
		Modes: []string{"SEL"},
		Keys:  keyBinding(char('K')),
	})
	r.RegisterCommand(actRemoveSelectionsMatching, command.Command{
		DocString: "Remove selections matching regex",
		Run: Continuation(model.RegexAction(
			"remove:", action.RemoveSelectionsMatching,
		)),
		Modes: []string{"SEL"},
		Keys:  keyBinding(alt('K')),
	})
	r.RegisterCommand(actSplitSelectionOnNewline, command.Command{
		DocString: "Split selection on newlines",
		Run:       Runner(action.SplitSelectionOnNewline),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt('s')),
	})
	r.RegisterCommand(actMergeSelections, command.Command{
		DocString: "Merge selections",
		Run:       Runner(action.MergeSelections),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt('-')),
	})
	r.RegisterCommand(actMergeConsecutiveSelections, command.Command{
		DocString: "Merge consecutive selections",
		Run:       Runner(action.MergeConsecutive),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(alt('_')),
	})

	r.RegisterCommand(actMatchBrackets, command.Command{
		DocString: "Goto matching bracket",
		Run:       Runner(action.MatchBrackets),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(m(char('m'))),
	})
	r.RegisterCommand(actSurroundAdd, command.Command{
		DocString: "Surround add",
		Run:       Continuation(surroundAddAction),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(m(char('s'))),
	})
	r.RegisterCommand(actSurroundReplace, command.Command{
		DocString: "Surround replace",
		Run:       Continuation(surroundReplaceAction),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(m(char('r'))),
	})
	r.RegisterCommand(actSurroundDelete, command.Command{
		DocString: "Surround delete",
		Run:       Continuation(surroundDeleteAction),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(m(char('d'))),
	})
	r.RegisterCommand(actSelectTextObjectAround, command.Command{
		DocString: "Select around object",
		Run:       Continuation(textObjectAction(true)),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(m(char('a'))),
	})
	r.RegisterCommand(actSelectTextObjectInside, command.Command{
		DocString: "Select inside object",
		Run:       Continuation(textObjectAction(false)),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(m(char('i'))),
	})

	r.RegisterCommand(actAddNewlineAbove, command.Command{
		DocString: "Add newline above",
		Run:       Runner(action.AddNewlineAbove),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(prev(char(' '))),
	})
	r.RegisterCommand(actAddNewlineBelow, command.Command{
		DocString: "Add newline below",
		Run:       Runner(action.AddNewlineBelow),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(next(char(' '))),
	})

	r.RegisterCommand(actSelectRegister, command.Command{
		DocString: "Select register",
		Run:       Continuation(selectRegisterAction),
		Modes:     []string{"SEL"},
		Keys:      keyBinding(char('"')),
	})
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
