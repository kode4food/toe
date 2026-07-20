package action

import (
	"strings"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// AlignSelections inserts spaces before each cursor so all cursors sit at the
// same visual column (the maximum column among all cursors). Only operates
// when there are multiple selection ranges, all on different lines
func AlignSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	if len(ranges) < 2 {
		return
	}

	cols := make([]int, len(ranges))
	maxCol := 0
	for i, r := range ranges {
		pos := r.Cursor(text)
		line, err := text.CharToLine(pos)
		if err != nil {
			return
		}
		lineStart, err := text.LineToChar(line)
		if err != nil {
			return
		}
		col := pos - lineStart
		cols[i] = col
		if col > maxCol {
			maxCol = col
		}
	}

	changes := make([]core.Change, 0, len(ranges))
	for i, r := range ranges {
		pad := maxCol - cols[i]
		if pad <= 0 {
			continue
		}
		pos := r.Cursor(text)
		changes = append(changes,
			core.TextChange(pos, pos, strings.Repeat(" ", pad)))
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

// ReplaceChar replaces every grapheme in each selection with ch and exits
// select mode
func ReplaceChar(e *view.Editor, ch rune) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	replacement := string(ch)

	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		if r.Empty() {
			continue
		}
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		var b strings.Builder
		for range utf8.RuneCountInString(frag) {
			b.WriteString(replacement)
		}
		changes = append(changes, core.TextChange(r.From(), r.To(), b.String()))
	}
	applyChangesFrom(e, applyChangesFromArgs{
		text:    text,
		sel:     sel,
		ranges:  ranges,
		changes: changes,
	})
}
