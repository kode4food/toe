package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// Indent inserts one indentation unit at the start of each selected line
func Indent(e *view.Editor) {
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
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	unit := doc.IndentStyle().AsStr()
	n := max(e.Count(), 1)
	indent := strings.Repeat(unit, n)

	style := doc.IndentStyle()
	indentWidth := style.IndentWidth(doc.TabWidth())
	lines := selectionLines(text, sel)
	changes := make([]core.Change, 0, len(lines))
	for _, line := range lines {
		lineRope, err := text.Line(line)
		if err != nil {
			continue
		}
		if isBlankLine(lineRope.String()) {
			continue
		}
		pos, err := text.LineToChar(line)
		if err != nil {
			continue
		}
		ins := indent
		if !style.IsTabs() {
			// Find the column of the first non-whitespace char and
			// align to the next indent stop
			firstNonWS := 0
			for _, ch := range lineRope.String() {
				if ch != ' ' && ch != '\t' {
					break
				}
				firstNonWS++
			}
			offset := firstNonWS % indentWidth
			if offset > 0 && offset < len(ins) {
				ins = ins[offset:]
			}
		}
		changes = append(changes, core.TextChange(pos, pos, ins))
	}
	applyWithSelMap(e, text, sel, changes)
}

// Unindent removes one indentation unit from the start of each selected line
func Unindent(e *view.Editor) {
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
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	n := max(e.Count(), 1)
	tabWidth := doc.TabWidth()
	indentWidth := n * doc.IndentStyle().IndentWidth(tabWidth)

	lines := selectionLines(text, sel)
	changes := make([]core.Change, 0, len(lines))
	for _, line := range lines {
		lineRope, err := text.Line(line)
		if err != nil {
			continue
		}
		lineStr := lineRope.String()
		width := 0
		pos := 0
		for _, ch := range lineStr {
			switch ch {
			case ' ':
				width++
			case '\t':
				width = (width/tabWidth + 1) * tabWidth
			default:
				goto doneUnindent
			}
			pos++
			if width >= indentWidth {
				goto doneUnindent
			}
		}
	doneUnindent:
		if pos == 0 {
			continue
		}
		start, err := text.LineToChar(line)
		if err != nil {
			continue
		}
		changes = append(changes, core.DeleteChange(start, start+pos))
	}
	applyWithSelMap(e, text, sel, changes)
}

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

func applyWithSelMap(
	e *view.Editor, text core.Rope, sel core.Selection, changes []core.Change,
) {
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
	e.SetMode(view.ModeNormal)
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}
