package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

type (
	jumpResolver func(*view.View) (view.DocumentId, int, bool)
	rangeMover   func(core.Rope, core.Range) core.Range
)

const statusNoSelectionsKey i18n.Key = "status.noSelections"

// GotoLineEndNewline moves each cursor to the end of its current line,
// landing on the newline character (for use in insert mode)
func GotoLineEndNewline(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		if lineEnd, err := doc.LineEndCharIndex(line); err == nil {
			return r.PutCursor(doc, lineEnd, false)
		}
		return r
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
		if lineEnd, err := doc.LineEndCharIndex(line); err == nil {
			return r.PutCursor(doc, lineEnd, true)
		}
		return r
	})
}

// SaveSelection pushes the current cursor position to the view's jump list
func SaveSelection(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	v.PushJump(v.DocID(), sel.Primary().Cursor(text), sel)
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
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	sel := doc.SelectionFor(v.ID())
	if len(sel.Ranges()) == 1 {
		e.SetStatusMsg(i18n.Text(statusNoSelectionsKey))
		return
	}
	if newSel, err := sel.Remove(sel.PrimaryIndex()); err == nil {
		doc.SetSelectionFor(v.ID(), newSel)
	}
}

// MergeSelections merges all selection ranges into one spanning range
func MergeSelections(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	sel := doc.SelectionFor(v.ID())
	doc.SetSelectionFor(v.ID(), sel.MergeRanges())
}

// MergeConsecutive merges overlapping or adjacent selection ranges
func MergeConsecutive(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
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
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
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

// ApplySelection applies sel to the focused document as a selection-only
// transaction, so cursor moves driven by input (mouse, jumps) go through the
// same insert-group and history bookkeeping as an edit
func ApplySelection(e *view.Editor, sel core.Selection) {
	if doc := e.FocusedDocument(); doc != nil {
		_ = e.Apply(core.NewTransaction(doc.Text()).WithSelection(sel))
	}
}

func jumpTo(e *view.Editor, fn jumpResolver) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	docID, pos, ok := fn(v)
	if !ok {
		return
	}
	if docID != v.DocID() && !e.SwitchBuffer(docID) {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
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
		lr, err := r.LineSpan(text)
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

func selectionIsLinewise(text core.Rope, sel core.Selection) bool {
	nLines := text.LenLines()
	for _, r := range sel.Ranges() {
		lr, err := r.LineSpan(text)
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

func applyMove(e *view.Editor, fn rangeMover) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = fn(text, r)
	}
	if newSel, err := core.NewSelection(
		ranges, sel.PrimaryIndex(),
	); err == nil {
		doc.SetSelectionFor(v.ID(), newSel)
	}
}
