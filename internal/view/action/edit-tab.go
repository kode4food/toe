package action

import (
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// InLeadingWhitespace reports whether every cursor has only whitespace between
// it and the start of its line
func InLeadingWhitespace(e *view.Editor) bool {
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
	for _, r := range sel.Ranges() {
		cursor := r.Cursor(text)
		lineNum, err := text.CharToLine(cursor)
		if err != nil {
			continue
		}
		lineStart, err := text.LineToChar(lineNum)
		if err != nil {
			continue
		}
		left, err := text.Slice(core.Span{From: lineStart, To: cursor})
		if err != nil {
			continue
		}
		if strings.ContainsFunc(left.String(), notSpace) {
			return false
		}
	}
	return true
}

// InsertTab inserts one indentation unit (tab or spaces) at each cursor
func InsertTab(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if doc.ReadOnly() {
		return
	}
	tab := doc.IndentStyle().AsStr()
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	changes := make([]core.Change, 0, len(sel.Ranges()))
	seen := map[int]bool{}
	for _, r := range sel.Ranges() {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(core.Span{
			From: pos,
			To:   pos,
		}, tab))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	if newSel, err := sel.Map(cs); err == nil {
		_ = e.Apply(
			core.NewTransaction(text).WithChanges(cs).WithSelection(newSel),
		)
	}
}

func notSpace(r rune) bool {
	return !unicode.IsSpace(r)
}
