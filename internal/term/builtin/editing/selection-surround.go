package editing

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

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

// syntaxSurroundPos uses Tree-sitter for surrounding brackets, falling back
// to plaintext for each range it cannot handle
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
