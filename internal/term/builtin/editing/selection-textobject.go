package editing

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func textObjectAction(around bool) command.KeyAction {
	h, inside := "ma", false
	if !around {
		h, inside = "mi", true
	}
	return func(e *view.Editor) command.Continuation {
		e.SetHint(h + " ...")
		return func(e *view.Editor, k command.KeyEvent) command.Continuation {
			if k.Code.Char != 0 && k.Mods == command.ModNone {
				ch := k.Code.Char
				if !syntaxTextObjectSelect(e, ch, inside) {
					if inside {
						action.SelectTextObjectInside(e, ch)
					} else {
						action.SelectTextObjectAround(e, ch)
					}
				}
			}
			e.SetHint("")
			return nil
		}
	}
}

func syntaxTextObjectSelect(e *view.Editor, ch rune, inside bool) bool {
	v, ok := e.FocusedView()
	if !ok {
		return false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return false
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
		return false
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return false
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return true
}
