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
	char  rune
	label string
}

var textObjectEntries = []textObjectEntry{
	{char: 'f', label: "function"},
	{char: 't', label: "type definition"},
	{char: 'a', label: "argument/parameter"},
	{char: 'c', label: "call"},
	{char: 'e', label: "data structure entry"},
	{char: 'w', label: "word"},
	{char: 'W', label: "WORD"},
	{char: 'p', label: "paragraph"},
	{char: 'm', label: "closest surrounding pair"},
	{char: '(', label: "parentheses"},
	{char: ')', label: "parentheses"},
	{char: '{', label: "curly braces"},
	{char: '}', label: "curly braces"},
	{char: '[', label: "square brackets"},
	{char: ']', label: "square brackets"},
	{char: '<', label: "angled brackets"},
	{char: '>', label: "angled brackets"},
	{char: '"', label: "double quotes"},
	{char: '\'', label: "single quotes"},
	{char: '`', label: "backticks"},
	{char: '|', label: "pipes"},
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
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('C')),
			},
			{
				Name:      actCopyOnPrevLine,
				DocString: "Copy selection on previous line",
				Run:       kit.Runner(action.CopyOnPrevLine),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('C')),
			},
			{
				Name:      actSelectWithinRegex,
				DocString: "Select all regex matches inside selections",
				Run: kit.Runner(model.RegexAction(
					i18n.Text(i18n.PromptSelect),
					action.SelectWithinRegex,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('s')),
			},
			{
				Name:      actSplitSelectionByRegex,
				DocString: "Split selections on regex matches",
				Run: kit.Runner(model.RegexAction(
					i18n.Text(i18n.PromptSplit),
					action.SplitSelectionByRegex,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('S')),
			},
			{
				Name:      actKeepSelectionsMatching,
				DocString: "Keep selections matching regex",
				Run: kit.Runner(model.RegexAction(
					i18n.Text(i18n.PromptKeep),
					action.KeepSelectionsMatching,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('K')),
			},
			{
				Name:      actRemoveSelectionsMatching,
				DocString: "Remove selections matching regex",
				Run: kit.Runner(model.RegexAction(
					i18n.Text(i18n.PromptRemove),
					action.RemoveSelectionsMatching,
				)),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Alt('K')),
			},
			{
				Name:      actSplitSelectionOnNewline,
				DocString: "Split selection on newlines",
				Run:       kit.Runner(action.SplitSelectionOnNewline),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('s')),
			},
			{
				Name:      actMergeSelections,
				DocString: "Merge selections",
				Run:       kit.Runner(action.MergeSelections),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('-')),
			},
			{
				Name:      actMergeConsecutiveSelections,
				DocString: "Merge consecutive selections",
				Run:       kit.Runner(action.MergeConsecutive),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('_')),
			},
			{
				Name:      actCollapseSelection,
				DocString: "Collapse selection into single cursor",
				Run:       kit.Runner(action.CollapseSelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char(';')),
			},
			{
				Name:      actFlipSelections,
				DocString: "Flip selection cursor and anchor",
				Run:       kit.Runner(action.FlipSelections),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt(';')),
			},
			{
				Name:      actSelectAll,
				DocString: "Select whole document",
				Run:       kit.Runner(action.SelectAll),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('%')),
			},
			{
				Name:      actSelectLineAbove,
				DocString: "Select line above",
				Run:       kit.Runner(action.SelectLineAbove),
				Modes:     command.DocNormalModes,
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actSelectLineBelow,
				DocString: "Select line below",
				Run:       kit.Runner(action.SelectLineBelow),
				Modes:     command.DocNormalModes,
				Signature: command.DefaultSignature(),
			},
			{
				Name: actExtendLineBelow,
				DocString: "Select current line, if already " +
					"selected, extend" +
					" to next line",
				Run:   kit.Runner(action.ExtendLineBelow),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('x')),
			},
			{
				Name:      actExtendToLineBounds,
				DocString: "Extend selection to line bounds",
				Run:       kit.Runner(action.ExtendToLineBounds),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('X')),
			},
			{
				Name:      actShrinkToLineBounds,
				DocString: "Shrink selection to line bounds",
				Run:       kit.Runner(action.ShrinkToLineBounds),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('x')),
			},
			{
				Name:      actExpandSelection,
				DocString: "Expand selection to syntax node",
				Run:       kit.Runner(syntaxExpandSelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('o')),
			},
			{
				Name:      actShrinkSelection,
				DocString: "Shrink selection to syntax node",
				Run:       kit.Runner(syntaxShrinkSelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('i')),
			},
			{
				Name:      actKeepPrimarySelection,
				DocString: "Keep primary selection",
				Run:       kit.Runner(action.KeepPrimarySelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char(',')),
			},
			{
				Name:      actRemovePrimarySelection,
				DocString: "Remove primary selection",
				Run:       kit.Runner(action.RemovePrimarySelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt(',')),
			},
			{
				Name:      actMatchBrackets,
				DocString: "Goto matching bracket",
				Run:       kit.Runner(syntaxMatchBrackets),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(m(kit.Char('m'))),
			},
			{
				Name:      actSurroundAdd,
				DocString: "Surround add",
				Run:       kit.Continuation(surroundAddAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(m(kit.Char('s'))),
			},
			{
				Name:      actSurroundReplace,
				DocString: "Surround replace",
				Run:       kit.Continuation(surroundReplaceAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(m(kit.Char('r'))),
			},
			{
				Name:      actSurroundDelete,
				DocString: "Surround delete",
				Run:       kit.Continuation(surroundDeleteAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(m(kit.Char('d'))),
			},
			{
				Name:      actSelectObjectAround,
				DocString: "Select around object",
				Run:       kit.Continuation(textObjectAction(true)),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(m(kit.Char('a'))),
			},
			{
				Name:      actSelectObjectInner,
				DocString: "Select inside object",
				Run:       kit.Continuation(textObjectAction(false)),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(m(kit.Char('i'))),
			},
			{
				Name:      actAddNewlineAbove,
				DocString: "Add newline above",
				Run:       kit.Runner(action.AddNewlineAbove),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(prev(kit.Char(' '))),
			},
			{
				Name:      actAddNewlineBelow,
				DocString: "Add newline below",
				Run:       kit.Runner(action.AddNewlineBelow),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(next(kit.Char(' '))),
			},
			{
				Name:      actSelectRegister,
				DocString: "Select register",
				Run:       kit.Continuation(selectRegisterAction),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Char('"')),
			},
		},
	}
	mod.Labels = []command.PrefixLabel{
		kit.Label(
			"Match", kit.Char('m'), command.DocNormalModes,
		),
		kit.Label(
			"Match around", m(kit.Char('a')), command.DocNormalModes,
		),
		kit.Label(
			"Match inside", m(kit.Char('i')), command.DocNormalModes,
		),
	}
	for _, e := range textObjectEntries {
		mod.Labels = append(mod.Labels,
			kit.Label(e.label,
				m(append(kit.Char('a'), kit.Char(e.char)...)),
				command.DocNormalModes),
			kit.Label(e.label,
				m(append(kit.Char('i'), kit.Char(e.char)...)),
				command.DocNormalModes),
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
