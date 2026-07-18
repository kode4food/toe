package editing

import (
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
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
	actExtendLineBelow            = "extend_line_below"
	actExtendToLineBounds         = "extend_to_line_bounds"
	actShrinkToLineBounds         = "shrink_to_line_bounds"
	actExpandSelection            = "expand_selection"
	actShrinkSelection            = "shrink_selection"
	actKeepPrimarySelection       = "keep_primary_selection"
	actRemovePrimarySelection     = "remove_primary_selection"
	actMatchBrackets              = "match_brackets"
	actSurroundAdd                = "surround_add"
	actSurroundReplace            = "surround_replace"
	actSurroundDelete             = "surround_delete"
	actSelectObjectAround         = "select_textobject_around"
	actSelectObjectInner          = "select_textobject_inner"
	actAddNewlineAbove            = "add_newline_above"
	actAddNewlineBelow            = "add_newline_below"
	actSelectRegister             = "select_register"
)

type textObjectEntry struct {
	ch    rune
	label string
}

var textObjectEntries = []textObjectEntry{
	{ch: 'f', label: "function"},
	{ch: 't', label: "type definition"},
	{ch: 'a', label: "argument/parameter"},
	{ch: 'c', label: "call"},
	{ch: 'e', label: "data structure entry"},
	{ch: 'w', label: "word"},
	{ch: 'W', label: "WORD"},
	{ch: 'p', label: "paragraph"},
	{ch: 'm', label: "closest surrounding pair"},
	{ch: '(', label: "parentheses"},
	{ch: ')', label: "parentheses"},
	{ch: '{', label: "curly braces"},
	{ch: '}', label: "curly braces"},
	{ch: '[', label: "square brackets"},
	{ch: ']', label: "square brackets"},
	{ch: '<', label: "angled brackets"},
	{ch: '>', label: "angled brackets"},
	{ch: '"', label: "double quotes"},
	{ch: '\'', label: "single quotes"},
	{ch: '`', label: "backticks"},
	{ch: '|', label: "pipes"},
}

