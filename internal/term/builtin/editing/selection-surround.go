package editing

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func surroundAddAction(_ *view.Editor) command.Continuation {
	return command.ReadChar(func(e *view.Editor, ch rune) command.Continuation {
		action.SurroundAdd(e, ch)
		return nil
	})
}

func surroundReplaceAction(_ *view.Editor) command.Continuation {
	return command.ReadChar(func(
		_ *view.Editor, from rune,
	) command.Continuation {
		return command.ReadChar(func(
			e *view.Editor, to rune,
		) command.Continuation {
			if positions, ok := syntaxSurroundPos(e, from); ok {
				if doc := e.FocusedDocument(); doc != nil {
					action.SurroundReplaceAt(
						e, doc.Text(), positions, to,
					)
					return nil
				}
			}
			action.SurroundReplace(action.SurroundReplaceArgs{
				Editor:  e,
				Current: from,
				Wanted:  to,
			})
			return nil
		})
	})
}

func surroundDeleteAction(_ *view.Editor) command.Continuation {
	return command.ReadChar(func(e *view.Editor, ch rune) command.Continuation {
		if positions, ok := syntaxSurroundPos(e, ch); ok {
			if doc := e.FocusedDocument(); doc != nil {
				action.SurroundDeleteAt(e, doc.Text(), positions)
				return nil
			}
		}
		action.SurroundDelete(e, ch)
		return nil
	})
}

func syntaxSurroundPos(e *view.Editor, ch rune) ([]int, bool) {
	v := e.FocusedView()
	if v == nil {
		return nil, false
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return nil, false
	}
	text := doc.Text()
	src := text.String()
	lang := doc.Lang()
	sel := doc.SelectionFor(v.ID())
	skip := e.CountOr(1)

	var positions []int
	for _, r := range sel.Ranges() {
		cursor := r.Cursor(text)
		var res syntax.Range
		var found bool
		if ch == 'm' {
			res, found = syntax.FindSurroundPair(syntax.FindSurroundPairArgs{
				Source: core.Source{Text: src, Lang: lang},
				Cursor: cursor,
				Skip:   skip,
			})
		} else {
			res, found = syntax.FindSurroundPairFor(
				syntax.FindSurroundPairForArgs{
					Text:   src,
					Lang:   lang,
					Cursor: cursor,
					Char:   ch,
					Skip:   skip,
				},
			)
		}
		if !found {
			var pair core.Range
			var err error
			if ch == 'm' {
				pair, err = core.FindNthClosestPairsPos(text, r, skip)
			} else {
				pair, err = core.FindNthPairsPos(text, ch, r, skip)
			}
			if err != nil {
				return nil, false
			}
			res = syntax.Range{From: pair.From(), To: pair.To()}
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
