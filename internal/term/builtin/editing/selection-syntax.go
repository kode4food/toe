package editing

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
)

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
