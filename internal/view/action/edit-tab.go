package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// InsertTab inserts one indentation unit (tab or spaces) at each cursor
func InsertTab(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
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
		changes = append(changes, core.TextChange(pos, pos, tab))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// SmartTab inserts a tab when all cursors have only whitespace to their left;
// otherwise jumps to the next snippet tabstop or parent node end (no-op when
// those subsystems are absent)
func SmartTab(e *view.Editor) {
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
	allWhitespace := true
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
		left, err := text.Slice(lineStart, cursor)
		if err != nil {
			continue
		}
		for _, ch := range left.String() {
			if ch != ' ' && ch != '\t' {
				allWhitespace = false
				break
			}
		}
		if !allWhitespace {
			break
		}
	}
	if !allWhitespace {
		return
	}
	InsertTab(e)
}
