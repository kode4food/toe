package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// NormalMode returns to normal mode, running cleanup like stripping blank-line
// auto-indent and moving the cursor off the past-end position
func NormalMode(e *view.Editor) {
	if e.Mode() == view.ModeNormal {
		return
	}
	v, ok := e.FocusedView()
	if !ok {
		e.SetMode(view.ModeNormal)
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		e.SetMode(view.ModeNormal)
		return
	}
	e.SetMode(view.ModeNormal)

	// Strip trailing whitespace on a blank line that auto-indent may have left
	tryRestoreIndent(e, doc, v)

	if doc.RestoreCursor() {
		doc.SetRestoreCursor(false)

		text := doc.Text()
		sel := doc.SelectionFor(v.ID())
		ranges := sel.Ranges()
		for i, r := range ranges {
			head := r.To()
			if r.Head > r.Anchor {
				head = core.PrevGraphemeBoundary(text, head)
			}
			ranges[i] = core.NewRange(r.From(), head)
		}
		newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
		if err == nil {
			doc.SetSelectionFor(v.ID(), newSel)
		}
	}

	e.CommitInsertHistory()
}

// ExitSelectMode returns to normal mode from select mode without performing
// the insert-mode cleanup that NormalMode does
func ExitSelectMode(e *view.Editor) {
	if e.Mode() == view.ModeSelect {
		e.SetMode(view.ModeNormal)
	}
}

// SelectMode enters select mode, adjusting any empty end-of-document selection
// to be one grapheme wide so it is always visible
func SelectMode(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	nChars := text.LenChars()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changed := false
	for i, r := range ranges {
		if r.Empty() && r.Head == nChars && nChars > 0 {
			prev := core.PrevGraphemeBoundary(text, r.Anchor)
			ranges[i] = core.NewRange(prev, r.Head)
			changed = true
		}
	}
	if changed {
		newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
		if err == nil {
			doc.SetSelectionFor(v.ID(), newSel)
		}
	}
	e.SetMode(view.ModeSelect)
}

// InsertMode enters insert mode with cursors at the start of each selection
func InsertMode(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		// Cursor lands at the start (from) of the selection
		ranges[i] = core.PointRange(r.From())
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
	e.SetMode(view.ModeInsert)
}

// AppendMode enters insert mode with cursors one grapheme past the end of
// each selection
func AppendMode(e *view.Editor) {
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
		ins := r.To()
		if r.Empty() {
			ins = core.NextGraphemeBoundary(text, ins)
		}
		head := core.NextGraphemeBoundary(text, ins)
		if head == ins {
			ranges[i] = core.PointRange(ins)
		} else {
			ranges[i] = core.NewRange(r.From(), head)
		}
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
	doc.SetRestoreCursor(true)
	e.SetMode(view.ModeInsert)
}

// InsertAtLineStart moves each cursor to the first non-whitespace char of
// its current line and enters insert mode
func InsertAtLineStart(e *view.Editor) {
	MoveLineNonWhitespace(e)
	InsertMode(e)
}

// AppendToLine moves each cursor to the end of its current line and enters
// insert mode (append after last char)
func AppendToLine(e *view.Editor) {
	MoveLineEnd(e)
	AppendMode(e)
}

func tryRestoreIndent(_ *view.Editor, doc *view.Document, v *view.View) {
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary()
	cursor := primary.Cursor(text)

	line, err := text.CharToLine(cursor)
	if err != nil {
		return
	}
	lineEnd, err := text.LineEndCharIndex(line)
	if err != nil {
		return
	}
	if cursor != lineEnd {
		return
	}
	lineStart, err := text.LineToChar(line)
	if err != nil {
		return
	}
	// Check that the line contains only whitespace
	onlySpace := true
	for pos := lineStart; pos < lineEnd; pos++ {
		ch, err := text.CharAt(pos)
		if err != nil || (ch != ' ' && ch != '\t') {
			onlySpace = false
			break
		}
	}
	if !onlySpace || lineStart == lineEnd {
		return
	}
	// Delete the whitespace-only content of this line
	cs, err := core.NewChangeSetFromChanges(text, []core.Change{
		core.DeleteChange(lineStart, lineEnd),
	})
	if err != nil {
		return
	}
	_ = doc.Apply(core.Transaction{}.WithChanges(cs), v.ID())
}
