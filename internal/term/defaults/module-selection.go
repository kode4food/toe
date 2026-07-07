package defaults

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type textObjectEntry struct {
	ch    rune
	label string
}

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
	actAddNewlineAbove            = "add_newline_above"
	actAddNewlineBelow            = "add_newline_below"
	actSelectRegister             = "select_register"
)

var (
	textObjectEntries = []textObjectEntry{
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

	textObjectCharIDs = map[rune]string{
		'(':  "lparen",
		')':  "rparen",
		'{':  "lbrace",
		'}':  "rbrace",
		'[':  "lbracket",
		']':  "rbracket",
		'<':  "langle",
		'>':  "rangle",
		'"':  "dquote",
		'\'': "squote",
		'`':  "backtick",
		'|':  "pipe",
	}
)

func selectionModule(model ui.Model) command.Module {
	m := prefixed(char('m'))
	prev := prefixed(char('['))
	next := prefixed(char(']'))

	mod := command.Module{
		Commands: []command.Command{
			{
				Name:      actCopyOnNextLine,
				DocString: "Copy selection on next line",
				Run:       Runner(action.CopyOnNextLine),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('C')),
			},
			{
				Name:      actCopyOnPrevLine,
				DocString: "Copy selection on previous line",
				Run:       Runner(action.CopyOnPrevLine),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('C')),
			},
			{
				Name:      actSelectWithinRegex,
				DocString: "Select all regex matches inside selections",
				Run: Continuation(model.RegexAction(
					"select:", action.SelectWithinRegex,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('s')),
			},
			{
				Name:      actSplitSelectionByRegex,
				DocString: "Split selections on regex matches",
				Run: Continuation(model.RegexAction(
					"split:", action.SplitSelectionByRegex,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('S')),
			},
			{
				Name:      actKeepSelectionsMatching,
				DocString: "Keep selections matching regex",
				Run: Continuation(model.RegexAction(
					"keep:", action.KeepSelectionsMatching,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('K')),
			},
			{
				Name:      actRemoveSelectionsMatching,
				DocString: "Remove selections matching regex",
				Run: Continuation(model.RegexAction(
					"remove:", action.RemoveSelectionsMatching,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(alt('K')),
			},
			{
				Name:      actSplitSelectionOnNewline,
				DocString: "Split selection on newlines",
				Run:       Runner(action.SplitSelectionOnNewline),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('s')),
			},
			{
				Name:      actMergeSelections,
				DocString: "Merge selections",
				Run:       Runner(action.MergeSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('-')),
			},
			{
				Name:      actMergeConsecutiveSelections,
				DocString: "Merge consecutive selections",
				Run:       Runner(action.MergeConsecutive),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('_')),
			},
			{
				Name:      actCollapseSelection,
				DocString: "Collapse selection into single cursor",
				Run:       Runner(action.CollapseSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(';')),
			},
			{
				Name:      actFlipSelections,
				DocString: "Flip selection cursor and anchor",
				Run:       Runner(action.FlipSelections),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt(';')),
			},
			{
				Name:      actSelectAll,
				DocString: "Select whole document",
				Run:       Runner(action.SelectAll),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('%')),
			},
			{
				Name:      actSelectLineAbove,
				DocString: "Select line above",
				Run:       Runner(action.SelectLineAbove),
				Signature: sig(),
			},
			{
				Name:      actSelectLineBelow,
				DocString: "Select line below",
				Run:       Runner(action.SelectLineBelow),
				Signature: sig(),
			},
			{
				Name: actExtendLineBelow,
				DocString: "Select current line, if already selected, extend" +
					" to next line",
				Run:   Runner(action.ExtendLineBelow),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('x')),
			},
			{
				Name:      actExtendToLineBounds,
				DocString: "Extend selection to line bounds",
				Run:       Runner(action.ExtendToLineBounds),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('X')),
			},
			{
				Name:      actShrinkToLineBounds,
				DocString: "Shrink selection to line bounds",
				Run:       Runner(action.ShrinkToLineBounds),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('x')),
			},
			{
				Name:      actExpandSelection,
				DocString: "Expand selection to syntax node",
				Run:       Runner(syntaxExpandSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('o')),
			},
			{
				Name:      actShrinkSelection,
				DocString: "Shrink selection to syntax node",
				Run:       Runner(syntaxShrinkSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('i')),
			},
			{
				Name:      actKeepPrimarySelection,
				DocString: "Keep primary selection",
				Run:       Runner(action.KeepPrimarySelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(',')),
			},
			{
				Name:      actRemovePrimarySelection,
				DocString: "Remove primary selection",
				Run:       Runner(action.RemovePrimarySelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt(',')),
			},
			{
				Name:      actMatchBrackets,
				DocString: "Goto matching bracket",
				Run:       Runner(syntaxMatchBrackets),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('m'))),
			},
			{
				Name:      actSurroundAdd,
				DocString: "Surround add",
				Run:       Continuation(surroundAddAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('s'))),
			},
			{
				Name:      actSurroundReplace,
				DocString: "Surround replace",
				Run:       Continuation(surroundReplaceAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('r'))),
			},
			{
				Name:      actSurroundDelete,
				DocString: "Surround delete",
				Run:       Continuation(surroundDeleteAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(m(char('d'))),
			},
			{
				Name:      actAddNewlineAbove,
				DocString: "Add newline above",
				Run:       Runner(action.AddNewlineAbove),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(prev(char(' '))),
			},
			{
				Name:      actAddNewlineBelow,
				DocString: "Add newline below",
				Run:       Runner(action.AddNewlineBelow),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(next(char(' '))),
			},
			{
				Name:      actSelectRegister,
				DocString: "Select register",
				Run:       Continuation(selectRegisterAction),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('"')),
			},
		},
	}
	mod.Commands = append(mod.Commands, textObjectCommands(m)...)
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
				to := k.Code.Char
				if positions, ok := syntaxSurroundPos(e, from); ok {
					if doc, dOK := e.FocusedDocument(); dOK {
						action.SurroundReplaceAt(e, doc.Text(), positions, to)
						e.SetHint("")
						return nil
					}
				}
				action.SurroundReplace(e, from, to)
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
			ch := k.Code.Char
			if positions, ok := syntaxSurroundPos(e, ch); ok {
				if doc, dOK := e.FocusedDocument(); dOK {
					action.SurroundDeleteAt(e, doc.Text(), positions)
					e.SetHint("")
					return nil
				}
			}
			action.SurroundDelete(e, ch)
		}
		e.SetHint("")
		return nil
	}
}