// SelectionModule returns the selection, surround, and text-object commands
func SelectionModule(model ui.Model) command.Module {
	m := kit.Prefixed(kit.Char('m'))
	prev := kit.Prefixed(kit.Char('['))
	next := kit.Prefixed(kit.Char(']'))

	mod := command.Module{
		Commands: []command.Command{
			{
				Name:      actCopyOnNextLine,
				DocString: "Copy selection on next line",
				Run:       kit.Runner(action.CopyOnNextLine),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('C')),
			},
			{
				Name:      actCopyOnPrevLine,
				DocString: "Copy selection on previous line",
				Run:       kit.Runner(action.CopyOnPrevLine),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('C')),
			},
			{
				Name:      actSelectWithinRegex,
				DocString: "Select all regex matches inside selections",
				Run: kit.Continuation(model.RegexAction(
					i18n.Text(i18n.PromptSelect),
					action.SelectWithinRegex,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(kit.Char('s')),
			},
			{
				Name:      actSplitSelectionByRegex,
				DocString: "Split selections on regex matches",
				Run: kit.Continuation(model.RegexAction(
					i18n.Text(i18n.PromptSplit),
					action.SplitSelectionByRegex,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(kit.Char('S')),
			},
			{
				Name:      actKeepSelectionsMatching,
				DocString: "Keep selections matching regex",
				Run: kit.Continuation(model.RegexAction(
					i18n.Text(i18n.PromptKeep),
					action.KeepSelectionsMatching,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(kit.Char('K')),
			},
			{
				Name:      actRemoveSelectionsMatching,
				DocString: "Remove selections matching regex",
				Run: kit.Continuation(model.RegexAction(
					i18n.Text(i18n.PromptRemove),
					action.RemoveSelectionsMatching,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(kit.Alt('K')),
			},
			{
				Name:      actSplitSelectionOnNewline,
				DocString: "Split selection on newlines",
				Run:       kit.Runner(action.SplitSelectionOnNewline),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('s')),
			},
			{
				Name:      actMergeSelections,
				DocString: "Merge selections",
				Run:       kit.Runner(action.MergeSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('-')),
			},
			{
				Name:      actMergeConsecutiveSelections,
				DocString: "Merge consecutive selections",
				Run:       kit.Runner(action.MergeConsecutive),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('_')),
			},
			{
				Name:      actCollapseSelection,
				DocString: "Collapse selection into single cursor",
				Run:       kit.Runner(action.CollapseSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char(';')),
			},
			{
				Name:      actFlipSelections,
				DocString: "Flip selection cursor and anchor",
				Run:       kit.Runner(action.FlipSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt(';')),
			},
			{
				Name:      actSelectAll,
				DocString: "Select whole document",
				Run:       kit.Runner(action.SelectAll),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('%')),
			},
			{
				Name:      actSelectLineAbove,
				DocString: "Select line above",
				Run:       kit.Runner(action.SelectLineAbove),
				Modes:     []string{"NOR", "SEL"},
				Signature: kit.Sig(),
			},
			{
				Name:      actSelectLineBelow,
				DocString: "Select line below",
				Run:       kit.Runner(action.SelectLineBelow),
				Modes:     []string{"NOR", "SEL"},
				Signature: kit.Sig(),
			},
			{
				Name: actExtendLineBelow,
				DocString: "Select current line, if already " +
					"selected, extend" +
					" to next line",
				Run:   kit.Runner(action.ExtendLineBelow),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(kit.Char('x')),
			},
			{
				Name:      actExtendToLineBounds,
				DocString: "Extend selection to line bounds",
				Run:       kit.Runner(action.ExtendToLineBounds),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('X')),
			},
			{
				Name:      actShrinkToLineBounds,
				DocString: "Shrink selection to line bounds",
				Run:       kit.Runner(action.ShrinkToLineBounds),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('x')),
			},
			{
				Name:      actExpandSelection,
				DocString: "Expand selection to syntax node",
				Run:       kit.Runner(syntaxExpandSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('o')),
			},
			{
				Name:      actShrinkSelection,
				DocString: "Shrink selection to syntax node",
				Run:       kit.Runner(syntaxShrinkSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt('i')),
			},
			{
				Name:      actKeepPrimarySelection,
				DocString: "Keep primary selection",
				Run:       kit.Runner(action.KeepPrimarySelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char(',')),
			},
			{
				Name:      actRemovePrimarySelection,
				DocString: "Remove primary selection",
				Run:       kit.Runner(action.RemovePrimarySelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Alt(',')),
			},
			{
				Name:      actMatchBrackets,
				DocString: "Goto matching bracket",
				Run:       kit.Runner(syntaxMatchBrackets),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(m(kit.Char('m'))),
			},
			{
				Name:      actSurroundAdd,
				DocString: "Surround add",
				Run:       kit.Continuation(surroundAddAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(m(kit.Char('s'))),
			},
			{
				Name:      actSurroundReplace,
				DocString: "Surround replace",
				Run:       kit.Continuation(surroundReplaceAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(m(kit.Char('r'))),
			},
			{
				Name:      actSurroundDelete,
				DocString: "Surround delete",
				Run:       kit.Continuation(surroundDeleteAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(m(kit.Char('d'))),
			},
			{
				Name:      actSelectObjectAround,
				DocString: "Select around object",
				Run:       kit.Continuation(textObjectAction(true)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(m(kit.Char('a'))),
			},
			{
				Name:      actSelectObjectInner,
				DocString: "Select inside object",
				Run:       kit.Continuation(textObjectAction(false)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(m(kit.Char('i'))),
			},
			{
				Name:      actAddNewlineAbove,
				DocString: "Add newline above",
				Run:       kit.Runner(action.AddNewlineAbove),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(prev(kit.Char(' '))),
			},
			{
				Name:      actAddNewlineBelow,
				DocString: "Add newline below",
				Run:       kit.Runner(action.AddNewlineBelow),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(next(kit.Char(' '))),
			},
			{
				Name:      actSelectRegister,
				DocString: "Select register",
				Run:       kit.Continuation(selectRegisterAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('"')),
			},
		},
	}
	mod.Labels = []command.PrefixLabel{
		kit.Label("Match", kit.Char('m'), "NOR", "SEL"),
		kit.Label("Match around", m(kit.Char('a')), "NOR", "SEL"),
		kit.Label("Match inside", m(kit.Char('i')), "NOR", "SEL"),
	}
	for _, e := range textObjectEntries {
		mod.Labels = append(mod.Labels,
			kit.Label(e.label,
				m(append(kit.Char('a'), kit.Char(e.ch)...)), "NOR", "SEL"),
			kit.Label(e.label,
				m(append(kit.Char('i'), kit.Char(e.ch)...)), "NOR", "SEL"),
		)
	}
	return mod
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
