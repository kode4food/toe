package editing

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func textObjectAction(around bool) command.KeyAction {
	inside := !around
	return func(_ *view.Editor) command.Continuation {
		return command.ReadChar(func(
			e *view.Editor, ch rune,
		) command.Continuation {
			if !syntaxTextObjectSelect(e, ch, inside) {
				if inside {
					action.SelectTextObjectInside(e, ch)
				} else {
					action.SelectTextObjectAround(e, ch)
				}
			}
			return nil
		})
	}
}

func syntaxTextObjectSelect(e *view.Editor, ch rune, inside bool) bool {
	v := e.FocusedView()
	if v == nil {
		return false
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return false
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changed := false
	for i, r := range ranges {
		res, ok := syntax.FindTextObject(syntax.FindTextObjectArgs{
			Text:   text.String(),
			Lang:   doc.Lang(),
			Cursor: r.Cursor(text),
			Char:   ch,
			Inside: inside,
		})
		if !ok {
			continue
		}
		nr := core.Range{Anchor: res.From, Head: res.To}
		if nr == r {
			continue
		}
		ranges[i] = nr
		changed = true
	}
	if !changed {
		return false
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return false
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return true
}