// syntaxSurroundPos finds surrounding bracket positions for all selection
// ranges, using Tree-sitter for structural brackets and falling back to
// plaintext for each range that tree-sitter cannot handle
func syntaxSurroundPos(e *view.Editor, ch rune) ([]int, bool) {
	v, ok := e.FocusedView()
	if !ok {
		return nil, false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil, false
	}
	text := doc.Text()
	src := text.String()
	lang := doc.Lang()
	sel := doc.SelectionFor(v.ID())
	skip := max(e.Count(), 1)

	var positions []int
	for _, r := range sel.Ranges() {
		cursor := r.Cursor(text)
		var res syntax.Range
		var found bool
		if ch == 'm' {
			res, found = syntax.FindSurroundPair(src, lang, cursor, skip)
		} else {
			res, found = syntax.FindSurroundPairFor(
				src, lang, cursor, ch, skip,
			)
		}
		if !found {
			var coreFrom, coreTo int
			var err error
			if ch == 'm' {
				coreFrom, coreTo, err = core.FindNthClosestPairsPos(
					text, r, skip,
				)
			} else {
				coreFrom, coreTo, err = core.FindNthPairsPos(text, ch, r, skip)
			}
			if err != nil {
				return nil, false
			}
			res = syntax.Range{
				From: min(coreFrom, coreTo),
				To:   max(coreFrom, coreTo),
			}
		}
		anchor, head := min(res.From, res.To), max(res.From, res.To)
		for _, p := range positions {
			if p == anchor || p == head {
				return nil, false
			}
		}
		positions = append(positions, anchor, head)
	}
	return positions, true
}

func textObjectCommands(
	m func(...[]command.KeyEvent) []command.KeyEvent,
) []command.Command {
	maSeq := func(ch rune) []command.KeyEvent {
		return append(m(char('a')), char(ch)...)
	}
	miSeq := func(ch rune) []command.KeyEvent {
		return append(m(char('i')), char(ch)...)
	}
	cmds := make([]command.Command, 0, len(textObjectEntries)*2)
	for _, e := range textObjectEntries {
		for _, inside := range []bool{false, true} {
			ch, lbl := e.ch, e.label
			dir, pfx, seq := "around", "select_textobject_around_", maSeq
			if inside {
				dir, pfx, seq = "inside", "select_textobject_inside_", miSeq
			}
			id, ok := textObjectCharIDs[ch]
			if !ok {
				id = string(ch)
			}
			cmds = append(cmds, command.Command{
				Name:      pfx + id,
				DocString: "Select " + dir + " " + lbl,
				Run: Runner(func(ed *view.Editor) {
					if syntax.IsTextObjectChar(ch) {
						syntaxTextObjectSelect(ed, ch, inside)
					} else if inside {
						action.SelectTextObjectInside(ed, ch)
					} else {
						action.SelectTextObjectAround(ed, ch)
					}
				}),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(seq(ch)),
			})
		}
	}
	return cmds
}

func syntaxTextObjectSelect(e *view.Editor, ch rune, inside bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changed := false
	for i, r := range ranges {
		res, ok := syntax.FindTextObject(
			text.String(), doc.Lang(), r.Cursor(text), ch, inside,
		)
		if !ok {
			continue
		}
		nr := core.NewRange(res.From, res.To)
		if nr == r {
			continue
		}
		ranges[i] = nr
		changed = true
	}
	if !changed {
		return
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

func syntaxExpandSelection(e *view.Editor) {
	syntaxSelect(e, syntax.ExpandSelection)
}

func syntaxShrinkSelection(e *view.Editor) {
	syntaxSelect(e, syntax.ShrinkSelection)
}

func syntaxSelect(
	e *view.Editor, fn func(syntax.SelectionArgs) (syntax.Range, bool),
) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	src := text.String()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changed := false
	for i, r := range ranges {
		res, ok := fn(syntax.SelectionArgs{
			Text:   src,
			Lang:   doc.Lang(),
			Cursor: r.Cursor(text),
			Range: syntax.Range{
				From: r.From(),
				To:   r.To(),
			},
		})
		if !ok {
			continue
		}
		ranges[i] = core.NewRange(res.From, res.To).WithDirection(r.Direction())
		changed = changed || ranges[i] != r
	}
	if !changed {
		return
	}
	sel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
}

func syntaxMatchBrackets(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	src := text.String()
	lang := doc.Lang()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changed := false
	for i, r := range ranges {
		pos := r.Cursor(text)
		match, ok := syntax.FindMatchingBracket(src, lang, pos)
		if !ok {
			match, ok = core.FindMatchingBracket(text, pos)
		}
		if !ok {
			continue
		}
		nr := r.PutCursor(text, match, false)
		if nr != r {
			ranges[i] = nr
			changed = true
		}
	}
	if !changed {
		return
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}
