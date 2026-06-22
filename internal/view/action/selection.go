package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// GotoLineEndNewline moves each cursor to the end of its current line,
// landing on the newline character (for use in insert mode)
func GotoLineEndNewline(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, lineEnd, false)
	})
}

// ExtendToLineEndNewline extends each selection to the end of its current line,
// landing on the newline character
func ExtendToLineEndNewline(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, lineEnd, true)
	})
}

// SaveSelection pushes the current cursor position to the view's jump list
func SaveSelection(e *view.Editor) {
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
	v.PushJump(v.DocID(), sel.Primary().Cursor(text))
}

// CommitUndoCheckpoint explicitly commits any pending insert-mode changes to
// history, creating an undo boundary mid-session
func CommitUndoCheckpoint(e *view.Editor) {
	e.CommitInsertHistory()
}

// JumpBackward navigates to the previous position in the view's jump list
func JumpBackward(e *view.Editor) {
	jumpTo(e, (*view.View).JumpBackward)
}

// JumpForward navigates to the next position in the view's jump list
func JumpForward(e *view.Editor) {
	jumpTo(e, (*view.View).JumpForward)
}

// RemovePrimarySelection removes the primary selection range. If only one
// range exists, the command is a no-op
func RemovePrimarySelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	if len(sel.Ranges()) == 1 {
		e.SetStatusMsg("no selections remaining")
		return
	}
	newSel, err := sel.Remove(sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// MergeSelections merges all selection ranges into one spanning range
func MergeSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	doc.SetSelectionFor(v.ID(), sel.MergeRanges())
}

// MergeConsecutive merges overlapping or adjacent selection ranges
func MergeConsecutive(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	doc.SetSelectionFor(v.ID(), sel.MergeConsecutiveRanges())
}

// EnsureForward forces all selection ranges to have a forward
// direction (anchor <= head)
func EnsureForward(e *view.Editor) {
	applyMove(e, func(_ core.Rope, r core.Range) core.Range {
		return r.WithDirection(core.DirectionForward)
	})
}

// GotoLastModification moves each cursor to the position of the most recent
// committed change in the current document
func GotoLastModification(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	pos := doc.LastEditPos()
	text := doc.Text()
	extend := e.Mode() == view.ModeSelect
	SaveSelection(e)
	newSel, err := core.NewSelection(
		[]core.Range{core.PointRange(pos).PutCursor(text, pos, extend)},
		0,
	)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

func jumpTo(e *view.Editor, fn func(*view.View) (view.DocumentId, int, bool)) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	_, pos, ok := fn(v)
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	newSel, err := core.NewSelection([]core.Range{core.PointRange(pos)}, 0)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

func selectionLines(text core.Rope, sel core.Selection) []int {
	seen := map[int]bool{}
	var lines []int
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		for l := lr.From; l <= lr.To; l++ {
			if !seen[l] {
				seen[l] = true
				lines = append(lines, l)
			}
		}
	}
	return lines
}

func isBlankLine(s string) bool {
	for _, ch := range s {
		if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n' {
			return false
		}
	}
	return true
}

// selectionIsLinewise returns true when every range in sel spans at least
// two lines and starts/ends exactly on line boundaries (start of a line and
// start of the next line, i.e., covers whole lines including newlines)
func selectionIsLinewise(text core.Rope, sel core.Selection) bool {
	nLines := text.LenLines()
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			return false
		}
		startLine, endLine := lr.From, lr.To
		if endLine <= startLine {
			return false
		}
		start, err := text.LineToChar(startLine)
		if err != nil {
			return false
		}
		endLineNext := min(endLine+1, nLines)
		end, err := text.LineToChar(endLineNext)
		if err != nil {
			return false
		}
		if r.From() != start || r.To() != end {
			return false
		}
	}
	return true
}

func applyMove(e *view.Editor, fn func(core.Rope, core.Range) core.Range) {
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
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = fn(text, r)
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}
