package editing

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
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
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
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
	if sel, err := core.NewSelection(ranges, sel.PrimaryIndex()); err == nil {
		doc.SetSelectionFor(v.ID(), sel)
	}
}

func syntaxMatchBrackets(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
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
	if newSel, err := core.NewSelection(
		ranges, sel.PrimaryIndex(),
	); err == nil {
		doc.SetSelectionFor(v.ID(), newSel)
	}
}

// smartTab indents inside leading whitespace, and otherwise moves each cursor
// past the enclosing syntax node; a literal tab is Shift+Tab
func smartTab(e *view.Editor) {
	if action.InLeadingWhitespace(e) {
		action.InsertTab(e)
		return
	}
	syntaxMoveParentNodeEnd(e)
}

func syntaxMoveParentNodeEnd(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	text := doc.Text()
	src := text.String()
	lang := doc.Lang()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changed := false
	for i, r := range ranges {
		end, ok := syntax.ParentNodeEnd(syntax.SelectionArgs{
			Text:   src,
			Lang:   lang,
			Cursor: r.Cursor(text),
		})
		if !ok {
			continue
		}
		nr := r.PutCursor(text, end, false)
		if nr != r {
			ranges[i] = nr
			changed = true
		}
	}
	if !changed {
		return
	}
	if newSel, err := core.NewSelection(
		ranges, sel.PrimaryIndex(),
	); err == nil {
		doc.SetSelectionFor(v.ID(), newSel)
	}
}
